package tailscale

import (
	"reflect"
	"sort"
	"testing"
)

func TestParsePeersOnlineIPv4Only(t *testing.T) {
	data := []byte(`{
		"Peer": {
			"key1": {"TailscaleIPs": ["100.64.0.1", "fd7a:115c::1"], "Online": true},
			"key2": {"TailscaleIPs": ["100.64.0.2"], "Online": false},
			"key3": {"TailscaleIPs": ["fd7a:115c::3"], "Online": true},
			"key4": {"TailscaleIPs": ["100.64.0.4"], "Online": true}
		}
	}`)
	got := parsePeers(data)
	sort.Strings(got)
	want := []string{"100.64.0.1", "100.64.0.4"} // online + has IPv4; offline and v6-only excluded
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsePeers = %v, want %v", got, want)
	}
}

func TestParsePeersBadJSON(t *testing.T) {
	if got := parsePeers([]byte("not json")); got != nil {
		t.Errorf("bad JSON should yield nil, got %v", got)
	}
}

func TestIsIPv4(t *testing.T) {
	cases := map[string]bool{
		"100.64.0.1":   true,
		"1.2.3.4":      true,
		"fd7a:115c::1": false,
		"1.2.3":        false,
		"":             false,
		"abc":          false,
	}
	for in, want := range cases {
		if got := isIPv4(in); got != want {
			t.Errorf("isIPv4(%q) = %v, want %v", in, got, want)
		}
	}
}
