package scanner_test

import (
	"testing"

	"laptudirm.com/x/mtor/pkg/bencode/scanner"
)

var validTests = []struct {
	input string
	valid bool
}{
	// no value
	{"", false},

	// non-closed value
	{"d", false},
	{"l", false},
	{"i", false},
	{"1", false},

	// closed multiple times
	{"dee", false},
	{"lee", false},
	{"iee", false},

	// data missing
	{"ie", false},
	{"1:", false},

	// proper values
	{"de", true},
	{"le", true},
	{"i1e", true},
	{"i-1e", true},
	{"i0e", true},
	{"0:", true},
	{"1:a", true},

	// invalid values
	{"i01e", false},
	{"i-0e", false},

	// multiple top-level values
	{"dede", false},
}

func TestValid(t *testing.T) {
	for _, test := range validTests {
		t.Run(test.input, func(t *testing.T) {
			valid := scanner.Valid([]byte(test.input))
			if valid != test.valid {
				t.Errorf("Valid(%#v): returned %v", test.input, valid)
			}
		})
	}
}
