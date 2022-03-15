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

package bitfield

// Bitfield represents a single mutable bitfield.
type Bitfield []byte

// Has checks if the ith bit of the bitfield b is set.
func (b Bitfield) Has(i int) bool {
	atByte := i / 8     // 8 pieces per byte
	byteOffset := i % 8 // offset in byte

	// index is outside of bitfield's range
	if atByte < 0 || atByte > len(b) {
		return false
	}

	// b[atByte]:        get byte with bit i
	// >>(7-byteOffset): get rid of bits after i
	// &1:               get rid of bits before i
	return b[atByte]>>(7-byteOffset)&1 != 0
}

// Set sets the ith bit of the bitfield b.
func (b Bitfield) Set(i int) {
	atByte := i / 8     // 8 pieces per byte
	byteOffset := i % 8 // offset in byte

	if atByte < 0 || atByte > len(b) {
		return
	}

	// set ith bit
	b[atByte] |= 1 << (7 - byteOffset)
}
