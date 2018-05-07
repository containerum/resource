package host2dnslabel

import (
	"testing"
)

func TestHost2DNSLabel(test *testing.T) {
	var hosts = []string{
		"google.com",
		"123.com",
		"asndl0-😏ю.loc",
		"as=-0e20 -doqd- 3-- -s.saalc=asd.cpks",
		"asdasd d-ds d----- --.net",
		"ßßßß",
		"GOOGLE.ComА",
		"test_test.com",
	}
	for _, host := range hosts {
		DNSlabel, err := Host2DNSLabel(host)
		if err != nil {
			test.Logf("%q -> %q (Error: %q)", host, DNSlabel, err)
		} else {
			test.Logf("%q -> %q", host, DNSlabel)
		}
	}
}
