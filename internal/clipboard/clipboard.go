// Package clipboard reads and writes the system clipboard by shelling out to
// whichever helper is available: wl-clipboard (Wayland), xclip, xsel, or — when
// running inside tmux on a headless box — tmux's own paste buffer (which can
// reach the real clipboard via tmux's set-clipboard/OSC52). The binary stays
// dependency-free and static; these tools are optional, so a box with none
// simply reports the clipboard unavailable.
package clipboard

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// ErrUnavailable means no supported clipboard helper is reachable.
var ErrUnavailable = errors.New("no clipboard available (need wl-clipboard, xclip, xsel, or tmux)")

// inTmux reports whether we're running inside a tmux session.
func inTmux() bool { return os.Getenv("TMUX") != "" }

// candidate is one clipboard helper invocation. eligible gates it beyond the
// binary simply existing on PATH (used to require an active tmux session).
type candidate struct {
	bin      string
	args     []string
	eligible bool
}

// Read returns the current clipboard text, or ErrUnavailable if no helper is
// reachable. A single trailing newline is trimmed. System tools take priority;
// tmux's buffer is the fallback for headless sessions.
func Read() (string, error) {
	for _, c := range []candidate{
		{"wl-paste", []string{"--no-newline"}, true},
		{"xclip", []string{"-selection", "clipboard", "-o"}, true},
		{"xsel", []string{"--clipboard", "--output"}, true},
		{"tmux", []string{"save-buffer", "-"}, inTmux()},
	} {
		if !c.eligible {
			continue
		}
		if _, err := exec.LookPath(c.bin); err != nil {
			continue
		}
		out, err := exec.Command(c.bin, c.args...).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(out), "\n"), nil
	}
	return "", ErrUnavailable
}

// Write sets the clipboard to text, or returns ErrUnavailable if no helper is
// reachable.
func Write(text string) error {
	for _, c := range []candidate{
		{"wl-copy", nil, true},
		{"xclip", []string{"-selection", "clipboard"}, true},
		{"xsel", []string{"--clipboard", "--input"}, true},
		// -w also pushes the buffer to the outer terminal's clipboard when
		// tmux's set-clipboard is on; we fall back to a plain load on failure.
		{"tmux", []string{"load-buffer", "-w", "-"}, inTmux()},
	} {
		if !c.eligible {
			continue
		}
		if _, err := exec.LookPath(c.bin); err != nil {
			continue
		}
		if err := run(c.bin, c.args, text); err != nil {
			if c.bin == "tmux" {
				return run("tmux", []string{"load-buffer", "-"}, text) // older tmux: no -w
			}
			return err
		}
		return nil
	}
	return ErrUnavailable
}

func run(bin string, args []string, stdin string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = bytes.NewBufferString(stdin)
	return cmd.Run()
}
