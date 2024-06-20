package fcgiprotocol

import (
	"encoding/binary"
	"errors"
	"io"
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
	MaxWrite        = 65535
	MaxKeyPairLen   = 255
	MaxValuePairLen = 255
	MaxPad          = 255
)

type Header struct {
	Version       uint8
	Type          uint8
	Id            uint16
	ContentLength uint16
	PaddingLength uint8
	Reserved      uint8
}

func NewHeader(recType uint8, reqId uint16, contentLength int) Header {
	return Header{
		Version:       VERSION_1,
		Type:          recType,
		Id:            reqId,
		ContentLength: uint16(contentLength),
		PaddingLength: uint8(-contentLength & 7),
	}
}

type Record struct {
	Header Header
	Buf    []byte
}

func (rec *Record) Read(r io.Reader) (err error) {
	if err = binary.Read(r, binary.BigEndian, &rec.Header); err != nil {
		return err
	}
	if rec.Header.Version != VERSION_1 {
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
