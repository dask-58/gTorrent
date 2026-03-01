package peer

import (
	"fmt"
	"io"
	"net"
	"time"
)

const protocolStr = "BitTorrent protocol"

type Handshake struct {
	InfoHash [20]byte
	PeerID   [20]byte
}

// [1][19][8][20][20]
// pstrlen | pstr | reserved | infohash | peerID
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, 68)
	buf[0] = 19
	copy(buf[1:20], protocolStr)
	// buf[20:28] reserved, already zero
	copy(buf[28:48], h.InfoHash[:])
	copy(buf[48:68], h.PeerID[:])
	return buf
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	pstrLen := make([]byte, 1)
	_, err := io.ReadFull(r, pstrLen)
	if err != nil {
		return nil, err
	}
	if pstrLen[0] != 19 {
		return nil, fmt.Errorf("unexpected pstrlen: %d", pstrLen[0])
	}

	rest := make([]byte, 67)
	_, err = io.ReadFull(r, rest)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte
	copy(infoHash[:], rest[27:47])
	copy(peerID[:], rest[47:67])

	return &Handshake{InfoHash: infoHash, PeerID: peerID}, nil
}

func Connect(addr string, infoHash [20]byte, peerID [20]byte) (net.Conn, *Handshake, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, nil, err
	}

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	h := &Handshake{InfoHash: infoHash, PeerID: peerID}
	_, err = conn.Write(h.Serialize())
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	resp, err := ReadHandshake(conn)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	if resp.InfoHash != infoHash {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("infohash mismatch")
	}

	_ = conn.SetDeadline(time.Time{}) // clear deadline after handshake
	return conn, resp, nil
}
