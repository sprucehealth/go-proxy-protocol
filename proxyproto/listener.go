package proxyproto

import (
	"bufio"
	"io"
	"net"
	"sync"
)

var bufioReaderCache sync.Pool

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

type conn struct {
	net.Conn
	proxyHeader *Header
	rd          *bufio.Reader
	mu          sync.Mutex
	err         error
}

func (c *conn) RemoteAddr() net.Addr {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.readHeader() // can't really do anything with the error at this point, but it's fine since it gets recorded in c.err
	if c.proxyHeader == nil {
		return c.Conn.RemoteAddr()
	}
	return c.proxyHeader.SrcAddr
}

func (c *conn) Read(b []byte) (int, error) {
	// Can't defer the unlock here because we can't hold the lock during the c.Conn.Read
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return 0, c.err
	}
	if err := c.readHeader(); err != nil {
		c.mu.Unlock()
		return 0, err
	}
	rd := c.rd
	c.mu.Unlock()

	if rd != nil {
		bn := rd.Buffered()
		// No data left in buffer so switch to underlying connection
		if bn == 0 {
			putBufioReader(rd)
			c.mu.Lock()
			c.rd = nil
			c.mu.Unlock()
			return c.Conn.Read(b)
		}
		// If reading less than buffered data then just let it go through
		// since we'll need to continue using the bufio reader
		if bn < len(b) {
			return rd.Read(b)
		}
		// Drain the bufio and switch to the connection directly. This will
		// read less data than requested, but that's allowed by the io.Reader
		// interface contract.
		n, err := rd.Read(b[:bn])
		if err == nil {
			putBufioReader(rd)
			c.mu.Lock()
			c.rd = nil
			c.mu.Unlock()
		}
		return n, err
	}

	return c.Conn.Read(b)
}

func (c *conn) Close() error {
	c.mu.Lock()
	rd := c.rd
	c.rd = nil
	c.mu.Unlock()
	if rd != nil {
		putBufioReader(rd)
	}
	return c.Conn.Close()
}

// readHeader looks for the proxy header if it hasn't already been read or an error
// hasn't already occured. The lock is expected to be held at this point.
func (c *conn) readHeader() error {
	if c.proxyHeader != nil || c.err != nil {
		return nil
	}
	c.proxyHeader, c.err = ReadHeader(c.rd)
	if c.err != nil {
		putBufioReader(c.rd)
		c.rd = nil
		return c.err
	}
	return nil
}

// Listener wraps the net.Listener interface to provide proxy protocl support.
type Listener struct {
	net.Listener
}

// Listen creates a new listener that implements the proxy protocol for client connections.
func Listen(n, laddr string) (net.Listener, error) {
	ln, err := net.Listen(n, laddr)
	if err != nil {
		return nil, err
	}
	return &Listener{Listener: ln}, nil
}

// Accept implements the Accept method in the Listener interface.
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &conn{Conn: c, rd: newBufioReader(c)}, nil
}
