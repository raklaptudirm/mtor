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

type Conn struct {
	Conn     net.Conn
	Choked   bool
	Peer     Peer
	Bitfield bitfield.Bitfield
	InfoHash [20]byte
	Name     [20]byte
}

func (c *Conn) Read() (*message.Message, error) {
	return message.Read(c.Conn)
}

func (c *Conn) UnChoke() error {
	m := &message.Message{Identifier: message.UnChoke}
	_, err := c.Conn.Write(m.Serialize())
	return err
}

func (c *Conn) Interested() error {
	m := &message.Message{Identifier: message.Interested}
	_, err := c.Conn.Write(m.Serialize())
	return err
}

func (c *Conn) Request(index, begin, length int) error {
	req := message.NewReqest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}

func (p *Peer) handshake(conn net.Conn, hash, name [20]byte) (*message.Handshake, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	req := message.NewHandshake(hash, name)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := message.ReadHandshake(conn)
	if err != nil {
		return nil, err
	}

	if err := res.Verify(hash); err != nil {
		return nil, err
	}

	return res, nil
}

func getBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}

	if msg.Identifier != message.Bitfield {
		return nil, fmt.Errorf("expected bitfield message, received %v", msg.Identifier)
	}

	return msg.Payload, nil
}

func NewConn(peer Peer, hash, name [20]byte) (*Conn, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = peer.handshake(conn, hash, name)
	if err != nil {
		return nil, err
	}

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
