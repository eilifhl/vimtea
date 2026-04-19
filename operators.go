package vimtea

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *editorModel) handleOperatorKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	now := time.Now()
	if now.Sub(m.lastKeyTime) > 750*time.Millisecond && len(m.operatorSequence) > 0 {
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.resetCountPrefix()
		return m, nil
	}
	m.lastKeyTime = now

	if msg.Type == tea.KeyEsc {
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.resetCountPrefix()
		return m, nil
	}

	keyStr := msg.String()
	if keyStr == "" {
		return m, nil
	}

	m.operatorSequence = append(m.operatorSequence, keyStr)
	if len(m.operatorSequence) == 1 && m.operatorSequence[0] == m.pendingOperator {
		cmd := executeOperatorMotion(m, m.pendingOperator, motionSpec{kind: "same-line", count: 1})
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.keySequence = nil
		m.resetCountPrefix()
		return m, cmd
	}

	motion, pending, ok := parseOperatorMotion(m.operatorSequence)
	if pending {
		return m, nil
	}
	if ok {
		cmd := executeOperatorMotion(m, m.pendingOperator, motion)
		m.pendingOperator = ""
		m.operatorSequence = nil
		m.keySequence = nil
		m.resetCountPrefix()
		return m, cmd
	}

	m.pendingOperator = ""
	m.operatorSequence = nil
	m.keySequence = nil
	m.resetCountPrefix()
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
		if candidate == full || strings.HasPrefix(candidate, full) {
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

	selectedText := model.buffer.getRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		model.buffer.deleteRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
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

	selectedText := model.buffer.getRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		model.buffer.deleteRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
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

	selectedText := model.buffer.getRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		model.buffer.deleteRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
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
		selectedText = model.buffer.getRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
	}
	model.yankBuffer = selectedText
	writeClipboardText(model.yankBuffer)

	switch operation {
	case "yank":
		model.statusMessage = fmt.Sprintf("yanked %d characters", len(selectedText))
	case "delete", "change":
		if !empty {
			model.buffer.deleteRange(Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end})
		}
		model.cursor.Col = start
		if operation == "change" {
			return switchMode(model, ModeInsert)
		}
	}

	return nil
}

func getInnerParenBoundary(model *editorModel) (int, int, bool) {
	start, end, ok := getDelimitedBoundary(model, '(', ')', false)
	if !ok {
		return 0, 0, false
	}
	return start, end, true
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

	var pairs []boundaryPair
	for i := 0; i+1 < len(quotePositions); i += 2 {
		pairs = append(pairs, boundaryPair{open: quotePositions[i], close: quotePositions[i+1]})
	}

	bestOpen, bestClose, ok := choosePairForCursor(model.cursor.Col, pairs)
	if !ok {
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
	for end < len(line) && isWordSeparator(line[end]) {
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

func getInnerBracketBoundary(model *editorModel) (int, int, bool) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0, false
	}

	type bracketPair struct {
		open  int
		close int
	}
	matchingClose := map[byte]byte{'(': ')', '[': ']', '{': '}', '<': '>'}
	matchingOpen := map[byte]byte{')': '(', ']': '[', '}': '{', '>': '<'}
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
