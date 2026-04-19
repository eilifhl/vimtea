// Package vimtea provides a Vim-like text editor component for terminal applications
package vimtea

import tea "github.com/charmbracelet/bubbletea"

// Command is a function that performs an action on the editor model
// and returns a bubbletea command
type Command func(m *editorModel) tea.Cmd

// KeyBinding represents a key binding that can be registered with the editor
// This is the public API for adding key bindings
type KeyBinding struct {
	Key         string               // The key sequence to bind (e.g. "j", "dd", "ctrl+f")
	Mode        EditorMode           // Which editor mode this binding is active in
	Description string               // Human-readable description for help screens
	Handler     func(Buffer) tea.Cmd // Function to execute when the key is pressed
}

// UndoRedoMsg is sent when an undo or redo operation is performed
// It contains the new cursor position and operation status
type UndoRedoMsg struct {
	NewCursor Cursor // New cursor position after undo/redo
	Success   bool   // Whether the operation succeeded
	IsUndo    bool   // True for undo, false for redo
}

// internalKeyBinding is the internal representation of a key binding
// used by the binding registry
type internalKeyBinding struct {
	Key     string     // The key sequence
	Command Command    // The command function to execute
	Mode    EditorMode // The editor mode this binding is active in
	Help    string     // Help text describing the binding
}

// CommandRegistry stores and manages commands that can be executed in command mode
// Commands are invoked by typing ":command" in command mode
type CommandRegistry struct {
	commands map[string]Command // Map of command names to command functions
}

// BindingRegistry manages key bindings for the editor
// It supports exact matches and prefix detection for multi-key sequences
type BindingRegistry struct {
	// Maps EditorMode -> key sequence -> binding
	exactBindings map[EditorMode]map[string]internalKeyBinding

	// Maps EditorMode -> key prefix -> true
	// Used to detect if a key sequence could be a prefix of a longer binding
	prefixBindings map[EditorMode]map[string]bool

	// List of all bindings for help display
	allBindings []internalKeyBinding
}

// newBindingRegistry creates a new empty binding registry
func newBindingRegistry() *BindingRegistry {
	return &BindingRegistry{
		exactBindings:  make(map[EditorMode]map[string]internalKeyBinding),
		prefixBindings: make(map[EditorMode]map[string]bool),
		allBindings:    []internalKeyBinding{},
	}
}

// newCommandRegistry creates a new empty command registry
func newCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]Command),
	}
}

// Add registers a new key binding with the registry
// It automatically builds prefix maps for multi-key sequences
func (r *BindingRegistry) Add(key string, cmd Command, mode EditorMode, help string) {
	binding := internalKeyBinding{
		Key:     key,
		Command: cmd,
		Mode:    mode,
		Help:    help,
	}

	// Initialize mode map if needed
	if r.exactBindings[mode] == nil {
		r.exactBindings[mode] = make(map[string]internalKeyBinding)
	}
	r.exactBindings[mode][key] = binding

	// Initialize prefix map if needed
	if r.prefixBindings[mode] == nil {
		r.prefixBindings[mode] = make(map[string]bool)
	}

	// Register all prefixes of the key sequence
	// For example, for "dw", register "d" as a prefix
	for i := 1; i < len(key); i++ {
		prefix := key[:i]
		r.prefixBindings[mode][prefix] = true
	}

	// Add to the list of all bindings
	r.allBindings = append(r.allBindings, binding)
}

// FindExact looks for an exact match for the given key sequence in the specified mode
// It can handle numeric prefixes by ignoring them when looking for the command
func (r *BindingRegistry) FindExact(keySeq string, mode EditorMode) *internalKeyBinding {
	// Find where the numeric prefix ends (if any)
	nonDigitStart := 0
	for i, c := range keySeq {
		if c < '0' || c > '9' {
			nonDigitStart = i
			break
		}
	}

	// If the sequence is all digits, it's not a command
	if nonDigitStart == len(keySeq) {
		return nil
	}

	// Try to match without the numeric prefix
	cmdPart := keySeq[nonDigitStart:]
	if modeBindings, ok := r.exactBindings[mode]; ok {
		if binding, ok := modeBindings[cmdPart]; ok {
			return &binding
		}
	}

	// Try to match the full sequence (including any numeric prefix)
	if modeBindings, ok := r.exactBindings[mode]; ok {
		if binding, ok := modeBindings[keySeq]; ok {
			return &binding
		}
	}

	return nil
}

