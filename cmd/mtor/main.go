package main

import (
	"fmt"
	"os"
	"time"

	"github.com/raklaptudirm/mtor/pkg/file"
	"github.com/raklaptudirm/mtor/pkg/torrent"
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

	ps := Pieces{
		size:   t.PieceLength,
		buffer: make([]byte, t.Length),
	}
	err = t.Download(&ps, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	w, err := os.Create(string(f.Info.Name))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer w.Close()

	for i := range t.PieceHashes {
		block, err := ps.Get(i)
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = w.Write(block)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

// structure to put stuff together
type Pieces struct {
	size   int
	length int
	buffer []byte
}

func (p *Pieces) Put(index int, block []byte) error {
	p.length++
	copy(p.buffer[index*p.size:], block)
	return nil
}

func (p *Pieces) Get(index int) ([]byte, error) {
	return p.buffer[index*p.size : (index+1)*p.size], nil
}
