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

					reqID := int32(binary.LittleEndian.Uint32(payload[0:4]))
					pktType := int32(binary.LittleEndian.Uint32(payload[4:8]))
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
	length := int32(4 + 4 + len(bodyBytes) + 2)
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
		"list": "There are 2 out of 50 players online: Steve, Alex",
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
}
