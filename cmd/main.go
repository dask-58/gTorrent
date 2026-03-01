package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dask-58/gTorrent/internal/torrentfile"
	"github.com/dask-58/gTorrent/internal/tracker"
)

func main() {
	f, err := os.Open("data/debian-13.3.0-arm64-netinst.iso.torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tf, err := torrentfile.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	peerID, err := tracker.GeneratePeerID()
	if err != nil {
		log.Fatal(err)
	}

	resp, err := tracker.Request(tf, peerID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Got %d peers\n", len(resp.Peers))
	for _, p := range resp.Peers {
		fmt.Printf("  %s:%d\n", p.IP, p.Port)
	}
	fmt.Printf("Interval: %d seconds\n", resp.Interval)
}
