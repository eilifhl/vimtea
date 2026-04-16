// Package vimtea provides a Vim-like text editor component for terminal applications
package vimtea

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandFn is a function that can be executed when a command is run in command mode
// It takes a buffer reference and command arguments, and returns a bubbletea command
type CommandFn func(Buffer, []string) tea.Cmd

// CommandMsg is sent when a command is executed from command mode
// It contains the command name that should be looked up in the CommandRegistry
type CommandMsg struct {
	Command string // Command name without arguments
}

// withCountPrefix executes a function multiple times based on the numeric prefix
// This implements commands like "5j" to move down 5 lines
func withCountPrefix(model *editorModel, fn func()) {
	count := model.countPrefix
	for range count {
		fn()
	}
	model.countPrefix = 1
}

// switchMode changes the editor mode and performs necessary setup for the new mode
// Different modes require different cursor handling and UI state
func switchMode(model *editorModel, newMode EditorMode) tea.Cmd {
	model.mode = newMode
	model.pendingReplace = false
	model.pendingOperator = ""
	model.operatorSequence = nil

	switch newMode {
	case ModeNormal:
		// In normal mode, cursor can't be at end of line
		if model.buffer.lineLength(model.cursor.Row) > 0 &&
			model.cursor.Col >= model.buffer.lineLength(model.cursor.Row) {
			model.cursor.Col = max(0, model.buffer.lineLength(model.cursor.Row)-1)
		}
		model.isVisualLine = false
		model.statusMessage = ""
	case ModeCommand:
		// Reset command buffer when entering command mode
		model.commandBuffer = ""
	}

	return func() tea.Msg {
		return EditorModeMsg{newMode}
	}
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

		m.registry.Add(" ", moveCursorRightOrNextLine, mode, "Move cursor right")
		m.registry.Add("0", moveToStartOfLine, mode, "Move to start of line")
		m.registry.Add("^", moveToFirstNonWhitespace, mode, "Move to first non-whitespace character")
		m.registry.Add("$", moveToEndOfLine, mode, "Move to end of line")
		m.registry.Add("gg", moveToStartOfDocument, mode, "Move to document start")
		m.registry.Add("G", moveToEndOfDocument, mode, "Move to document end")

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
	m.registry.Add("alt+backspace", handleInsertDeletePreviousWord, ModeInsert, "Delete previous word")
	m.registry.Add("ctrl+t", handleInsertTransposeCharacters, ModeInsert, "Transpose characters")
	m.registry.Add("ctrl+_", undo, ModeInsert, "Undo")
	m.registry.Add("tab", handleInsertTab, ModeInsert, "Tab")
	m.registry.Add("enter", handleInsertEnterKey, ModeInsert, "Enter")
	m.registry.Add("ctrl+a", moveToStartOfLine, ModeInsert, "Move to start of line")
	m.registry.Add("ctrl+e", moveToEndOfLine, ModeInsert, "Move to end of line")
	m.registry.Add("ctrl+b", moveCursorLeft, ModeInsert, "Move cursor left")
	m.registry.Add("ctrl+f", moveCursorRight, ModeInsert, "Move cursor right")
	m.registry.Add("ctrl+p", moveCursorUp, ModeInsert, "Move cursor up")
	m.registry.Add("ctrl+n", moveCursorDown, ModeInsert, "Move cursor down")
	m.registry.Add("alt+b", moveToPrevWordStart, ModeInsert, "Move to previous word")
	m.registry.Add("alt+f", moveToNextWordStart, ModeInsert, "Move to next word")
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

func toggleRelativeLineNumbers(model *editorModel) tea.Cmd {
	model.relativeNumbers = !model.relativeNumbers
	if model.relativeNumbers {
		return SetStatusMsg("relative line numbers: on")
	} else {
		return SetStatusMsg("relative line numbers: off")
	}
}

func clearBuffer(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)
	model.buffer.clear()
	model.cursor = newCursor(0, 0)
	return SetStatusMsg("buffer cleared")
}

func resetEditor(model *editorModel) tea.Cmd {
	return model.Reset()
}

func moveToFirstNonWhitespace(model *editorModel) tea.Cmd {
	line := model.buffer.Line(model.cursor.Row)
	for i, char := range line {
		if char != ' ' && char != '\t' {
			model.cursor.Col = i
			model.desiredCol = model.cursor.Col
			break
		}
	}
	return nil
}

func deleteToEndOfLine(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	col := model.cursor.Col
	line := model.buffer.Line(row)

	if len(line) > 0 {

		model.buffer.saveUndoState(model.cursor)

		start := Cursor{Row: row, Col: col}
		end := Cursor{Row: row, Col: len(line) - 1}

		model.yankBuffer = model.buffer.deleteRange(start, end)
		writeClipboardText(model.yankBuffer)
	}

	return nil
}

func replaceCharacterAtCursor(model *editorModel, replacement string) (tea.Model, tea.Cmd) {
	model.buffer.saveUndoState(model.cursor)

	line := model.buffer.Line(model.cursor.Row)
	if model.cursor.Col < 0 || model.cursor.Col >= len(line) {
		return model, nil
	}

	model.buffer.setLine(
		model.cursor.Row,
		line[:model.cursor.Col]+replacement+line[model.cursor.Col+1:],
	)

	return model, nil
}

func exitModeCommand(model *editorModel) tea.Cmd {
	return switchMode(model, ModeNormal)
}

func exitModeVisual(model *editorModel) tea.Cmd {
	return switchMode(model, ModeNormal)
}

func exitModeInsert(model *editorModel) tea.Cmd {
	return switchMode(model, ModeNormal)
}

func enterModeInsert(model *editorModel) tea.Cmd {
	return switchMode(model, ModeInsert)
}

func enterModeCommand(model *editorModel) tea.Cmd {
	return switchMode(model, ModeCommand)
}

func beginVisualSelection(model *editorModel) tea.Cmd {
	model.visualStart = model.cursor.Clone()
	model.isVisualLine = false
	model.statusMessage = "-- VISUAL --"
	return switchMode(model, ModeVisual)
}

func beginVisualLineSelection(model *editorModel) tea.Cmd {
	model.visualStart = newCursor(model.cursor.Row, 0)
	model.isVisualLine = true
	model.statusMessage = "-- VISUAL LINE --"
	return switchMode(model, ModeVisual)
}

func appendAfterCursor(model *editorModel) tea.Cmd {
	if model.cursor.Col < model.buffer.lineLength(model.cursor.Row) {
		model.cursor.Col++
	}
	return switchMode(model, ModeInsert)
}

func appendAtEndOfLine(model *editorModel) tea.Cmd {
	model.cursor.Col = model.buffer.lineLength(model.cursor.Row)
	return switchMode(model, ModeInsert)
}

func insertAtStartOfLine(model *editorModel) tea.Cmd {
	model.cursor.Col = 0
	return switchMode(model, ModeInsert)
}

func openLineBelow(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	model.buffer.insertLine(model.cursor.Row+1, "")
	model.cursor.Row++
	model.cursor.Col = 0
	model.ensureCursorVisible()
	return switchMode(model, ModeInsert)
}

func openLineAbove(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	model.buffer.insertLine(model.cursor.Row, "")
	model.cursor.Col = 0
	model.ensureCursorVisible()
	return switchMode(model, ModeInsert)
}

