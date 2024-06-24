package sniff

import (
	"app/fcgi/fcgiprotocol"
	"bytes"
	"reflect"
	"testing"
)

func TestReadFullRequest(t *testing.T) {
	var buf bytes.Buffer
	err := fcgiprotocol.WriteRequest(
		fcgiprotocol.RawRecordWriter(&buf),
		0,
		map[string]string{"test": "test"},
		"body",
	)
	if err != nil {
		t.Fatalf("WriteRequest failed: %v", err)
	}

	records, err := ReadFullRequest(&buf)
	if err != nil {
		t.Fatalf("ReadFullRequest failed: %v", err)
	}

	expected := []fcgiprotocol.Record{
		{
			Header: fcgiprotocol.NewHeader(fcgiprotocol.FCGI_BEGIN_REQUEST, 0, 8),
			Buf:    []byte{0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			Header: fcgiprotocol.NewHeader(fcgiprotocol.FCGI_PARAMS, 0, 10),
			Buf:    fcgiprotocol.MustBuildPairWithPadding(map[string]string{"test": "test"}, 6),
		},
		{
			Header: fcgiprotocol.NewHeader(fcgiprotocol.FCGI_PARAMS, 0, 0),
			Buf:    []byte{},
		},
		{
			Header: fcgiprotocol.NewHeader(fcgiprotocol.FCGI_STDIN, 0, 4),
			Buf:    []byte{'b', 'o', 'd', 'y', 0, 0, 0, 0},
		},
	}
	if !reflect.DeepEqual(records, expected) {
		t.Fatalf("got \n%#v\n, exp \n%#v\n", records, expected)
	}
}
