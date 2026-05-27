// Package app bridges the Tea-agnostic domain layer (discovery, server, client)
// into Bubble Tea messages. Domain components emit on Go channels; a bridge
// goroutine forwards them via tea.Program.Send.
package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"omarchy-send/internal/discovery"
	"omarchy-send/internal/server"
	"omarchy-send/internal/transfer"
)

// PeerFoundMsg is delivered when a peer is discovered or its address changes.
type PeerFoundMsg struct{ Peer discovery.Peer }

// PeerLostMsg is delivered when a peer ages out (post-M1).
type PeerLostMsg struct{ Fingerprint string }

// IncomingMsg is delivered when a peer asks to send us files. The TUI shows an
// accept prompt and answers via the carried Reply channel.
type IncomingMsg struct{ Req server.AcceptRequest }

// TransferMsg reports progress/lifecycle of a transfer (either direction).
type TransferMsg struct{ Ev transfer.Event }

// MessageMsg is delivered when a peer sends us a plain-text message.
type MessageMsg struct{ Msg server.ReceivedMessage }

// BridgeServer forwards the server's accept requests, transfer events, and
// received messages to the Tea program until ctx is cancelled.
func BridgeServer(ctx context.Context, accepts <-chan server.AcceptRequest, transfers <-chan transfer.Event, messages <-chan server.ReceivedMessage, send func(tea.Msg)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-accepts:
				send(IncomingMsg{Req: req})
			case ev := <-transfers:
				send(TransferMsg{Ev: ev})
			case m := <-messages:
				send(MessageMsg{Msg: m})
			}
		}
	}()
}

// BridgeTransfers forwards a transfer event channel (e.g. the sender's) to the
// Tea program until ctx is cancelled.
func BridgeTransfers(ctx context.Context, transfers <-chan transfer.Event, send func(tea.Msg)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-transfers:
				send(TransferMsg{Ev: ev})
			}
		}
	}()
}

// BridgeDiscovery forwards discovery events to the Tea program until ctx is
// cancelled. send is typically tea.Program.Send.
func BridgeDiscovery(ctx context.Context, events <-chan discovery.Event, send func(tea.Msg)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-events:
				switch ev.Kind {
				case discovery.PeerFound:
					send(PeerFoundMsg{Peer: ev.Peer})
				case discovery.PeerLost:
					send(PeerLostMsg{Fingerprint: ev.Peer.Info.Fingerprint})
				}
			}
		}
	}()
}
