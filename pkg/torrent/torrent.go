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

// Identifier generates a random client identifier for use.
func Identifier() [20]byte {
	var id [20]byte
	rand.Read(id[:])

	return id
}