func insertCharacter(model *editorModel, char string) (tea.Model, tea.Cmd) {
	model.buffer.saveUndoState(model.cursor)

	if model.cursor.Col > model.buffer.lineLength(model.cursor.Row) {
		model.cursor.Col = model.buffer.lineLength(model.cursor.Row)
	}

	line := model.buffer.Line(model.cursor.Row)
	newLine := line[:model.cursor.Col] + char + line[model.cursor.Col:]
	model.buffer.setLine(model.cursor.Row, newLine)
	model.cursor.Col++

	return model, nil
}

func handleInsertBackspace(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	if model.cursor.Col > 0 {

		model.buffer.deleteAt(model.cursor.Row, model.cursor.Col-1, model.cursor.Row, model.cursor.Col-1)
		model.cursor.Col--
	} else if model.cursor.Row > 0 {

		prevLineLen := model.buffer.lineLength(model.cursor.Row - 1)

		model.buffer.deleteAt(model.cursor.Row-1, prevLineLen, model.cursor.Row, 0)

		model.cursor.Row--
		model.cursor.Col = prevLineLen
	}
	return nil
}

func handleInsertDeleteForward(model *editorModel) tea.Cmd {
	line := model.buffer.Line(model.cursor.Row)
	if model.cursor.Col < 0 || model.cursor.Col >= len(line) {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	model.buffer.deleteAt(model.cursor.Row, model.cursor.Col, model.cursor.Row, model.cursor.Col)
	return nil
}

func handleInsertKillToEndOfLine(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 {
		if row < model.buffer.lineCount()-1 {
			model.buffer.saveUndoState(model.cursor)
			model.yankBuffer = "\n"
			writeClipboardText(model.yankBuffer)
			model.buffer.joinLines(row, row+1)
		}
		return nil
	}

	if model.cursor.Col >= len(line) {
		if row < model.buffer.lineCount()-1 {
			model.buffer.saveUndoState(model.cursor)
			model.yankBuffer = "\n"
			writeClipboardText(model.yankBuffer)
			model.buffer.joinLines(row, row+1)
		}
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	deleted := line[model.cursor.Col:]
	model.yankBuffer = deleted
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[:model.cursor.Col])
	return nil
}

func handleInsertKillToStartOfLine(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 || model.cursor.Col <= 0 {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	deleted := line[:model.cursor.Col]
	model.yankBuffer = deleted
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[model.cursor.Col:])
	model.cursor.Col = 0
	return nil
}

func handleInsertDeletePreviousWord(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 || model.cursor.Col <= 0 {
		return nil
	}

	start := previousWordStart(line, model.cursor.Col)
	if start == model.cursor.Col {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	deleted := line[start:model.cursor.Col]
	model.yankBuffer = deleted
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[:start]+line[model.cursor.Col:])
	model.cursor.Col = start
	return nil
}

func handleInsertDeleteNextWord(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 || model.cursor.Col >= len(line) {
		return nil
	}

	end := getDeleteWordEnd(line, model.cursor.Col)
	if end < model.cursor.Col {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	deleted := line[model.cursor.Col : end+1]
	model.yankBuffer = deleted
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[:model.cursor.Col]+line[end+1:])
	return nil
}

func handleInsertYank(model *editorModel) tea.Cmd {
	text := readClipboardText()
	if text == "" {
		text = model.yankBuffer
	}
	if text == "" {
		return nil
	}

	return insertTextAtCursor(model, text)
}

func handleInsertTransposeCharacters(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) < 2 {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)

	pos := model.cursor.Col
	if pos <= 0 {
		pos = 1
	}
	if pos >= len(line) {
		pos = len(line) - 1
	}

	left := pos - 1
	right := pos
	if right >= len(line) {
		right = len(line) - 1
		left = right - 1
	}

	if left < 0 || right >= len(line) {
		return nil
	}

	runes := []rune(line)
	runes[left], runes[right] = runes[right], runes[left]
	model.buffer.setLine(row, string(runes))
	model.cursor.Col = min(pos+1, len(line))
	return nil
}

func insertTextAtCursor(model *editorModel, text string) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)
	model.buffer.insertAt(model.cursor.Row, model.cursor.Col, text)

	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		model.cursor.Col += len(text)
		model.ensureCursorVisible()
		return nil
	}

	model.cursor.Row += len(lines) - 1
	model.cursor.Col = len(lines[len(lines)-1])
	model.ensureCursorVisible()
	return nil
}

func handleInsertTab(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	line := model.buffer.Line(model.cursor.Row)
	newLine := line[:model.cursor.Col] + "\t" + line[model.cursor.Col:]
	model.buffer.setLine(model.cursor.Row, newLine)
	model.cursor.Col += 1
	return nil
}

func handleInsertEnterKey(m *editorModel) tea.Cmd {
	m.buffer.saveUndoState(m.cursor)

	currentLine := m.buffer.Line(m.cursor.Row)
	newLine := ""

	if m.cursor.Col < len(currentLine) {
		newLine = currentLine[m.cursor.Col:]
		m.buffer.setLine(m.cursor.Row, currentLine[:m.cursor.Col])
	}

	m.buffer.insertLine(m.cursor.Row+1, newLine)
	m.cursor.Row++
	m.cursor.Col = 0
	m.ensureCursorVisible()
	return nil
}

func moveCursorLeft(model *editorModel) tea.Cmd {
	withCountPrefix(model, func() {
		if model.cursor.Col > 0 {
			model.cursor.Col--
		}
	})
	model.desiredCol = model.cursor.Col
	return nil
}

func moveCursorDown(model *editorModel) tea.Cmd {
	withCountPrefix(model, func() {
		if model.cursor.Row < model.buffer.lineCount()-1 {
			model.cursor.Row++
			lineLen := model.buffer.lineLength(model.cursor.Row)
			if lineLen == 0 {
				model.cursor.Col = 0
			} else {
				model.cursor.Col = min(model.desiredCol, lineLen-1)
			}
		}
	})
	model.ensureCursorVisible()
	return nil
}

func moveCursorUp(model *editorModel) tea.Cmd {
	withCountPrefix(model, func() {
		if model.cursor.Row > 0 {
			model.cursor.Row--
			lineLen := model.buffer.lineLength(model.cursor.Row)
			if lineLen == 0 {
				model.cursor.Col = 0
			} else {
				model.cursor.Col = min(model.desiredCol, lineLen-1)
			}
		}
	})
	model.ensureCursorVisible()
	return nil
}

func moveCursorRight(model *editorModel) tea.Cmd {
	lineLen := model.buffer.lineLength(model.cursor.Row)

	withCountPrefix(model, func() {
		if lineLen > 0 && model.cursor.Col < lineLen-1 {
			model.cursor.Col++
		}
	})
	model.desiredCol = model.cursor.Col
	return nil
}

func moveCursorRightOrNextLine(model *editorModel) tea.Cmd {
	lineLen := model.buffer.lineLength(model.cursor.Row)
	if lineLen > 0 && model.cursor.Col < lineLen-1 {
		model.cursor.Col++
	} else if model.cursor.Row < model.buffer.lineCount()-1 {
		model.cursor.Row++
		model.cursor.Col = 0
	}
	model.desiredCol = model.cursor.Col
	model.ensureCursorVisible()
	return nil
}

