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

package bencode

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"laptudirm.com/x/mtor/pkg/bencode/scanner"
	"laptudirm.com/x/mtor/pkg/bencode/token"
)

// Unmarshal unmarshals bencode data into v.
func Unmarshal(data []byte, v any) error {
	d := &decoder{scanner: scanner.New(data)}
	return d.unmarshal(v)
}

// Valid checks if the provided data is valid bencode.
func Valid(data []byte) bool {
	return scanner.Valid(data)
}

// decoder is a state machine which goes through the tokens generated by its
// scanner and unmarshals them into the provided destination.
type decoder struct {
	scanner *scanner.Scanner

	offset int         // offset in token stream
	curr   token.Token // current token
}

// syntaxPanicMsg is the message used to panic when the decoder receives
// invalid tokens from the scanner without an error.
var syntaxPanicMsg = "bencode: invalid syntax without scanner error"

// UnmarshalTypeError represents an error where a bencode type is being
// unmarshalled into an invalid go type.
type UnmarshalTypeError struct {
	Value  string       // the bencode type
	Type   reflect.Type // the go type
	Offset int          // offset of the literal
}

func (e *UnmarshalTypeError) Error() string {
	return fmt.Sprintf("bencode: cannot unmarshal %s into Go value of type %s", e.Value, e.Type)
}

// InvalidUnmarshalError represents an error where data is getting
// unmarshalled into an invalid go type.
type InvalidUnmarshalError struct {
	Type reflect.Type // the invalid type
}

func (e *InvalidUnmarshalError) Error() string {
	switch {
	case e.Type == nil:
		return "bencode: Unmarshal(nil)"
	case e.Type.Kind() != reflect.Pointer:
		return fmt.Sprintf("bencode: Unmarshal(non-pointer %s)", e.Type)
	default:
		return fmt.Sprintf("bencode: Unmarshal(nil %s)", e.Type)
	}
}

// Unmarshal scans the next value from the decoder and unmarshals it into v.
func (d *decoder) unmarshal(v any) error {
	rv := reflect.ValueOf(v)
	// check if rv is valid
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidUnmarshalError{rv.Type()}
	}

	// tokenize and verify the bencode data
	err := d.scanner.Valid()
	if err != nil {
		return err
	}

	// unmarshal the next value into rv
	return d.value(rv)
}

// value unmarshals the next value from the decoder's token stream into v.
func (d *decoder) value(v reflect.Value) error {
	switch d.peek().Type {
	case token.DICT:
		return d.dict(v)
	case token.LIST:
		return d.list(v)
	case token.NUMBER:
		return d.number(v)
	case token.STRING:
		return d.string(v)
	default:
		panic(syntaxPanicMsg)
	}
}

// valueInterface is like value but instead of unmarshalling into a variable
// it unmarshals it into an any value and returns it.
func (d *decoder) valueInterface() (any, error) {
	switch d.peek().Type {
	case token.DICT:
		return d.dictInterface()
	case token.LIST:
		return d.listInterface()
	case token.NUMBER:
		return d.numberInterface()
	case token.STRING:
		return d.stringInterface()
	default:
		panic(syntaxPanicMsg)
	}
}

