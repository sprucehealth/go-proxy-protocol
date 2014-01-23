package proxyproto

import (
	"net"
)

type Conn struct {
	net.Conn
	ProxyHeader *Header
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.ProxyHeader.SrcAddr
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
	head, err := ReadHeader(conn)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: conn, ProxyHeader: head}, nil
}
