// Package prompt provides interactive terminal prompting helpers built on
// charmbracelet/huh. Every prompt has a corresponding flag/env-var equivalent
// so CI pipelines never need to interact.
package prompt

import (
	"os"

	"golang.org/x/term"
)

// IsInteractive returns true when stdin is a terminal, meaning
// the user can respond to interactive prompts.  In CI or piped
// contexts it returns false so commands should require explicit flags.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
