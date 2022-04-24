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
	"sort"
	"strings"
)

// field stores necessary data about a struct field required during
// marshalling and unmarshalling.
type field struct {
	index []int // index in struct

	// tag information
	name    string // bencode name
	options string // tag options
}

// contains checks if the receiver field contains the given tag.
func (f *field) contains(target string) bool {
	rest := f.options

	for {
		// get leading option from rest
		option, rest, _ := strings.Cut(rest, ",")

		// check if option equals target
		if option == target {
			return true
		}

		// check if no options are left
		if rest == "" {
			return false
		}
	}
}

// parseField parses a reflect.StructField and its tags as a field.
func parseField(f reflect.StructField) (field, bool) {
	// get bencode tag
	tag := f.Tag.Get("bencode")

	var name, options string

	// return false if field is to be ignored: "-"
	if tag == "-" {
		return field{}, false
	}

	// `bencode:"name,option1,option2"`
	name, options, _ = strings.Cut(tag, ",")

	// if tag does not specify name, use field name
	if name == "" {
		name = f.Name
	}

	return field{
		index:   f.Index,
		name:    name,
		options: options,
	}, true
}

// structFields stores necessary data about a struct's fields required
// during marshalling and unmarshalling.
type structFields struct {
	fields []field        // list of fields of the structure
	names  map[string]int // list of names to find exact match
}

// order sorts the fields slice of the structFields according to their
// names using lexicographical ordering.
func (s *structFields) order() {
	sort.Slice(s.fields, func(i, j int) bool {
		return s.fields[i].name < s.fields[j].name
	})
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
		f, ok := parseField(v.Type().Field(i))

		// if not ok, ignore field
		if !ok {
			continue
		}

		s.fields = append(s.fields, f) // add field to list
		s.names[f.name] = i            // store index as name
	}

	return s
}
