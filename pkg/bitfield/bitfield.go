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

type Bitfield []byte

func (b Bitfield) Has(i int) bool {
	atByte := i / 8     // 8 pieces per byte
	byteOffset := i % 8 // offset in byte

	if atByte < 0 || atByte > len(b) {
		return false
	}

	return b[atByte]>>(7-byteOffset)&1 != 0
}

func (b Bitfield) Set(i int) {
	atByte := i / 8     // 8 pieces per byte
	byteOffset := i % 8 // offset in byte

	if atByte < 0 || atByte > len(b) {
		return
	}

	b[atByte] |= 1 << (7 - byteOffset)
}
