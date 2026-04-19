package vimtea

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

func exitModeInsert(model *editorModel) tea.Cmd {
	return switchMode(model, ModeNormal)
}

func enterModeInsert(model *editorModel) tea.Cmd {
	return switchMode(model, ModeInsert)
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

	if closer, ok := autoInsertClosingDelimiter(char); ok {
		newLine := line[:model.cursor.Col] + char + closer + line[model.cursor.Col:]
		model.buffer.setLine(model.cursor.Row, newLine)
		model.cursor.Col++
		model.noteInsertAction(insertActionSelfInsert)
		return model, nil
	}

	if shouldSkipClosingDelimiter(line, model.cursor.Col, char) {
		model.cursor.Col++
		model.noteInsertAction(insertActionSelfInsert)
		return model, nil
	}

	newLine := line[:model.cursor.Col] + char + line[model.cursor.Col:]
	model.buffer.setLine(model.cursor.Row, newLine)
	model.cursor.Col++
	model.noteInsertAction(insertActionSelfInsert)

	return model, nil
}

func handleInsertBackspace(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	if model.cursor.Col > 0 {
		line := model.buffer.Line(model.cursor.Row)
		if shouldDeleteAutoInsertedPair(line, model.cursor.Col) {
			model.buffer.deleteAt(model.cursor.Row, model.cursor.Col-1, model.cursor.Row, model.cursor.Col)
			model.cursor.Col--
			model.noteInsertAction(insertActionSelfInsert)
			return nil
		}

		model.buffer.deleteAt(model.cursor.Row, model.cursor.Col-1, model.cursor.Row, model.cursor.Col-1)
		model.cursor.Col--
	} else if model.cursor.Row > 0 {

		prevLineLen := model.buffer.lineLength(model.cursor.Row - 1)

		model.buffer.deleteAt(model.cursor.Row-1, prevLineLen, model.cursor.Row, 0)

		model.cursor.Row--
		model.cursor.Col = prevLineLen
	}
	model.noteInsertAction(insertActionSelfInsert)
	return nil
}

func autoInsertClosingDelimiter(char string) (string, bool) {
	switch char {
	case "(":
		return ")", true
	case "[":
		return "]", true
	case "{":
		return "}", true
	case `"`:
		return `"`, true
	case "'":
		return "'", true
	default:
		return "", false
	}
}

func shouldSkipClosingDelimiter(line string, col int, char string) bool {
	if col < 0 || col >= len(line) {
		return false
	}

	switch char {
	case ")", "]", "}", `"`, "'":
		return string(line[col]) == char
	default:
		return false
	}
}

func shouldDeleteAutoInsertedPair(line string, col int) bool {
	if col <= 0 || col >= len(line) {
		return false
	}

	open := line[col-1]
	close := line[col]

	switch open {
	case '(':
		return close == ')'
	case '[':
		return close == ']'
	case '{':
		return close == '}'
	case '"':
		return close == '"'
	case '\'':
		return close == '\''
	default:
		return false
	}
}

func handleInsertDeleteForward(model *editorModel) tea.Cmd {
	line := model.buffer.Line(model.cursor.Row)
	if model.cursor.Col < 0 || model.cursor.Col >= len(line) {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	model.buffer.deleteAt(model.cursor.Row, model.cursor.Col, model.cursor.Row, model.cursor.Col)
	model.noteInsertAction(insertActionSelfInsert)
	return nil
}

func handleInsertKillToEndOfLine(model *editorModel) tea.Cmd {
	row := model.cursor.Row
	line := model.buffer.Line(row)
	if len(line) == 0 {
		if row < model.buffer.lineCount()-1 {
			model.buffer.saveUndoState(model.cursor)
			model.pushKill("\n", false)
			model.buffer.joinLines(row, row+1)
		}
		return nil
	}

	if model.cursor.Col >= len(line) {
		if row < model.buffer.lineCount()-1 {
			model.buffer.saveUndoState(model.cursor)
			model.pushKill("\n", false)
			model.buffer.joinLines(row, row+1)
		}
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	deleted := line[model.cursor.Col:]
	model.pushKill(deleted, false)
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
	model.pushKill(deleted, true)
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
	model.pushKill(deleted, true)
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
	model.pushKill(deleted, false)
	model.buffer.setLine(row, line[:model.cursor.Col]+line[end+1:])
	return nil
}

func handleInsertYank(model *editorModel) tea.Cmd {
	text := model.currentKill()
	if text == "" {
		return nil
	}

	insertKilledText(model, text)
	return nil
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
	model.noteInsertAction(insertActionSelfInsert)
	return nil
}

func insertTextAtCursor(model *editorModel, text string) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)
	model.buffer.insertAt(model.cursor.Row, model.cursor.Col, text)

	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		model.cursor.Col += len(text)
		model.noteInsertAction(insertActionSelfInsert)
		model.ensureCursorVisible()
		return nil
	}

	model.cursor.Row += len(lines) - 1
	model.cursor.Col = len(lines[len(lines)-1])
	model.noteInsertAction(insertActionSelfInsert)
	model.ensureCursorVisible()
	return nil
}

func handleInsertTab(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)

	line := model.buffer.Line(model.cursor.Row)
	newLine := line[:model.cursor.Col] + "\t" + line[model.cursor.Col:]
	model.buffer.setLine(model.cursor.Row, newLine)
	model.cursor.Col += 1
	model.noteInsertAction(insertActionSelfInsert)
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
	m.noteInsertAction(insertActionSelfInsert)
	m.ensureCursorVisible()
	return nil
}
