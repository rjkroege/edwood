// Package command provides command dispatch functionality for edwood.
// This package contains the dispatch table for user-facing commands (like Cut, Put, Get)
// and edit script commands (like 'a', 'd', 's' in the sam-like Edit language).
package command

import (
	"testing"
)

// =============================================================================
// Tests for CommandEntry type
// =============================================================================

// TestCommandEntryNew tests that a new CommandEntry is properly initialized.
func TestCommandEntryNew(t *testing.T) {
	ce := NewCommandEntry("Test", true, true, false)
	if ce == nil {
		t.Fatal("NewCommandEntry returned nil")
	}

	if ce.Name() != "Test" {
		t.Errorf("Name should be 'Test'; got %q", ce.Name())
	}
	if !ce.Mark() {
		t.Error("Mark should be true")
	}
	if !ce.Flag1() {
		t.Error("Flag1 should be true")
	}
	if ce.Flag2() {
		t.Error("Flag2 should be false")
	}
}

// TestCommandEntryFields tests the CommandEntry fields.
func TestCommandEntryFields(t *testing.T) {
	tests := []struct {
		name  string
		mark  bool
		flag1 bool
		flag2 bool
	}{
		{"Cut", true, true, true},
		{"Snarf", false, true, false},
		{"Del", false, false, true},
		{"Put", false, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ce := NewCommandEntry(tc.name, tc.mark, tc.flag1, tc.flag2)
			if ce.Name() != tc.name {
				t.Errorf("Name should be %q; got %q", tc.name, ce.Name())
			}
			if ce.Mark() != tc.mark {
				t.Errorf("Mark should be %v; got %v", tc.mark, ce.Mark())
			}
			if ce.Flag1() != tc.flag1 {
				t.Errorf("Flag1 should be %v; got %v", tc.flag1, ce.Flag1())
			}
			if ce.Flag2() != tc.flag2 {
				t.Errorf("Flag2 should be %v; got %v", tc.flag2, ce.Flag2())
			}
		})
	}
}

// =============================================================================
// Tests for EditEntry type
// =============================================================================

// TestEditEntryNew tests that a new EditEntry is properly initialized.
func TestEditEntryNew(t *testing.T) {
	ee := NewEditEntry('a', true, false, false, 0, DefaddrDot)
	if ee == nil {
		t.Fatal("NewEditEntry returned nil")
	}

	if ee.Char() != 'a' {
		t.Errorf("Char should be 'a'; got %c", ee.Char())
	}
	if !ee.Text() {
		t.Error("Text should be true")
	}
	if ee.Regexp() {
		t.Error("Regexp should be false")
	}
	if ee.Addr() {
		t.Error("Addr should be false")
	}
	if ee.DefCmd() != 0 {
		t.Errorf("DefCmd should be 0; got %c", ee.DefCmd())
	}
	if ee.DefAddr() != DefaddrDot {
		t.Errorf("DefAddr should be DefaddrDot; got %v", ee.DefAddr())
	}
}

// TestEditEntryFields tests the EditEntry fields for common edit commands.
func TestEditEntryFields(t *testing.T) {
	tests := []struct {
		char    rune
		text    bool
		regexp  bool
		addr    bool
		defcmd  rune
		defaddr Defaddr
	}{
		{'a', true, false, false, 0, DefaddrDot},    // append
		{'c', true, false, false, 0, DefaddrDot},    // change
		{'d', false, false, false, 0, DefaddrDot},   // delete
		{'s', false, true, false, 0, DefaddrDot},    // substitute (takes regexp)
		{'m', false, false, true, 0, DefaddrDot},    // move (takes address)
		{'x', false, true, false, 'p', DefaddrDot},  // extract (has default cmd)
		{'w', false, false, false, 0, DefaddrAll},   // write (default is all)
		{'e', false, false, false, 0, DefaddrNone},  // edit (no default addr)
	}

	for _, tc := range tests {
		t.Run(string(tc.char), func(t *testing.T) {
			ee := NewEditEntry(tc.char, tc.text, tc.regexp, tc.addr, tc.defcmd, tc.defaddr)
			if ee.Char() != tc.char {
				t.Errorf("Char should be %c; got %c", tc.char, ee.Char())
			}
			if ee.Text() != tc.text {
				t.Errorf("Text should be %v; got %v", tc.text, ee.Text())
			}
			if ee.Regexp() != tc.regexp {
				t.Errorf("Regexp should be %v; got %v", tc.regexp, ee.Regexp())
			}
			if ee.Addr() != tc.addr {
				t.Errorf("Addr should be %v; got %v", tc.addr, ee.Addr())
			}
			if ee.DefCmd() != tc.defcmd {
				t.Errorf("DefCmd should be %c; got %c", tc.defcmd, ee.DefCmd())
			}
			if ee.DefAddr() != tc.defaddr {
				t.Errorf("DefAddr should be %v; got %v", tc.defaddr, ee.DefAddr())
			}
		})
	}
}

