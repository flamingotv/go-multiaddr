package multiaddr

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func stringToBytes(s string) ([]byte, error) {

	// consume trailing slashes
	s = strings.TrimRight(s, "/")

	b := []byte{}
	sp := strings.Split(s, "/")

	if sp[0] != "" {
		return nil, fmt.Errorf("invalid multiaddr, must begin with /")
	}

	// consume first empty elem
	sp = sp[1:]

	for len(sp) > 0 {
		p := ProtocolWithName(sp[0])
		if p.Code == 0 {
			return nil, fmt.Errorf("no protocol with name %s", sp[0])
		}
		b = append(b, CodeToVarint(p.Code)...)
		sp = sp[1:]

		if p.Size > 0 {
			if len(sp) < 1 {
				return nil, fmt.Errorf("protocol requires address, none given: %s", sp)
			}
			a, err := addressStringToBytes(p, sp[0])
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %s %s", p.Name, sp[0], err)
			}
			b = append(b, a...)
			sp = sp[1:]
		}
	}
	return b, nil
}

func bytesToString(b []byte) (ret string, err error) {
	// panic handler, in case we try accessing bytes incorrectly.
	defer func() {
		if e := recover(); e != nil {
			ret = ""
			err = e.(error)
		}
	}()

	s := ""

	for len(b) > 0 {

		code, n := ReadVarintCode(b)
		b = b[n:]
		p := ProtocolWithCode(code)
		if p.Code == 0 {
			return "", fmt.Errorf("no protocol with code %d", code)
		}
		s = strings.Join([]string{s, "/", p.Name}, "")

		if p.Size > 0 {
			a := addressBytesToString(p, b[:(p.Size/8)])
			if len(a) > 0 {
				s = strings.Join([]string{s, "/", a}, "")
			}
			b = b[(p.Size / 8):]
		}
	}

	return s, nil
}

func bytesSplit(b []byte) (ret [][]byte, err error) {
	// panic handler, in case we try accessing bytes incorrectly.
	defer func() {
		if e := recover(); e != nil {
			ret = [][]byte{}
			err = e.(error)
		}
	}()

	ret = [][]byte{}
	for len(b) > 0 {
		code, n := ReadVarintCode(b)
		p := ProtocolWithCode(code)
		if p.Code == 0 {
			return [][]byte{}, fmt.Errorf("no protocol with code %d", b[0])
		}

		length := n + (p.Size / 8)
		ret = append(ret, b[:length])
		b = b[length:]
	}

	return ret, nil
}

func addressStringToBytes(p Protocol, s string) ([]byte, error) {
	switch p.Code {

	case P_IP4: // ipv4
		i := net.ParseIP(s).To4()
		if i == nil {
			return nil, fmt.Errorf("failed to parse ip4 addr: %s", s)
		}
		return i, nil

	case P_IP6: // ipv6
		i := net.ParseIP(s).To16()
		if i == nil {
			return nil, fmt.Errorf("failed to parse ip6 addr: %s", s)
		}
		return i, nil

	// tcp udp dccp sctp
	case P_TCP, P_UDP, P_DCCP, P_SCTP:
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s addr: %s", p.Name, err)
		}
		if i >= 65536 {
			return nil, fmt.Errorf("failed to parse %s addr: %s", p.Name, "greater than 65536")
		}
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(i))
		return b, nil
	}

	return []byte{}, fmt.Errorf("failed to parse %s addr: unknown", p.Name)
}

func addressBytesToString(p Protocol, b []byte) string {
	switch p.Code {

	// ipv4,6
	case P_IP4, P_IP6:
		return net.IP(b).String()

	// tcp udp dccp sctp
	case P_TCP, P_UDP, P_DCCP, P_SCTP:
		i := binary.BigEndian.Uint16(b)
		return strconv.Itoa(int(i))
	}

	return ""
}
