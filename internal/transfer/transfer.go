// Package transfer defines the shared progress-event vocabulary used by both
// the receiver (server) and the sender (client), so the TUI renders incoming
// and outgoing transfers uniformly.
package transfer

import "errors"

// ErrPinRequired is reported by the sender when a peer rejects prepare-upload
// with 401, i.e. it needs a PIN. The TUI prompts for one and retries.
var ErrPinRequired = errors.New("pin required")

// Direction is whether a transfer is incoming (we receive) or outgoing (we send).
type Direction int

const (
	Incoming Direction = iota
	Outgoing
)

// Kind classifies a transfer lifecycle event.
type Kind int

const (
	Start Kind = iota
	Progress
	FileDone
	Error
	Cancel
)

// Event reports the progress/lifecycle of one file. A transfer row is keyed by
// ID (sessionId:fileId).
type Event struct {
	Dir      Direction
	Kind     Kind
	ID       string
	FileName string
	Received int64
	Total    int64
	Err      error
}
