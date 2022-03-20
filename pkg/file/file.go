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
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/raklaptudirm/mtor/pkg/torrent"
)

// Port is the port the client is listening on.
const Port = 6881

// file represents a .torrent metainfo file.
type file struct {
	Info     info   `bencode:"info"`     // info section of metainfo
	Announce string `bencode:"announce"` // tracker announce url

	Date    int64  `bencode:"creation date"` // creation timestamp
	Comment string `bencode:"comment"`       // free-form comment
	Author  string `bencode:"created by"`    // author of the metainfo
}

// info represents the info section of a metainfo file.
type info struct {
	// common fields
	PieceLen int    `bencode:"piece length"` // length of each piece
	Pieces   string `bencode:"pieces"`       // hash of each piece
	// file name in single-file torrent, directory name in multi-file torrent
	Name string `bencode:"name"`

	// single-file only
	Length int `bencode:"length"` // length of file in single-file torrent

	// multi-file only
	Files []singleFile `bencode:"files"` // files in multi-file torrent
}

// file represtents a single file in multi-file torrent.
type singleFile struct {
	Length int      `bencode:"length"` // length of the file
	Path   []string `bencode:"path"`   // path of the file
}

// Save saves the torrent as a file or directory, fetching pieces from the
// provided piece manager.
func (f *file) Save(pieces torrent.PieceManager, dst string) error {
	if f.isSingleFile() {
		return f.saveSingleFile(pieces, dst)
	}

	// TODO: add support for saving multifile torrents
	return nil
}

// saveSingleFile saves a single-file torrent as a file, fetching the pieces
// from the provided piece manager.
func (f *file) saveSingleFile(pieces torrent.PieceManager, dst string) error {
	length := len(f.Info.Pieces) / 20 // each hash is 20 bytes

	file, err := os.Create(path.Join(dst, f.Info.Name))
	if err != nil {
		return err
	}
	defer file.Close()

	// get each piece
	for i := 0; i < length; i++ {
		piece, err := pieces.Get(i)
		if err != nil {
			return err
		}

		// write each piece to the file
		_, err = file.Write(piece)
		if err != nil {
			return err
		}
	}
	return nil
}

// Torrent converts a file into a torrent.Torrent.
func (f *file) Torrent() (*torrent.Torrent, error) {
	hash, err := f.Info.hash()
	if err != nil {
		return nil, err
	}

	hashes, err := f.Info.hashes()
	if err != nil {
		return nil, err
	}

	// generate random user id
	var id [20]byte
	rand.Seed(time.Now().Unix())
	rand.Read(id[:])

	return &torrent.Torrent{
		Announce:    f.Announce,
		InfoHash:    hash,
		PieceHashes: hashes,
		PieceLength: f.Info.PieceLen,
		Length:      f.length(),
		Port:        Port,
		Name:        id,
	}, nil
}

// hash calculates the infohash of info.
func (i *info) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}

	// this is really bad code, fix the bad code
	// TODO: bad hack, remove as soon as possible
	b := removeExcess(buf.Bytes())
	return sha1.Sum(b), nil
}

// remove excess removes the files key from the bencode for single file
// torrents.
func removeExcess(buf []byte) []byte {
	res := make([]byte, len(buf)-9)

	// literally remove the files field
	copy(res, buf[:1])
	copy(res[1:], buf[10:])
	return res
}

// hashes returns an array containing the hash of each piece in the
// info.
func (i *info) hashes() ([][20]byte, error) {
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

func (f *file) length() int {
	if f.isSingleFile() {
		return f.Info.Length
	}

	length := 0
	for _, file := range f.Info.Files {
		length += file.Length
	}

	return length
}

func (f *file) isSingleFile() bool {
	return len(f.Info.Files) == 0
}

// Open opens a io.Reader as a .torrent metainfo file.
func Open(r io.Reader) (*file, error) {
	var f file

	err := bencode.Unmarshal(r, &f)
	if err != nil {
		return nil, err
	}

	return &f, nil
}
