// Package tailscale discovers peer addresses from a local Tailscale daemon, so
// omarchy-send can reach devices that share a tailnet but not a LAN subnet (and
// therefore can't be found by multicast). It shells out to the `tailscale` CLI
// and is a no-op when that isn't present.
package tailscale

import (
	"context"
	"encoding/json"
	"os/exec"
)

// status is the subset of `tailscale status --json` we care about.
type status struct {
	Peer map[string]struct {
		TailscaleIPs []string `json:"TailscaleIPs"`
		Online       bool     `json:"Online"`
	} `json:"Peer"`
}

// Available reports whether the tailscale CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("tailscale")
	return err == nil
}

// Peers returns the IPv4 Tailscale address of each online peer in the tailnet.
// It returns nil (no error) when tailscale isn't installed, isn't running, or
// the output can't be parsed — callers treat Tailscale discovery as best-effort.
func Peers(ctx context.Context) []string {
	if !Available() {
		return nil
	}
	out, err := exec.CommandContext(ctx, "tailscale", "status", "--json").Output()
	if err != nil {
		return nil
	}
	return parsePeers(out)
}

// parsePeers extracts each online peer's first IPv4 address from the JSON of
// `tailscale status --json`. Split out so it can be tested without the CLI.
func parsePeers(data []byte) []string {
	var st status
	if err := json.Unmarshal(data, &st); err != nil {
		return nil
	}
	var hosts []string
	for _, p := range st.Peer {
		if !p.Online {
			continue
		}
		for _, ip := range p.TailscaleIPs {
			if isIPv4(ip) {
				hosts = append(hosts, ip)
				break // one address per peer is enough to probe
			}
		}
	}
	return hosts
}

// isIPv4 reports whether s looks like a dotted-quad (cheap check — avoids
// pulling in net just to skip the IPv6 entries Tailscale also reports).
func isIPv4(s string) bool {
	dots := 0
	for _, c := range s {
		switch {
		case c == '.':
			dots++
		case c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return dots == 3
}
