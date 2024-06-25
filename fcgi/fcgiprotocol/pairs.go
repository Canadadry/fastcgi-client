package fcgiprotocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
)

func MustBuildPairWithPadding(pairs map[string]string, padding int) []byte {
	buf := &bytes.Buffer{}
	err := buildPair(buf, pairs)
	if err != nil {
		panic(fmt.Errorf("cannot build pair :%w", err))
	}
	for _ = range padding {
		buf.Write([]byte{0})
	}
	return buf.Bytes()
}
func buildPair(w io.Writer, pairs map[string]string) error {
	b := make([]byte, 8)
	keys := make([]string, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := pairs[k]
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
