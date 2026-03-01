package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dask-58/gTorrent/internal/logger"
	"github.com/dask-58/gTorrent/internal/peer"
	"github.com/dask-58/gTorrent/internal/torrent"
	"github.com/dask-58/gTorrent/internal/torrentfile"
	"github.com/dask-58/gTorrent/internal/tracker"
)

func main() {
	if err := logger.Init(); err != nil {
		panic(err)
	}

	f, err := os.Open("data/debian-13.3.0-arm64-netinst.iso.torrent")
	if err != nil {
		logger.Log.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	tf, err := torrentfile.Parse(f)
	if err != nil {
		logger.Log.Fatal(err)
	}

	peerID, err := tracker.GeneratePeerID()
	if err != nil {
		logger.Log.Fatal(err)
	}

	resp, err := tracker.Request(tf, peerID)
	if err != nil {
		logger.Log.Fatal(err)
	}

	logger.Log.Infof("Attempting handshake with %d peers...", len(resp.Peers))

	type result struct {
		addr string
		conn net.Conn
	}

	results := make(chan result, len(resp.Peers))

	for _, p := range resp.Peers {
		go func(p tracker.Peer) {
			addr := fmt.Sprintf("%s:%d", p.IP, p.Port)
			conn, _, err := peer.Connect(addr, tf.InfoHash, peerID)
			if err != nil {
				results <- result{addr, nil}
				return
			}
			results <- result{addr, conn}
		}(p)
	}

	var goodConns []net.Conn
	var bad int
	for range resp.Peers {
		r := <-results
		if r.conn != nil {
			goodConns = append(goodConns, r.conn)
			logger.Log.Infof("Handshake with %s", r.addr)
		} else {
			bad++
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Log.Info("\nReceived interrupt signal. Shutting down gracefully...")
		cancel()
	}()

	err = torrent.Download(ctx, tf, goodConns, "output/"+tf.Name)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Log.Info("Shutdown complete.")
		} else {
			logger.Log.Fatal(err)
		}
	}
}
