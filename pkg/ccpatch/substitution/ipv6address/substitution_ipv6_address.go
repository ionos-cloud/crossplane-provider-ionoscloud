package ccpatch

import (
	"math/big"
	"net"
	"strings"

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
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	nextIP := big.NewInt(0).SetBytes(ipnet.IP)

	// Calculate the first host address within the CIDR range
	start := big.NewInt(1)
	nextIP.Add(nextIP, start)

	// Increment the nextIP until finding an unused IP address
	for {
		// Convert the nextIP to IPv6 format
		nextIPBytes := nextIP.Bytes()
		nextIPBytesPadded := make([]byte, net.IPv6len)
		copy(nextIPBytesPadded[net.IPv6len-len(nextIPBytes):], nextIPBytes)
		nextIPAddr := net.IP(nextIPBytesPadded)

		// Check if the next IP address is within the CIDR range and not in the usedAddresses list
		if ipnet.Contains(nextIPAddr) && !isUsedIPv6(nextIPAddr, usedAddresses) {
			return nextIPAddr, nil
		}

		// Increment the nextIP
		nextIP.Add(nextIP, big.NewInt(1))
	}
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
