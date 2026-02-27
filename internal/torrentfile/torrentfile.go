package torrentfile

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"

	"github.com/dask-58/gTorrent/internal/bencode"
)

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

// Parse Open the file in main.go, pass the io.Reader here
func Parse(r io.Reader) (TorrentFile, error) {
	val, err := bencode.Decode(r)
	if err != nil {
		return TorrentFile{}, err
	}

	root, ok := val.(map[string]interface{})
	if !ok {
		return TorrentFile{}, errors.New("root bencode is not a dictionary")
	}

	return extractMeta(root)
}

func extractMeta(root map[string]interface{}) (TorrentFile, error) {
	announceBuf, ok := root["announce"].([]byte)
	if !ok {
		return TorrentFile{}, errors.New("missing announce")
	}
	announce := string(announceBuf)

	info, ok := root["info"].(map[string]interface{})
	if !ok {
		return TorrentFile{}, errors.New("missing info dictionary")
	}

	nameBuf, ok := info["name"].([]byte)
	if !ok {
		return TorrentFile{}, errors.New("missing name")
	}
	name := string(nameBuf)

	pieceLengthRaw, ok := info["piece length"].(int64)
	if !ok {
		return TorrentFile{}, errors.New("missing or invalid piece length")
	}

	lengthRaw, ok := info["length"].(int64)
	if !ok {
		return TorrentFile{}, errors.New("missing or invalid length")
	}

	piecesBuf, ok := info["pieces"].([]byte)
	if !ok {
		return TorrentFile{}, errors.New("missing pieces")
	}

	infoHash, err := hashInfoDict(info)
	if err != nil {
		return TorrentFile{}, err
	}

	pieceHashes, err := splitPieceHashes(piecesBuf)
	if err != nil {
		return TorrentFile{}, err
	}

	return TorrentFile{
		Announce:    announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: int(pieceLengthRaw),
		Length:      int(lengthRaw),
		Name:        name,
	}, nil
}

func hashInfoDict(info map[string]interface{}) ([20]byte, error) {
	encodedInfo, err := bencode.Encode(info)
	if err != nil {
		return [20]byte{}, err
	}
	return sha1.Sum(encodedInfo), nil
}

func splitPieceHashes(buf []byte) ([][20]byte, error) {
	const hashLen = 20

	if len(buf)%hashLen != 0 {
		return nil, fmt.Errorf("malformed pieces string")
	}

	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)
	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}
