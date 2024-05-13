package ccpatch

import (
	"fmt"
	"net"
	"strings"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/ipnet"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

func init() {
	substitution.RegisterSubstitution(&ipv6Address{})
}

type ipv6Address struct{}

var _ substitution.Handler = &ipv6Address{}

func (i *ipv6Address) Type() string {
	return "ipv6Address"
}

func (i *ipv6Address) WriteState(identifier substitution.Identifier, gs *substitution.GlobalState, sub substitution.Substitution) error {
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

	nip, err := getNextIPv6(value, used)
	if err != nil {
		return err
	}

	gs.Set(identifier, sub.Key, nip.String())

	return nil
}

// getNextIPv6 calculates the next usable IPv6 address based on the CIDR and used addresses list
func getNextIPv6(cidr string, usedAddresses []string) (net.IP, error) {
	gen, err := ipnet.New(cidr)
	if err != nil {
		return nil, err
	}

	if !isUsedIPv6(gen.IP, usedAddresses) {
		return gen.IP, nil
	}

	for {
		nextIP := gen.Next()
		if !isUsedIPv6(nextIP, usedAddresses) {
			return nextIP, nil
		}

		if nextIP == nil {
			break
		}
	}

	return nil, fmt.Errorf("no more available IPv6 addresses in %s", cidr)
}

// isUsedIPv6 checks if an IPv6 address is in the usedAddresses list
func isUsedIPv6(ip net.IP, usedAddresses []string) bool {
	ipString := ip.String()
	for _, usedIP := range usedAddresses {
		if strings.EqualFold(ipString, usedIP) {
			return true
		}
	}
	return false
}
