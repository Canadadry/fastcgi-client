package decoder

import (
	"app/fcgiclient"
	"bytes"
	"reflect"
	"testing"
)

// Assume writePairs and readPairs are in the same package or imported accordingly
func TestDecodeEnv(t *testing.T) {
	tests := map[string]struct {
		Data map[string]string
	}{
		"simple case": {
			Data: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		"empty case": {
			Data: map[string]string{},
		},
		"complex case": {
			Data: map[string]string{
				"key1":           "value1",
				"longkey2":       "longvalue2",
				"key with space": "value with space",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			// Encode the map
			err := fcgiclient.BuildPair(&buf, tt.Data)
			if err != nil {
				t.Fatalf("writePairs failed: %v", err)
			}

			// Decode the map
			decodedPairs, err := decodeEnv(&buf)
			if err != nil {
				t.Fatalf("readPairs failed: %v", err)
			}

			// Check if the original map and the decoded map are equal
			if !reflect.DeepEqual(tt.Data, decodedPairs) {
				t.Fatalf("decoded map does not match the original map: got %v, want %v", decodedPairs, tt.Data)
			}
		})
	}
}
