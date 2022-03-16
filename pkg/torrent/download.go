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

// download represents the state of a torrent thats being downloaded.
type download struct {
	// communication channels
	work    workChan   // work channel
	results resultChan // results channel

	// state information
	torrent *Torrent     // the torrent being downloaded
	manager PieceManager // the piece manager
	peers   []peer.Peer  // the peerlist
	peerNum int          // number of peers connected to

	// config information
	maxBacklog   int // maximum request backlog
	maxBlockSize int // maximum data per request in bytes
	maxPeers     int // number of peers to request
}

// workChan represtents a work channel consisting of pieces which need to be
// downloaded.
type workChan chan *piece

// resultChan represents a result channel consisting of pieces that have
// been downloaded.
type resultChan chan *pieceResult

// start starts downloading the provided download
func (d *download) start() error {
	length := cap(d.work)

	// get peers
	err := d.loadPeers()
	if err != nil {
		return err
	}

	// start connections with peers
	go d.startConns()

	// send pieces into work channel
	go d.putPieces()

	for done := 0; done < length; done++ {
		res := <-d.results
		fmt.Printf("mtor: downloaded piece %v, %v peers\n", res.index, d.peerNum)
		d.manager.Put(res.index, res.value)
	}
	// all pieces downloaded
	close(d.work)

	return nil
}

// putPieces starts putting the torrent pieces in the work channel.
func (d *download) putPieces() {
	for index, hash := range d.torrent.PieceHashes {
		d.work <- &piece{
			index:  index,
			hash:   hash,
			length: d.torrent.pieceLen(index),
		}
	}
}

// loadPeers fetches the peers of the torrent being downloaded, and puts
// them in the state.
func (d *download) loadPeers() error {
	// get peers from tracker
	peers, err := d.torrent.Peers(MaxPeers)
	d.peers = peers
	return err
}

// startConns starts connections with the peers in the state.
func (d *download) startConns() error {
	// start peer connections
	for _, peer := range d.peers {
		go d.connectToPeer(peer)
	}

	return nil
}

// connectToPeer tries to connect to the peer p, and if successful, downloads
// the torrent pieces from that peer.
func (d *download) connectToPeer(p peer.Peer) {
	d.peerNum++
	defer func() { d.peerNum-- }()

	// try to connect to peer
	conn, err := peer.NewConn(p, d.torrent.InfoHash, d.torrent.Name)
	if err != nil {
		return
	}
	defer conn.Conn.Close()

	conn.UnChoke() // un-choke peer
	conn.Interested()

	fmt.Printf("mtor: connected to peer %s\n", p)

	// get pieces from work channel
	for piece := range d.work {
		// check if peer has piece
		if !conn.Bitfield.Has(piece.index) {
			d.work <- piece
			continue
		}

		// download piece from peer
		block, err := downloadBlock(conn, piece)
		if err != nil {
			d.work <- piece
			return
		}

		// check the integrity of downloaded piece
		if !checkIntegrity(piece, block) {
			d.work <- piece
			continue
		}

		// send downloaded piece to results channel
		d.results <- &pieceResult{
			index: piece.index,
			value: block,
		}
	}
}

// downloadBlock downloads a piece from a peer connection.
func downloadBlock(conn *peer.Conn, p *piece) ([]byte, error) {
	progress := pieceProgress{
		index: p.index,
		buf:   make([]byte, p.length),
		conn:  conn,
	}

	// set download deadline
	conn.Conn.SetDeadline(time.Now().Add(20 * time.Second))
	defer conn.Conn.SetDeadline(time.Time{}) // disable deadline

	// repeat till number of bytes downloaded is less than total
	for progress.downloaded < p.length {
		if !conn.Choked {
			for progress.backlog < MaxBacklog && progress.requested < p.length {
				// calculate block size
				size := MaxBlockSize
				// last block is of irregular size
				if p.length-progress.requested < size {
					size = p.length - progress.requested
				}

				// request block
				err := conn.Request(p.index, progress.requested, size)
				if err != nil {
					return nil, err
				}
				progress.backlog++
				progress.requested += size
			}
		}

		err := progress.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return progress.buf, nil
}

// checkIntegrity checks if the dowloaded piece's hash matches the expected
// hash.
func checkIntegrity(p *piece, block []byte) bool {
	return p.hash == sha1.Sum(block)
}

// pieceLen calculates the length of the piece with the provided index.
func (t *Torrent) pieceLen(index int) int {
	begin := index * t.PieceLength // beginning of piece
	end := begin + t.PieceLength   // end of piece

	// last piece is irregular in length
	if end > t.Length {
		return t.Length - begin
	}

	// not last piece, default length
	return t.PieceLength
}

var (
	// MaxBacklog represents the maximum number of requests that can be in backlog.
	MaxBacklog = 20
	// MaxBlock size represents the maximum number of bytes that can be requested
	// at a time.
	MaxBlockSize = 16384 // 16 kb
	// MaxPeers represents the number of peers to request from the tracker.
	MaxPeers = 500
)

// Download downloads the t torrent and stores the downloaded pieces into
// the provided PieceManager.
func (t *Torrent) Download(p PieceManager) error {
	start := time.Now()

	download := download{
		work:         make(workChan, len(t.PieceHashes)),
		results:      make(resultChan),
		torrent:      t,
		manager:      p,
		maxBacklog:   MaxBacklog,
		maxBlockSize: MaxBacklog,
		maxPeers:     MaxPeers,
	}

	err := download.start()
	if err != nil {
		return err
	}

	duration := time.Since(start)
	fmt.Println("mtor: download complete")
	fmt.Printf("mtor: %s taken", duration)

	return nil
}

// Identifier generates a random client identifier for use.
func Identifier() [20]byte {
	var id [20]byte
	rand.Read(id[:])

	return id
}
