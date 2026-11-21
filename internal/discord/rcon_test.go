package discord_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestDefaultConsoleExecutor(t *testing.T) {
	ctx := context.Background()

	t.Run("valid target servers", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{}
		for _, srv := range []string{"proxy", "lobby", "survival"} {
			out, err := exec.Execute(ctx, srv, "list")
			if err != nil || out == "" {
				t.Fatalf("unexpected failure for server %s: out=%s err=%v", srv, out, err)
			}
		}
	})

	t.Run("invalid target server", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{}
		_, err := exec.Execute(ctx, "nonexistent", "list")
		if err == nil {
			t.Fatalf("expected error for invalid server, got nil")
		}
	})

	t.Run("mock error output", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{
			MockErr: errors.New("connection refused"),
		}
		_, err := exec.Execute(ctx, "lobby", "say hello")
		if err == nil || err.Error() != "connection refused" {
			t.Fatalf("expected connection refused error, got: %v", err)
		}
	})

	t.Run("custom mock output", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{
			MockOutput: "There are 4 players online",
		}
		out, err := exec.Execute(ctx, "proxy", "glist")
		if err != nil || out != "There are 4 players online" {
			t.Fatalf("unexpected mock output: %s (err: %v)", out, err)
		}
	})
}

func startMockRconServer(t *testing.T, expectedPassword string, commandResponses map[string]string) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on loopback: %v", err)
	}

	done := make(chan struct{})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}

			go func(c net.Conn) {
				defer c.Close()
				authenticated := false

				for {
					var length int32
					if err := binary.Read(c, binary.LittleEndian, &length); err != nil {
						return
					}
					payload := make([]byte, length)
					if _, err := io.ReadFull(c, payload); err != nil {
						return
					}

					var reqID, pktType int32
					_ = binary.Read(bytes.NewReader(payload[0:4]), binary.LittleEndian, &reqID)
					_ = binary.Read(bytes.NewReader(payload[4:8]), binary.LittleEndian, &pktType)
					bodyBytes := payload[8:]
					if len(bodyBytes) >= 2 {
						bodyBytes = bodyBytes[:len(bodyBytes)-2]
					}
					body := string(bodyBytes)

					if pktType == 3 { // AUTH
						if body == expectedPassword {
							authenticated = true
							// Send auth success
							sendMockPacket(c, reqID, 2, "")
						} else {
							// Send auth failure
							sendMockPacket(c, -1, 2, "")
							return
						}
					} else if pktType == 2 { // EXEC
						if !authenticated {
							return
						}
						respBody, ok := commandResponses[body]
						if !ok {
							respBody = "Executed: " + body
						}
						sendMockPacket(c, reqID, 0, respBody)
					}
				}
			}(conn)
		}
	}()

	cleanup := func() {
		close(done)
		_ = ln.Close()
	}

	return ln.Addr().String(), cleanup
}

func sendMockPacket(w io.Writer, id, pktType int32, body string) {
	bodyBytes := []byte(body)
	length := int32(4 + 4 + len(bodyBytes) + 2) // #nosec G115
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, length)
	_ = binary.Write(buf, binary.LittleEndian, id)
	_ = binary.Write(buf, binary.LittleEndian, pktType)
	buf.Write(bodyBytes)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	_, _ = w.Write(buf.Bytes())
}

