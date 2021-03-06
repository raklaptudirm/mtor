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

package token

import (
	"fmt"
	"strings"
)

// Type indicates the type of a Token.
type Type int

const (
	ILLEGAL = iota

	NUMBER // i123e
	STRING // 3:cat

	DICT // d
	LIST // l

	END // e
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",

	NUMBER: "NUMBER",
	STRING: "STRING",

	DICT: "d",
	LIST: "l",

	END: "e",
}

// String converts a Type into a readable string from the tokens array if it
// is present in it. Otherwise, it formats it as token(<index>).
func (tok Type) String() string {
	s := ""
	if 0 <= tok && tok < Type(len(tokens)) {
		s = tokens[tok]
	}
	if s == "" {
		s = fmt.Sprintf("token(%d)", tok)
	}
	return s
}

// Token represents a token from a bencode source. It is used by the parser
// to parse the source into meaningful structures.
type Token struct {
	Type    Type   // type of the token
	Literal string // the literal representation from the source
	Offset  int    // the offest of the token in the source
}

// RawString extracts the raw string from a bencode string literal.
//
// Examples:
// 3:cat -> cat
// 4:boat -> boat
func (t *Token) RawString() string {
	if t.Type != STRING {
		panic("invalid token type in receiver to token.String()")
	}

	_, s, ok := strings.Cut(t.Literal, ":")
	if !ok {
		panic("invalid string literal without ':'")
	}

	return s
}

// RawNumber removes the start and end markers from a bencode number
// literal.
//
// Examples:
// i123e -> 123
// i-12e -> -12
// i0e -> 0
func (t *Token) RawNumber() string {
	if t.Type != NUMBER {
		panic("invalid token type in receiver to token.Number()")
	}

	return t.Literal[1 : len(t.Literal)-1]
}
