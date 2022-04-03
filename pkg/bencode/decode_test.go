package bencode_test

import (
	"reflect"
	"testing"

	"laptudirm.com/x/mtor/pkg/bencode"
)

var tests = []struct {
	in  string
	ptr any
	out any
	err error
}{
	// basic values
	{in: "i123e", ptr: new(int), out: 123},
	{in: "i-123e", ptr: new(int), out: -123},
	{in: "i0e", ptr: new(int), out: 0},
	{in: "0:", ptr: new(string), out: ""},
	{in: "3:cat", ptr: new(string), out: "cat"},
	{in: "le", ptr: new([]any), out: *new([]any)},
	{in: "li123e3:cate", ptr: new([]any), out: []any{int64(123), "cat"}},
	{in: "lli123e3:catee", ptr: new([][]any), out: [][]any{{int64(123), "cat"}}},
	{in: "de", ptr: new(map[string]any), out: make(map[string]any)},
	{in: "d3:cati123e3:dogi-123ee", ptr: new(map[string]any), out: map[string]any{"cat": int64(123), "dog": int64(-123)}},
	{in: "d1:ad1:ai123e1:b3:catee", ptr: new(any), out: map[string]any{"a": map[string]any{"a": int64(123), "b": "cat"}}},
}

func TestDecode(t *testing.T) {
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			err := bencode.Unmarshal([]byte(test.in), test.ptr)

			if err != test.err {
				t.Errorf("Unmarshal(%#v): returned error %v did not match %v", test.in, err, test.err)
				return
			}

			v := reflect.ValueOf(test.ptr)
			c := v.Elem().Interface()
			if !reflect.DeepEqual(c, test.out) {
				t.Errorf("Unmarshal(%#v): data %#v did not match %#v", test.in, c, test.out)
			}
		})
	}
}
