package proxyproto

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestV1ParseTCP4(t *testing.T) {
	data := "PROXY TCP4 127.0.0.1 127.0.0.2 111 222\r\n"
	rd := strings.NewReader(data)
	head, err := ReadHeader(bufio.NewReader(rd))
	if err != nil {
		t.Fatal(err)
	}
	if head.Protocol != TCP4 {
		t.Fatalf("Expected TCP4, got %s", head.Protocol)
	}
	if head.SrcAddr.Network() != "tcp" {
		t.Fatalf("Expected tcp, got %s", head.SrcAddr.Network())
	}
	if head.DstAddr.Network() != "tcp" {
		t.Fatalf("Expected tcp, got %s", head.DstAddr.Network())
	}
	if head.SrcAddr.String() != "127.0.0.1:111" {
		t.Fatalf("Expected 127.0.0.1, got %s", head.SrcAddr)
	}
	if head.DstAddr.String() != "127.0.0.2:222" {
		t.Fatalf("Expected 127.0.0.2, got %s", head.DstAddr)
	}
}

func TestV1ParseTCP6(t *testing.T) {
	data := "PROXY TCP4 2607:f8b0:4010:801::1004%en0 127.0.0.2 111 222\r\n"
	rd := strings.NewReader(data)
	head, err := ReadHeader(bufio.NewReader(rd))
	if err != nil {
		t.Fatal(err)
	}
	if head.Protocol != TCP4 {
		t.Fatalf("Expected TCP4, got %s", head.Protocol)
	}
	if head.SrcAddr.Network() != "tcp" {
		t.Fatalf("Expected tcp, got %s", head.SrcAddr.Network())
	}
	if head.DstAddr.Network() != "tcp" {
		t.Fatalf("Expected tcp, got %s", head.DstAddr.Network())
	}
	if head.SrcAddr.String() != "[2607:f8b0:4010:801::1004%en0]:111" {
		t.Fatalf("Expected 127.0.0.1, got %s", head.SrcAddr)
	}
	if head.DstAddr.String() != "127.0.0.2:222" {
		t.Fatalf("Expected 127.0.0.2, got %s", head.DstAddr)
	}
}

func BenchmarkParser(b *testing.B) {
	b.ReportAllocs()
	rd := bytes.NewReader([]byte("PROXY TCP4 127.0.0.1 127.0.0.2 111 222\r\n"))
	for i := 0; i < b.N; i++ {
		rd.Seek(0, 0)
		br := newBufioReader(rd)
		if _, err := ReadHeader(br); err != nil {
			b.Fatal(err)
		}
		putBufioReader(br)
	}
}
