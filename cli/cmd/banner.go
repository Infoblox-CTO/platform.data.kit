// Package cmd provides CLI commands for dk.
//
// banner.go renders a styled ASCII-art "DataKit" banner during interactive
// prompting sessions.  It is suppressed in non-TTY, piped, or narrow terminals.
package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// bannerArt is the ASCII art rendered during interactive prompting.
// Uses only standard ASCII characters for maximum terminal compatibility.
const bannerArt = `
  ____        _        _  ___ _
 |  _ \  __ _| |_ __ _| |/ (_) |_
 | | | |/ _` + "`" + ` | __/ _` + "`" + ` | ' /| | __|
 | |_| | (_| | || (_| | . \| | |_
 |____/ \__,_|\__\__,_|_|\_\_|\__|
`

// minBannerWidth is the minimum terminal width required to display the banner.
// Below this threshold the banner is omitted to avoid wrapping artifacts.
const minBannerWidth = 40

// ShowBanner renders the DataKit ASCII-art banner to stdout when running in
// an interactive TTY with sufficient width and color support.
//
// The banner is intentionally a no-op when:
//   - stdout is not a terminal (CI, piped output, redirected to file)
//   - the terminal is narrower than minBannerWidth columns
func ShowBanner() {
	fd := int(os.Stdout.Fd())

	// Only display in a real terminal.
	if !term.IsTerminal(fd) {
		return
	}

	// Check terminal width — skip if too narrow.
	width, _, err := term.GetSize(fd)
	if err == nil && width < minBannerWidth {
		return
	}

	// Render with lipgloss styling.
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")). // light blue / cyan
		Bold(true)

	fmt.Fprint(os.Stdout, style.Render(bannerArt))
	fmt.Fprintln(os.Stdout) // trailing newline for spacing
}