func moveToStartOfLine(model *editorModel) tea.Cmd {
	model.cursor.Col = 0
	model.desiredCol = model.cursor.Col
	return nil
}

func moveToEndOfLine(model *editorModel) tea.Cmd {
	lineLen := model.buffer.lineLength(model.cursor.Row)
	if lineLen > 0 {
		model.cursor.Col = lineLen - 1
	} else {
		model.cursor.Col = 0
	}
	model.desiredCol = model.cursor.Col
	return nil
}

func deleteWord(model *editorModel) tea.Cmd {
	withCountPrefix(model, func() {
		deleteWordOnce(model)
	})
	return nil
}

func changeWord(model *editorModel) tea.Cmd {
	withCountPrefix(model, func() {
		deleteWordOnce(model)
	})
	return switchMode(model, ModeInsert)
}

func deleteWordOnce(model *editorModel) {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 {
		return
	}

	start := model.cursor.Col
	if start >= len(line) {
		start = len(line) - 1
	}
	end := getDeleteWordEnd(line, start)
	if end < start {
		return
	}

	model.buffer.saveUndoState(model.cursor)
	deleted := model.buffer.deleteRange(Cursor{Row: row, Col: start}, Cursor{Row: row, Col: end})
	model.yankBuffer = deleted
	writeClipboardText(model.yankBuffer)

	lineLen := model.buffer.lineLength(row)
	if lineLen == 0 {
		model.cursor.Col = 0
		return
	}

	if start >= lineLen {
		model.cursor.Col = lineLen - 1
	} else {
		model.cursor.Col = start
	}
	model.desiredCol = model.cursor.Col
}

func moveToStartOfDocument(model *editorModel) tea.Cmd {
	model.cursor.Row = 0
	lineLen := model.buffer.lineLength(model.cursor.Row)
	if lineLen == 0 {
		model.cursor.Col = 0
	} else {
		model.cursor.Col = min(model.desiredCol, lineLen-1)
	}
	model.keySequence = []string{}
	model.ensureCursorVisible()
	return nil
}

func moveToEndOfDocument(model *editorModel) tea.Cmd {
	model.cursor.Row = model.buffer.lineCount() - 1
	lineLen := model.buffer.lineLength(model.cursor.Row)
	if lineLen == 0 {
		model.cursor.Col = 0
	} else {
		model.cursor.Col = min(model.desiredCol, lineLen-1)
	}
	model.keySequence = []string{}
	model.ensureCursorVisible()
	return nil
}

func handleArrowKeys(key string) func(*editorModel) tea.Cmd {
	return func(m *editorModel) tea.Cmd {
		switch key {
		case "up":
			return moveCursorUp(m)
		case "down":
			return moveCursorDown(m)
		case "left":
			return moveCursorLeft(m)
		case "right":
			return moveCursorRight(m)
		}
		return nil
	}
}

func executeCommand(model *editorModel) tea.Cmd {
	command := model.commandBuffer
	model.commandBuffer = ""
	return func() tea.Msg {
		return CommandMsg{command}
	}
}

func addCommandCharacter(model *editorModel, char string) (tea.Model, tea.Cmd) {
	model.commandBuffer += char
	return model, nil
}

func commandBackspace(model *editorModel) tea.Cmd {
	if len(model.commandBuffer) > 0 {
		model.commandBuffer = model.commandBuffer[:len(model.commandBuffer)-1]
	}
	return nil
}

func moveToNextWordStart(model *editorModel) tea.Cmd {
	currRow := model.cursor.Row
	if currRow >= model.buffer.lineCount() {
		return nil
	}

	line := model.buffer.Line(currRow)
	startPos := model.cursor.Col + 1

	if startPos >= len(line) {
		if currRow < model.buffer.lineCount()-1 {
			model.cursor.Row++
			model.cursor.Col = 0
			model.ensureCursorVisible()
		}
		model.desiredCol = model.cursor.Col
		return nil
	}

	for i := startPos; i < len(line); i++ {
		if (i == 0 || isWordSeparator(line[i-1])) && !isWordSeparator(line[i]) {
			model.cursor.Col = i
			model.desiredCol = model.cursor.Col
			return nil
		}
	}

	model.cursor.Col = max(0, len(line)-1)
	model.desiredCol = model.cursor.Col
	return nil
}

func moveToPrevWordStart(model *editorModel) tea.Cmd {
	currRow := model.cursor.Row
	if currRow >= model.buffer.lineCount() {
		return nil
	}

	line := model.buffer.Line(currRow)
	if model.cursor.Col <= 0 {
		if currRow > 0 {
			model.cursor.Row--
			prevLineLen := model.buffer.lineLength(model.cursor.Row)
			model.cursor.Col = max(0, prevLineLen-1)
			model.desiredCol = model.cursor.Col
			model.ensureCursorVisible()
		}
		return nil
	}

	for i := model.cursor.Col - 1; i >= 0; i-- {
		if (i == 0 || isWordSeparator(line[i-1])) && !isWordSeparator(line[i]) {
			model.cursor.Col = i
			model.desiredCol = model.cursor.Col
			return nil
		}
	}

	model.cursor.Col = 0
	model.desiredCol = model.cursor.Col
	return nil
}

func (m *editorModel) handleOperatorKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	now := time.Now()
	if now.Sub(m.lastKeyTime) > 750*time.Millisecond && len(m.operatorSequence) > 0 {
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.countPrefix = 1
		return m, nil
	}
	m.lastKeyTime = now

	if msg.Type == tea.KeyEsc {
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.countPrefix = 1
		return m, nil
	}

	keyStr := msg.String()
	if keyStr == "" {
		return m, nil
	}

	m.operatorSequence = append(m.operatorSequence, keyStr)
	seq := strings.Join(m.operatorSequence, "")

	if cmd, ok := m.executePendingOperator(seq); ok {
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.countPrefix = 1
		return m, cmd
	}

	if isPendingOperatorPrefix(m.pendingOperator, seq) {
		return m, nil
	}

	m.pendingOperator = ""
	m.operatorSequence = nil
	m.countPrefix = 1
	return m, nil
}

func (m *editorModel) executePendingOperator(seq string) (tea.Cmd, bool) {
	switch m.pendingOperator {
	case "d":
		return executeDeleteOperator(m, seq)
	case "y":
		return executeYankOperator(m, seq)
	case "c":
		return executeChangeOperator(m, seq)
	default:
		return nil, false
	}
}

func isPendingOperatorPrefix(op, seq string) bool {
	known := []string{
		"d", "dd", "dl", "dh", "dj", "dk", "dw", "db", "d0", "d^", "d$", "dgg", "dG", "dx", "diw", "daw", "di(", "da(", "di\"", "da\"", "di'", "da'",
		"y", "yy", "yl", "yh", "yj", "yk", "yw", "yb", "y0", "y^", "y$", "ygg", "yG", "yx", "yiw", "yaw", "yi(", "ya(", "yi\"", "ya\"", "yi'", "ya'",
		"c", "cc", "cl", "ch", "cj", "ck", "cw", "cb", "c0", "c^", "c$", "cgg", "cG", "cx", "ciw", "caw", "ci(", "ca(", "ci\"", "ca\"", "ci'", "ca'",
	}

	full := op + seq
	for _, candidate := range known {
		if candidate == full {
			return true
		}
		if strings.HasPrefix(candidate, full) {
			return true
		}
	}
	return false
}