// =============================================================================
// Tests for Dispatcher
// =============================================================================

// TestDispatcherNew tests that a new Dispatcher is properly initialized.
func TestDispatcherNew(t *testing.T) {
	d := NewDispatcher()
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}
}

// TestDispatcherRegisterCommand tests registering user commands.
func TestDispatcherRegisterCommand(t *testing.T) {
	d := NewDispatcher()

	cmd := NewCommandEntry("Cut", true, true, true)
	d.RegisterCommand(cmd)

	// Should be able to look it up
	found := d.LookupCommand("Cut")
	if found == nil {
		t.Fatal("LookupCommand returned nil for registered command")
	}
	if found.Name() != "Cut" {
		t.Errorf("found command Name should be 'Cut'; got %q", found.Name())
	}
}

// TestDispatcherRegisterEditCommand tests registering edit commands.
func TestDispatcherRegisterEditCommand(t *testing.T) {
	d := NewDispatcher()

	cmd := NewEditEntry('a', true, false, false, 0, DefaddrDot)
	d.RegisterEditCommand(cmd)

	// Should be able to look it up
	found := d.LookupEditCommand('a')
	if found == nil {
		t.Fatal("LookupEditCommand returned nil for registered command")
	}
	if found.Char() != 'a' {
		t.Errorf("found command Char should be 'a'; got %c", found.Char())
	}
}

// TestDispatcherLookupCommandNotFound tests lookup for non-existent command.
func TestDispatcherLookupCommandNotFound(t *testing.T) {
	d := NewDispatcher()

	// Looking up a command that doesn't exist should return nil
	found := d.LookupCommand("NonExistent")
	if found != nil {
		t.Error("LookupCommand should return nil for non-existent command")
	}
}

// TestDispatcherLookupEditCommandNotFound tests lookup for non-existent edit command.
func TestDispatcherLookupEditCommandNotFound(t *testing.T) {
	d := NewDispatcher()

	// Looking up a command that doesn't exist should return nil
	found := d.LookupEditCommand('z')
	if found != nil {
		t.Error("LookupEditCommand should return nil for non-existent command")
	}
}

// TestDispatcherLookupCommandNormalization tests that command lookup normalizes whitespace.
func TestDispatcherLookupCommandNormalization(t *testing.T) {
	d := NewDispatcher()
	cmd := NewCommandEntry("Cut", true, true, true)
	d.RegisterCommand(cmd)

	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"exact match", "Cut", true},
		{"leading whitespace", "  Cut", true},
		{"trailing content", "Cut extra args", true},
		{"tabs and newlines", "\t\nCut", true},
		{"wrong case", "cut", false}, // Commands are case-sensitive
		{"partial match", "Cu", false},
		{"no match", "Paste", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			found := d.LookupCommand(tc.input)
			if tc.found && found == nil {
				t.Errorf("expected to find command for %q", tc.input)
			}
			if !tc.found && found != nil {
				t.Errorf("expected nil for %q, got %v", tc.input, found)
			}
		})
	}
}

// TestDispatcherMultipleCommands tests registering and looking up multiple commands.
func TestDispatcherMultipleCommands(t *testing.T) {
	d := NewDispatcher()

	// Register multiple commands
	commands := []struct {
		name  string
		mark  bool
		flag1 bool
		flag2 bool
	}{
		{"Cut", true, true, true},
		{"Paste", true, true, false},
		{"Get", false, true, false},
		{"Put", false, false, false},
		{"Del", false, false, true},
	}

	for _, c := range commands {
		d.RegisterCommand(NewCommandEntry(c.name, c.mark, c.flag1, c.flag2))
	}

	// Verify each can be looked up
	for _, c := range commands {
		found := d.LookupCommand(c.name)
		if found == nil {
			t.Errorf("LookupCommand returned nil for %q", c.name)
			continue
		}
		if found.Name() != c.name {
			t.Errorf("found.Name() should be %q; got %q", c.name, found.Name())
		}
		if found.Mark() != c.mark {
			t.Errorf("%s: Mark should be %v; got %v", c.name, c.mark, found.Mark())
		}
	}
}

