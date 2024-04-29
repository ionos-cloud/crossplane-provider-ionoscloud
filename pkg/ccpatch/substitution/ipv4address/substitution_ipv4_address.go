package ccpatch

import (
	"crypto/rand"
	"fmt"
	"net"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

func init() {
	substitution.RegisterSubstitution(&ipv4Address{})
}

type ipv4Address struct{}

var _ substitution.Handler = &ipv4Address{}

func (i *ipv4Address) Type() string {
	return "ipv4Address"
}

func (i *ipv4Address) WriteState(identifier substitution.Identifier, state *substitution.GlobalState, sub substitution.Substitution) error {
	value, ok := sub.AdditionalProperties["cidr"]
	if !ok {
		return substitution.ErrMissingCIDR
	}

	ip, err := randomIPv4FromCIDR(value)
	if err != nil {
		return fmt.Errorf("error generating random IPv4 address: %w", err)
	}

	fmt.Println(ip.String())

	return nil
}

func randomIPv4FromCIDR(cidr string) (net.IP, error) {

jump:
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("error parsing CIDR: %w", err)
	}

	ones, _ := ipnet.Mask.Size()
	quotient := ones / 8
	remainder := ones % 8

	r := make([]byte, 4)
	rand.Read(r)

	for i := 0; i <= quotient; i++ {
		if i == quotient {
			shifted := r[i] >> remainder
			r[i] = ^ipnet.IP[i] & shifted
		} else {
			r[i] = ipnet.IP[i]
		}
	}
	ip = net.IPv4(r[0], r[1], r[2], r[3])

	if ip.Equal(ipnet.IP) /*|| ip.Equal(broadcast) */ {
		// we got unlucky. Theu host portion of our ipv4 address was
		// either all 0s (the network address) or all 1s (the broadcast address)
		goto jump
	}
	return ip, nil
}
