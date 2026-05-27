package clipboard

import (
	"errors"
	"testing"
)

// With no helpers reachable on PATH, both operations report ErrUnavailable
// rather than panicking or hanging. (We empty PATH instead of touching the real
// clipboard, so the test is safe to run anywhere.)
func TestUnavailableWithoutTools(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := Read(); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Read with no tools: got %v, want ErrUnavailable", err)
	}
	if err := Write("x"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Write with no tools: got %v, want ErrUnavailable", err)
	}
}
