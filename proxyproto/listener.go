package proxyproto

import (
	"bufio"
	"io"
	"net"
	"sync"
)

var (
	bufioReaderCache = &sync.Pool{}
)

func newBufioReader(r io.Reader) *bufio.Reader {
	if br, ok := bufioReaderCache.Get().(*bufio.Reader); ok {
		br.Reset(r)
		return br
	}
	return bufio.NewReader(r)
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderCache.Put(br)
}

type Conn struct {
	net.Conn
	ProxyHeader *Header
	rd          *bufio.Reader
	mu          sync.Mutex
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.ProxyHeader.SrcAddr
}

func (c *Conn) Read(b []byte) (int, error) {
	c.mu.Lock()
	rd := c.rd
	c.mu.Unlock()
	if rd != nil {
		return rd.Read(b)
	} else {
		return c.Conn.Read(b)
	}
}

func (c *Conn) Close() error {
	c.mu.Lock()
	rd := c.rd
	c.rd = nil
	c.mu.Unlock()
	if rd != nil {
		putBufioReader(rd)
	}
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
