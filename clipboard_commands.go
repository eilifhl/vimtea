package vimtea

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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
	clipboard := readClipboardText()
	if clipboard != "" {
		model.yankBuffer = clipboard
	}
	if model.yankBuffer == "" {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	if strings.HasPrefix(model.yankBuffer, "\n") {
		return pasteLineAfter(model)
	}

	currLine := model.buffer.Line(model.cursor.Row)
	insertPos := model.cursor.Col

	if strings.Contains(model.yankBuffer, "\n") {
		lines := strings.Split(model.yankBuffer, "\n")
		firstLine := lines[0]
		remainderOfLine := ""
		if insertPos < len(currLine) {
			remainderOfLine = currLine[insertPos+1:]
		}

		if insertPos >= len(currLine) {
			model.buffer.setLine(model.cursor.Row, currLine+firstLine)
		} else {
			model.buffer.setLine(model.cursor.Row, currLine[:insertPos+1]+firstLine)
		}

		row := model.cursor.Row
		for i := 1; i < len(lines)-1; i++ {
			model.buffer.insertLine(row+i, lines[i])
		}

		if len(lines) > 1 {
			lastLine := lines[len(lines)-1]
			model.buffer.insertLine(row+len(lines)-1, lastLine+remainderOfLine)
		} else {
			currLineContent := model.buffer.Line(model.cursor.Row)
			model.buffer.setLine(model.cursor.Row, currLineContent+remainderOfLine)
		}

		model.cursor.Row = row + len(lines) - 1
		if len(lines) > 1 {
			model.cursor.Col = len(lines[len(lines)-1])
		} else {
			model.cursor.Col = insertPos + len(firstLine) + 1
		}

		if model.mode != ModeInsert && model.cursor.Col > 0 {
			model.cursor.Col--
		}
	} else {
		if insertPos >= len(currLine) {
			model.buffer.setLine(model.cursor.Row, currLine+model.yankBuffer)
		} else {
			model.buffer.setLine(model.cursor.Row, currLine[:insertPos+1]+model.yankBuffer+currLine[insertPos+1:])
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
	clipboard := readClipboardText()
	if clipboard != "" {
		model.yankBuffer = clipboard
	}
	if model.yankBuffer == "" {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	if strings.HasPrefix(model.yankBuffer, "\n") {
		return pasteLineBefore(model)
	}

	currLine := model.buffer.Line(model.cursor.Row)
	insertPos := model.cursor.Col

	if strings.Contains(model.yankBuffer, "\n") {
		lines := strings.Split(model.yankBuffer, "\n")
		firstLine := lines[0]
		newFirstLine := currLine[:insertPos] + firstLine
		model.buffer.setLine(model.cursor.Row, newFirstLine)

		if len(lines) == 1 {
			model.buffer.setLine(model.cursor.Row, newFirstLine+currLine[insertPos:])
		} else {
			lastLineIndex := len(lines) - 1
			lastLine := lines[lastLineIndex] + currLine[insertPos:]

			row := model.cursor.Row
			for i := 1; i < lastLineIndex; i++ {
				model.buffer.insertLine(row+i, lines[i])
			}
			model.buffer.insertLine(row+lastLineIndex, lastLine)
		}

		model.cursor.Col = insertPos + len(firstLine)
		if model.mode != ModeInsert && model.cursor.Col > 0 {
			model.cursor.Col--
		}
	} else {
		model.buffer.setLine(model.cursor.Row, currLine[:insertPos]+model.yankBuffer+currLine[insertPos:])
		model.cursor.Col = max(insertPos+len(model.yankBuffer)-1, 0)
	}

	model.ensureCursorVisible()
	return nil
}

func pasteLineAfter(model *editorModel) tea.Cmd {
	clipboard := readClipboardText()
	if clipboard != "" {
		model.yankBuffer = clipboard
	}
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
	clipboard := readClipboardText()
	if clipboard != "" {
		model.yankBuffer = clipboard
	}
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
		model.buffer.setLine(model.cursor.Row, currLine[:insertPos]+model.yankBuffer+currLine[insertPos:])
		model.cursor.Col = max(insertPos+len(model.yankBuffer)-1, 0)
	}

	model.ensureCursorVisible()
	return switchMode(model, ModeNormal)
}
