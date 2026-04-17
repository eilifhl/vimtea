package vimtea

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type insertActionKind int

const (
	insertActionNone insertActionKind = iota
	insertActionSelfInsert
	insertActionMotion
	insertActionKill
	insertActionYank
)

func (m *editorModel) noteInsertAction(action insertActionKind) {
	m.lastInsertAction = action
	if action != insertActionYank {
		m.lastYankRange = nil
	}
	if action != insertActionKill {
		m.killRingIndex = 0
	}
}

func (m *editorModel) pushKill(text string, backward bool) {
	if text == "" {
		return
	}

	if m.lastInsertAction == insertActionKill && len(m.killRing) > 0 {
		if backward {
			m.killRing[0] = text + m.killRing[0]
		} else {
			m.killRing[0] += text
		}
	} else {
		m.killRing = append([]string{text}, m.killRing...)
		if len(m.killRing) > 30 {
			m.killRing = m.killRing[:30]
		}
	}

	m.killRingIndex = 0
	m.yankBuffer = m.killRing[0]
	writeClipboardText(m.yankBuffer)
	m.noteInsertAction(insertActionKill)
}

func (m *editorModel) currentKill() string {
	if len(m.killRing) == 0 {
		return m.yankBuffer
	}
	if m.killRingIndex < 0 || m.killRingIndex >= len(m.killRing) {
		m.killRingIndex = 0
	}
	return m.killRing[m.killRingIndex]
}

func insertCursorForPointOffset(buf *buffer, offset int) Cursor {
	if offset < 0 {
		return newCursor(0, 0)
	}
	for row := 0; row < buf.lineCount(); row++ {
		lineLen := buf.lineLength(row)
		if offset <= lineLen {
			return newCursor(row, offset)
		}
		offset -= lineLen + 1
	}
	lastRow := buf.lineCount() - 1
	return newCursor(lastRow, buf.lineLength(lastRow))
}

func handleInsertYankPop(model *editorModel) tea.Cmd {
	if model.lastInsertAction != insertActionYank || model.lastYankRange == nil || len(model.killRing) < 2 {
		return nil
	}

	model.killRingIndex = (model.killRingIndex + 1) % len(model.killRing)
	replacement := model.currentKill()

	model.buffer.saveUndoState(model.cursor)
	start := model.lastYankRange.Start
	model.buffer.deleteRange(model.lastYankRange.Start, model.lastYankRange.End)
	model.buffer.insertAt(start.Row, start.Col, replacement)

	startOffset := cursorToTextOffset(model.buffer, start)
	model.cursor = insertCursorForPointOffset(model.buffer, startOffset+len(replacement))
	model.lastYankRange = &TextRange{
		Start: start,
		End:   cursorForTextOffset(model, startOffset+len(replacement)-1),
	}
	model.yankBuffer = replacement
	model.ensureCursorVisible()
	return nil
}

func handleInsertTransposeWords(model *editorModel) tea.Cmd {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return nil
	}

	leftStart, leftEnd, rightStart, rightEnd, ok := adjacentWordBounds(line, model.cursor.Col)
	if !ok {
		return nil
	}

	model.buffer.saveUndoState(model.cursor)
	newLine := line[:leftStart] + line[rightStart:rightEnd] + line[leftEnd:rightStart] + line[leftStart:leftEnd] + line[rightEnd:]
	model.buffer.setLine(model.cursor.Row, newLine)
	model.cursor.Col = rightEnd
	model.noteInsertAction(insertActionSelfInsert)
	return nil
}

func adjacentWordBounds(line string, col int) (int, int, int, int, bool) {
	if len(line) == 0 {
		return 0, 0, 0, 0, false
	}

	col = max(0, min(len(line), col))
	leftEnd := col
	for leftEnd > 0 && isWhitespace(line[leftEnd-1]) {
		leftEnd--
	}
	leftStart := leftEnd
	for leftStart > 0 && !isWhitespace(line[leftStart-1]) {
		leftStart--
	}
	if leftStart == leftEnd {
		return 0, 0, 0, 0, false
	}

	rightStart := col
	for rightStart < len(line) && isWhitespace(line[rightStart]) {
		rightStart++
	}
	rightEnd := rightStart
	for rightEnd < len(line) && !isWhitespace(line[rightEnd]) {
		rightEnd++
	}
	if rightStart == rightEnd {
		return 0, 0, 0, 0, false
	}

	return leftStart, leftEnd, rightStart, rightEnd, true
}

func handleInsertOpenLine(model *editorModel) tea.Cmd {
	model.buffer.saveUndoState(model.cursor)
	model.buffer.insertAt(model.cursor.Row, model.cursor.Col, "\n")
	model.noteInsertAction(insertActionSelfInsert)
	model.ensureCursorVisible()
	return nil
}

func insertKilledText(model *editorModel, text string) {
	if text == "" {
		return
	}

	startOffset := cursorToTextOffset(model.buffer, model.cursor)
	start := insertCursorForPointOffset(model.buffer, startOffset)
	model.buffer.saveUndoState(model.cursor)
	model.buffer.insertAt(model.cursor.Row, model.cursor.Col, text)
	model.cursor = insertCursorForPointOffset(model.buffer, startOffset+len(text))
	end := cursorForTextOffset(model, startOffset+len(text)-1)
	model.lastYankRange = &TextRange{Start: start, End: end}
	model.noteInsertAction(insertActionYank)
	model.ensureCursorVisible()
}

func bigPreviousWordStart(line string, col int) int {
	if col > len(line) {
		col = len(line)
	}
	i := col - 1
	for i > 0 && isWhitespace(line[i]) {
		i--
	}
	for i > 0 && !isWhitespace(line[i-1]) {
		i--
	}
	return max(0, i)
}

func bigNextWordStart(line string, col int) int {
	i := col
	if i < len(line) && !isWhitespace(line[i]) {
		for i < len(line) && !isWhitespace(line[i]) {
			i++
		}
	}
	for i < len(line) && isWhitespace(line[i]) {
		i++
	}
	return min(len(line), i)
}

func joinKillPieces(parts ...string) string {
	return strings.Join(parts, "")
}
