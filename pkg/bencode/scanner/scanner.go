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

package scanner

import (
	"fmt"
	"strconv"
	"unicode"

	"laptudirm.com/x/mtor/pkg/bencode/token"
)

// New creates a new Scanner with the provided data and returns a pointer
// to it.
func New(data []byte) *Scanner {
	return &Scanner{Data: data}
}

// Valid checks if the provided data is valid bencode. It returns true if
// calling s.Valid on a Scanner initialized with the provided data does not
// return any error.
func Valid(data []byte) bool {
	s := New(data) // create a new scanner
	return s.Valid() == nil
}

// Scanner is scanning state machine on bencode data. Callers will call
// s.Next or s.Valid to tokenize the source. Scanner will go through the
// data, while checking syntax, and appending all emitted tokens to the
// s.Tokens array.
type Scanner struct {
	Data []byte // data to scan

	ch       rune        // current byte
	offset   int         // start of current token
	rdOffset int         // current read offset
	last     token.Token // the last emitted token

	Tokens []token.Token // output array
}

const eof = -1 // end of file

// SyntaxError represents a bencode syntax error.
type SyntaxError struct {
	msg    string // error message
	Offset int    // error position
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%d: %s", e.Offset, e.msg)
}

// Next scans the next bencode value from the scanner's source. s.Next
// will not return an error if there is other data after the first top
// level bencode value. See s.Valid for that.
func (s *Scanner) Next() error {
	return s.scanNext()
}

// scanNext tries to scan the next bytes in the scanner as a bencode value.
// It also checks for any syntax errors.
func (s *Scanner) scanNext() error {
	r := s.peek()
	switch {
	case r == 'd':
		return s.scanDict()
	case r == 'l':
		return s.scanList()
	case r == 'i':
		return s.scanInt()
	case unicode.IsDigit(r):
		return s.scanStr()
	case r == eof:
		return s.error("unexpected end of input")
	default:
		return s.error("looking for beginning of value")
	}
}

// scanDict tries to scan the next bytes in the scanner as a bencode
// dictionary. It also checks for any syntax errors. A proper bencode
// dictionary has the format:
// d <string key> <value>... e
func (s *Scanner) scanDict() error {
	// try to consume opening 'd'
	if !s.consume('d') {
		return s.error("looking for beginning of dictionary")
	}

	s.emit(token.DICT)

	// prev stores the previous key to check for proper ordering of the
	// dictionary's keys, while first records if this is the first key,
	// which can be anything
	prev, first := "", true

	// exit only on 'e' or eof
	for r := s.peek(); r != 'e' && r != eof; r = s.peek() {
		// scan the key string
		err := s.scanStr()
		if err != nil {
			return err
		}

		// get the raw string literal
		key := s.last.RawString()

		// key is not the first key and is lexicographically the same or
		// below the previous key, so ordering is improper
		if !first && key <= prev {
			return &SyntaxError{
				msg:    fmt.Sprintf("improper ordering of dictionary keys, %#v seen after %#v", key, prev),
				Offset: s.last.Offset,
			}
		}

		// update key data
		prev, first = key, false

		// scan the key's value
		err = s.scanNext()
		if err != nil {
			return err
		}
	}

	// try to consume ending 'e'
	if !s.consume('e') {
		// loop exits only on 'e' or eof, so r == eof
		return s.error("unexpected end of input while scanning dictionary")
	}

	s.emit(token.END)
	return nil
}

// scanList tries to scan the next bytes in the scanner as a bencode list.
// It also checks for any syntax errors. A proper bencode list has the
// format: l <value>... e
func (s *Scanner) scanList() error {
	// try to consume opening 'l'
	if !s.consume('l') {
		return s.error("looking for beginning of list")
	}

	s.emit(token.LIST)

	// exit only of 'e' or eof
	for r := s.peek(); r != 'e' && r != eof; r = s.peek() {
		// scan next value
		err := s.scanNext()
		if err != nil {
			return err
		}
	}

	// try to consume ending 'e'
	if !s.consume('e') {
		// loop exits only on 'e' or eof, so r == eof
		return s.error("unexpected end of input while scanning list")
	}

	s.emit(token.END)
	return nil
}

// scanInt tries to scan the next bytes in the scanner as a bencode integer.
// It also checks for any syntax errors. A proper bencode integer has the
// format: i <number> e
func (s *Scanner) scanInt() error {
	// try to consume opening 'i'
	if !s.consume('i') {
		return s.error("looking for beginning of integer")
	}

	// scan enclosed number with delimeter 'e'
	err := s.scanNumber('e')
	if err != nil {
		return err
	}

	s.emit(token.NUMBER)
	return nil
}

