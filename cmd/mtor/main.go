package main

import (
	"fmt"
	"os"
	"time"

	"laptudirm.com/x/mtor/internal/build"
	"laptudirm.com/x/mtor/pkg/file"
	"laptudirm.com/x/mtor/pkg/torrent"
)

func main() {
	// basic config
	config := &torrent.DownloadConfig{
		Backlog:     25,
		PeerAmt:     500,
		DownTimeout: 20 * time.Second,
		ConnTimeout: 5 * time.Second,
	}

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mtor [torrent]")
		os.Exit(1)
	}

	r, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	f, err := file.Open(r)
	if err != nil {
		fmt.Println(err)
		return
	}

	t, err := f.Torrent()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("torrent %x - %d pieces\n", t.InfoHash, len(t.PieceHashes))

	ps := build.PieceManager
	err = ps.Init()
	if err != nil {
		fmt.Println(err)
	}
	defer ps.Close()

	err = t.DownloadPieces(ps, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = f.Save(ps, ".") // save in cwd
	if err != nil {
		fmt.Println(err)
		return
	}
}
