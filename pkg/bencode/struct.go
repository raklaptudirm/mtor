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
	"reflect"
	"strings"
)

// field stores necessary data about a struct field required during
// marshalling and unmarshalling.
type field struct {
	index []int // index in struct

	// tag information
	ignore    bool   // ignore field
	name      string // bencode name
	omitempty bool   // omit if empty
}

// parseField parses a reflect.StructField and its tags as a field.
func parseField(f reflect.StructField) field {
	// get bencode tag
	tag := f.Tag.Get("bencode")

	var name, options string
	var omitempty bool

	// check if field is to be ignored: "-"
	ignore := tag == "-"

	if !ignore {
		// `bencode:"name,omitempty"`
		name, options, _ = strings.Cut(tag, ",")

		// if tag does not specify name, use field name
		if name == "" {
			name = f.Name
		}

		// check if field is to be omitted if it is the zero value during
		// marshalling
		omitempty = options == "omitempty"
	}

	return field{
		index:     f.Index,
		ignore:    ignore,
		name:      name,
		omitempty: omitempty,
	}
}

// structFields stores necessary data about a struct's fields required
// during marshalling and unmarshalling.
type structFields struct {
	fields []field        // list of fields of the structure
	names  map[string]int // list of names to find exact match
}

// fields parses a reflect.Value of Kind Struct into a structFields value.
func fields(v reflect.Value) *structFields {
	// only reflect.Struct is supported
	if v.Kind() != reflect.Struct {
		panic("invalid type provided to fields()")
	}

	// init value
	s := &structFields{names: make(map[string]int)}

	n := v.NumField()
	// iterate through the fields
	for i := 0; i < n; i++ {
		f := parseField(v.Type().Field(i))

		s.fields = append(s.fields, f) // add field to list
		s.names[f.name] = i            // store index as name
	}

	return s
}
