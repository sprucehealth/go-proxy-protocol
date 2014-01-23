package proxyproto

import (
	"bufio"
	"io"
	"net"
)

// TODO: use a sync.Cache instead
var (
	bufioReaderCache = make(chan *bufio.Reader, 4)
)

func newBufioReader(r io.Reader) *bufio.Reader {
	select {
	case p := <-bufioReaderCache:
		p.Reset(r)
		return p
	default:
		return bufio.NewReader(r)
	}
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	select {
	case bufioReaderCache <- br:
	default:
	}
}

type Conn struct {
	net.Conn
	ProxyHeader *Header
	rd          *bufio.Reader
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.ProxyHeader.SrcAddr
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.rd.Read(b)
}

func (c *Conn) Close() error {
	putBufioReader(c.rd)
	return c.Conn.Close()
}

type Listener struct {
	net.Listener
}

func Listen(n, laddr string) (net.Listener, error) {
	ln, err := net.Listen(n, laddr)
	if err != nil {
		return nil, err
	}
	return &Listener{Listener: ln}, nil
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	rd := newBufioReader(conn)
	head, err := ReadHeader(rd)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: conn, ProxyHeader: head, rd: rd}, nil
}
