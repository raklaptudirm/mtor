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

package manager

import (
	"errors"
	"fmt"
	"os"
	"path"
)

// piece represents the piece manager.
type piece struct {
	src string // storage directory
}

// ErrManagerClosed is returned when the manager is not initialized,
// or closed.
var ErrManagerClosed = errors.New("the manager is closed")

// Init initializes the manager.
func (p *piece) Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// create storage directory
	dir, err := os.MkdirTemp(home, "mtor pieces ")
	if err != nil {
		return err
	}

	p.src = dir
	return nil
}

// Put stores a piece in the manager.
func (p *piece) Put(i int, buf []byte) error {
	if p.isClosed() {
		return ErrManagerClosed
	}

	file := path.Join(p.src, fmt.Sprintf("%x", i))
	return os.WriteFile(file, buf, 0600)
}

// Get fetches a piece from the manager.
func (p *piece) Get(i int) ([]byte, error) {
	if p.isClosed() {
		return nil, ErrManagerClosed
	}

	file := path.Join(p.src, fmt.Sprintf("%x", i))
	return os.ReadFile(file)
}

// Close closes the manager.
func (p *piece) Close() error {
	if p.isClosed() {
		return ErrManagerClosed
	}

	// free space
	return os.RemoveAll(p.src)
}

// isClosed checks if the manager is closed.
func (p *piece) isClosed() bool {
	return p.src == ""
}

// New returns a new and un-initialzed instance of the manager.
func New() *piece {
	return &piece{}
}