func executeDeleteOperator(model *editorModel, seq string) (tea.Cmd, bool) {
	switch seq {
	case "d":
		return deleteLineWithCount(model), true
	case "w":
		return deleteWord(model), true
	case "b":
		return deletePreviousWord(model), true
	case "x", "l", "h", "j", "k", "0", "^", "$", "gg", "G":
		return deleteMotion(model, seq), true
	case "iw":
		return deleteInnerWord(model), true
	case "aw":
		return deleteAroundWord(model), true
	case "i(":
		return deleteInnerParen(model), true
	case "a(":
		return deleteAroundParen(model), true
	case "i\"":
		return deleteInnerQuote(model, '"'), true
	case "a\"":
		return deleteAroundQuote(model, '"'), true
	case "i'":
		return deleteInnerQuote(model, '\''), true
	case "a'":
		return deleteAroundQuote(model, '\''), true
	default:
		return nil, false
	}
}

func executeYankOperator(model *editorModel, seq string) (tea.Cmd, bool) {
	switch seq {
	case "y":
		return yankLinesWithCount(model), true
	case "w":
		return yankInnerWord(model), true
	case "b":
		return yankPreviousWord(model), true
	case "x", "l", "h", "j", "k", "0", "^", "$", "gg", "G":
		return yankMotion(model, seq), true
	case "iw":
		return yankInnerWord(model), true
	case "aw":
		return yankAroundWord(model), true
	case "i(":
		return yankInnerParen(model), true
	case "a(":
		return yankAroundParen(model), true
	case "i\"":
		return yankInnerQuote(model, '"'), true
	case "a\"":
		return yankAroundQuote(model, '"'), true
	case "i'":
		return yankInnerQuote(model, '\''), true
	case "a'":
		return yankAroundQuote(model, '\''), true
	default:
		return nil, false
	}
}

func executeChangeOperator(model *editorModel, seq string) (tea.Cmd, bool) {
	switch seq {
	case "c":
		return changeLinesWithCount(model), true
	case "w":
		return changeWord(model), true
	case "b":
		return changePreviousWord(model), true
	case "x", "l", "h", "j", "k", "0", "^", "$", "gg", "G":
		return changeMotion(model, seq), true
	case "iw":
		return changeInnerWord(model), true
	case "aw":
		return changeAroundWord(model), true
	case "i(":
		return changeInnerParen(model), true
	case "a(":
		return changeAroundParen(model), true
	case "i\"":
		return changeInnerQuote(model, '"'), true
	case "a\"":
		return changeAroundQuote(model, '"'), true
	case "i'":
		return changeInnerQuote(model, '\''), true
	case "a'":
		return changeAroundQuote(model, '\''), true
	default:
		return nil, false
	}
}

