package fcgiprotocol

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

const FCGI_LISTENSOCK_FILENO uint8 = 0
const FCGI_HEADER_LEN uint8 = 8
const VERSION_1 uint8 = 1
const FCGI_NULL_REQUEST_ID uint8 = 0
const FCGI_KEEP_CONN uint8 = 1

const (
	FCGI_BEGIN_REQUEST     uint8               = iota + 1 // 1
	FCGI_ABORT_REQUEST                                    // 2
	FCGI_END_REQUEST                                      // 3
	FCGI_PARAMS                                           // 4
	FCGI_STDIN                                            // 5
	FCGI_STDOUT                                           // 6
	FCGI_STDERR                                           // 7
	FCGI_DATA                                             // 8
	FCGI_GET_VALUES                                       // 9
	FCGI_GET_VALUES_RESULT                                // 10
	FCGI_UNKNOWN_TYPE                                     // 11
	FCGI_MAXTYPE           = FCGI_UNKNOWN_TYPE            // 11
)

const (
	FCGI_RESPONDER uint8 = iota + 1
	FCGI_AUTHORIZER
	FCGI_FILTER
)

const (
	FCGI_REQUEST_COMPLETE uint8 = iota
	FCGI_CANT_MPX_CONN
	FCGI_OVERLOADED
	FCGI_UNKNOWN_ROLE
)

const (
	FCGI_MAX_CONNS  string = "MAX_CONNS"
	FCGI_MAX_REQS   string = "MAX_REQS"
	FCGI_MPXS_CONNS string = "MPXS_CONNS"
)

const (
	MaxWrite = 65535 // maximum record body
	MaxPad   = 255
)

type Header struct {
	Version       uint8
	Type          uint8
	Id            uint16
	ContentLength uint16
	PaddingLength uint8
	Reserved      uint8
}

// for padding so we don't have to allocate all the time
// not synchronized because we don't care what the contents are
var pad [MaxPad]byte

func (h *Header) init(recType uint8, reqId uint16, contentLength int) {
	h.Version = 1
	h.Type = recType
	h.Id = reqId
	h.ContentLength = uint16(contentLength)
	h.PaddingLength = uint8(-contentLength & 7)
}

type Record struct {
	Header Header
	Buf    []byte
}

func (rec *Record) Read(r io.Reader) (err error) {
	if err = binary.Read(r, binary.BigEndian, &rec.Header); err != nil {
		return err
	}
	if rec.Header.Version != 1 {
		return errors.New("fcgi: invalid header version")
	}
	n := int(rec.Header.ContentLength) + int(rec.Header.PaddingLength)
	if n > MaxWrite+MaxPad {
		return errors.New("fcgi: response is too long")
	}
	rec.Buf = make([]byte, n)
	if _, err = io.ReadFull(r, rec.Buf[:n]); err != nil {
		return err
	}
	return nil
}

func (r *Record) Content() []byte {
	return r.Buf[:r.Header.ContentLength]
}

type FCGIClient struct {
	mutex     sync.Mutex
	rwc       io.ReadWriter
	h         Header
	buf       bytes.Buffer
	keepAlive bool
}

func New(rwc io.ReadWriter) *FCGIClient {
	return &FCGIClient{
		rwc:       rwc,
		keepAlive: false,
	}
}

func (this *FCGIClient) writeRecord(recType uint8, reqId uint16, content []byte) (err error) {
	fmt.Printf("writeRecord of %d with len %d uint16(%d) : %s\n", recType, len(content), uint16(len(content)), base64.StdEncoding.EncodeToString(compress(content)))
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.buf.Reset()
	this.h.init(recType, reqId, len(content))
	if err := binary.Write(&this.buf, binary.BigEndian, this.h); err != nil {
		return err
	}
	if _, err := this.buf.Write(content); err != nil {
		return err
	}
	if _, err := this.buf.Write(pad[:this.h.PaddingLength]); err != nil {
		return err
	}
	_, err = this.rwc.Write(this.buf.Bytes())
	return err
}