// TestDispatcherMultipleEditCommands tests registering and looking up multiple edit commands.
func TestDispatcherMultipleEditCommands(t *testing.T) {
	d := NewDispatcher()

	// Register multiple edit commands
	commands := []struct {
		char    rune
		text    bool
		regexp  bool
		defaddr Defaddr
	}{
		{'a', true, false, DefaddrDot},
		{'c', true, false, DefaddrDot},
		{'d', false, false, DefaddrDot},
		{'s', false, true, DefaddrDot},
		{'w', false, false, DefaddrAll},
	}

	for _, c := range commands {
		d.RegisterEditCommand(NewEditEntry(c.char, c.text, c.regexp, false, 0, c.defaddr))
	}

	// Verify each can be looked up
	for _, c := range commands {
		found := d.LookupEditCommand(c.char)
		if found == nil {
			t.Errorf("LookupEditCommand returned nil for %c", c.char)
			continue
		}
		if found.Char() != c.char {
			t.Errorf("found.Char() should be %c; got %c", c.char, found.Char())
		}
		if found.Text() != c.text {
			t.Errorf("%c: Text should be %v; got %v", c.char, c.text, found.Text())
		}
	}
}

// =============================================================================
// Tests for CommandList
// =============================================================================

// TestCommandListCommands tests the Commands() method.
func TestCommandListCommands(t *testing.T) {
	d := NewDispatcher()

	// Register some commands
	d.RegisterCommand(NewCommandEntry("Cut", true, true, true))
	d.RegisterCommand(NewCommandEntry("Paste", true, true, false))
	d.RegisterCommand(NewCommandEntry("Get", false, true, false))

	commands := d.Commands()
	if len(commands) != 3 {
		t.Errorf("Commands() should return 3 commands; got %d", len(commands))
	}
}

// TestEditCommandListCommands tests the EditCommands() method.
func TestEditCommandListCommands(t *testing.T) {
	d := NewDispatcher()

	// Register some edit commands
	d.RegisterEditCommand(NewEditEntry('a', true, false, false, 0, DefaddrDot))
	d.RegisterEditCommand(NewEditEntry('d', false, false, false, 0, DefaddrDot))
	d.RegisterEditCommand(NewEditEntry('s', false, true, false, 0, DefaddrDot))

	commands := d.EditCommands()
	if len(commands) != 3 {
		t.Errorf("EditCommands() should return 3 commands; got %d", len(commands))
	}
}

// =============================================================================
// Tests for Defaddr constants
// =============================================================================

// TestDefaddrConstants tests that Defaddr constants are distinct.
func TestDefaddrConstants(t *testing.T) {
	if DefaddrNone == DefaddrDot {
		t.Error("DefaddrNone should not equal DefaddrDot")
	}
	if DefaddrDot == DefaddrAll {
		t.Error("DefaddrDot should not equal DefaddrAll")
	}
	if DefaddrNone == DefaddrAll {
		t.Error("DefaddrNone should not equal DefaddrAll")
	}
}

// =============================================================================
// Tests for ParseCommandName
// =============================================================================

// TestParseCommandName tests extracting command name from input string.
func TestParseCommandName(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		args    string
		wantErr bool
	}{
		{"Cut", "Cut", "", false},
		{"Cut something", "Cut", "something", false},
		{"  Cut  ", "Cut", "", false},
		{"Put file.txt", "Put", "file.txt", false},
		{"Edit x/foo/bar/", "Edit", "x/foo/bar/", false},
		{"", "", "", true},       // empty input
		{"   ", "", "", true},    // whitespace only
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			name, args, err := ParseCommandName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if name != tc.name {
				t.Errorf("name should be %q; got %q", tc.name, name)
			}
			if args != tc.args {
				t.Errorf("args should be %q; got %q", tc.args, args)
			}
		})
	}
}

// =============================================================================
// Tests for integration with existing system
// =============================================================================

