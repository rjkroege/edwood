// Package command provides command dispatch functionality for edwood.
// This package contains the dispatch table for user-facing commands (like Cut, Put, Get)
// and edit script commands (like 'a', 'd', 's' in the sam-like Edit language).
package command

import (
	"errors"
	"regexp"
	"strings"
)

// Defaddr represents the default address type for an edit command.
type Defaddr int

const (
	// DefaddrNone means the command takes no address
	DefaddrNone Defaddr = iota
	// DefaddrDot means the command defaults to the current selection (dot)
	DefaddrDot
	// DefaddrAll means the command defaults to the entire file
	DefaddrAll
)

// CommandEntry represents a user-facing command (like Cut, Put, Get, etc.)
// that can be invoked via middle-click in the editor.
type CommandEntry struct {
	name  string
	mark  bool // Command is undoable (requires establishing an undo point)
	flag1 bool // Command-specific flag
	flag2 bool // Command-specific flag
}

// NewCommandEntry creates a new CommandEntry.
func NewCommandEntry(name string, mark, flag1, flag2 bool) *CommandEntry {
	return &CommandEntry{
		name:  name,
		mark:  mark,
		flag1: flag1,
		flag2: flag2,
	}
}

// Name returns the command name.
func (c *CommandEntry) Name() string { return c.name }

// Mark returns whether this command is undoable.
func (c *CommandEntry) Mark() bool { return c.mark }

// Flag1 returns the first command-specific flag.
func (c *CommandEntry) Flag1() bool { return c.flag1 }

// Flag2 returns the second command-specific flag.
func (c *CommandEntry) Flag2() bool { return c.flag2 }

// EditEntry represents an edit script command (like 'a', 'd', 's', etc.)
// that can be invoked via the Edit command.
type EditEntry struct {
	char    rune    // Command character
	text    bool    // Takes a textual argument?
	regexp  bool    // Takes a regular expression?
	addr    bool    // Takes an address (m or t)?
	defcmd  rune    // Default subcommand; 0 means none
	defaddr Defaddr // Default address
}

// NewEditEntry creates a new EditEntry.
func NewEditEntry(char rune, text, regexp, addr bool, defcmd rune, defaddr Defaddr) *EditEntry {
	return &EditEntry{
		char:    char,
		text:    text,
		regexp:  regexp,
		addr:    addr,
		defcmd:  defcmd,
		defaddr: defaddr,
	}
}

// Char returns the command character.
func (e *EditEntry) Char() rune { return e.char }

// Text returns whether this command takes a textual argument.
func (e *EditEntry) Text() bool { return e.text }

// Regexp returns whether this command takes a regular expression.
func (e *EditEntry) Regexp() bool { return e.regexp }

// Addr returns whether this command takes an address.
func (e *EditEntry) Addr() bool { return e.addr }

// DefCmd returns the default subcommand (0 if none).
func (e *EditEntry) DefCmd() rune { return e.defcmd }

// DefAddr returns the default address type.
func (e *EditEntry) DefAddr() Defaddr { return e.defaddr }

// Dispatcher manages command lookup and dispatch for both
// user-facing commands and edit script commands.
type Dispatcher struct {
	commands     []*CommandEntry
	editCommands []*EditEntry
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		commands:     make([]*CommandEntry, 0),
		editCommands: make([]*EditEntry, 0),
	}
}

// RegisterCommand adds a user-facing command to the dispatcher.
func (d *Dispatcher) RegisterCommand(cmd *CommandEntry) {
	d.commands = append(d.commands, cmd)
}

// RegisterEditCommand adds an edit script command to the dispatcher.
func (d *Dispatcher) RegisterEditCommand(cmd *EditEntry) {
	d.editCommands = append(d.editCommands, cmd)
}

// wsre matches whitespace for command name normalization.
var wsre = regexp.MustCompile(`[ \t\n]+`)

// LookupCommand finds a user-facing command by name.
// The name is normalized (whitespace collapsed, leading whitespace trimmed).
// Returns nil if the command is not found.
func (d *Dispatcher) LookupCommand(name string) *CommandEntry {
	// Normalize whitespace
	name = wsre.ReplaceAllString(name, " ")
	name = strings.TrimLeft(name, " ")

	if name == "" {
		return nil
	}

	// Extract just the command name (first word)
	words := strings.SplitN(name, " ", 2)
	cmdName := words[0]

	for _, cmd := range d.commands {
		if cmd.name == cmdName {
			return cmd
		}
	}
	return nil
}

// LookupEditCommand finds an edit script command by character.
// Returns nil if the command is not found.
func (d *Dispatcher) LookupEditCommand(char rune) *EditEntry {
	for _, cmd := range d.editCommands {
		if cmd.char == char {
			return cmd
		}
	}
	return nil
}

// Commands returns all registered user-facing commands.
func (d *Dispatcher) Commands() []*CommandEntry {
	return d.commands
}

// EditCommands returns all registered edit script commands.
func (d *Dispatcher) EditCommands() []*EditEntry {
	return d.editCommands
}

// ErrEmptyInput is returned when parsing an empty command string.
var ErrEmptyInput = errors.New("empty command input")

// ParseCommandName extracts the command name and arguments from an input string.
// Returns the command name, remaining arguments, and any error.
func ParseCommandName(input string) (name, args string, err error) {
	// Normalize whitespace
	input = wsre.ReplaceAllString(input, " ")
	input = strings.TrimSpace(input)

	if input == "" {
		return "", "", ErrEmptyInput
	}

	// Split into command name and arguments
	words := strings.SplitN(input, " ", 2)
	name = words[0]
	if len(words) > 1 {
		args = words[1]
	}

	return name, args, nil
}
