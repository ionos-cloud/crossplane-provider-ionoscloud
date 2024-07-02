package ipnet

import (
	"net"
	"testing"
)

var ipTable = [][2]string{
	{"192.168.100.1", "192.168.100.2"},
	{"0.0.0.0", "0.0.0.1"},
	{"192.168.100.255", "192.168.101.0"},
	{"10.10.255.255", "10.11.0.0"},
	{"10.255.255.255", "11.0.0.0"},
	{"::1", "::2"},
	{"fe80::ffff", "fe80::1:0"},
	{"fe80::ffff:ffff", "fe80::1:0:0"},
}

func TestIncrement(t *testing.T) {
	for _, ips := range ipTable {
		ip := net.ParseIP(ips[0])
		if ip == nil {
			t.Fatalf("ipTable: Bad IP Address: %s", ips[0])
		}
		Increment(ip)
		if ip.String() != ips[1] {
			t.Errorf("IP Mismatch: Expected: %s, Got: %s", ips[1], ip.String())
		}
	}
}

func TestNewIPNetGenerator(t *testing.T) {
	// bad CIDRs
	_, err := New("stuff")
	if err == nil {
		t.Errorf("Expected error, got nil for CIDR: stuff")
	}
	_, err = New("192.168.100.10")
	if err == nil {
		t.Errorf("Expected error, got nil for CIDR: 192.168.100.10")
	}
	_, err = New("::1")
	if err == nil {
		t.Errorf("Expected error, got nil for CIDR: ::1")
	}

	// good CIDRs
	gen, err := New("192.168.100.0/24")
	if err != nil {
		t.Errorf("Unexpected error with CIDR 192.168.100.0/24: %#v", err)
	}

	if gen.current.String() != "192.168.100.0" {
		t.Errorf("Expected: current = 192.168.100.0, Got: current = %s", gen.current.String())
	}

	if gen.count.String() != "256" {
		t.Errorf("Expected: count = 256, Got: count = %s", gen.count.String())
	}

	gen, err = New("10.10.10.245/25")
	if err != nil {
		t.Errorf("Unexpected error with CIDR 192.168.100.0/24: %#v", err)
	}

	if gen.current.String() != "10.10.10.128" {
		t.Errorf("Expected: current = 10.10.10.128, Got: current = %s", gen.current.String())
	}

	if gen.count.String() != "128" {
		t.Errorf("Expected: count = 128, Got: count = %s", gen.count.String())
	}

	gen, err = New("fe80::/112")
	if err != nil {
		t.Errorf("Unexpected error with CIDR fe80::/112: %#v", err)
	}

	if gen.current.String() != "fe80::" {
		t.Errorf("Expected: current = fe80::, Got: current = %s", gen.current.String())
	}

	if gen.count.String() != "65536" {
		t.Errorf("Expected: count = 65536, Got: count = %s", gen.count.String())
	}

	gen, err = New("0100::/64")
	if err != nil {
		t.Errorf("Unexpected error with CIDR 0100::/64: %#v", err)
	}

	if gen.current.String() != "100::" {
		t.Errorf("Expected: current = 0100::, Got: current = %s", gen.current.String())
	}

	if gen.count.String() != "18446744073709551616" {
		t.Errorf("Expected: count = 18446744073709551616, Got: count = %s", gen.count.String())
	}
}

func TestNextIPv4(t *testing.T) {
	gen, err := New("192.168.100.0/24")
	if err != nil {
		t.Errorf("Unexpected error with CIDR 192.168.100.0/24: %#v", err)
	}

	current := gen.current
	next := gen.Next()
	if &(current[0]) == &(next[0]) {
		t.Errorf("Error: current should not be equal to next")
	}

	if next.String() != "192.168.100.0" {
		t.Errorf("Expected: next = 192.168.100.0, Got: next = %s", next.String())
	}

	newNext := gen.Next()
	if &(next[0]) == &(newNext[0]) {
		t.Errorf("Error: next should not be equal to newNext")
	}

	if newNext.String() != "192.168.100.1" {
		t.Errorf("Expected: next = 192.168.100.1, Got: next = %s", newNext.String())
	}

	for i := 0; i < 254; i++ {
		gen.Next()
	}

	last := gen.Next()
	if last != nil {
		t.Errorf("Expected: next = nil, Got next = %s", last.String())
	}
}

func TestNextIPv6(t *testing.T) {
	gen, err := New("fe80::/112")
	if err != nil {
		t.Errorf("Unexpected error with CIDR fe80::/112: %#v", err)
	}

	current := gen.current
	next := gen.Next()
	if &(current[0]) == &(next[0]) {
		t.Errorf("Error: current should not be equal to next")
	}

	if next.String() != "fe80::" {
		t.Errorf("Expected: next = fe80::, Got: next = %s", next.String())
	}

	newNext := gen.Next()
	if &(next[0]) == &(newNext[0]) {
		t.Errorf("Error: next should not be equal to newNext")
	}

	if newNext.String() != "fe80::1" {
		t.Errorf("Expected: next = fe80::1, Got: next = %s", newNext.String())
	}

	for i := 0; i < 65534; i++ {
		gen.Next()
	}

	last := gen.Next()
	if last != nil {
		t.Errorf("Expected: next = nil, Got next = %s", last.String())
	}
}
