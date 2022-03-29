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
	"crypto/sha1"
	"errors"
	"fmt"
	"time"

	"laptudirm.com/x/mtor/pkg/peer"
)

// download represents the state of a torrent thats being downloaded.
type download struct {
	// communication channels
	work   workChan   // work channel
	pieces pieceChan  // pieces channel
	death  deathChan  // death channel
	result resultChan // result channel

	// state information
	torrent *Torrent     // the torrent being downloaded
	manager PieceManager // the piece manager
	peers   []peer.Peer  // the peerlist
	peerNum int          // number of peers connected to

	// config information
	config *DownloadConfig
}

type DownloadConfig struct {
	Backlog int // number of requests to keep in backlog
	PeerAmt int // number of peers to request from tracker

	DownTimeout time.Duration // download timeout
	ConnTimeout time.Duration // connection timeout
}

// workChan represtents a work channel consisting of pieces which need to be
// downloaded.
type workChan chan *piece

// pieceChan represents a piece channel consisting of pieces that have
// been downloaded.
type pieceChan chan *pieceResult

// deathChan represents the channel where dead workers report their death.
type deathChan chan *peer.Peer

// resultChan represents the channel from which the main goroutine receives
// the results of the download.
type resultChan chan result

// result represents a result of the download.
type result int

const (
	resultDownloadComplete result = iota // download successful
	resultAllWorkersDead                 // all workers died
)

var ErrWorkersDead = errors.New("download: all workers are dead")

const MaxBlockSize = 16384 // 16 kb

// start starts downloading the provided download
func (d *download) start() error {
	d.init() // initialize channels

	// get peers
	err := d.loadPeers()
	if err != nil {
		return err
	}

	go d.checkWorkers() // check if workers are working
	go d.managePieces() // manage the downloaded pieces
	go d.scheduleWork() // schedule pieces to download
	go d.startWorkers() // start workers with peers

	switch <-d.result {
	case resultDownloadComplete: // download complete
		err = nil
	case resultAllWorkersDead: // all workers are dead
		err = ErrWorkersDead
	default: // unreachable
		panic("fatal: unknown download result")
	}

	return err
}

// init initializes the channels in the provided download.
func (d *download) init() {
	pieceNum := len(d.torrent.PieceHashes)

	d.work = make(workChan, pieceNum)
	d.pieces = make(pieceChan, pieceNum)
	d.death = make(deathChan)
	d.result = make(resultChan)
}

// loadPeers fetches the peers of the torrent being downloaded, and puts
// them in the state.
func (d *download) loadPeers() error {
	// get peers from tracker
	peers, err := d.torrent.Peers(d.config.PeerAmt)
	d.peers = peers
	return err
}

// checkWorkers manages the lifetime of the workers, and checks if all the
// workers are dead or not.
func (d *download) checkWorkers() {
	for range d.death {
		d.peerNum--

		if d.peerNum == 0 {
			d.result <- resultAllWorkersDead
			close(d.death) // no death left to report
			return
		}
	}
}

// managePieces manages the downloaded pieces from the piece channel.
func (d *download) managePieces() {
	length := cap(d.work)
	for done := 0; done < length; done++ {
		piece := <-d.pieces
		fmt.Printf("mtor: downloaded piece %v, %v peers\n", piece.index, d.peerNum)
		d.manager.Put(piece.index, piece.value)
	}

	close(d.work)   // no work left to schedule
	close(d.pieces) // no pieces left to download

	// all pieces downloaded
	d.result <- resultDownloadComplete
}

// scheduleWork starts putting the torrent pieces in the work channel.
func (d *download) scheduleWork() {
	for index, hash := range d.torrent.PieceHashes {
		d.work <- &piece{
			index:  index,
			hash:   hash,
			length: d.torrent.pieceLen(index),
		}
	}
}

// startWorkers starts connections with the peers in the state.
func (d *download) startWorkers() error {
	d.peerNum = len(d.peers)

	// start peer connections
	for _, peer := range d.peers {
		go d.connectToPeer(peer)
	}

	return nil
}

// connectToPeer tries to connect to the peer p, and if successful, downloads
// the torrent pieces from that peer.
func (d *download) connectToPeer(p peer.Peer) {
	defer func() {
		d.death <- &p // report death
	}()

	// try to connect to peer
	conn, err := peer.NewConn(p, d.torrent.InfoHash, d.torrent.Name, d.config.ConnTimeout)
	if err != nil {
		fmt.Println(err)
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
		block, err := d.downloadPiece(conn, piece)
		if err != nil {
			d.work <- piece
			fmt.Println(err)
			return
		}

		// check the integrity of downloaded piece
		if !checkIntegrity(piece, block) {
			d.work <- piece
			continue
		}

		// send downloaded piece to pieces channel
		d.pieces <- &pieceResult{
			index: piece.index,
			value: block,
		}
	}
}

// downloadBlock downloads a piece from a peer connection.
func (d *download) downloadPiece(conn *peer.Conn, p *piece) ([]byte, error) {
	progress := pieceProgress{
		index: p.index,
		buf:   make([]byte, p.length),
		conn:  conn,
	}

	// set download deadline
	conn.Conn.SetDeadline(time.Now().Add(d.config.DownTimeout))
	defer conn.Conn.SetDeadline(time.Time{}) // disable deadline

	// repeat till number of bytes downloaded is less than total
	for progress.downloaded < p.length {
		if !conn.Choked {
			for progress.backlog < d.config.Backlog && progress.requested < p.length {
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

// DownloadPieces downloads the pieces of the provided torrent and stores
// them into the provided PieceManager.
func (t *Torrent) DownloadPieces(p PieceManager, c *DownloadConfig) error {
	start := time.Now()

	err := t.newDownload(p, c).start()
	if err != nil {
		return err
	}

	duration := time.Since(start)
	fmt.Println("mtor: download complete")
	fmt.Printf("mtor: %s taken", duration)

	return nil
}

func (t *Torrent) newDownload(p PieceManager, c *DownloadConfig) *download {
	return &download{
		torrent: t,
		manager: p,
		config:  c,
	}
}
