// Package vimtea provides a Vim-like text editor component for terminal applications
package vimtea

import tea "github.com/charmbracelet/bubbletea"

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
	model.pendingFind = nil

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

	if newMode != ModeInsert {
		model.lastInsertAction = insertActionNone
		model.lastYankRange = nil
	}

	return func() tea.Msg {
		return EditorModeMsg{newMode}
	}
}

func toggleRelativeLineNumbers(model *editorModel) tea.Cmd {
	model.relativeNumbers = !model.relativeNumbers
	if model.relativeNumbers {
		return SetStatusMsg("relative line numbers: on")
	}
	return SetStatusMsg("relative line numbers: off")
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

func exitModeCommand(model *editorModel) tea.Cmd {
	return switchMode(model, ModeNormal)
}

func exitModeVisual(model *editorModel) tea.Cmd {
	return switchMode(model, ModeNormal)
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

func moveCursorLeft(model *editorModel) tea.Cmd {
	withCountPrefix(model, func() {
		if model.cursor.Col > 0 {
			model.cursor.Col--
		}
	})
	model.desiredCol = model.cursor.Col
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
	return nil
}

func moveToStartOfLine(model *editorModel) tea.Cmd {
	model.cursor.Col = 0
	model.desiredCol = model.cursor.Col
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
		if model.mode == ModeInsert {
			model.noteInsertAction(insertActionMotion)
		}
		return nil
	}

	for i := startPos; i < len(line); i++ {
		if (i == 0 || isWordSeparator(line[i-1])) && !isWordSeparator(line[i]) {
			model.cursor.Col = i
			model.desiredCol = model.cursor.Col
			if model.mode == ModeInsert {
				model.noteInsertAction(insertActionMotion)
			}
			return nil
		}
	}

	if model.mode == ModeInsert {
		model.cursor.Col = len(line)
	} else {
		model.cursor.Col = max(0, len(line)-1)
	}
	model.desiredCol = model.cursor.Col
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
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
		if model.mode == ModeInsert {
			model.noteInsertAction(insertActionMotion)
		}
		return nil
	}

	for i := model.cursor.Col - 1; i >= 0; i-- {
		if (i == 0 || isWordSeparator(line[i-1])) && !isWordSeparator(line[i]) {
			model.cursor.Col = i
			model.desiredCol = model.cursor.Col
			if model.mode == ModeInsert {
				model.noteInsertAction(insertActionMotion)
			}
			return nil
		}
	}

	model.cursor.Col = 0
	model.desiredCol = model.cursor.Col
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
	return nil
}
