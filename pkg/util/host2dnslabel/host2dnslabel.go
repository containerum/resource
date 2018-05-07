package host2dnslabel

import (
	"golang.org/x/net/idna"
)

func Host2DNSLabel(hostname string) (string, error) {
	return idna.Lookup.ToASCII(hostname)
}
