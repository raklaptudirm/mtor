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
	"fmt"
	"io"
)

const ProtocolName = "BitTorrent protocol"

type Handshake struct {
	Protocol   string
	Reserved   [8]byte
	InfoHash   [20]byte
	Identifier [20]byte
}

func (h *Handshake) Serialize() []byte {
	length := byte(len(h.Protocol))

	buffer := make([]byte, 1+length)

	buffer[0] = length
	copy(buffer[1:], []byte(h.Protocol))

	metadata := make([]byte, 48) // 8 + 20 + 20
	copy(metadata[:8], h.Reserved[:])
	copy(metadata[8:28], h.InfoHash[:])
	copy(metadata[28:48], h.Identifier[:])

	return append(buffer, metadata...)
}

func (h *Handshake) Verify(hash [20]byte) error {
	switch {
	case h.Protocol != ProtocolName:
		return fmt.Errorf("invalid protocol %v", h.Protocol)
	case h.InfoHash != hash:
		return fmt.Errorf("invalid infohash %x", h.InfoHash)
	default:
		return nil
	}
}

func NewHandshake(hash, name [20]byte) *Handshake {
	return &Handshake{
		Protocol:   ProtocolName,
		Reserved:   [8]byte{},
		InfoHash:   hash,
		Identifier: name,
	}
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	lenBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	length := lenBuf[0]

	protocolbuf := make([]byte, length)
	_, err = io.ReadFull(r, protocolbuf)
	if err != nil {
		return nil, err
	}
	protocol := string(protocolbuf)

	var reservedBytes [8]byte
	_, err = io.ReadFull(r, reservedBytes[:])
	if err != nil {
		return nil, err
	}

	var hashBytes [20]byte
	_, err = io.ReadFull(r, hashBytes[:])
	if err != nil {
		return nil, err
	}

	var idBytes [20]byte
	_, err = io.ReadFull(r, idBytes[:])
	if err != nil {
		return nil, err
	}

	return &Handshake{
		Protocol:   protocol,
		Reserved:   reservedBytes,
		InfoHash:   hashBytes,
		Identifier: idBytes,
	}, nil
}
