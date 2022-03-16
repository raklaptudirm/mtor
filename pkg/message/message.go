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

package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

// id represents the various message types.
type id byte

// various message types and their identifiers.
const (
	Choke         id = 0
	UnChoke       id = 1
	Interested    id = 2
	NotInterested id = 3
	Have          id = 4
	Bitfield      id = 5
	Request       id = 6
	Piece         id = 7
	Cancel        id = 8
)

// Message represents a bittorrent p2p message.
type Message struct {
	Identifier id     // message identifier
	Payload    []byte // message payload
}

// Serialize serializes a message into a byte slice.
// [length] [id] [payload]
func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1)
	msg := make([]byte, length+4)

	binary.BigEndian.PutUint32(msg[:4], length)
	msg[4] = byte(m.Identifier)
	copy(msg[5:], m.Payload)

	return msg
}

// Read reads a serialized message from a io.Reader.
func Read(r io.Reader) (*Message, error) {
	// read length
	lenBuf := make([]byte, 4) // 4 byte length prefix
	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	// read id and payload
	msgBuf := make([]byte, length)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return nil, err
	}

	return &Message{
		Identifier: id(msgBuf[0]),
		Payload:    msgBuf[1:],
	}, nil
}

// NewRequest formats a request message into a Message value.
func NewReqest(index, begin, length int) *Message {
	payload := make([]byte, 12)

	// [index] [begin] [length]
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{
		Identifier: Request,
		Payload:    payload,
	}
}

// ParseHave parses a Have Message to get the piece index.
func ParseHave(msg *Message) (int, error) {
	if msg.Identifier != Have {
		return 0, fmt.Errorf("expected Have message, received %v", msg.Identifier)
	}

	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload of length 4, received %v", len(msg.Payload))
	}

	return int(binary.BigEndian.Uint32(msg.Payload)), nil
}

// ParsePiece parses a PieceMessage and puts the payload into the provided buffer.
func ParsePiece(index int, buf []byte, msg *Message) (int, error) {
	if msg.Identifier != Piece {
		return 0, fmt.Errorf("expected Piece message, received %v", msg.Identifier)
	}

	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("payload too short with length %v", len(msg.Payload))
	}

	recIndex := int(binary.BigEndian.Uint32(msg.Payload[:4]))
	if recIndex != index {
		return 0, fmt.Errorf("expected piece %v, received %v", index, recIndex)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("begin index too high at %v", begin)
	}

	block := msg.Payload[8:]
	if begin+len(block) > len(buf) {
		return 0, fmt.Errorf("block size too big at %v bytes", len(block))
	}

	copy(buf[begin:], block)
	return len(block), nil
}
