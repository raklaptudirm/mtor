// Copyright © 2021 Rak Laptudirm <raklaptudirm@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package torrent

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/raklaptudirm/mtor/pkg/peer"
)

// Torrent represents the data required to fetch peers and download a torrent
// from a tracker.
type Torrent struct {
	Announce string   // the announce url of the tracker
	InfoHash [20]byte // hash of the info section of the torrent

	PieceHashes [][20]byte // hash of each torrent piece
	PieceLength int        // length of each piece in bytes
	Length      int        // total length of the file

	Name [20]byte // client identifier
	Port uint16   // port the client is listening on
}

func (t *Torrent) startClient(p peer.Peer, w chan *Piece, r chan *PieceResult) {
	conn, err := peer.NewConn(p, t.InfoHash, t.Name)
	if err != nil {
		//fmt.Printf("mtor: failed connection to peer %s\n", p)
		//fmt.Println(err)
		return
	}
	defer conn.Conn.Close()

	conn.UnChoke()
	conn.Interested()

	fmt.Printf("mtor: connected to peer %s\n", p)

	for piece := range w {
		if !conn.Bitfield.Has(piece.index) {
			w <- piece
			continue
		}

		block, err := downloadBlock(conn, piece)
		if err != nil {
			w <- piece
			return
		}

		if !checkIntegrity(piece, block) {
			w <- piece
			continue
		}

		r <- &PieceResult{
			index: piece.index,
			value: block,
		}
	}
}

const (
	MaxBacklog   = 20
	MaxBlockSize = 16384 // 16 kb
)

func downloadBlock(conn *peer.Conn, p *Piece) ([]byte, error) {
	progress := PieceProgress{
		index: p.index,
		buf:   make([]byte, p.length),
		conn:  conn,
	}

	conn.Conn.SetDeadline(time.Now().Add(20 * time.Second))
	defer conn.Conn.SetDeadline(time.Time{})

	for progress.downloaded < p.length {
		if !conn.Choked {
			for progress.backlog < MaxBacklog && progress.requested < p.length {
				size := MaxBlockSize
				if p.length-progress.requested < size {
					size = p.length - progress.requested
				}

				err := conn.Request(p.index, progress.requested, size)
				if err != nil {
					return nil, err
				}
				progress.backlog++
				progress.requested += size
			}
		}

		err := progress.ReadMessage()
		if err != nil {
			return nil, err
		}
	}

	return progress.buf, nil
}

func checkIntegrity(p *Piece, block []byte) bool {
	hash := sha1.Sum(block)
	return p.hash == hash
}

func (t *Torrent) Download(p PieceManager) error {
	start := time.Now()

	length := len(t.PieceHashes)

	peers, err := t.Peers()
	if err != nil {
		return err
	}

	pieces := make(chan *Piece, length)
	result := make(chan *PieceResult)
	for index, hash := range t.PieceHashes {
		pieces <- &Piece{
			index:  index,
			hash:   hash,
			length: t.pieceLen(index),
		}
	}

	for _, peer := range peers {
		go t.startClient(peer, pieces, result)
	}

	completed := 0
	for completed < length {
		res := <-result
		fmt.Printf("mtor: downloaded piece %v\n", res.index)
		p.Put(res.index, res.value)
		completed++
	}
	close(pieces)

	duration := time.Since(start)
	fmt.Println("mtor: download complete")
	fmt.Printf("mtor: %s taken", duration)

	return nil
}

func (t *Torrent) pieceLen(index int) int {
	begin := index * t.PieceLength
	end := begin + t.PieceLength

	if end > t.Length {
		return t.Length - begin
	}
	return t.PieceLength
}

// Identifier generates a random client identifier for use.
func Identifier() [20]byte {
	var id [20]byte
	rand.Read(id[:])

	return id
}