// dict unmarshals a dictionary from the decoder's token stream into v.
func (d *decoder) dict(v reflect.Value) error {
	v, ok := indirect(v)
	if !ok {
		return &UnmarshalTypeError{Value: "dictionary", Type: v.Type(), Offset: d.curr.Offset}
	}

	// fs stores the fields of v if it is a struct
	var fs *structFields

	switch v.Kind() {
	case reflect.Map:
		// only maps with string keys are supported
		if v.Type().Key().Kind() != reflect.String {
			return &UnmarshalTypeError{Value: "dictionary", Type: v.Type(), Offset: d.curr.Offset}
		}

		// if map is nil, allocate a new map
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	case reflect.Struct:
		// store field info into fs
		fs = fields(v)
	case reflect.Interface:
		if isAny(v) {
			value, err := d.dictInterface()
			if err != nil {
				return err
			}

			v.Set(reflect.ValueOf(value))
			return nil
		}

		// only interface{} is supported
		fallthrough
	default:
		return &UnmarshalTypeError{Value: "dictionary", Type: v.Type(), Offset: d.curr.Offset}
	}

	// consume the leading DICT token
	d.mustConsume(token.DICT)

	// loop while there is a STRING key
	for d.consume(token.STRING) {
		// extract key string from literal
		key := d.curr.RawString()

		switch v.Kind() {
		case reflect.Map:
			// allocate temporary pointer
			f := reflect.New(v.Type().Elem())

			err := d.value(f)
			if err != nil {
				return err
			}

			v.SetMapIndex(reflect.ValueOf(key), f.Elem())
		case reflect.Struct:
			// try to find exact match
			if i, ok := fs.names[key]; ok {
				if err := d.value(v.Field(i)); err != nil {
					return err
				}

				break
			}

			// exact match not found, try iterating to find case folded match
			for _, f := range fs.fields {
				if strings.EqualFold(key, f.name) {
					if err := d.value(v.FieldByIndex(f.index)); err != nil {
						return err
					}

					break
				}
			}
		}
	}

	// consume END token
	d.mustConsume(token.END)
	return nil
}

// dictInterface is like dict but instead of unmarshalling into a variable
// it unmarshals it into an any value and returns it.
func (d *decoder) dictInterface() (any, error) {
	// consume the leading DICT token
	d.mustConsume(token.DICT)

	v := make(map[string]any)

	// loop while there is a STRING key
	for d.consume(token.STRING) {
		// extract key string from literal
		key := d.curr.RawString()
		value, err := d.valueInterface()
		if err != nil {
			return nil, err
		}

		v[key] = value
	}

	// consume END token
	d.mustConsume(token.END)
	return v, nil
}

// list unmarshals a list from the decoder's token stream into v.
func (d *decoder) list(v reflect.Value) error {
	v, ok := indirect(v)
	if !ok {
		return &UnmarshalTypeError{Value: "list", Type: v.Type(), Offset: d.curr.Offset}
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice:
	case reflect.Interface:
		// switch to interface mode
		if isAny(v) {
			value, err := d.listInterface()
			if err != nil {
				return err
			}

			v.Set(reflect.ValueOf(value))
			return nil
		}

		// only interaface{} is supported
		fallthrough
	default:
		return &UnmarshalTypeError{Value: "list", Type: v.Type(), Offset: d.curr.Offset}
	}

	// consume leading LIST token
	d.mustConsume(token.LIST)

	// loop while there are values
	for i := 0; !d.match(token.END) && !d.match(token.ILLEGAL); i++ {
		// only slices can be grown
		if v.Kind() == reflect.Slice {
			// grow slice if necessary
			if i >= v.Cap() {
				newcap := v.Cap() + v.Cap()/2
				if newcap < 8 {
					newcap = 8
				}

				newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
				reflect.Copy(newv, v)
				v.Set(newv)
			}

			if i >= v.Len() {
				// capacity has been grown enough
				// to increase length
				v.SetLen(i + 1)
			}
		}

		if i < v.Len() {
			if err := d.value(v.Index(i)); err != nil {
				return err
			}
		} else {
			// end of fixed length array, skip rest
			if err := d.value(reflect.Value{}); err != nil {
				return err
			}
		}
	}

	// consume END token
	d.mustConsume(token.END)
	return nil
}

// listInterface is like list but instead of unmarshalling into a variable
// it unmarshals it into an any value and returns it.
func (d *decoder) listInterface() (any, error) {
	// consume leading LIST token
	d.mustConsume(token.LIST)

	var v []any

	// loop while end is not reached
	for !d.consume(token.END) {
		value, err := d.valueInterface()
		if err != nil {
			return nil, err
		}

		v = append(v, value)
	}

	return v, nil
}

