package torrent

import (
	"context"
	"crypto/sha1"
	"net"
	"os"
	"path/filepath"

	"github.com/dask-58/gTorrent/internal/logger"
	"github.com/dask-58/gTorrent/internal/peer"
	"github.com/dask-58/gTorrent/internal/torrentfile"
	"github.com/schollz/progressbar/v3"
)

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	data  []byte
}

func Download(ctx context.Context, tf torrentfile.TorrentFile, conns []net.Conn, outPath string) error {
	// build work queue
	workQueue := make(chan *pieceWork, len(tf.PieceHashes))
	results := make(chan *pieceResult)

	for i, hash := range tf.PieceHashes {
		length := pieceLength(tf, i)
		workQueue <- &pieceWork{i, hash, length}
	}

	// spawn one worker per connection
	for _, conn := range conns {
		go worker(ctx, conn, workQueue, results)
	}

	// collect results and write to file
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	// pre-allocate full file size
	if err := file.Truncate(int64(tf.Length)); err != nil {
		return err
	}

	done := 0
	total := len(tf.PieceHashes)

	bar := progressbar.Default(int64(total), "Downloading pieces")

	for done < total {
		select {
		case <-ctx.Done():
			logger.Log.Info("\nDownload interrupted. Flushing file...")
			return ctx.Err()
		case res := <-results:
			offset := int64(res.index) * int64(tf.PieceLength)
			_, err := file.WriteAt(res.data, offset)
			if err != nil {
				return err
			}
			done++
			_ = bar.Add(1)
		}
	}
	logger.Log.Info("Download completed!")
	return nil
}

func worker(ctx context.Context, conn net.Conn, workQueue chan *pieceWork, results chan<- *pieceResult) {
	defer func() { _ = conn.Close() }()
	for {
		select {
		case <-ctx.Done():
			return // abort worker
		case work, ok := <-workQueue:
			if !ok {
				return // queue closed
			}
			data, err := peer.DownloadPiece(ctx, conn, work.index, work.length)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				workQueue <- work // re-queue on failure
				return            // this worker is done, conn is probably dead
			}

			hash := sha1.Sum(data)
			if hash != work.hash {
				workQueue <- work // re-queue bad piece
				continue
			}

			select {
			case <-ctx.Done():
				return
			case results <- &pieceResult{work.index, data}:
			}
		}
	}
}

// last piece is usually smaller than PieceLength
func pieceLength(tf torrentfile.TorrentFile, index int) int {
	begin := index * tf.PieceLength
	end := begin + tf.PieceLength
	if end > tf.Length {
		end = tf.Length
	}
	return end - begin
}
