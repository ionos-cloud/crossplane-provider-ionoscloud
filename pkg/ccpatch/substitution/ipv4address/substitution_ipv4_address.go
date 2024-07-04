package ccpatch

import (
	"fmt"
	"net"
	"slices"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/ipnet"
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

func (i *ipv4Address) WriteGlobalState(identifier substitution.Identifier, gs *substitution.GlobalState, sub substitution.Substitution) error {
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
	gen, err := ipnet.New(cidr)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(used, gen.IP.String()) && !gen.First().Equal(gen.IP) {
		return gen.IP, nil
	}

	for {
		next := gen.Next()

		if next == nil {
			break
		}

		if !slices.Contains(used, next.String()) && !gen.First().Equal(next) {
			return next, nil
		}
	}

	return nil, fmt.Errorf("no more available IPv4 addresses in %s", cidr)
}
