package fcgiprotocol

import (
	"encoding/binary"
	"io"
)

func GetPairSize(pairs map[string]string) int {
	b := make([]byte, 8)

	writterBufSize := 0
	for k, v := range pairs {
		kLen := len(k)
		vLen := len(v)

		n := encodeSize(b, uint32(kLen))
		n += encodeSize(b[n:], uint32(vLen))

		writterBufSize += (n + kLen + vLen)
	}
	return writterBufSize
}

func BuildPair(w io.Writer, pairs map[string]string) error {
	b := make([]byte, 8)

	for k, v := range pairs {
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
