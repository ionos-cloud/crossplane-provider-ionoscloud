package ccpatch

import (
	"crypto/rand"
	"fmt"
	"net"
	"slices"

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

func (i *ipv4Address) WriteState(identifier substitution.Identifier, gs *substitution.GlobalState, sub substitution.Substitution) error {
	value, ok := sub.AdditionalProperties["cidr"]
	if !ok {
		return substitution.ErrMissingCIDR
	}

	if gs.Exists(identifier, sub.Key) {
		return nil
	}

	used := []string{}

	if sub.Unique {
		gs.Each(func(key substitution.Identifier, state []substitution.State) {
			for _, s := range state {
				if s.Key == sub.Key {
					used = append(used, s.Value)
				}
			}
		})
	}

	nip, err := getNextIPv4(value, used)
	if err != nil {
		return fmt.Errorf("error generating random IPv4 address: %w", err)
	}

	gs.Set(identifier, sub.Key, nip.String())

	return nil
}

func getNextIPv4(cidr string, used []string) (net.IP, error) {

jump:
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("error parsing CIDR: %w", err)
	}

	ones, _ := ipnet.Mask.Size()
	quotient := ones / 8
	remainder := ones % 8

	r := make([]byte, 4)
	_, err = rand.Read(r)
	if err != nil {
		return nil, fmt.Errorf("while reading %w", err)
	}

	for i := 0; i <= quotient; i++ {
		if i == quotient {
			shifted := r[i] >> remainder
			r[i] = ^ipnet.IP[i] & shifted
		} else {
			r[i] = ipnet.IP[i]
		}
	}
	ip = net.IPv4(r[0], r[1], r[2], r[3])

	if ip.Equal(ipnet.IP) || slices.Contains(
		used,
		ip.String(),
	) /*|| ip.Equal(broadcast) */ {
		// we got unlucky. The host portion of our ipv4 address was
		// either all 0s (the network address) or all 1s (the broadcast address)
		goto jump
	}
	return ip, nil
}
