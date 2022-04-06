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
	"sort"
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
		return e.marshalMap(v)
	case reflect.Struct:
		return e.marshalStruct(v)
	case reflect.String:
		e.marshalString(v)
	case reflect.Array, reflect.Slice:
		return e.marshalArray(v)
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

// marshalMap marshals a map into the encoder.
func (e *encoder) marshalMap(v reflect.Value) error {
	if v.Kind() != reflect.Map {
		panic("non-map input to encoder.marshalMap()")
	}

	// key should be of string type
	if v.Type().Key().Kind() != reflect.String {
		return &UnsupportedTypeError{v.Type()}
	}

	// write leading 'd'
	e.data += "d"

	// get sorted key list
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	// marshal elements
	for _, key := range keys {
		// marshal key
		e.marshalString(key)

		// marshal value
		err := e.marshal(v.MapIndex(key))
		if err != nil {
			return err
		}
	}

	// write ending 'e'
	e.data += "e"
	return nil
}

// marshalStruct marshals a struct into the encoder.
func (e *encoder) marshalStruct(v reflect.Value) error {
	if v.Kind() != reflect.Struct {
		panic("non-struct input to encoder.marshalStruct()")
	}

	// write leading 'd'
	e.data += "d"

	// get sorted key list
	keys := fields(v)
	keys.order()

	// marshal elements
	for _, key := range keys.fields {
		if key.ignore {
			continue
		}

		d := v.FieldByIndex(key.index)

		if key.omitempty && isEmpty(d) {
			continue
		}

		// marshal key
		e.marshalString(reflect.ValueOf(key.name))

		// marshal value
		err := e.marshal(d)
		if err != nil {
			return err
		}
	}

	// write ending 'e'
	e.data += "e"
	return nil
}

// isEmpty checks if the value is empty and should be omitted. An empty
// value is defined as 0, a nil pointer, a nil interface value, and any
// empty array, slice, map, or string.
func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
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

// marshalArray marshals an array or slice into the encoder.
func (e *encoder) marshalArray(v reflect.Value) error {
	switch v.Kind() {
	// check if v is array or slice
	case reflect.Array, reflect.Slice:
		// write leading 'l'
		e.data += "l"

		length := v.Len()
		for i := 0; i < length; i++ {
			// marshal each element
			err := e.marshal(v.Index(i))
			if err != nil {
				return err
			}
		}

		// write ending 'e'
		e.data += "e"
		return nil
	default:
		panic("non-array input to encoder.marshalArray()")
	}
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
