package proxyproto

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

var (
	ErrBadHeaderMagic     = errors.New("proxyproto: not a V1 or V2 proxy header")
	ErrInvalidAddress     = errors.New("proxyproto: invalid address for protocol")
	ErrInvalidHeader      = errors.New("proxyproto: invalid or corrupt proxy header")
	ErrInvalidPort        = errors.New("proxyproto: invalid port")
	ErrUnsupportedVersion = errors.New("proxyproto: unsupported proxy protocol version")
)

var (
	v1Magic  = []byte("PROXY ")
	v2Magic1 = []byte{0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D}
	v2Magic2 = []byte{0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A}
)

const crlf = "\r\n"

type Protocol string

const (
	TCP4    Protocol = "TCP4"
	TCP6    Protocol = "TCP6"
	UNKNOWN Protocol = "UNKNOWN"
)

func (p Protocol) IPNet() string {
	switch p {
	case TCP4:
		return "ip4"
	case TCP6:
		return "ip6"
	}
	return "ip"
}

type UnknownAddr struct {
	net.IPAddr
	Port int
}

func (a *UnknownAddr) Network() string {
	return "unknown"
}

func (a *UnknownAddr) String() string {
	return fmt.Sprintf("%s:%d", a.IP.String(), a.Port)
}

type Header struct {
	Protocol         Protocol
	SrcAddr, DstAddr net.Addr
}

func HeaderFromTCPAddr(src, dst string) (*Header, error) {
	srcAddr, err := net.ResolveTCPAddr("tcp", src)
	if err != nil {
		return nil, err
	}
	dstAddr, err := net.ResolveTCPAddr("tcp", dst)
	if err != nil {
		return nil, err
	}
	proto := TCP4
	if len(srcAddr.IP) != 4 {
		proto = TCP6
	}
	return &Header{
		Protocol: proto,
		SrcAddr:  srcAddr,
		DstAddr:  dstAddr,
	}, nil
}

func addrSplit(addr net.Addr) (Protocol, string, int) {
	switch a := addr.(type) {
	case *net.TCPAddr:
		proto := TCP4
		if len(a.IP) != 4 {
			proto = TCP6
		}
		return proto, a.IP.String(), a.Port
	case *UnknownAddr:
		proto := TCP4
		if len(a.IP) != 4 {
			proto = TCP6
		}
		return proto, a.IP.String(), a.Port
	default:
		return UNKNOWN, addr.String(), 0
	}
}

func (h *Header) String() string {
	srcProto, srcIP, srcPort := addrSplit(h.SrcAddr)
	_, dstIP, dstPort := addrSplit(h.DstAddr)
	return fmt.Sprintf("PROXY %s %s %s %d %d", srcProto, srcIP, dstIP, srcPort, dstPort)
}

func (h *Header) WriteV1(wr io.Writer) (int, error) {
	return io.WriteString(wr, h.String()+crlf)
}

func ReadHeader(bufRd *bufio.Reader) (*Header, error) {
	// Reader the header magic to identify the protocol version

	if magic, err := bufRd.Peek(6); err != nil {
		return nil, err
	} else if bytes.Equal(magic, v1Magic) {
		return readV1(bufRd)
	} else if bytes.Equal(magic, v2Magic1) {
		if magic, err := bufRd.Peek(6); err != nil {
			return nil, err
		} else if bytes.Equal(magic, v2Magic2) {
			return readV2(bufRd)
		}
	}
	return nil, ErrBadHeaderMagic
}

func readV1(bufRd *bufio.Reader) (*Header, error) {
	// magic "PROXY " has already been read
	line, err := bufRd.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(line, crlf) {
		return nil, ErrInvalidHeader
	}
	parts := strings.Split(line[:len(line)-2], " ")
	if len(parts) < 6 {
		return nil, ErrInvalidHeader
	}
	head := Header{Protocol: Protocol(parts[1])}
	if head.Protocol != TCP4 && head.Protocol != TCP6 {
		head.Protocol = UNKNOWN
	}
	if head.SrcAddr, err = parseAddress(head.Protocol, parts[2], parts[4]); err != nil {
		return nil, err
	}
	if head.DstAddr, err = parseAddress(head.Protocol, parts[3], parts[5]); err != nil {
		return nil, err
	}
	return &head, nil
}

func readV2(bufRd *bufio.Reader) (*Header, error) {
	// magic 12-bytes has already been read
	// TODO
	return nil, ErrUnsupportedVersion
}

func parseAddress(proto Protocol, ip, portStr string) (net.Addr, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	} else if port < 0 || port > 65535 {
		return nil, ErrInvalidPort
	}

	ipAddr, err := net.ResolveIPAddr(proto.IPNet(), ip)
	if err != nil {
		return nil, err
	}
	switch proto {
	case "TCP4", "TCP6":
		return &net.TCPAddr{IP: ipAddr.IP, Zone: ipAddr.Zone, Port: port}, nil
	default:
		return &UnknownAddr{IPAddr: *ipAddr, Port: port}, nil
	}
}