// number unmarshals a number from the decoder's token stream into v.
func (d *decoder) number(v reflect.Value) error {
	// consume the NUMBER token
	d.mustConsume(token.NUMBER)

	// extract number from number literal
	literal := d.curr.RawNumber()

	v, ok := indirect(v)
	if !ok {
		return &UnmarshalTypeError{Value: "number", Type: v.Type(), Offset: d.curr.Offset}
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// parse literal as an int
		n, err := strconv.ParseInt(literal, 10, 64)
		if err == nil && !v.OverflowInt(n) {
			v.SetInt(n)
			return nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// parse literal as uint
		n, err := strconv.ParseUint(literal, 10, 64)
		if err == nil && !v.OverflowUint(n) {
			v.SetUint(n)
			return nil
		}

	case reflect.Float32, reflect.Float64:
		// parse literal as float
		n, err := strconv.ParseFloat(literal, v.Type().Bits())
		if err == nil && !v.OverflowFloat(n) {
			v.SetFloat(n)
			return nil
		}

	case reflect.Interface:
		if !isAny(v) {
			// only interface{} is supported
			break
		}

		// parse as int by default
		n, err := strconv.ParseInt(literal, 10, 64)
		if err != nil {
			return err
		}

		v.Set(reflect.ValueOf(n))
		return nil
	}

	return &UnmarshalTypeError{Value: "number", Type: v.Type(), Offset: d.curr.Offset}
}

// numberInterface is like number but instead of unmarshalling into a
// variable it unmarshals it into an any value and returns it.
func (d *decoder) numberInterface() (any, error) {
	// consume the NUMBER token
	d.mustConsume(token.NUMBER)

	lit := d.curr.RawNumber()
	return strconv.ParseInt(lit, 10, 64)
}

// string unmarshals a string from the decoder's token stream into v.
func (d *decoder) string(v reflect.Value) error {
	// consume the STRING token
	d.mustConsume(token.STRING)

	// extract string bytes from string literal
	literal := d.curr.RawString()

	v, ok := indirect(v)
	if !ok {
		return &UnmarshalTypeError{Value: "string", Type: v.Type(), Offset: d.curr.Offset}
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(literal)
		return nil

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// []byte or []uint8
			v.SetBytes([]byte(literal))
			return nil
		}

	case reflect.Interface:
		// only interface{} is supported
		if isAny(v) {
			v.Set(reflect.ValueOf(literal))
			return nil
		}
	}

	return &UnmarshalTypeError{Value: "string", Type: v.Type(), Offset: d.curr.Offset}
}

// stringInterface is like string but instead of unmarshalling into a
// variable it unmarshals it into an any value and returns it.
func (d *decoder) stringInterface() (any, error) {
	// consume the STRING token
	d.mustConsume(token.STRING)

	// extract string bytes from string literal
	return d.curr.RawString(), nil
}

// mustConsume tries to consume a token of type t. If it can't it panics
// with syntaxPanicMsg.
func (d *decoder) mustConsume(t token.Type) {
	if !d.consume(t) {
		panic(syntaxPanicMsg)
	}
}

// consume tries to consume a token of type t, and returns whether it
// succeeded or not.
func (d *decoder) consume(t token.Type) bool {
	if !d.match(t) {
		return false
	}

	d.next()
	return true
}

// next consumes the next token from the token stream.
func (d *decoder) next() {
	d.curr = d.peek()

	if !d.atEnd() {
		d.offset++
	}
}

// match checks if the next token matches the type t.
func (d *decoder) match(t token.Type) bool {
	return d.peek().Type == t
}

// peek returns the next token from the token stream. It returns a
// token.ILLEGAL if it reaches the end of the token stream.
func (d *decoder) peek() token.Token {
	if d.atEnd() {
		return token.Token{Type: token.ILLEGAL}
	}

	return d.scanner.Tokens[d.offset]
}

// atEnd checks whether the end of the token stream has been reached.
func (d *decoder) atEnd() bool {
	return d.offset >= len(d.scanner.Tokens)
}

// indirect indirects the value v while it is a pointer. When it reaches
// a non pointer, it returns v along with whether v is a valid settable
// value.
func indirect(v reflect.Value) (reflect.Value, bool) {
	v0 := v
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		v = reflect.Indirect(v)
	}

	if v.IsValid() && v.CanSet() {
		return v, true
	}

	return v0, false
}

// isAny checks if the provided reflect.Value has a type of any.
func isAny(v reflect.Value) bool {
	return v.Kind() == reflect.Interface && v.NumMethod() == 0
}
