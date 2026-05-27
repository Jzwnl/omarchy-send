package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fileItem adapts an entry in the receive directory to bubbles/list.Item.
type fileItem struct {
	path    string
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (i fileItem) Title() string       { return i.name }
func (i fileItem) Description() string  { return "" }
func (i fileItem) FilterValue() string { return i.name }

// receivedFiles lists the top-level entries in dir, newest first. In-progress
// transfers (".part" temp files) are skipped so the manage view never offers to
// delete a file that is still being written. A missing directory yields an empty
// slice and no error — nothing has been received yet.
func receivedFiles(dir string) ([]list.Item, error) {
	dir = expandHome(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	files := make([]fileItem, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".part") {
			continue // a transfer in flight; renamed into place on success
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileItem{
			path:    filepath.Join(dir, e.Name()),
			name:    e.Name(),
			size:    info.Size(),
			modTime: info.ModTime(),
			isDir:   e.IsDir(),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = f
	}
	return items, nil
}

// Manage list column widths.
const (
	colFileName = 36
	colFileSize = 12
)

// fileHeader is the dim column-header row shown above the received-files list.
// The leading pad covers the cursor bar (2) + mark column (2).
func fileHeader() string {
	h := lipgloss.NewStyle().Foreground(muted)
	return "    " +
		h.Width(colFileName).Render("Name") +
		h.Width(colFileSize).Render("Size") +
		h.Render("Received")
}

// fileDelegate renders each received file as one aligned table row:
//
//	▌ ✓ <name>   <size>   <received-at>
//
// It shares the Model's marked set by reference so toggles show immediately.
type fileDelegate struct{ marked map[string]bool }

func (fileDelegate) Height() int                         { return 1 }
func (fileDelegate) Spacing() int                        { return 0 }
func (fileDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d fileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(fileItem)
	if !ok {
		return
	}
	name := it.name
	if it.isDir {
		name += "/"
	}
	name = truncate(name, colFileName-2)

	check := "  "
	if d.marked[it.path] {
		check = lipgloss.NewStyle().Foreground(good).Render("✓ ")
	}
	size := "—"
	if !it.isDir {
		size = humanBytes(it.size)
	}
	date := it.modTime.Format("2006-01-02 15:04")

	nameSt := lipgloss.NewStyle().Width(colFileName)
	sizeSt := lipgloss.NewStyle().Width(colFileSize)

	if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().Foreground(accent).Render("▌ ")+check+
			nameSt.Foreground(accent).Bold(true).Render(name)+
			sizeSt.Foreground(accent).Render(size)+
			lipgloss.NewStyle().Foreground(accent).Render(date))
		return
	}
	fmt.Fprint(w, "  "+check+
		nameSt.Foreground(text).Render(name)+
		sizeSt.Foreground(dim).Render(size)+
		lipgloss.NewStyle().Foreground(muted).Render(date))
}

// manageView renders the received-files list plus a one-line status (delete
// error, or count marked) on the spare row beneath it.
func (m Model) manageView() string {
	var b strings.Builder
	b.WriteString(fileHeader())
	b.WriteString("\n")
	b.WriteString(m.fileList.View())
	switch {
	case m.manageErr != "":
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(bad).Render(m.manageErr))
	case len(m.marked) > 0:
		b.WriteString("\n" + titleStyle.Render(fmt.Sprintf("%d marked for deletion", len(m.marked))))
	}
	return b.String()
}

// confirmDeleteView is the centered card shown before files are removed.
func (m Model) confirmDeleteView() string {
	const maxShow = 8
	var b strings.Builder
	b.WriteString(titleStyle.Render("Delete files"))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render(fmt.Sprintf("Permanently delete %d item(s) from the receive folder?", len(m.delTargets))))
	b.WriteString("\n\n")
	for i, p := range m.delTargets {
		if i == maxShow {
			b.WriteString(headerStyle.Render(fmt.Sprintf("  … and %d more", len(m.delTargets)-maxShow)))
			b.WriteByte('\n')
			break
		}
		b.WriteString("  • " + filepath.Base(p) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("y/enter delete   ·   n/esc cancel"))
	return b.String()
}