func deleteCharacterMotion(model *editorModel, rowDelta, colDelta int) tea.Cmd {
	start, end, ok := motionRange(model, rowDelta, colDelta)
	if !ok {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	model.yankBuffer = model.buffer.deleteRange(start, end)
	writeClipboardText(model.yankBuffer)
	model.cursor = start
	model.ensureCursorVisible()
	return nil
}

func yankCharacterMotion(model *editorModel, rowDelta, colDelta int) tea.Cmd {
	start, end, ok := motionRange(model, rowDelta, colDelta)
	if !ok {
		return nil
	}
	model.yankBuffer = model.buffer.getRange(start, end)
	writeClipboardText(model.yankBuffer)
	return nil
}

func changeCharacterMotion(model *editorModel, rowDelta, colDelta int) tea.Cmd {
	cmd := deleteCharacterMotion(model, rowDelta, colDelta)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func deleteMotion(model *editorModel, seq string) tea.Cmd {
	start, end, ok := motionRangeForSequence(model, seq)
	if !ok {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	model.yankBuffer = model.buffer.deleteRange(start, end)
	writeClipboardText(model.yankBuffer)
	model.cursor = start
	model.ensureCursorVisible()
	return nil
}

func yankMotion(model *editorModel, seq string) tea.Cmd {
	start, end, ok := motionRangeForSequence(model, seq)
	if !ok {
		return nil
	}
	model.yankBuffer = model.buffer.getRange(start, end)
	writeClipboardText(model.yankBuffer)
	return nil
}

func changeMotion(model *editorModel, seq string) tea.Cmd {
	start, end, ok := motionRangeForSequence(model, seq)
	if !ok {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	model.yankBuffer = model.buffer.deleteRange(start, end)
	writeClipboardText(model.yankBuffer)
	model.cursor = start
	model.ensureCursorVisible()
	return switchMode(model, ModeInsert)
}

func motionRangeForSequence(model *editorModel, seq string) (Cursor, Cursor, bool) {
	temp := *model
	temp.cursor = model.cursor.Clone()
	temp.desiredCol = model.desiredCol
	temp.countPrefix = model.countPrefix

	switch seq {
	case "h":
		moveCursorLeft(&temp)
	case "l", "x":
		moveCursorRight(&temp)
	case "j":
		moveCursorDown(&temp)
	case "k":
		moveCursorUp(&temp)
	case "0":
		moveToStartOfLine(&temp)
	case "^":
		moveToFirstNonWhitespace(&temp)
	case "$":
		moveToEndOfLine(&temp)
	case "gg":
		moveToStartOfDocument(&temp)
	case "G":
		moveToEndOfDocument(&temp)
	default:
		return Cursor{}, Cursor{}, false
	}

	start, end := model.cursor, temp.cursor
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	if start == end {
		return Cursor{}, Cursor{}, false
	}
	return start, end, true
}

func motionRange(model *editorModel, rowDelta, colDelta int) (Cursor, Cursor, bool) {
	temp := *model
	temp.cursor = model.cursor.Clone()
	temp.desiredCol = model.desiredCol
	temp.countPrefix = model.countPrefix

	if rowDelta < 0 || colDelta < 0 {
		return Cursor{}, Cursor{}, false
	}

	for range model.countPrefix {
		temp.cursor.Row += rowDelta
		temp.cursor.Col += colDelta
	}

	temp.adjustCursorPosition()
	if temp.cursor.Row == model.cursor.Row && temp.cursor.Col == model.cursor.Col {
		return Cursor{}, Cursor{}, false
	}

	start, end := model.cursor, temp.cursor
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	return start, end, true
}

func deleteLineWithCount(model *editorModel) tea.Cmd {
	count := max(1, model.countPrefix)
	startRow := model.cursor.Row
	endRow := min(model.buffer.lineCount()-1, startRow+count-1)

	model.buffer.saveUndoState(model.cursor)
	if startRow == endRow {
		model.yankBuffer = "\n" + model.buffer.Line(startRow)
		model.buffer.deleteLine(startRow)
	} else {
		lines := make([]string, 0, endRow-startRow+1)
		for row := startRow; row <= endRow; row++ {
			lines = append(lines, model.buffer.Line(row))
		}
		model.yankBuffer = "\n" + strings.Join(lines, "\n")
		for i := 0; i <= endRow-startRow; i++ {
			model.buffer.deleteLine(startRow)
		}
	}
	if model.buffer.lineCount() == 0 {
		model.buffer.insertLine(0, "")
	}
	if model.cursor.Row >= model.buffer.lineCount() {
		model.cursor.Row = model.buffer.lineCount() - 1
	}
	model.cursor.Col = min(model.cursor.Col, max(0, model.buffer.lineLength(model.cursor.Row)-1))
	writeClipboardText(model.yankBuffer)
	model.ensureCursorVisible()
	return nil
}

func yankLinesWithCount(model *editorModel) tea.Cmd {
	count := max(1, model.countPrefix)
	startRow := model.cursor.Row
	endRow := min(model.buffer.lineCount()-1, startRow+count-1)
	lines := make([]string, 0, endRow-startRow+1)
	for row := startRow; row <= endRow; row++ {
		lines = append(lines, model.buffer.Line(row))
	}
	model.yankBuffer = "\n" + strings.Join(lines, "\n")
	writeClipboardText(model.yankBuffer)
	return nil
}

func changeLinesWithCount(model *editorModel) tea.Cmd {
	cmd := deleteLineWithCount(model)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func deletePreviousWord(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 || model.cursor.Col <= 0 {
		return nil
	}
	start := previousWordStart(line, model.cursor.Col)
	if start == model.cursor.Col {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	model.yankBuffer = line[start:model.cursor.Col]
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[:start]+line[model.cursor.Col:])
	model.cursor.Col = start
	return nil
}

func yankPreviousWord(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 || model.cursor.Col <= 0 {
		return nil
	}
	start := previousWordStart(line, model.cursor.Col)
	model.yankBuffer = line[start:model.cursor.Col]
	writeClipboardText(model.yankBuffer)
	return nil
}

func changePreviousWord(model *editorModel) tea.Cmd {
	cmd := deletePreviousWord(model)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func deleteToLineStart(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if model.cursor.Col <= 0 || len(line) == 0 {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	model.yankBuffer = line[:model.cursor.Col]
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[model.cursor.Col:])
	model.cursor.Col = 0
	return nil
}

func deleteToFirstNonWhitespace(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 {
		return nil
	}
	idx := 0
	for idx < len(line) && (line[idx] == ' ' || line[idx] == '\t') {
		idx++
	}
	if idx >= model.cursor.Col {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	model.yankBuffer = line[:model.cursor.Col]
	writeClipboardText(model.yankBuffer)
	model.buffer.setLine(row, line[model.cursor.Col:])
	model.cursor.Col = 0
	return nil
}

func changeToLineStart(model *editorModel) tea.Cmd {
	cmd := deleteToLineStart(model)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func changeToFirstNonWhitespace(model *editorModel) tea.Cmd {
	cmd := deleteToFirstNonWhitespace(model)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func deleteToDocumentStart(model *editorModel) tea.Cmd {
	if model.cursor.Row == 0 && model.cursor.Col == 0 {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	text := model.buffer.text()
	idx := 0
	for i := 0; i < model.cursor.Row; i++ {
		idx += len(model.buffer.Line(i)) + 1
	}
	idx += model.cursor.Col
	model.yankBuffer = text[:idx]
	writeClipboardText(model.yankBuffer)
	model.buffer.lines = []string{text[idx:]}
	model.cursor = newCursor(0, 0)
	return nil
}

func deleteToDocumentEnd(model *editorModel) tea.Cmd {
	if model.cursor.Row >= model.buffer.lineCount() {
		return nil
	}
	model.buffer.saveUndoState(model.cursor)
	text := model.buffer.text()
	idx := 0
	for i := 0; i < model.cursor.Row; i++ {
		idx += len(model.buffer.Line(i)) + 1
	}
	idx += model.cursor.Col
	model.yankBuffer = text[idx:]
	writeClipboardText(model.yankBuffer)
	model.buffer.lines = []string{text[:idx]}
	if len(model.buffer.lines) == 0 {
		model.buffer.lines = []string{""}
	}
	return nil
}

func changeToDocumentStart(model *editorModel) tea.Cmd {
	cmd := deleteToDocumentStart(model)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func changeToDocumentEnd(model *editorModel) tea.Cmd {
	cmd := deleteToDocumentEnd(model)
	if cmd != nil {
		return cmd
	}
	return switchMode(model, ModeInsert)
}

func yankToLineStart(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if model.cursor.Col <= 0 || len(line) == 0 {
		return nil
	}
	model.yankBuffer = line[:model.cursor.Col]
	writeClipboardText(model.yankBuffer)
	return nil
}

func yankToFirstNonWhitespace(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 {
		return nil
	}
	idx := 0
	for idx < len(line) && (line[idx] == ' ' || line[idx] == '\t') {
		idx++
	}
	if idx >= model.cursor.Col {
		return nil
	}
	model.yankBuffer = line[:model.cursor.Col]
	writeClipboardText(model.yankBuffer)
	return nil
}

func yankToEndOfLine(model *editorModel) tea.Cmd {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 || model.cursor.Col >= len(line) {
		return nil
	}
	model.yankBuffer = line[model.cursor.Col:]
	writeClipboardText(model.yankBuffer)
	return nil
}

func yankToDocumentStart(model *editorModel) tea.Cmd {
	if model.cursor.Row == 0 && model.cursor.Col == 0 {
		return nil
	}
	text := model.buffer.text()
	idx := 0
	for i := 0; i < model.cursor.Row; i++ {
		idx += len(model.buffer.Line(i)) + 1
	}
	idx += model.cursor.Col
	model.yankBuffer = text[:idx]
	writeClipboardText(model.yankBuffer)
	return nil
}

func yankToDocumentEnd(model *editorModel) tea.Cmd {
	text := model.buffer.text()
	idx := 0
	for i := 0; i < model.cursor.Row; i++ {
		idx += len(model.buffer.Line(i)) + 1
	}
	idx += model.cursor.Col
	model.yankBuffer = text[idx:]
	writeClipboardText(model.yankBuffer)
	return nil
}

func previousWordStart(line string, col int) int {
	if len(line) == 0 {
		return 0
	}

	if col > len(line) {
		col = len(line)
	}
	if col <= 0 {
		return 0
	}

	i := col - 1
	for i > 0 && isWordSeparator(line[i]) {
		i--
	}
	for i > 0 && !isWordSeparator(line[i-1]) {
		i--
	}
	if isWordSeparator(line[i]) {
		for i < col && isWordSeparator(line[i]) {
			i++
		}
	}
	return i
}

func undo(model *editorModel) tea.Cmd {
	return model.buffer.undo(model.cursor)
}

func redo(model *editorModel) tea.Cmd {
	return model.buffer.redo(model.cursor)
}

func deleteCharAtCursor(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	lineLen := model.buffer.lineLength(model.cursor.Row)
	if lineLen > 0 && model.cursor.Col < lineLen {

		model.buffer.deleteAt(model.cursor.Row, model.cursor.Col, model.cursor.Row, model.cursor.Col)

		newLineLen := model.buffer.lineLength(model.cursor.Row)
		if model.cursor.Col >= newLineLen && newLineLen > 0 {
			model.cursor.Col = newLineLen - 1
		}
	}
	return nil
}

func setupYankHighlight(model *editorModel, start, end Cursor, text string, isLinewise bool) {
	model.yankBuffer = text
	writeClipboardText(model.yankBuffer)
	model.statusMessage = fmt.Sprintf("yanked %d characters", len(text))
	model.yankHighlight.Start = start
	model.yankHighlight.End = end
	model.yankHighlight.StartTime = time.Now()
	model.yankHighlight.IsLinewise = isLinewise
	model.yankHighlight.Active = true
}

func yankLine(model *editorModel) tea.Cmd {
	line := model.buffer.Line(model.cursor.Row)

	setupYankHighlight(
		model,
		Cursor{model.cursor.Row, 0},
		Cursor{model.cursor.Row, max(0, len(line)-1)},
		"\n"+line,
		true,
	)

	model.keySequence = []string{}
	return nil
}

func deleteLine(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	row := model.cursor.Row
	lineContent := model.buffer.Line(row)
	model.yankBuffer = "\n" + lineContent
	writeClipboardText(model.yankBuffer)

	model.buffer.deleteLine(row)

	if model.buffer.lineCount() == 0 {
		model.buffer.insertLine(0, "")
	}

	if model.cursor.Row >= model.buffer.lineCount() {
		model.cursor.Row = model.buffer.lineCount() - 1
	}
	if model.cursor.Col >= model.buffer.lineLength(model.cursor.Row) {
		model.cursor.Col = max(0, model.buffer.lineLength(model.cursor.Row)-1)
	}

	model.ensureCursorVisible()
	return nil
}

func pasteAfter(model *editorModel) tea.Cmd {
	model.yankBuffer = readClipboardText()
	if model.yankBuffer == "" {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)

	// Line-wise paste
	if strings.HasPrefix(model.yankBuffer, "\n") {
		return pasteLineAfter(model)
	}

	// Character-wise paste
	currLine := model.buffer.Line(model.cursor.Row)
	insertPos := model.cursor.Col

	// Check if the yanked text contains newlines (multi-line character-wise yank)
	if strings.Contains(model.yankBuffer, "\n") {
		// Split the yanked text by newlines
		lines := strings.Split(model.yankBuffer, "\n")

		// Handle the first line - insert at cursor position in current line
		firstLine := lines[0]
		remainderOfLine := ""
		if insertPos < len(currLine) {
			remainderOfLine = currLine[insertPos+1:]
		}

		// Set the first line with the content before cursor + first part of yanked text
		if insertPos >= len(currLine) {
			model.buffer.setLine(model.cursor.Row, currLine+firstLine)
		} else {
			model.buffer.setLine(model.cursor.Row,
				currLine[:insertPos+1]+firstLine)
		}

		// Insert middle lines as new lines
		row := model.cursor.Row
		for i := 1; i < len(lines)-1; i++ {
			model.buffer.insertLine(row+i, lines[i])
		}

		// Handle the last line separately
		if len(lines) > 1 {
			lastLine := lines[len(lines)-1]
			model.buffer.insertLine(row+len(lines)-1, lastLine+remainderOfLine)
		} else {
			// If only one line, append the remainder to the current line
			currLineContent := model.buffer.Line(model.cursor.Row)
			model.buffer.setLine(model.cursor.Row, currLineContent+remainderOfLine)
		}

		// Position cursor at the end of the last inserted line
		model.cursor.Row = row + len(lines) - 1
		if len(lines) > 1 {
			// For multi-line pastes, position at the end of the last line's content
			model.cursor.Col = len(lines[len(lines)-1])
		} else {
			// For single line pastes, position at the end of what was pasted
			model.cursor.Col = insertPos + len(firstLine) + 1
		}

		if model.mode != ModeInsert && model.cursor.Col > 0 {
			model.cursor.Col--
		}
	} else {
		// Single-line paste - original behavior
		if insertPos >= len(currLine) {
			model.buffer.setLine(model.cursor.Row, currLine+model.yankBuffer)
		} else {
			model.buffer.setLine(model.cursor.Row,
				currLine[:insertPos+1]+model.yankBuffer+currLine[insertPos+1:])
		}

		model.cursor.Col = insertPos + len(model.yankBuffer) + 1
		if model.mode != ModeInsert && model.cursor.Col > 0 {
			model.cursor.Col--
		}
	}

	model.ensureCursorVisible()
	return nil
}

func pasteBefore(model *editorModel) tea.Cmd {
	model.yankBuffer = readClipboardText()
	if model.yankBuffer == "" {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)

	// Line-wise paste
	if strings.HasPrefix(model.yankBuffer, "\n") {
		return pasteLineBefore(model)
	}

	// Character-wise paste
	currLine := model.buffer.Line(model.cursor.Row)
	insertPos := model.cursor.Col

	// Check if the yanked text contains newlines (multi-line character-wise yank)
	if strings.Contains(model.yankBuffer, "\n") {
		// Split the yanked text by newlines
		lines := strings.Split(model.yankBuffer, "\n")

		// Handle the first line - insert at cursor position in current line
		firstLine := lines[0]
		newFirstLine := currLine[:insertPos] + firstLine
		model.buffer.setLine(model.cursor.Row, newFirstLine)

		// If this is the last line, append the remainder of the original line
		if len(lines) == 1 {
			model.buffer.setLine(model.cursor.Row, newFirstLine+currLine[insertPos:])
		} else {
			// Handle the last line - combine with remainder of current line
			lastLineIndex := len(lines) - 1
			lastLine := lines[lastLineIndex] + currLine[insertPos:]

			// Insert middle and last lines as new lines
			row := model.cursor.Row
			for i := 1; i < lastLineIndex; i++ {
				model.buffer.insertLine(row+i, lines[i])
			}
			model.buffer.insertLine(row+lastLineIndex, lastLine)
		}

		// Position cursor appropriately depending on where the paste ended
		if len(lines) > 1 {
			// For multi-line pastes in pasteBefore, cursor stays at the insertion point
			model.cursor.Col = insertPos + len(firstLine)
		} else {
			// For single line pastes, position at the end of what was pasted
			model.cursor.Col = insertPos + len(firstLine)
		}

		if model.mode != ModeInsert && model.cursor.Col > 0 {
			model.cursor.Col--
		}
	} else {
		// Single-line paste - original behavior
		model.buffer.setLine(model.cursor.Row,
			currLine[:insertPos]+model.yankBuffer+currLine[insertPos:])

		model.cursor.Col = max(insertPos+len(model.yankBuffer)-1, 0)
	}

	model.ensureCursorVisible()
	return nil
}

func pasteLineAfter(model *editorModel) tea.Cmd {
	model.yankBuffer = readClipboardText()
	lines := strings.Split(model.yankBuffer[1:], "\n")
	row := model.cursor.Row

	for i := range lines {
		model.buffer.insertLine(row+1+i, lines[i])
	}

	model.cursor.Row = row + 1
	model.cursor.Col = 0
	model.ensureCursorVisible()
	return nil
}

func pasteLineBefore(model *editorModel) tea.Cmd {
	model.yankBuffer = readClipboardText()
	lines := strings.Split(model.yankBuffer[1:], "\n")
	row := model.cursor.Row

	for i := range lines {
		model.buffer.insertLine(row+i, lines[i])
	}

	model.cursor.Col = 0
	model.ensureCursorVisible()
	return nil
}

func yankVisualSelection(model *editorModel) tea.Cmd {
	start, end := model.GetSelectionBoundary()
	selectedText := model.buffer.getRange(start, end)

	if model.isVisualLine {
		selectedText = "\n" + selectedText
	}

	setupYankHighlight(model, start, end, selectedText, model.isVisualLine)
	return switchMode(model, ModeNormal)
}

func deleteVisualSelection(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)
	start, end := model.GetSelectionBoundary()

	selectedText := model.buffer.getRange(start, end)
	if model.isVisualLine {
		selectedText = "\n" + selectedText
	}
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	model.buffer.deleteRange(start, end)

	model.cursor = start
	model.ensureCursorVisible()

	return switchMode(model, ModeNormal)
}

func replaceVisualSelectionWithYank(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)
	start, end := model.GetSelectionBoundary()
	oldSelection := model.buffer.deleteRange(start, end)
	model.yankBuffer = oldSelection
	writeClipboardText(model.yankBuffer)

	model.cursor = start

	if strings.Contains(model.yankBuffer, "\n") {
		pasteLineBefore(model)
	} else {
		currLine := model.buffer.Line(model.cursor.Row)
		insertPos := model.cursor.Col
		model.buffer.setLine(model.cursor.Row,
			currLine[:insertPos]+model.yankBuffer+currLine[insertPos:])
		model.cursor.Col = max(insertPos+len(model.yankBuffer)-1, 0)
	}

	model.ensureCursorVisible()
	return switchMode(model, ModeNormal)
}

func performWordOperation(model *editorModel, operation string) tea.Cmd {
	start, end := getWordBoundary(model)
	if start == end {
		return nil
	}

	word := model.buffer.Line(model.cursor.Row)[start:end]

	if operation == "delete" || operation == "change" {
		model.buffer.saveUndoState(model.cursor)
	}

	model.yankBuffer = word
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked word: %s", model.yankBuffer)
		model.yankHighlight.Start = Cursor{model.cursor.Row, start}
		model.yankHighlight.End = Cursor{model.cursor.Row, end - 1}
		model.yankHighlight.StartTime = time.Now()
		model.yankHighlight.IsLinewise = false
		model.yankHighlight.Active = true
	case "delete", "change":

		line := model.buffer.Line(model.cursor.Row)
		newLine := line[:start] + line[end:]
		model.buffer.setLine(model.cursor.Row, newLine)
		model.cursor.Col = start

		if operation == "change" {
			return switchMode(model, ModeInsert)
		}
	}

	model.keySequence = []string{}
	return nil
}

func deleteInnerWord(model *editorModel) tea.Cmd {
	return performWordOperation(model, "delete")
}

func yankInnerWord(model *editorModel) tea.Cmd {
	return performWordOperation(model, "yank")
}

func changeInnerWord(model *editorModel) tea.Cmd {
	return performWordOperation(model, "change")
}

func deleteInnerParen(model *editorModel) tea.Cmd {
	return performInnerParenOperation(model, "delete")
}

func yankInnerParen(model *editorModel) tea.Cmd {
	return performInnerParenOperation(model, "yank")
}

func changeInnerParen(model *editorModel) tea.Cmd {
	return performInnerParenOperation(model, "change")
}

func performInnerParenOperation(model *editorModel, operation string) tea.Cmd {
	start, end, ok := getInnerParenBoundary(model)
	if !ok {
		return nil
	}

	if operation == "delete" || operation == "change" {
		model.buffer.saveUndoState(model.cursor)
	}

	selectedText := model.buffer.getRange(
		Cursor{Row: model.cursor.Row, Col: start},
		Cursor{Row: model.cursor.Row, Col: end},
	)
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		model.buffer.deleteRange(
			Cursor{Row: model.cursor.Row, Col: start},
			Cursor{Row: model.cursor.Row, Col: end},
		)
		model.cursor.Col = start
		if operation == "change" {
			return switchMode(model, ModeInsert)
		}
	}

	return nil
}

func deleteAroundWord(model *editorModel) tea.Cmd {
	return performAroundWordOperation(model, "delete")
}

func yankAroundWord(model *editorModel) tea.Cmd {
	return performAroundWordOperation(model, "yank")
}

func changeAroundWord(model *editorModel) tea.Cmd {
	return performAroundWordOperation(model, "change")
}

func performAroundWordOperation(model *editorModel, operation string) tea.Cmd {
	start, end, ok := getAroundWordBoundary(model)
	if !ok {
		return nil
	}

	if operation == "delete" || operation == "change" {
		model.buffer.saveUndoState(model.cursor)
	}

	selectedText := model.buffer.getRange(
		Cursor{Row: model.cursor.Row, Col: start},
		Cursor{Row: model.cursor.Row, Col: end},
	)
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		model.buffer.deleteRange(
			Cursor{Row: model.cursor.Row, Col: start},
			Cursor{Row: model.cursor.Row, Col: end},
		)
		model.cursor.Col = start
		if operation == "change" {
			return switchMode(model, ModeInsert)
		}
	}

	return nil
}

func deleteAroundParen(model *editorModel) tea.Cmd {
	return performAroundParenOperation(model, "delete")
}

func yankAroundParen(model *editorModel) tea.Cmd {
	return performAroundParenOperation(model, "yank")
}

func changeAroundParen(model *editorModel) tea.Cmd {
	return performAroundParenOperation(model, "change")
}

func performAroundParenOperation(model *editorModel, operation string) tea.Cmd {
	start, end, ok := getAroundParenBoundary(model)
	if !ok {
		return nil
	}

	if operation == "delete" || operation == "change" {
		model.buffer.saveUndoState(model.cursor)
	}

	selectedText := model.buffer.getRange(
		Cursor{Row: model.cursor.Row, Col: start},
		Cursor{Row: model.cursor.Row, Col: end},
	)
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		model.buffer.deleteRange(
			Cursor{Row: model.cursor.Row, Col: start},
			Cursor{Row: model.cursor.Row, Col: end},
		)
		model.cursor.Col = start
		if operation == "change" {
			return switchMode(model, ModeInsert)
		}
	}

	return nil
}

func deleteInnerQuote(model *editorModel, quote byte) tea.Cmd {
	return performQuoteOperation(model, quote, false, "delete")
}

func yankInnerQuote(model *editorModel, quote byte) tea.Cmd {
	return performQuoteOperation(model, quote, false, "yank")
}

func changeInnerQuote(model *editorModel, quote byte) tea.Cmd {
	return performQuoteOperation(model, quote, false, "change")
}

func deleteAroundQuote(model *editorModel, quote byte) tea.Cmd {
	return performQuoteOperation(model, quote, true, "delete")
}

func yankAroundQuote(model *editorModel, quote byte) tea.Cmd {
	return performQuoteOperation(model, quote, true, "yank")
}

func changeAroundQuote(model *editorModel, quote byte) tea.Cmd {
	return performQuoteOperation(model, quote, true, "change")
}

func performQuoteOperation(model *editorModel, quote byte, around bool, operation string) tea.Cmd {
	start, end, ok, empty := getQuoteBoundary(model, quote, around)
	if !ok {
		return nil
	}

	if operation == "delete" || operation == "change" {
		model.buffer.saveUndoState(model.cursor)
	}

	selectedText := ""
	if !empty {
		selectedText = model.buffer.getRange(
			Cursor{Row: model.cursor.Row, Col: start},
			Cursor{Row: model.cursor.Row, Col: end},
		)
	}
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		if !empty {
			model.buffer.deleteRange(
				Cursor{Row: model.cursor.Row, Col: start},
				Cursor{Row: model.cursor.Row, Col: end},
			)
		}
		model.cursor.Col = start
		if operation == "change" {
			return switchMode(model, ModeInsert)
		}
	}

	return nil
}

func getInnerParenBoundary(model *editorModel) (int, int, bool) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0, false
	}

	type pair struct {
		open  int
		close int
	}

	var stack []int
	var pairs []pair

	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '(':
			stack = append(stack, i)
		case ')':
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			pairs = append(pairs, pair{open: open, close: i})
		}
	}

	cursorCol := model.cursor.Col
	bestOpen, bestClose := -1, -1
	bestSpan := 0

	for _, p := range pairs {
		if cursorCol < p.open || cursorCol > p.close {
			continue
		}

		innerOpen := p.open + 1
		innerClose := p.close - 1
		if innerOpen > innerClose {
			continue
		}

		span := p.close - p.open
		if bestOpen == -1 || span < bestSpan {
			bestOpen = innerOpen
			bestClose = innerClose
			bestSpan = span
		}
	}

	if bestOpen == -1 {
		return 0, 0, false
	}

	return bestOpen, bestClose, true
}

