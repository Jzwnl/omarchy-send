package discovery

import (
	"context"
	"testing"
	"time"
)

// FindPeer returns immediately for a peer already in the snapshot.
func TestFindPeerAlreadyKnown(t *testing.T) {
	d := New(mkInfo("self", "selffp", 0))
	d.NotePeer(mkInfo("Strong Onion", "peerfp", 53317), "192.168.1.50")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Match is case-insensitive and whitespace-trimmed, like the CLI.
	got, err := d.FindPeer(ctx, func(p Peer) bool {
		return p.Info.Alias == "Strong Onion"
	})
	if err != nil {
		t.Fatalf("FindPeer: %v", err)
	}
	if got.IP != "192.168.1.50" {
		t.Errorf("IP = %q, want 192.168.1.50", got.IP)
	}
}

// FindPeer resolves a peer that only appears after the call starts.
func TestFindPeerArrivesLater(t *testing.T) {
	d := New(mkInfo("self", "selffp", 0))

	go func() {
		time.Sleep(50 * time.Millisecond)
		d.NotePeer(mkInfo("Latecomer", "latefp", 53317), "10.0.0.9")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := d.FindPeer(ctx, func(p Peer) bool { return p.Info.Alias == "Latecomer" })
	if err != nil {
		t.Fatalf("FindPeer: %v", err)
	}
	if got.IP != "10.0.0.9" {
		t.Errorf("IP = %q, want 10.0.0.9", got.IP)
	}
}

// FindPeer returns the context error when no match arrives in time.
func TestFindPeerTimeout(t *testing.T) {
	d := New(mkInfo("self", "selffp", 0))
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if _, err := d.FindPeer(ctx, func(Peer) bool { return false }); err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
}
