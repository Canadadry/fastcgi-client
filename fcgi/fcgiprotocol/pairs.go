package fcgiprotocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

func BuildPair(w io.Writer, pairs map[string]string) error {
	b := make([]byte, 8)

	for k, v := range pairs {
		if len(k) > MaxKeyPairLen {
			return fmt.Errorf("failed a key has len of %d > max len of %d", len(k), MaxKeyPairLen)
		}
		if len(v) > MaxValuePairLen {
			return fmt.Errorf("failed a value has len of %d > max len of %d", len(v), MaxValuePairLen)
		}
		n := encodeSize(b, uint32(len(k)))
		n += encodeSize(b[n:], uint32(len(v)))
		if _, err := w.Write(b[:n]); err != nil {
			return err
		}
		if _, err := io.WriteString(w, k); err != nil {
			return err
		}
		if _, err := io.WriteString(w, v); err != nil {
			return err
		}
	}
	return nil
}

func encodeSize(b []byte, size uint32) int {
	if size > 127 {
		size |= 1 << 31
		binary.BigEndian.PutUint32(b, size)
		return 4
	}
	b[0] = byte(size)
	return 1
}
