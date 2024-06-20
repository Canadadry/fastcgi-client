package fcgiprotocol

import (
	"reflect"
	"testing"
)

func TestEncodeSize(t *testing.T) {
	tests := map[string]struct {
		In          uint32
		Expected    []byte
		ExpectedLen int
	}{
		"base case small size": {
			In:          127,
			Expected:    []byte{127},
			ExpectedLen: 1,
		},
		"overflow case large size": {
			In:          128,
			Expected:    []byte{0x80, 0x00, 0x00, 0x80},
			ExpectedLen: 4,
		},
		"overflow case large size 256": {
			In:          256,
			Expected:    []byte{0x80, 0x00, 0x01, 0x00},
			ExpectedLen: 4,
		},
		"base case small size 0": {
			In:          0,
			Expected:    []byte{0},
			ExpectedLen: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b := make([]byte, 4)
			l := encodeSize(b, tt.In)
			if l != tt.ExpectedLen {
				t.Fatalf("len want \n%#v\ngot \n%#v\n", tt.ExpectedLen, l)
			}
			result := b[:len(tt.Expected)]
			if !reflect.DeepEqual(tt.Expected, result) {
				t.Fatalf("buf want \n%#v\ngot \n%#v\n", tt.Expected, b)
			}
		})
	}
}
