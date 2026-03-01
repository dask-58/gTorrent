package peer

import (
	"context"
	"net"
	"time"
)

const blockSize = 16384 // 16KB

func DownloadPiece(ctx context.Context, conn net.Conn, index, pieceLength int) ([]byte, error) {
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// send interested
	_, err := conn.Write(NewInterested().Serialize())
	if err != nil {
		return nil, err
	}

	// wait for unchoke, drain other messages
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
		msg, err := ReadMessage(conn)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return nil, err
		}
		if msg == nil {
			continue // keepalive
		}
		if msg.ID == MsgUnchoke {
			break
		}
		// ignore others
	}

	// request all blocks for this piece
	buf := make([]byte, pieceLength)
	downloaded := 0

	for downloaded < pieceLength {
		begin := downloaded
		length := blockSize
		if pieceLength-begin < blockSize {
			length = pieceLength - begin // last block may be smaller
		}

		req := NewRequest(index, begin, length)
		_, err := conn.Write(req.Serialize())
		if err != nil {
			return nil, err
		}

		// read piece response
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			msg, err := ReadMessage(conn)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return nil, err
			}
			if msg == nil {
				continue
			}
			if msg.ID != MsgPiece {
				continue // ignore others
			}
			n, err := ParsePiece(index, buf, msg)
			if err != nil {
				return nil, err
			}
			downloaded += n
			break
		}
	}

	return buf, nil
}
