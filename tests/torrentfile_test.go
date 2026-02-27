package tests

import (
	"bytes"
	"testing"

	"github.com/dask-58/gTorrent/internal/bencode"
	"github.com/dask-58/gTorrent/internal/torrentfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createValidTorrentDict() map[string]interface{} {
	return map[string]interface{}{
		"announce": []byte("http://tracker.example.com:80/announce"),
		"info": map[string]interface{}{
			"name":         []byte("test.txt"),
			"piece length": int64(262144),
			"length":       int64(1048576),
			// 40 bytes = 2 * 20 bytes piece hashes
			"pieces": []byte("12345678901234567890abcdefghijklmnopqrst"),
		},
	}
}

func encodeToBuffer(t *testing.T, dict map[string]interface{}) *bytes.Buffer {
	encoded, err := bencode.Encode(dict)
	require.NoError(t, err)
	return bytes.NewBuffer(encoded)
}

func TestParseValid(t *testing.T) {
	dict := createValidTorrentDict()
	buf := encodeToBuffer(t, dict)

	tf, err := torrentfile.Parse(buf)
	require.NoError(t, err)

	assert.Equal(t, "http://tracker.example.com:80/announce", tf.Announce)
	assert.Equal(t, "test.txt", tf.Name)
	assert.Equal(t, 262144, tf.PieceLength)
	assert.Equal(t, 1048576, tf.Length)
	assert.Len(t, tf.PieceHashes, 2)

	expectedPiece1 := [20]byte{}
	copy(expectedPiece1[:], []byte("12345678901234567890"))
	assert.Equal(t, expectedPiece1, tf.PieceHashes[0])

	expectedPiece2 := [20]byte{}
	copy(expectedPiece2[:], []byte("abcdefghijklmnopqrst"))
	assert.Equal(t, expectedPiece2, tf.PieceHashes[1])
}

func TestParseInvalidBencode(t *testing.T) {
	buf := bytes.NewBufferString("invalid string")
	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error parsing invalid bencode")

	// Missing root dict
	buf = bytes.NewBufferString("i42e")
	_, err = torrentfile.Parse(buf)
	require.Error(t, err, "Expected error when root is not a dictionary")
}

func TestParseMissingAnnounce(t *testing.T) {
	dict := createValidTorrentDict()
	delete(dict, "announce")
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on missing announce")
	assert.Contains(t, err.Error(), "missing announce")
}

func TestParseMissingInfo(t *testing.T) {
	dict := createValidTorrentDict()
	delete(dict, "info")
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on missing info dict")
	assert.Contains(t, err.Error(), "missing info dictionary")
}

func TestParseMissingName(t *testing.T) {
	dict := createValidTorrentDict()
	info := dict["info"].(map[string]interface{})
	delete(info, "name")
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on missing name")
	assert.Contains(t, err.Error(), "missing name")
}

func TestParseMissingPieceLength(t *testing.T) {
	dict := createValidTorrentDict()
	info := dict["info"].(map[string]interface{})
	delete(info, "piece length")
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on missing piece length")
	assert.Contains(t, err.Error(), "missing or invalid piece length")
}

func TestParseMissingLength(t *testing.T) {
	dict := createValidTorrentDict()
	info := dict["info"].(map[string]interface{})
	delete(info, "length")
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on missing length")
	assert.Contains(t, err.Error(), "missing or invalid length")
}

func TestParseMissingPieces(t *testing.T) {
	dict := createValidTorrentDict()
	info := dict["info"].(map[string]interface{})
	delete(info, "pieces")
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on missing pieces")
	assert.Contains(t, err.Error(), "missing pieces")
}

func TestParseMalformedPieces(t *testing.T) {
	dict := createValidTorrentDict()
	info := dict["info"].(map[string]interface{})
	info["pieces"] = []byte("short") // Not a multiple of 20
	buf := encodeToBuffer(t, dict)

	_, err := torrentfile.Parse(buf)
	require.Error(t, err, "Expected error on malformed pieces string")
	assert.Contains(t, err.Error(), "malformed pieces string")
}
