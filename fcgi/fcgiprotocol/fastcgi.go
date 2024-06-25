package fcgiprotocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// for padding so we don't have to allocate all the time
// not synchronized because we don't care what the contents are
var pad [MaxPad]byte

type recordWriter func(recType uint8, reqId uint16, content []byte) error

func Do(rwc io.ReadWriter, env map[string]string, reqStr string) ([]byte, []byte, error) {
	var reqId uint16 = 1
	buf := bufio.NewWriterSize(rwc, MaxWrite)
	err := WriteRequest(StreamRecordWriter(buf, MaxWrite), reqId, env, reqStr)
	if err != nil {
		return nil, nil, fmt.Errorf("cant write req : %w", err)
	}
	err = buf.Flush()
	if err != nil {
		return nil, nil, fmt.Errorf("while flushing, cant write req %w", err)
	}

	return readResponse(rwc)
}

func WriteRequest(w recordWriter, reqId uint16, env map[string]string, body string) error {
	buf := &bytes.Buffer{}
	err := BuildPair(buf, env)
	if err != nil {
		return fmt.Errorf("cant build pair : %w", err)
	}
	if buf.Len() > MaxPairLen {
		//fmt.Printf("pairs : %s\n", base64.StdEncoding.EncodeToString(buf.Bytes()))
		return fmt.Errorf("build pair len exceed MaxPairLen of (%d)", MaxPairLen)
	}

	err = writeBeginRequest(w, reqId)
	if err != nil {
		return fmt.Errorf("cant write begin req %w", err)
	}
	err = writePairs(w, reqId, buf.Bytes())
	if err != nil {
		return fmt.Errorf("cant write pairs req %w", err)
	}
	err = writeStdin(w, reqId, []byte(body))
	if err != nil {
		return fmt.Errorf("cant write stdin req %w", err)
	}
	return nil
}

func readResponse(r io.Reader) ([]byte, []byte, error) {
	var stdout, stderr []byte
	rec := &Record{}
	for {
		err := rec.Read(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("cannot read response : %w", err)
		}
		switch {
		case rec.Header.Type == FCGI_STDOUT:
			stdout = append(stdout, rec.Content()...)
		case rec.Header.Type == FCGI_STDERR:
			stderr = append(stderr, rec.Content()...)
		case rec.Header.Type == FCGI_END_REQUEST:
			break
		default:
			break
		}
	}

	return stdout, stderr, nil
}

func writeBeginRequest(w recordWriter, reqId uint16) error {
	role := uint16(FCGI_RESPONDER)
	flags := uint8(0)
	b := [8]byte{byte(role >> 8), byte(role), flags}
	return w(FCGI_BEGIN_REQUEST, reqId, b[:])
}

func writePairs(w recordWriter, reqId uint16, pairs []byte) error {
	err := w(FCGI_PARAMS, reqId, pairs)
	if err != nil {
		return fmt.Errorf("cannot write pair : %w", err)
	}
	return w(FCGI_PARAMS, reqId, nil)
}

func writeStdin(w recordWriter, reqId uint16, body []byte) error {
	var err error
	if len(body) > 0 {
		err = w(FCGI_STDIN, reqId, body)
	}
	return err
}

func RawRecordWriter(w io.Writer) recordWriter {
	return func(recType uint8, reqId uint16, content []byte) (err error) {
		//fmt.Fprintf("writeRecord of %d : %s\n", recType, base64.StdEncoding.EncodeToString(content))
		h := NewHeader(recType, reqId, len(content))
		if err := binary.Write(w, binary.BigEndian, h); err != nil {
			return err
		}
		if _, err := w.Write(content); err != nil {
			return err
		}
		if _, err := w.Write(pad[:h.PaddingLength]); err != nil {
			return err
		}
		return nil
	}
}

func StreamRecordWriter(w io.Writer, maxWrite int) recordWriter {
	rw := RawRecordWriter(w)
	return func(recType uint8, reqId uint16, content []byte) error {
		if len(content) == 0 {
			return rw(recType, reqId, nil)
		}
		for len(content) > 0 {
			n := len(content)
			if n > maxWrite {
				n = maxWrite
			}
			if err := rw(recType, reqId, content[:n]); err != nil {
				return err
			}
			content = content[n:]
		}
		return nil
	}
}
