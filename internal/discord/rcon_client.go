package discord

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	rconPacketAuth         int32 = 3
	rconPacketAuthResponse int32 = 2
	rconPacketExecCommand  int32 = 2
	rconPacketResponse     int32 = 0

	defaultRconDialTimeout = 5 * time.Second
	defaultRconReadTimeout = 10 * time.Second
)

var (
	// ErrRconAuthFailed indicates invalid credentials passed during RCON handshake.
	ErrRconAuthFailed = errors.New("rcon: authentication failed (bad password)")
	// ErrRconTargetNotFound indicates unknown server target name.
	ErrRconTargetNotFound = errors.New("rcon: server target not found")
)

type rconPacket struct {
	ID   int32
	Type int32
	Body string
}

func encodeRconPacket(p rconPacket) ([]byte, error) {
	bodyBytes := []byte(p.Body)
	// packet length: 4 (ID) + 4 (Type) + len(body) + 2 (terminating nulls)
	length := int32(4 + 4 + len(bodyBytes) + 2)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, length); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, p.ID); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, p.Type); err != nil {
		return nil, err
	}
	buf.Write(bodyBytes)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)

	return buf.Bytes(), nil
}

func decodeRconPacket(r io.Reader) (rconPacket, error) {
	var length int32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return rconPacket{}, err
	}
	if length < 10 || length > 65535 {
		return rconPacket{}, fmt.Errorf("invalid rcon packet length %d", length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return rconPacket{}, err
	}

	id := int32(binary.LittleEndian.Uint32(payload[0:4]))
	pktType := int32(binary.LittleEndian.Uint32(payload[4:8]))
	// null-terminated body
	bodyBytes := payload[8:]
	if len(bodyBytes) >= 2 && bodyBytes[len(bodyBytes)-1] == 0x00 && bodyBytes[len(bodyBytes)-2] == 0x00 {
		bodyBytes = bodyBytes[:len(bodyBytes)-2]
	} else if len(bodyBytes) >= 1 && bodyBytes[len(bodyBytes)-1] == 0x00 {
		bodyBytes = bodyBytes[:len(bodyBytes)-1]
	}

	return rconPacket{
		ID:   id,
		Type: pktType,
		Body: string(bodyBytes),
	}, nil
}

// RconClient manages a single TCP session with a Minecraft RCON endpoint.
type RconClient struct {
	conn   net.Conn
	nextID int32
	mu     sync.Mutex
}

// DialRcon establishes a TCP connection, completes the RCON authentication handshake, and returns an active client.
func DialRcon(ctx context.Context, address, password string, dialTimeout time.Duration) (*RconClient, error) {
	if dialTimeout <= 0 {
		dialTimeout = defaultRconDialTimeout
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial rcon at %s: %w", address, err)
	}

	client := &RconClient{conn: conn}
	if err := client.authenticate(ctx, password); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *RconClient) authenticate(ctx context.Context, password string) error {
	reqID := atomic.AddInt32(&c.nextID, 1)
	pkt := rconPacket{
		ID:   reqID,
		Type: rconPacketAuth,
		Body: password,
	}

	data, err := encodeRconPacket(pkt)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(defaultRconDialTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = c.conn.SetDeadline(deadline)

	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("write auth packet: %w", err)
	}

	resp, err := decodeRconPacket(c.conn)
	if err != nil {
		return fmt.Errorf("read auth response: %w", err)
	}

	// Some servers reply with empty response value before auth response
	if resp.Type == rconPacketResponse {
		resp, err = decodeRconPacket(c.conn)
		if err != nil {
			return fmt.Errorf("read auth response following empty packet: %w", err)
		}
	}

	if resp.ID == -1 || resp.ID != reqID {
		return ErrRconAuthFailed
	}
	return nil
}

// Execute sends an administrative console command and returns the output string.
func (c *RconClient) Execute(ctx context.Context, command string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	reqID := atomic.AddInt32(&c.nextID, 1)
	pkt := rconPacket{
		ID:   reqID,
		Type: rconPacketExecCommand,
		Body: command,
	}

	data, err := encodeRconPacket(pkt)
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(defaultRconReadTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = c.conn.SetDeadline(deadline)

	if _, err := c.conn.Write(data); err != nil {
		return "", fmt.Errorf("send rcon command: %w", err)
	}

	resp, err := decodeRconPacket(c.conn)
	if err != nil {
		return "", fmt.Errorf("read rcon response: %w", err)
	}

	out := strings.TrimSpace(resp.Body)
	if out == "" {
		out = "Command executed successfully (no output)."
	}
	return out, nil
}

// Close terminates the TCP session.
func (c *RconClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// NetworkRconExecutor executes console commands by opening short-lived or pooled RCON TCP connections.
type NetworkRconExecutor struct {
	Targets     map[string]string
	Password    string
	DialTimeout time.Duration
	Fallback    ConsoleExecutor
}

// NewNetworkRconExecutor constructs a NetworkRconExecutor with default network targets.
func NewNetworkRconExecutor(password string, targets map[string]string) *NetworkRconExecutor {
	if targets == nil {
		targets = map[string]string{
			"lobby":    "lobby:25575",
			"survival": "survival:25575",
		}
	}
	return &NetworkRconExecutor{
		Targets:     targets,
		Password:    password,
		DialTimeout: 4 * time.Second,
	}
}

// Execute resolves the target server, dials via RCON protocol, and returns command output.
func (e *NetworkRconExecutor) Execute(ctx context.Context, server, command string) (string, error) {
	server = strings.ToLower(strings.TrimSpace(server))
	addr, exists := e.Targets[server]
	if !exists {
		if e.Fallback != nil {
			return e.Fallback.Execute(ctx, server, command)
		}
		return "", fmt.Errorf("%w: %q (known targets: %v)", ErrRconTargetNotFound, server, mapKeys(e.Targets))
	}

	client, err := DialRcon(ctx, addr, e.Password, e.DialTimeout)
	if err != nil {
		if e.Fallback != nil {
			return e.Fallback.Execute(ctx, server, command)
		}
		return "", fmt.Errorf("rcon connect to %s (%s): %w", server, addr, err)
	}
	defer client.Close()

	return client.Execute(ctx, command)
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