func getAroundParenBoundary(model *editorModel) (int, int, bool) {
	line := model.buffer.Line(model.cursor.Row)
	start, end, ok := getInnerParenBoundary(model)
	if !ok {
		return 0, 0, false
	}
	if start <= 0 || end >= len(line)-1 {
		return 0, 0, false
	}
	return start - 1, end + 1, true
}

func getQuoteBoundary(model *editorModel, quote byte, around bool) (int, int, bool, bool) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0, false, false
	}

	var quotePositions []int
	for i := 0; i < len(line); i++ {
		if line[i] != quote || isEscapedQuote(line, i) {
			continue
		}
		quotePositions = append(quotePositions, i)
	}

	if len(quotePositions) < 2 {
		return 0, 0, false, false
	}

	cursorCol := model.cursor.Col
	bestOpen, bestClose := -1, -1
	bestSpan := 0
	for i := 0; i+1 < len(quotePositions); i += 2 {
		open := quotePositions[i]
		close := quotePositions[i+1]
		if cursorCol < open || cursorCol > close {
			continue
		}

		span := close - open
		if bestOpen == -1 || span < bestSpan {
			bestOpen = open
			bestClose = close
			bestSpan = span
		}
	}

	if bestOpen == -1 {
		return 0, 0, false, false
	}

	if around {
		return bestOpen, bestClose, true, false
	}

	innerStart := bestOpen + 1
	innerEnd := bestClose - 1
	if innerStart > innerEnd {
		return innerStart, innerEnd, true, true
	}
	return innerStart, innerEnd, true, false
}

