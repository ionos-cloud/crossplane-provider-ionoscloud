package ipnet

// credits: https://github.com/korylprince/ipnetgen/tree/master

import (
	"math/big"
	"net"
)

// Increment increments the given net.IP by one bit. Incrementing the last IP in an IP space (IPv4, IPV6) is undefined.
func Increment(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		// only add to the next byte if we overflowed
		if ip[i] != 0 {
			break
		}
	}
}

// Generator is a net.IPnet wrapper that you can iterate over
type Generator struct {
	*net.IPNet
	count *big.Int

	// state
	idx     *big.Int
	current net.IP
}

// New creates a new Generator from a CIDR string, or an error if the CIDR is invalid.
func New(cidr string) (*Generator, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	return NewFromIPNet(ipNet), nil
}

// NewFromIPNet creates a new Generator from a *net.IPNet
func NewFromIPNet(ipNet *net.IPNet) *Generator {
	ones, bits := ipNet.Mask.Size()

	newIP := make(net.IP, len(ipNet.IP))
	copy(newIP, ipNet.IP)

	count := big.NewInt(0)
	count.Exp(big.NewInt(2), big.NewInt(int64(bits-ones)), nil)

	return &Generator{
		IPNet:   ipNet,
		count:   count,
		idx:     big.NewInt(0),
		current: newIP,
	}
}

// Next returns the next net.IP in the subnet
func (g *Generator) Next() net.IP {
	g.idx.Add(g.idx, big.NewInt(1))
	if g.idx.Cmp(g.count) == 1 {
		return nil
	}
	current := make(net.IP, len(g.current))
	copy(current, g.current)
	Increment(g.current)

	return current
}

func (g *Generator) First() net.IP {
	return g.IPNet.IP
}