func compress(content []byte) []byte {

	buf := bytes.Buffer{}
	zipWriter := zip.NewWriter(&buf)
	zipFile, _ := zipWriter.Create("body.bin")
	_, _ = zipFile.Write(content)
	_ = zipWriter.Close()
	return buf.Bytes()
}

func (this *FCGIClient) writeBeginRequest(reqId uint16, role uint16, flags uint8) error {
	b := [8]byte{byte(role >> 8), byte(role), flags}
	return this.writeRecord(FCGI_BEGIN_REQUEST, reqId, b[:])
}

func (this *FCGIClient) writeEndRequest(reqId uint16, appStatus int, protocolStatus uint8) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(appStatus))
	b[4] = protocolStatus
	return this.writeRecord(FCGI_END_REQUEST, reqId, b)
}

func (this *FCGIClient) writePairs(recType uint8, reqId uint16, pairs map[string]string) error {
	w := newWriter(this, recType, reqId, GetPairSize(pairs))
	defer w.Close()
	return BuildPair(w, pairs)
}

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

// bufWriter encapsulates bufio.Writer but also closes the underlying stream when
// Closed.
type bufWriter struct {
	closer io.Closer
	*bufio.Writer
}

func (w *bufWriter) Close() error {
	if err := w.Writer.Flush(); err != nil {
		w.closer.Close()
		return err
	}
	return w.closer.Close()
}

func newWriter(c *FCGIClient, recType uint8, reqId uint16, bufSize int) *bufWriter {
	s := &streamWriter{c: c, recType: recType, reqId: reqId}
	w := bufio.NewWriterSize(s, bufSize)
	return &bufWriter{s, w}
}

// streamWriter abstracts out the separation of a stream into discrete records.
// It only writes maxWrite bytes at a time.
type streamWriter struct {
	c       *FCGIClient
	recType uint8
	reqId   uint16
}

func (w *streamWriter) Write(p []byte) (int, error) {
	nn := 0
	for len(p) > 0 {
		n := len(p)
		if n > MaxWrite {
			n = MaxWrite
		}
		if err := w.c.writeRecord(w.recType, w.reqId, p[:n]); err != nil {
			return nn, err
		}
		nn += n
		p = p[n:]
	}
	return nn, nil
}

func (w *streamWriter) Close() error {
	// send empty record to close the stream
	// return w.c.writeRecord(w.recType, w.reqId, nil)
	return nil
}

func (this *FCGIClient) Request(env map[string]string, reqStr string) ([]byte, []byte, error) {

	var reqId uint16 = 1

	err := this.writeBeginRequest(reqId, uint16(FCGI_RESPONDER), 0)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot write begin request")
	}
	err = this.writePairs(FCGI_PARAMS, reqId, env)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot write param request : %w", err)
	}
	err = this.writeRecord(FCGI_PARAMS, reqId, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot write end param request : %w", err)
	}
	if len(reqStr) > 0 {
		err = this.writeRecord(FCGI_STDIN, reqId, []byte(reqStr))
		if err != nil {
			return nil, nil, fmt.Errorf("cannot write stdin request : %w", err)
		}
	}

	fmt.Printf("finish sending request\n")

	rec := &Record{}
	var retout, reterr []byte

	// recive untill EOF or FCGI_END_REQUEST
	for {
		err := rec.Read(this.rwc)
		if err != nil {
			if err != io.EOF {
				return retout, reterr, fmt.Errorf("cannot read from server : %w", err)
			}
			break
		}
		fmt.Printf("reading of %d : %s\n", rec.Header.Type, base64.StdEncoding.EncodeToString(compress(rec.Content())))
		switch {
		case rec.Header.Type == FCGI_STDOUT:
			retout = append(retout, rec.Content()...)
		case rec.Header.Type == FCGI_STDERR:
			reterr = append(reterr, rec.Content()...)
		case rec.Header.Type == FCGI_END_REQUEST:
			fallthrough
		default:
			break
		}
	}

	return retout, reterr, nil
}
