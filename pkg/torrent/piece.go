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

type Piece struct {
	index  int
	hash   [20]byte
	length int
}

type PieceResult struct {
	index int
	value []byte
}

type PieceProgress struct {
	index      int
	buf        []byte
	conn       *peer.Conn
	downloaded int
	requested  int
	backlog    int
}

func (p *PieceProgress) ReadMessage() error {
	msg, err := p.conn.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
	}

	switch msg.Identifier {
	case message.Choke:
		p.conn.Choked = true
	case message.UnChoke:
		p.conn.Choked = false
	case message.Have:
		piece, err := message.ParseHave(msg)
		if err != nil {
			return err
		}

		p.conn.Bitfield.Set(piece)
	case message.Piece:
		n, err := message.ParsePiece(p.index, p.buf, msg)
		if err != nil {
			return err
		}

		p.downloaded += n
		p.backlog--
	}

	return nil
}

type PieceManager interface {
	Put(int, []byte) error
	Get(int) ([]byte, error)
}
