package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dask-58/gTorrent/internal/bencode"
	"github.com/dask-58/gTorrent/internal/torrentfile"
	"github.com/dask-58/gTorrent/internal/tracker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePeerID(t *testing.T) {
	peerID1, err := tracker.GeneratePeerID()
	require.NoError(t, err)
	assert.Len(t, peerID1, 20)
	assert.Equal(t, "-GT0001-", string(peerID1[:8]))

	peerID2, err := tracker.GeneratePeerID()
	require.NoError(t, err)
	assert.NotEqual(t, peerID1, peerID2, "Subsequent peer IDs should be unique")
}

func TestRequestSuccess(t *testing.T) {
	mockPeers := []byte{192, 168, 0, 1, 0x1A, 0xE1} // 192.168.0.1:6881

	respDict := map[string]interface{}{
		"interval": int64(1800),
		"peers":    mockPeers,
	}

	encodedResp, err := bencode.Encode(respDict)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/announce", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("compact"))
		assert.Equal(t, "6881", r.URL.Query().Get("port"))
		w.WriteHeader(http.StatusOK)
		w.Write(encodedResp)
	}))
	defer server.Close()

	tf := torrentfile.TorrentFile{
		Announce: server.URL + "/announce",
		InfoHash: [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		Length:   100,
	}

	peerID, _ := tracker.GeneratePeerID()
	resp, err := tracker.Request(tf, peerID)

	require.NoError(t, err)
	assert.Equal(t, 1800, resp.Interval)
	assert.Len(t, resp.Peers, 1)
	assert.Equal(t, "192.168.0.1", resp.Peers[0].IP.String())
	assert.Equal(t, uint16(6881), resp.Peers[0].Port)
}

func TestRequestServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tf := torrentfile.TorrentFile{
		Announce: server.URL + "/announce",
	}
	peerID, _ := tracker.GeneratePeerID()
	_, err := tracker.Request(tf, peerID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tracker returned error status")
}

func TestRequestFailureReason(t *testing.T) {
	respDict := map[string]interface{}{
		"failure reason": []byte("invalid info_hash"),
	}
	encodedResp, err := bencode.Encode(respDict)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(encodedResp)
	}))
	defer server.Close()

	tf := torrentfile.TorrentFile{
		Announce: server.URL + "/announce",
	}
	peerID, _ := tracker.GeneratePeerID()
	_, err = tracker.Request(tf, peerID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tracker: invalid info_hash")
}
