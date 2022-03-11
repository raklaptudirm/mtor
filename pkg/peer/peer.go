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
	"encoding/binary"
	"fmt"
	"net"
)

// Peer represents a torrent peer.
type Peer struct {
	IP   net.IP // ip of the peer
	Port uint16 // port of the peer
}

// String converts Peer to a string with the format ip:port.
func (p Peer) String() string {
	return fmt.Sprintf("%s:%v", p.IP, p.Port)
}

// Unmarshal parses peers from a byte array.
func Unmarshal(buffer []byte) ([]Peer, error) {
	const peerLen = 6

	length := len(buffer)
	number := length / peerLen
	if length%peerLen != 0 {
		return nil, fmt.Errorf("malformed peer list of length %v", length)
	}

	peers := make([]Peer, number)
	for i := 0; i < number; i++ {
		offset := i * peerLen
		peers[i].IP = net.IP(buffer[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(buffer[offset+4 : offset+6])
	}
	return peers, nil
}
