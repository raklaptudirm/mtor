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

package peer

import (
	"fmt"
	"net"
	"time"

	"github.com/raklaptudirm/mtor/pkg/bitfield"
	"github.com/raklaptudirm/mtor/pkg/message"
)

// Conn represents a p2p connection to a peer.
type Conn struct {
	Conn     net.Conn          // the connection with the peer
	Choked   bool              // wether the peer is choking
	Peer     Peer              // the peer with the connection
	Bitfield bitfield.Bitfield // peer's bitfield
	InfoHash [20]byte          // torrent infohash
	Name     [20]byte          // peer's identifier
}

// Read reads a Message from the Conn.
func (c *Conn) Read() (*message.Message, error) {
	return message.Read(c.Conn)
}

// UnChoke sends an UnChoke message to the Conn.
func (c *Conn) UnChoke() error {
	m := &message.Message{Identifier: message.UnChoke}
	_, err := c.Conn.Write(m.Serialize())
	return err
}

// Interested sends an Interested message to the Conn.
func (c *Conn) Interested() error {
	m := &message.Message{Identifier: message.Interested}
	_, err := c.Conn.Write(m.Serialize())
	return err
}

// Request sends a Request message to the Conn.
func (c *Conn) Request(index, begin, length int) error {
	req := message.NewReqest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}

// handshake tries to complete a proper handshake with the peer.
func (p *Peer) handshake(conn net.Conn, hash, name [20]byte) (*message.Handshake, error) {
	// set handshake deadline
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // disable deadline

	// send a handshake to the peer
	req := message.NewHandshake(hash, name)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	// await a handshake from the peer
	res, err := message.ReadHandshake(conn)
	if err != nil {
		return nil, err
	}

	// verify the peer's handshake
	if err := res.Verify(hash); err != nil {
		return nil, err
	}

	return res, nil
}

// getBitfield reads a serialized bitfield from the Conn.
func getBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	// set bitfield deadline
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // disable deadline

	// await message from peer
	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}

	// expect Message of type Bitfield
	if msg.Identifier != message.Bitfield {
		return nil, fmt.Errorf("expected bitfield message, received %v", msg.Identifier)
	}

	return msg.Payload, nil
}

// NewConn creates a new p2p Conn with the provided peer.
func NewConn(peer Peer, hash, name [20]byte) (*Conn, error) {
	// dial a tcp connection with peer
	conn, err := net.DialTimeout("tcp", peer.String(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	// try to complete handshake with peer
	_, err = peer.handshake(conn, hash, name)
	if err != nil {
		return nil, err
	}

	// get peer's bitfield
	b, err := getBitfield(conn)
	if err != nil {
		return nil, err
	}

	return &Conn{
		Conn:     conn,
		Choked:   true,
		Peer:     peer,
		Bitfield: b,
		InfoHash: hash,
		Name:     name,
	}, nil
}