func isEscapedQuote(line string, idx int) bool {
	if idx <= 0 {
		return false
	}

	backslashes := 0
	for i := idx - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func getWordBoundary(model *editorModel) (int, int) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0
	}

	col := model.cursor.Col
	if col >= len(line) {
		col = len(line) - 1
	}

	start := col

	if isWordSeparator(line[col]) {
		for start > 0 && isWordSeparator(line[start-1]) {
			start--
		}
	} else {
		for start > 0 && !isWordSeparator(line[start-1]) {
			start--
		}
	}

	end := col

	if isWordSeparator(line[col]) {
		for end < len(line)-1 && isWordSeparator(line[end+1]) {
			end++
		}
	} else {
		for end < len(line)-1 && !isWordSeparator(line[end+1]) {
			end++
		}
	}

	return start, end + 1
}

func getAroundWordBoundary(model *editorModel) (int, int, bool) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0, false
	}

	start, end := getWordBoundary(model)
	if start == end {
		return 0, 0, false
	}

	for start > 0 && isWordSeparator(line[start-1]) {
		start--
	}
	for end < len(line) && end < len(line) && isWordSeparator(line[end]) {
		end++
	}
	return start, end - 1, true
}

func getDeleteWordEnd(line string, col int) int {
	if len(line) == 0 {
		return -1
	}

	if col >= len(line) {
		col = len(line) - 1
	}
	end := col

	if isWordSeparator(line[end]) {
		for end < len(line)-1 && isWordSeparator(line[end]) {
			end++
		}
		if end >= len(line)-1 && isWordSeparator(line[end]) {
			return len(line) - 1
		}
	}

	for end < len(line)-1 && !isWordSeparator(line[end+1]) {
		end++
	}
	for end < len(line)-1 && isWordSeparator(line[end+1]) {
		end++
	}

	return end
}

