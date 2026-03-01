package peer

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageID uint8

const (
	MsgChoke         MessageID = 0
	MsgUnchoke       MessageID = 1
	MsgInterested    MessageID = 2
	MsgNotInterested MessageID = 3
	MsgHave          MessageID = 4
	MsgBitfield      MessageID = 5
	MsgRequest       MessageID = 6
	MsgPiece         MessageID = 7
	MsgCancel        MessageID = 8
)

type Message struct {
	ID      MessageID
	Payload []byte
}

// ReadMessage reads one length-prefixed message from the connection
// Format: [4 bytes length][1 byte ID][payload]
func ReadMessage(r io.Reader) (*Message, error) {
	// read 4-byte length prefix
	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lenBuf)
	if length == 0 {
		return nil, nil // keepalive, caller should ignore and loop
	}

	msgBuf := make([]byte, length)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      MessageID(msgBuf[0]),
		Payload: msgBuf[1:],
	}, nil
}

func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4) // keepalive
	}
	length := 1 + len(m.Payload)
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[:4], uint32(length))
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

// helper constructors
func NewInterested() *Message {
	return &Message{ID: MsgInterested}
}

func NewRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}

func ParsePiece(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("expected piece, got %d", msg.ID)
	}
	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("piece payload too short")
	}
	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("piece index mismatch: got %d want %d", parsedIndex, index)
	}
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("block out of bounds")
	}
	copy(buf[begin:], data)
	return len(data), nil
}