// scanStr tries to scan the next bytes in the scanner as a bencode string.
// It also checks for any syntax errors. A proper bencode string has the
// format: <length>:<string bytes>
func (s *Scanner) scanStr() error {
	// strings start with a positive number
	if !unicode.IsDigit(s.peek()) {
		return s.error("looking for beginning of string")
	}

	// scan length number with delimeter ':'
	err := s.scanNumber(':')
	if err != nil {
		return err
	}

	// parse length from string
	lenStr := s.literal()
	lenStr = lenStr[:len(lenStr)-1]
	length, err := strconv.Atoi(string(lenStr))
	if err != nil {
		// out of range errors
		return err
	}

	// check if length takes us past scanners end
	if len(s.Data)-s.rdOffset < length {
		s.rdOffset = len(s.Data)
		return s.error("unexpected end of input while scanning string")
	}

	s.rdOffset += length

	s.emit(token.STRING)
	return nil
}

// scanNumber tries to scan the next bytes in the scanner as a bencode number.
// It is used to scan the bencode string length and the number in a bencode
// integer. It also checks for any syntax errors. A proper bencode number has
// the format: 0 | [-] non_0_digit { digit }
func (s *Scanner) scanNumber(d rune) error {
	// consume '-' if any for negative numbers
	negative := s.consume('-')

	r := s.peek()
	switch {
	case r == d: // no number found
		return s.error("looking for a number")
	case !unicode.IsDigit(r): // non number byte
		return s.runeError("in number literal")
	case r == '0':
		// negative numbers can't start with a 0
		if negative {
			return s.error("leading 0 in negative number literal")
		}

		s.next()
		// leading 0s are invalid, so try to consume delimeter
		if !s.consume(d) {
			// context aware error reporting :D
			if unicode.IsDigit(s.peek()) {
				return s.error("leading zero in number")
			}

			return s.runeError("in number literal")
		}

		return nil
	}

	// exit only on delimeter or eof
	for r := s.peek(); r != d && r != eof; r = s.peek() {
		if !unicode.IsDigit(r) {
			// invalid character in number literal
			return s.runeError("in number literal")
		}

		s.next()
	}

	// try to scan ending delimeter
	if !s.consume(d) {
		// loop exits only on delimeter or eof, so r == eof
		return s.error("unexpected end of input while scanning number")
	}

	return nil
}

// Valid scans the next bencode value from the scanner and reports an error
// if the data is not valid bencode. It returns nil for all valid bencode data.
// s.Valid, unlike s.Next, will return an error if there is other data present
// after the first top-level bencode value.
func (s *Scanner) Valid() error {
	err := s.Next()
	if err != nil {
		return err
	}

	// check if end of data has been reached
	if !s.atEnd() {
		return s.runeError("after top-level value")
	}

	return nil
}

// literal returns the bytes present between the scanner's offset and read
// offset, or the currently scanned bytes.
func (s *Scanner) literal() []byte {
	return s.Data[s.offset:s.rdOffset]
}

// reset sets the scanner's offset to it's read offset as preparation to start
// scanning the next token.
func (s *Scanner) reset() {
	s.offset = s.rdOffset
}

// consume checks if the next byte in the source matches the provided rune. If
// it does, it calls s.next and returns true. Otherwise it returns false.
func (s *Scanner) consume(r rune) bool {
	if s.peek() == r {
		s.next()
		return true
	}

	return false
}

// next increases the scanner's read offset by 1 and stores the current byte in
// s.ch if the scanner is not at the end of it's data otherwise, it sets s.ch to
// eof(-1) and does not increase the read offset.
func (s *Scanner) next() {
	s.ch = s.peek()

	if s.ch != eof {
		s.rdOffset++
	}
}

// peek returns the byte that will be read next by the scanner. If the scanner is
// at the end of it's input, peek returns eof(-1).
func (s *Scanner) peek() rune {
	if s.atEnd() {
		return eof
	}

	return rune(s.Data[s.rdOffset])
}

// atEnd reports if the scanner has reached the end of it's input.
func (s *Scanner) atEnd() bool {
	return s.rdOffset >= len(s.Data)
}

// runeError returns a new SyntaxError with the message:
// invalid character <character> <context>
func (s *Scanner) runeError(msg string) error {
	return s.error(fmt.Sprintf("invalid character %q %s", s.peek(), msg))
}

// error returns a new SyntaxError with the provided message and the current read
// offset.
func (s *Scanner) error(msg string) error {
	return &SyntaxError{msg, s.rdOffset}
}

// emit creates a new token.Token with the provided type, the currently
// scanned literal and at the current offset. It appends the new token to
// the scanner's Tokens array and calls s.reset.
func (s *Scanner) emit(t token.Type) {
	tok := token.Token{
		Type:    t,
		Literal: string(s.literal()),
		Offset:  s.offset,
	}

	s.Tokens = append(s.Tokens, tok)
	s.last = tok
	s.reset()
}