func getInnerBracketBoundary(model *editorModel) (int, int, bool) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0, false
	}

	type bracketPair struct {
		open  int
		close int
	}

	matchingClose := map[byte]byte{
		'(': ')',
		'[': ']',
		'{': '}',
		'<': '>',
	}

	matchingOpen := map[byte]byte{
		')': '(',
		']': '[',
		'}': '{',
		'>': '<',
	}

	type stackEntry struct {
		ch  byte
		idx int
	}

	var stack []stackEntry
	var pairs []bracketPair

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if _, ok := matchingClose[ch]; ok {
			stack = append(stack, stackEntry{ch: ch, idx: i})
			continue
		}

		open, ok := matchingOpen[ch]
		if !ok || len(stack) == 0 {
			continue
		}

		last := stack[len(stack)-1]
		if last.ch == open {
			stack = stack[:len(stack)-1]
			pairs = append(pairs, bracketPair{open: last.idx, close: i})
		}
	}

	cursorCol := model.cursor.Col
	bestOpen, bestClose := -1, -1
	bestSpan := 0

	for _, pair := range pairs {
		if cursorCol < pair.open || cursorCol > pair.close {
			continue
		}

		innerOpen := pair.open + 1
		innerClose := pair.close - 1
		if innerOpen > innerClose {
			continue
		}

		span := pair.close - pair.open
		if bestOpen == -1 || span < bestSpan {
			bestOpen = innerOpen
			bestClose = innerClose
			bestSpan = span
		}
	}

	if bestOpen == -1 {
		return 0, 0, false
	}

	return bestOpen, bestClose, true
}

func isWordSeparator(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '.' || ch == ',' ||
		ch == ';' || ch == ':' || ch == '!' || ch == '?' ||
		ch == '(' || ch == ')' || ch == '[' || ch == ']' ||
		ch == '{' || ch == '}' || ch == '<' || ch == '>' ||
		ch == '/' || ch == '\\' || ch == '+' || ch == '-' ||
		ch == '*' || ch == '&' || ch == '^' || ch == '%' ||
		ch == '$' || ch == '#' || ch == '@' || ch == '=' ||
		ch == '|' || ch == '`' || ch == '~' || ch == '"' ||
		ch == '\''
}
