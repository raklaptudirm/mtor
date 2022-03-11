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

package file

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"

	"github.com/jackpal/bencode-go"
	"github.com/raklaptudirm/mtor/pkg/torrent"
)

// Port is the port the client is listening on.
const Port = 6881

// File represents a .torrent metainfo file.
type File struct {
	Info     TorrentInfo `bencode:"info"`     // info section of metainfo
	Announce string      `bencode:"announce"` // tracker announce url

	Date    int64  `bencode:"creation date"` // creation timestamp
	Comment string `bencode:"comment"`       // free-form comment
	Author  string `bencode:"created by"`    // author of the metainfo
}

// TorrentInfo represents the info section of a metainfo file.
type TorrentInfo struct {
	PieceLen int    `bencode:"piece length"` // length of each piece
	Pieces   string `bencode:"pieces"`       // hash of each piece
	Name     string `bencode:"name"`         // name of file
	Length   int    `bencode:"length"`       // length of file
	// TODO: add multi-file support
}

// Open opens a io.Reader as a .torrent metainfo file.
func Open(r io.Reader) (*File, error) {
	var f File

	err := bencode.Unmarshal(r, &f)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// Torrent converts a File into a torrent.Torrent.
func (f *File) Torrent() (*torrent.Torrent, error) {
	hash, err := f.Info.hash()
	if err != nil {
		return nil, err
	}

	hashes, err := f.Info.hashes()
	if err != nil {
		return nil, err
	}

	return &torrent.Torrent{
		Announce:    f.Announce,
		InfoHash:    hash,
		PieceHashes: hashes,
		PieceLength: f.Info.PieceLen,
		Length:      f.Info.Length,
		Port:        Port,
		Name:        torrent.Identifier(),
	}, nil
}

// hash calculates the infohash of TorrentInfo.
func (i *TorrentInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}

	return sha1.Sum(buf.Bytes()), nil
}

// hashes returns an array containing the hash of each piece in the
// TorrentInfo.
func (i *TorrentInfo) hashes() ([][20]byte, error) {
	buffer := []byte(i.Pieces)
	length := len(buffer)
	if length%20 != 0 {
		return nil, fmt.Errorf("malformed piece hash string of length %v", length)
	}

	n := length / 20
	hashes := make([][20]byte, n)

	for i := 0; i < n; i++ {
		copy(hashes[i][:], buffer[i*20:(i+1)*20])
	}
	return hashes, nil
}