// TestDispatcherIntegration tests that the Dispatcher works as expected
// when set up similarly to the main package's globalexectab.
func TestDispatcherIntegration(t *testing.T) {
	d := NewDispatcher()

	// Register commands similar to globalexectab
	// Note: These match the actual commands in exec.go
	userCommands := []struct {
		name  string
		mark  bool
		flag1 bool
		flag2 bool
	}{
		{"Cut", true, true, true},
		{"Del", false, false, true},
		{"Delcol", false, true, true},
		{"Delete", false, true, true},
		{"Get", false, true, true},
		{"New", false, true, true},
		{"Newcol", false, true, true},
		{"Paste", true, true, true},
		{"Markdeep", false, true, true},
		{"Put", false, true, true},
		{"Putall", false, true, true},
		{"Redo", false, false, true},
		{"Snarf", false, true, false},
		{"Sort", false, true, true},
		{"Undo", false, true, true},
		{"Zerox", false, true, true},
	}

	for _, c := range userCommands {
		d.RegisterCommand(NewCommandEntry(c.name, c.mark, c.flag1, c.flag2))
	}

	// Register edit commands similar to cmdtab
	editCommands := []struct {
		char    rune
		text    bool
		regexp  bool
		addr    bool
		defcmd  rune
		defaddr Defaddr
	}{
		{'\n', false, false, false, 0, DefaddrDot},
		{'a', true, false, false, 0, DefaddrDot},
		{'c', true, false, false, 0, DefaddrDot},
		{'d', false, false, false, 0, DefaddrDot},
		{'i', true, false, false, 0, DefaddrDot},
		{'m', false, false, true, 0, DefaddrDot},
		{'p', false, false, false, 0, DefaddrDot},
		{'s', false, true, false, 0, DefaddrDot},
		{'t', false, false, true, 0, DefaddrDot},
		{'w', false, false, false, 0, DefaddrAll},
		{'x', false, true, false, 'p', DefaddrDot},
	}

	for _, c := range editCommands {
		d.RegisterEditCommand(NewEditEntry(c.char, c.text, c.regexp, c.addr, c.defcmd, c.defaddr))
	}

	// Verify we can look up Cut command and it has mark=true
	cut := d.LookupCommand("Cut")
	if cut == nil {
		t.Fatal("Cut command not found")
	}
	if !cut.Mark() {
		t.Error("Cut command should have Mark=true (undoable)")
	}

	// Verify Snarf uses the same function but different flags
	snarf := d.LookupCommand("Snarf")
	if snarf == nil {
		t.Fatal("Snarf command not found")
	}
	if snarf.Mark() {
		t.Error("Snarf command should have Mark=false")
	}
	if !snarf.Flag1() {
		t.Error("Snarf command should have Flag1=true")
	}
	if snarf.Flag2() {
		t.Error("Snarf command should have Flag2=false")
	}

	// Verify 's' (substitute) command takes a regexp
	sub := d.LookupEditCommand('s')
	if sub == nil {
		t.Fatal("substitute command not found")
	}
	if !sub.Regexp() {
		t.Error("substitute command should take regexp")
	}
	if sub.Text() {
		t.Error("substitute command should not take text")
	}

	// Verify 'a' (append) command takes text
	app := d.LookupEditCommand('a')
	if app == nil {
		t.Fatal("append command not found")
	}
	if !app.Text() {
		t.Error("append command should take text")
	}
	if app.Regexp() {
		t.Error("append command should not take regexp")
	}

	// Verify 'm' (move) command takes an address
	mov := d.LookupEditCommand('m')
	if mov == nil {
		t.Fatal("move command not found")
	}
	if !mov.Addr() {
		t.Error("move command should take address")
	}

	// Verify 'w' (write) has default address of all
	wrt := d.LookupEditCommand('w')
	if wrt == nil {
		t.Fatal("write command not found")
	}
	if wrt.DefAddr() != DefaddrAll {
		t.Errorf("write command should have DefAddr=DefaddrAll; got %v", wrt.DefAddr())
	}
}

// TestDispatcherEmptyInput tests behavior with empty or whitespace input.
func TestDispatcherEmptyInput(t *testing.T) {
	d := NewDispatcher()
	d.RegisterCommand(NewCommandEntry("Test", false, false, false))

	// Empty string should return nil
	if d.LookupCommand("") != nil {
		t.Error("empty string should return nil")
	}

	// Whitespace only should return nil
	if d.LookupCommand("   ") != nil {
		t.Error("whitespace-only string should return nil")
	}

	if d.LookupCommand("\t\n") != nil {
		t.Error("tab/newline only string should return nil")
	}
}
