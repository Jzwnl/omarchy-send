// Package dbg provides opt-in debug logging to the file named by $OMARCHY_SEND_LOG.
// When the variable is unset, logging is a no-op. It is safe for concurrent use.
package dbg

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	once sync.Once
	mu   sync.Mutex
	f    *os.File
)

func setup() {
	path := os.Getenv("OMARCHY_SEND_LOG")
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}
	f = file
}

// Writer returns the debug log file if $OMARCHY_SEND_LOG is set, else io.Discard.
// It's used to redirect the standard logger away from the terminal so stray
// stdlib logging (e.g. net/http's "unsolicited response" notice) can't corrupt
// the TUI; the output is still captured in the debug log when enabled.
func Writer() io.Writer {
	once.Do(setup)
	if f == nil {
		return io.Discard
	}
	return f
}

// Logf appends a timestamped line to the debug log if $OMARCHY_SEND_LOG is set.
func Logf(format string, args ...any) {
	once.Do(setup)
	if f == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format("15:04:05.000")}, args...)...)
}
