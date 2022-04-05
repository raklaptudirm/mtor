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
)

// Marshal marshals v into a bencode string.
func Marshal(v any) (string, error) {
	e := &encoder{}
	err := e.marshal(reflect.ValueOf(v))
	return e.data, err
}

// encoder stores the current state of the marshalling.
type encoder struct {
	data string // result string
}

// UnsupportedTypeError is returned by Marshal when an unsupported go type is
// marshalled.
type UnsupportedTypeError struct {
	Type reflect.Type // the go type
}

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("bencode: unsupported type %s", e.Type)
}

// marshal marshals v into the encoder e and returns an error if any.
func (e *encoder) marshal(v reflect.Value) error {
marshal:
	switch v.Kind() {
	case reflect.Map:
		// TODO: e.marshalMap(v)
	case reflect.Struct:
		// TODO: e.marshalStruct(v)
	case reflect.String:
		e.marshalString(v)
	case reflect.Array, reflect.Slice:
		// TODO: e.marshalArray(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.marshalInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.marshalUint(v)
	case reflect.Pointer, reflect.Interface:
		v = v.Elem()
		goto marshal
	default:
		return &UnsupportedTypeError{v.Type()}
	}

	return nil
}

// marshalString marshals a string into the encoder.
func (e *encoder) marshalString(v reflect.Value) {
	if v.Kind() != reflect.String {
		panic("non-string input to encoder.marshalString()")
	}

	str := v.String()
	// <length>:<raw bytes>
	e.data += fmt.Sprintf("%d:%s", len(str), str)
}

// marshalInt marshals an int type into the encoder.
func (e *encoder) marshalInt(v reflect.Value) {
	// i<number>e
	e.data += fmt.Sprintf("i%de", v.Int())
}

// marshalUint marshals an uint type int the encoder.
func (e *encoder) marshalUint(v reflect.Value) {
	// i<number>e
	e.data += fmt.Sprintf("i%de", v.Uint())
}
