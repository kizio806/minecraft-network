package discord

import "io"

// Exported rconPacket helpers for package internal testing only.
func EncodeRconPacketForTest(id, pktType int32, body string) ([]byte, error) {
	return encodeRconPacket(rconPacket{
		ID:   id,
		Type: pktType,
		Body: body,
	})
}

func DecodeRconPacketForTest(r io.Reader) (struct {
	ID   int32
	Type int32
	Body string
}, error) {
	p, err := decodeRconPacket(r)
	return struct {
		ID   int32
		Type int32
		Body string
	}{
		ID:   p.ID,
		Type: p.Type,
		Body: p.Body,
	}, err
}
