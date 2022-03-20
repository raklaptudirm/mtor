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
	"github.com/raklaptudirm/mtor/pkg/message"
	"github.com/raklaptudirm/mtor/pkg/peer"
)

// piece represents a piece of a torrent that needs to be downloaded.
type piece struct {
	index  int      // the index of the piece
	hash   [20]byte // the hash of the piece
	length int      // the length of the piece
}

// pieceResult represents a piece that has been successfully downloaded.
type pieceResult struct {
	index int    // the index of the piece
	value []byte // the value of the piece
}

// PieceProgress represents the progress made on a piece that is currently
// being downloaded.
type pieceProgress struct {
	index      int        // index of the piece
	buf        []byte     // buffer to store value of the piece
	conn       *peer.Conn // connection to download the piece from
	downloaded int        // number of bytes dowloaded
	requested  int        // number of bytes requested
	backlog    int        // backlog of block requests
}

// readMessage reads a message from p's peer connection, and works according
// to the message.
func (p *pieceProgress) readMessage() error {
	// read message from connection
	msg, err := p.conn.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
	}

	switch msg.Identifier {
	case message.Choke:
		// peer un-choked us
		p.conn.Choked = true
	case message.UnChoke:
		// peer choked us
		p.conn.Choked = false
	case message.Have:
		// peer has a new piece
		piece, err := message.ParseHave(msg)
		if err != nil {
			return err
		}

		p.conn.Bitfield.Set(piece)
	case message.Piece:
		// peer sent a block of data
		n, err := message.ParsePiece(p.index, p.buf, msg)
		if err != nil {
			return err
		}

		p.downloaded += n
		p.backlog--
	}

	return nil
}

// PieceManager represents an interface which can handle the storage of the
// torrent's pieces.
type PieceManager interface {
	// Init initializes the manager to start storing pieces.
	Init() error
	// Put stores the buffer data with the provided piece index.
	Put(int, []byte) error
	// Get gets the data of the provided piece index.
	Get(int) ([]byte, error)
	// Close destroy's the manager's data. Call this when done.
	Close() error
}
