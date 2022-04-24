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

// Package bitfield implements a Bitfield structure, which can efficiently
// hold multiple flags values as a byte slice.
package bitfield

// Bitfield represents a single mutable bitfield.
type Bitfield struct {
	bits []byte
}

// New creates a new Bitfield from the provided bits.
func New(bits []byte) Bitfield {
	return Bitfield{bits: bits}
}

// Has checks if the ith bit of the bitfield b is set.
func (b Bitfield) Has(i int) bool {
	atByte, byteOffset, inRange := b.indexOf(i)
	if !inRange {
		return false
	}

	// b[atByte]:        get byte with bit i
	// >>(7-byteOffset): get rid of bits after i
	// &1:               get rid of bits before i
	return b.bits[atByte]>>(7-byteOffset)&1 != 0
}

// Set sets the ith bit of the bitfield b.
func (b Bitfield) Set(i int) {
	atByte, byteOffset, inRange := b.indexOf(i)
	if !inRange {
		return
	}

	// set ith bit
	b.bits[atByte] |= 1 << (7 - byteOffset)
}

// Clear sets the ith bit of the bitfield b.
func (b Bitfield) Clear(i int) {
	atByte, byteOffset, inRange := b.indexOf(i)
	if !inRange {
		return
	}

	// clear ith bit
	b.bits[atByte] &^= 1 << (7 - byteOffset)
}

// indexOf returns the byte index, byte offset, and whether i is inside the
// bitfield or not.
func (b Bitfield) indexOf(i int) (atByte int, byteOffset int, inRange bool) {
	atByte = i / 8     // 8 pieces per byte
	byteOffset = i % 8 // offset in byte
	inRange = atByte > 0 && atByte < len(b.bits)
	return
}