// IsPrefix checks if the key sequence is a prefix of any registered binding
// This is used to determine if we should wait for more input
func (r *BindingRegistry) IsPrefix(keySeq string, mode EditorMode) bool {
	if prefixes, ok := r.prefixBindings[mode]; ok {
		return prefixes[keySeq]
	}
	return false
}

// GetAll returns all registered key bindings
func (r *BindingRegistry) GetAll() []internalKeyBinding {
	return r.allBindings
}

// GetForMode returns all key bindings for the specified mode
func (r *BindingRegistry) GetForMode(mode EditorMode) []internalKeyBinding {
	var result []internalKeyBinding
	for _, binding := range r.allBindings {
		if binding.Mode == mode {
			result = append(result, binding)
		}
	}
	return result
}

// Register adds a command to the registry with the given name
func (r *CommandRegistry) Register(name string, cmd Command) {
	r.commands[name] = cmd
}

// Get retrieves a command by name, returning nil if not found
func (r *CommandRegistry) Get(name string) Command {
	cmd, ok := r.commands[name]
	if !ok {
		return nil
	}
	return cmd
}

// GetAll returns all registered commands as a map
func (r *CommandRegistry) GetAll() map[string]Command {
	return r.commands
}

func registerBindings(m *editorModel) {
	m.registry.Add("i", enterModeInsert, ModeNormal, "Enter insert mode")
	m.registry.Add("v", beginVisualSelection, ModeNormal, "Enter visual mode")
	m.registry.Add("V", beginVisualLineSelection, ModeNormal, "Enter visual line mode")
	m.registry.Add("x", deleteCharAtCursor, ModeNormal, "Delete character at cursor")
	if m.enableCommandMode {
		m.registry.Add(":", enterModeCommand, ModeNormal, "Enter command mode")
	}

	m.registry.Add("a", appendAfterCursor, ModeNormal, "Append after cursor")
	m.registry.Add("A", appendAtEndOfLine, ModeNormal, "Append at end of line")
	m.registry.Add("I", insertAtStartOfLine, ModeNormal, "Insert at start of line")
	m.registry.Add("o", openLineBelow, ModeNormal, "Open line below")
	m.registry.Add("O", openLineAbove, ModeNormal, "Open line above")

	m.registry.Add("yy", yankLine, ModeNormal, "Yank line")
	m.registry.Add("dd", deleteLine, ModeNormal, "Delete line")
	m.registry.Add("D", deleteToEndOfLine, ModeNormal, "Delete to end of line")
	m.registry.Add("dw", deleteWord, ModeNormal, "Delete word")
	m.registry.Add("cw", changeWord, ModeNormal, "Change word")
	m.registry.Add("p", pasteAfter, ModeNormal, "Paste after cursor")
	m.registry.Add("P", pasteBefore, ModeNormal, "Paste before cursor")

	m.registry.Add("u", undo, ModeNormal, "Undo")
	m.registry.Add("ctrl+r", redo, ModeNormal, "Redo")
	m.registry.Add("diw", deleteInnerWord, ModeNormal, "Delete inner word")
	m.registry.Add("yiw", yankInnerWord, ModeNormal, "Yank inner word")
	m.registry.Add("ciw", changeInnerWord, ModeNormal, "Change inner word")
	m.registry.Add("di\"", func(m *editorModel) tea.Cmd { return deleteInnerQuote(m, '"') }, ModeNormal, "Delete inner double quotes")
	m.registry.Add("yi\"", func(m *editorModel) tea.Cmd { return yankInnerQuote(m, '"') }, ModeNormal, "Yank inner double quotes")
	m.registry.Add("ci\"", func(m *editorModel) tea.Cmd { return changeInnerQuote(m, '"') }, ModeNormal, "Change inner double quotes")
	m.registry.Add("da\"", func(m *editorModel) tea.Cmd { return deleteAroundQuote(m, '"') }, ModeNormal, "Delete around double quotes")
	m.registry.Add("ya\"", func(m *editorModel) tea.Cmd { return yankAroundQuote(m, '"') }, ModeNormal, "Yank around double quotes")
	m.registry.Add("ca\"", func(m *editorModel) tea.Cmd { return changeAroundQuote(m, '"') }, ModeNormal, "Change around double quotes")
	m.registry.Add("di(", deleteInnerParen, ModeNormal, "Delete inner parentheses")
	m.registry.Add("yi(", yankInnerParen, ModeNormal, "Yank inner parentheses")
	m.registry.Add("ci(", changeInnerParen, ModeNormal, "Change inner parentheses")
	m.registry.Add("di'", func(m *editorModel) tea.Cmd { return deleteInnerQuote(m, '\'') }, ModeNormal, "Delete inner single quotes")
	m.registry.Add("yi'", func(m *editorModel) tea.Cmd { return yankInnerQuote(m, '\'') }, ModeNormal, "Yank inner single quotes")
	m.registry.Add("ci'", func(m *editorModel) tea.Cmd { return changeInnerQuote(m, '\'') }, ModeNormal, "Change inner single quotes")
	m.registry.Add("da'", func(m *editorModel) tea.Cmd { return deleteAroundQuote(m, '\'') }, ModeNormal, "Delete around single quotes")
	m.registry.Add("ya'", func(m *editorModel) tea.Cmd { return yankAroundQuote(m, '\'') }, ModeNormal, "Yank around single quotes")
	m.registry.Add("ca'", func(m *editorModel) tea.Cmd { return changeAroundQuote(m, '\'') }, ModeNormal, "Change around single quotes")
	m.registry.Add("dib", deleteInnerParen, ModeNormal, "Delete inner parentheses")
	m.registry.Add("yib", yankInnerParen, ModeNormal, "Yank inner parentheses")
	m.registry.Add("cib", changeInnerParen, ModeNormal, "Change inner parentheses")

	for _, mode := range []EditorMode{ModeNormal, ModeVisual} {
		m.registry.Add("h", moveCursorLeft, mode, "Move cursor left")
		m.registry.Add("j", moveCursorDown, mode, "Move cursor down")
		m.registry.Add("k", moveCursorUp, mode, "Move cursor up")
		m.registry.Add("l", moveCursorRight, mode, "Move cursor right")
		m.registry.Add("w", moveToNextWordStart, mode, "Move to next word")
		m.registry.Add("b", moveToPrevWordStart, mode, "Move to previous word")
		m.registry.Add("W", directMotionCommand("W"), mode, "Move to next WORD")
		m.registry.Add("B", directMotionCommand("B"), mode, "Move to previous WORD")
		m.registry.Add("e", directMotionCommand("e"), mode, "Move to end of word")
		m.registry.Add("E", directMotionCommand("E"), mode, "Move to end of WORD")
		m.registry.Add("ge", directMotionCommand("ge"), mode, "Move to end of previous word")
		m.registry.Add("gE", directMotionCommand("gE"), mode, "Move to end of previous WORD")

		m.registry.Add(" ", moveCursorRightOrNextLine, mode, "Move cursor right")
		m.registry.Add("0", moveToStartOfLine, mode, "Move to start of line")
		m.registry.Add("^", moveToFirstNonWhitespace, mode, "Move to first non-whitespace character")
		m.registry.Add("$", moveToEndOfLine, mode, "Move to end of line")
		m.registry.Add("g_", directMotionCommand("g_"), mode, "Move to last non-whitespace character")
		m.registry.Add("gg", moveToStartOfDocument, mode, "Move to document start")
		m.registry.Add("G", moveToEndOfDocument, mode, "Move to document end")
		m.registry.Add("%", directMotionCommand("%"), mode, "Move to matching bracket")
		m.registry.Add("{", directMotionCommand("{"), mode, "Move to previous paragraph")
		m.registry.Add("}", directMotionCommand("}"), mode, "Move to next paragraph")
		m.registry.Add("H", directMotionCommand("H"), mode, "Move to top of screen")
		m.registry.Add("M", directMotionCommand("M"), mode, "Move to middle of screen")
		m.registry.Add("L", directMotionCommand("L"), mode, "Move to bottom of screen")
		m.registry.Add("f", beginFindMotion("f"), mode, "Find character forward")
		m.registry.Add("F", beginFindMotion("F"), mode, "Find character backward")
		m.registry.Add("t", beginFindMotion("t"), mode, "Move before character forward")
		m.registry.Add("T", beginFindMotion("T"), mode, "Move after character backward")
		m.registry.Add(";", repeatLastFind(false), mode, "Repeat latest find")
		m.registry.Add(",", repeatLastFind(true), mode, "Repeat latest find in reverse")

		m.registry.Add("up", moveCursorUp, mode, "Move cursor up")
		m.registry.Add("down", moveCursorDown, mode, "Move cursor down")
		m.registry.Add("left", moveCursorLeft, mode, "Move cursor left")
		m.registry.Add("right", moveCursorRight, mode, "Move cursor right")
	}

	m.registry.Add("esc", exitModeVisual, ModeVisual, "Exit visual mode")
	m.registry.Add("v", exitModeVisual, ModeVisual, "Exit visual mode")
	m.registry.Add("V", exitModeVisual, ModeVisual, "Exit visual mode")
	m.registry.Add(":", enterModeCommand, ModeVisual, "Enter command mode")
	m.registry.Add("y", yankVisualSelection, ModeVisual, "Yank selection")
	m.registry.Add("d", deleteVisualSelection, ModeVisual, "Delete selection")
	m.registry.Add("x", deleteVisualSelection, ModeVisual, "Delete selection")
	m.registry.Add("p", replaceVisualSelectionWithYank, ModeVisual, "Replace with yanked text")

	m.registry.Add("esc", exitModeInsert, ModeInsert, "Exit insert mode")
	m.registry.Add("backspace", handleInsertBackspace, ModeInsert, "Backspace")
	m.registry.Add("del", handleInsertDeleteForward, ModeInsert, "Delete character under cursor")
	m.registry.Add("ctrl+d", handleInsertDeleteForward, ModeInsert, "Delete character under cursor")
	m.registry.Add("delete", handleInsertDeleteForward, ModeInsert, "Delete character under cursor")
	m.registry.Add("ctrl+k", handleInsertKillToEndOfLine, ModeInsert, "Kill to end of line")
	m.registry.Add("ctrl+u", handleInsertKillToStartOfLine, ModeInsert, "Kill to start of line")
	m.registry.Add("ctrl+w", handleInsertDeletePreviousWord, ModeInsert, "Delete previous word")
	m.registry.Add("ctrl+y", handleInsertYank, ModeInsert, "Yank text")
	m.registry.Add("alt+d", handleInsertDeleteNextWord, ModeInsert, "Delete next word")
	m.registry.Add("alt+delete", handleInsertDeleteNextWord, ModeInsert, "Delete next word")
	m.registry.Add("alt+backspace", handleInsertDeletePreviousWord, ModeInsert, "Delete previous word")
	m.registry.Add("alt+y", handleInsertYankPop, ModeInsert, "Cycle yanked text")
	m.registry.Add("ctrl+t", handleInsertTransposeCharacters, ModeInsert, "Transpose characters")
	m.registry.Add("alt+t", handleInsertTransposeWords, ModeInsert, "Transpose words")
	m.registry.Add("ctrl+_", undo, ModeInsert, "Undo")
	m.registry.Add("tab", handleInsertTab, ModeInsert, "Tab")
	m.registry.Add("enter", handleInsertEnterKey, ModeInsert, "Enter")
	m.registry.Add("ctrl+o", handleInsertOpenLine, ModeInsert, "Open line")
	m.registry.Add("ctrl+a", moveToStartOfLine, ModeInsert, "Move to start of line")
	m.registry.Add("ctrl+e", moveToEndOfLine, ModeInsert, "Move to end of line")
	m.registry.Add("ctrl+b", moveCursorLeft, ModeInsert, "Move cursor left")
	m.registry.Add("ctrl+f", moveCursorRight, ModeInsert, "Move cursor right")
	m.registry.Add("ctrl+p", moveCursorUp, ModeInsert, "Move cursor up")
	m.registry.Add("ctrl+n", moveCursorDown, ModeInsert, "Move cursor down")
	m.registry.Add("alt+b", moveToPrevWordStart, ModeInsert, "Move to previous word")
	m.registry.Add("alt+f", moveToNextWordStart, ModeInsert, "Move to next word")
	m.registry.Add("alt+<", directMotionCommand("gg"), ModeInsert, "Move to start of document")
	m.registry.Add("alt+>", directMotionCommand("G"), ModeInsert, "Move to end of document")
	m.registry.Add("ctrl+left", moveToPrevWordStart, ModeInsert, "Move to previous word")
	m.registry.Add("ctrl+right", moveToNextWordStart, ModeInsert, "Move to next word")
	m.registry.Add("up", handleArrowKeys("up"), ModeInsert, "Move cursor up")
	m.registry.Add("down", handleArrowKeys("down"), ModeInsert, "Move cursor down")
	m.registry.Add("left", handleArrowKeys("left"), ModeInsert, "Move cursor left")
	m.registry.Add("right", handleArrowKeys("right"), ModeInsert, "Move cursor right")

	m.registry.Add("esc", exitModeCommand, ModeCommand, "Exit command mode")
	m.registry.Add("enter", executeCommand, ModeCommand, "Execute command")
	m.registry.Add("backspace", commandBackspace, ModeCommand, "Backspace")

	m.commands.Register("zr", toggleRelativeLineNumbers)
	m.commands.Register("clear", clearBuffer)
	m.commands.Register("reset", resetEditor)
}
