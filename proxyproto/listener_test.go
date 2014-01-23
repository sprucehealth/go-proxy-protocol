package proxyproto

import (
	"net"
	"testing"
	"time"
)

func TestListener(t *testing.T) {
	ln, err := Listen("tcp", "127.0.0.1:7791")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan string, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				t.Fatal(err)
				return
			}
			ch <- conn.RemoteAddr().String()
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
			return
		}
	}()
	defer ln.Close()

	conn, err := net.Dial("tcp", "127.0.0.1:7791")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	src := "127.0.0.1:111"
	h, err := HeaderFromTCPAddr(src, "127.0.0.2:222")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.WriteV1(conn); err != nil {
		t.Fatal(err)
	}
	select {
	case addr := <-ch:
		if addr != src {
			t.Fatalf("Expected %s, got %s", src, addr)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out")
	}
}
