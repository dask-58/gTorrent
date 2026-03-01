package tracker

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/dask-58/gTorrent/internal/bencode"
	"github.com/dask-58/gTorrent/internal/torrentfile"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

type TrackerResponse struct {
	Peers    []Peer
	Interval int
}

func GeneratePeerID() ([20]byte, error) {
	var pID [20]byte
	copy(pID[:8], "-GT0001-")
	_, err := rand.Read(pID[8:])
	return pID, err
}

func encodeBinary(b []byte) string {
	var buf strings.Builder
	for _, c := range b {
		_, _ = fmt.Fprintf(&buf, "%%%02x", c)
	}
	return buf.String()
}

func buildAnnounceURL(announce string, infoHash [20]byte, peerID [20]byte, left int) (string, error) {
	base, err := url.Parse(announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"port":       []string{"6881"},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{fmt.Sprintf("%d", left)},
		"event":      []string{"started"},
		"compact":    []string{"1"},
	}

	base.RawQuery = params.Encode() + "&info_hash=" + encodeBinary(infoHash[:]) + "&peer_id=" + encodeBinary(peerID[:])
	return base.String(), nil
}

func parsePeers(raw []byte) ([]Peer, error) {
	if len(raw)%6 != 0 {
		return nil, errors.New("invalid peer list length")
	}

	peers := make([]Peer, len(raw)/6)
	for i := range peers {
		offset := i * 6
		peers[i] = Peer{
			IP:   net.IP(raw[offset : offset+4]),
			Port: binary.BigEndian.Uint16(raw[offset+4 : offset+6]),
		}
	}
	return peers, nil
}

func Request(tf torrentfile.TorrentFile, peerID [20]byte) (TrackerResponse, error) {
	rawURL, err := buildAnnounceURL(tf.Announce, tf.InfoHash, peerID, tf.Length)
	if err != nil {
		return TrackerResponse{}, err
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return TrackerResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return TrackerResponse{}, fmt.Errorf("tracker returned error status: %d %s", resp.StatusCode, resp.Status)
	}

	decoded, err := bencode.Decode(resp.Body)
	if err != nil {
		return TrackerResponse{}, err
	}

	dict, ok := decoded.(map[string]interface{})
	if !ok {
		return TrackerResponse{}, errors.New("invalid tracker response")
	}

	if failure, ok := dict["failure reason"].([]byte); ok {
		return TrackerResponse{}, fmt.Errorf("tracker: %s", failure)
	}

	peersRaw, ok := dict["peers"].([]byte)
	if !ok {
		return TrackerResponse{}, errors.New("missing peers field")
	}

	peers, err := parsePeers(peersRaw)
	if err != nil {
		return TrackerResponse{}, err
	}

	interval, _ := dict["interval"].(int64)
	return TrackerResponse{Interval: int(interval), Peers: peers}, nil
}