func TestNetworkRconClientAndExecutor(t *testing.T) {
	password := "supersecret123"
	responses := map[string]string{
		"list":               "There are 2 out of 50 players online: Steve, Alex",
		"ban Steve Griefing": "Steve has been banned.",
	}

	addr, cleanup := startMockRconServer(t, password, responses)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	t.Run("direct rcon client success", func(t *testing.T) {
		client, err := discord.DialRcon(ctx, addr, password, 2*time.Second)
		if err != nil {
			t.Fatalf("unexpected dial error: %v", err)
		}
		defer client.Close()

		out, err := client.Execute(ctx, "list")
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}
		if out != responses["list"] {
			t.Errorf("expected %q, got %q", responses["list"], out)
		}
	})

	t.Run("rcon client auth failure", func(t *testing.T) {
		_, err := discord.DialRcon(ctx, addr, "wrongpassword", 2*time.Second)
		if err == nil {
			t.Fatalf("expected error for wrong password, got nil")
		}
	})

	t.Run("network rcon executor success", func(t *testing.T) {
		targets := map[string]string{
			"lobby":    addr,
			"survival": addr,
		}
		executor := discord.NewNetworkRconExecutor(password, targets)

		out, err := executor.Execute(ctx, "lobby", "list")
		if err != nil {
			t.Fatalf("unexpected executor error: %v", err)
		}
		if out != responses["list"] {
			t.Errorf("expected %q, got %q", responses["list"], out)
		}

		outBan, err := executor.Execute(ctx, "survival", "ban Steve Griefing")
		if err != nil {
			t.Fatalf("unexpected ban error: %v", err)
		}
		if outBan != responses["ban Steve Griefing"] {
			t.Errorf("expected %q, got %q", responses["ban Steve Griefing"], outBan)
		}
	})

	t.Run("network rcon executor unknown target with fallback", func(t *testing.T) {
		fallback := &discord.DefaultConsoleExecutor{
			MockOutput: "Fallback response",
		}
		executor := &discord.NetworkRconExecutor{
			Targets:  map[string]string{"lobby": addr},
			Password: password,
			Fallback: fallback,
		}

		out, err := executor.Execute(ctx, "proxy", "alert reboot")
		if err != nil {
			t.Fatalf("unexpected error with fallback: %v", err)
		}
		if out != "Fallback response" {
			t.Errorf("expected fallback response, got %q", out)
		}
	})

	t.Run("network rcon executor unknown target without fallback", func(t *testing.T) {
		executor := &discord.NetworkRconExecutor{
			Targets:  map[string]string{"lobby": addr},
			Password: password,
		}

		_, err := executor.Execute(ctx, "unknown-server", "help")
		if err == nil {
			t.Fatalf("expected target not found error, got nil")
		}
	})

	t.Run("network rcon executor default targets and fallback on connection failure", func(t *testing.T) {
		execDefault := discord.NewNetworkRconExecutor("pwd123", nil)
		if len(execDefault.Targets) != 2 || execDefault.Targets["lobby"] == "" || execDefault.Targets["survival"] == "" {
			t.Errorf("expected default targets lobby and survival, got: %v", execDefault.Targets)
		}

		// Target connects to non-listening port with fallback
		fallback := &discord.DefaultConsoleExecutor{MockOutput: "fallback on conn error"}
		execFailedConn := &discord.NetworkRconExecutor{
			Targets:     map[string]string{"lobby": "127.0.0.1:54321"},
			Password:    "pwd",
			DialTimeout: 50 * time.Millisecond,
			Fallback:    fallback,
		}
		out, err := execFailedConn.Execute(ctx, "lobby", "list")
		if err != nil || out != "fallback on conn error" {
			t.Errorf("expected fallback on connection error, got out=%q err=%v", out, err)
		}

		// Without fallback
		execFailedConnNoFallback := &discord.NetworkRconExecutor{
			Targets:     map[string]string{"lobby": "127.0.0.1:54321"},
			Password:    "pwd",
			DialTimeout: 50 * time.Millisecond,
		}
		_, err = execFailedConnNoFallback.Execute(ctx, "lobby", "list")
		if err == nil {
			t.Errorf("expected connection error, got nil")
		}
	})

	t.Run("rcon client empty command output", func(t *testing.T) {
		emptyCmdResponses := map[string]string{
			"silent_command": "",
		}
		emptyAddr, emptyCleanup := startMockRconServer(t, password, emptyCmdResponses)
		defer emptyCleanup()

		client, err := discord.DialRcon(ctx, emptyAddr, password, 0)
		if err != nil {
			t.Fatalf("unexpected dial error: %v", err)
		}
		defer client.Close()

		out, err := client.Execute(ctx, "silent_command")
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}
		if out != "Command executed successfully (no output)." {
			t.Errorf("expected empty output placeholder, got %q", out)
		}
		// Test multiple close calls
		if err := client.Close(); err != nil {
			t.Errorf("expected clean close, got %v", err)
		}
		if err := client.Close(); err != nil {
			t.Errorf("expected idempotent close, got %v", err)
		}
	})

	t.Run("rcon packet encoding errors and decoding variations", func(t *testing.T) {
		// Test encode with oversized body
		oversized := make([]byte, 70000)
		_, err := discord.EncodeRconPacketForTest(1, 2, string(oversized))
		if err == nil {
			t.Errorf("expected error for oversized rcon payload, got nil")
		}

		// Test decode with length < 10
		shortBuf := new(bytes.Buffer)
		_ = binary.Write(shortBuf, binary.LittleEndian, int32(5))
		shortBuf.Write([]byte{0, 0, 0, 0, 0})
		_, err = discord.DecodeRconPacketForTest(shortBuf)
		if err == nil {
			t.Errorf("expected error for length < 10, got nil")
		}

		// Test decode with length > 65535
		hugeBuf := new(bytes.Buffer)
		_ = binary.Write(hugeBuf, binary.LittleEndian, int32(70000))
		_, err = discord.DecodeRconPacketForTest(hugeBuf)
		if err == nil {
			t.Errorf("expected error for length > 65535, got nil")
		}

		// Test decode with truncated payload
		truncatedBuf := new(bytes.Buffer)
		_ = binary.Write(truncatedBuf, binary.LittleEndian, int32(20))
		truncatedBuf.Write([]byte{1, 2, 3}) // only 3 bytes instead of 20
		_, err = discord.DecodeRconPacketForTest(truncatedBuf)
		if err == nil {
			t.Errorf("expected error for truncated payload, got nil")
		}

		// Test decode with single null terminator
		singleNullBuf := new(bytes.Buffer)
		singlePayload := new(bytes.Buffer)
		_ = binary.Write(singlePayload, binary.LittleEndian, int32(101))
		_ = binary.Write(singlePayload, binary.LittleEndian, int32(0))
		singlePayload.WriteString("hello")
		singlePayload.WriteByte(0x00)                                                    // only 1 null byte
		_ = binary.Write(singleNullBuf, binary.LittleEndian, int32(singlePayload.Len())) // #nosec G115
		singleNullBuf.Write(singlePayload.Bytes())
		pkt, err := discord.DecodeRconPacketForTest(singleNullBuf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pkt.Body != "hello" || pkt.ID != 101 {
			t.Errorf("expected body 'hello' and id 101, got body=%q id=%d", pkt.Body, pkt.ID)
		}

		// Test decode with no null terminator
		noNullBuf := new(bytes.Buffer)
		noNullPayload := new(bytes.Buffer)
		_ = binary.Write(noNullPayload, binary.LittleEndian, int32(102))
		_ = binary.Write(noNullPayload, binary.LittleEndian, int32(0))
		noNullPayload.WriteString("world")
		_ = binary.Write(noNullBuf, binary.LittleEndian, int32(noNullPayload.Len())) // #nosec G115
		noNullBuf.Write(noNullPayload.Bytes())
		pkt2, err := discord.DecodeRconPacketForTest(noNullBuf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pkt2.Body != "world" || pkt2.ID != 102 {
			t.Errorf("expected body 'world' and id 102, got body=%q id=%d", pkt2.Body, pkt2.ID)
		}
	})

	t.Run("rcon server responds with empty response before auth success", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()

			var length int32
			_ = binary.Read(conn, binary.LittleEndian, &length)
			payload := make([]byte, length)
			_, _ = io.ReadFull(conn, payload)

			var reqID int32
			_ = binary.Read(bytes.NewReader(payload[0:4]), binary.LittleEndian, &reqID)

			// Send empty response first (Type 0 = rconPacketResponse)
			sendMockPacket(conn, reqID, 0, "")
			// Then send auth response (Type 2 = rconPacketAuthResponse)
			sendMockPacket(conn, reqID, 2, "")
		}()

		client, err := discord.DialRcon(ctx, ln.Addr().String(), "secret", 500*time.Millisecond)
		if err != nil {
			t.Fatalf("expected successful dial with empty preceding packet, got err: %v", err)
		}
		_ = client.Close()
	})

	t.Run("rcon server closes connection following empty packet during auth", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()

			var length int32
			_ = binary.Read(conn, binary.LittleEndian, &length)
			payload := make([]byte, length)
			_, _ = io.ReadFull(conn, payload)

			var reqID int32
			_ = binary.Read(bytes.NewReader(payload[0:4]), binary.LittleEndian, &reqID)

			// Send empty response first and immediately close socket
			sendMockPacket(conn, reqID, 0, "")
		}()

		_, err = discord.DialRcon(ctx, ln.Addr().String(), "secret", 500*time.Millisecond)
		if err == nil {
			t.Fatalf("expected error when server closes after empty packet, got nil")
		}
	})

	t.Run("rcon server immediate disconnect on auth", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}()

		_, err = discord.DialRcon(ctx, ln.Addr().String(), "secret", 500*time.Millisecond)
		if err == nil {
			t.Fatalf("expected EOF error on immediate disconnect, got nil")
		}
	})

	t.Run("rcon execute on closed client connection", func(t *testing.T) {
		addr, cleanup := startMockRconServer(t, password, responses)
		defer cleanup()

		client, err := discord.DialRcon(ctx, addr, password, 0)
		if err != nil {
			t.Fatalf("unexpected dial error: %v", err)
		}
		_ = client.Close()

		_, err = client.Execute(ctx, "list")
		if err == nil {
			t.Fatalf("expected error executing on closed client, got nil")
		}
	})

	t.Run("rcon execute server disconnects before response", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()

			// Auth phase
			var length int32
			_ = binary.Read(conn, binary.LittleEndian, &length)
			payload := make([]byte, length)
			_, _ = io.ReadFull(conn, payload)

			var reqID int32
			_ = binary.Read(bytes.NewReader(payload[0:4]), binary.LittleEndian, &reqID)
			sendMockPacket(conn, reqID, 2, "")

			// Exec command read, then close without answering
			_ = binary.Read(conn, binary.LittleEndian, &length)
			payload = make([]byte, length)
			_, _ = io.ReadFull(conn, payload)
		}()

		client, err := discord.DialRcon(ctx, ln.Addr().String(), "secret", 500*time.Millisecond)
		if err != nil {
			t.Fatalf("unexpected dial error: %v", err)
		}
		defer client.Close()

		_, err = client.Execute(ctx, "say hello")
		if err == nil {
			t.Fatalf("expected error on server disconnect during execute, got nil")
		}
	})
}
