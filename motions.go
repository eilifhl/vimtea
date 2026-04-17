package vimtea

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type pendingFindState struct {
	key   string
	count int
}

type findMotionState struct {
	key  string
	char byte
}

type motionSpec struct {
	kind   string
	key    string
	count  int
	char   byte
	around bool
	object byte
}

func directMotionCommand(kind string) Command {
	return func(m *editorModel) tea.Cmd {
		moveByMotion(m, motionSpec{kind: kind}, max(1, m.countPrefix), m.hasCountPrefix)
		return nil
	}
}

func directMotionWithChar(kind string, ch byte) Command {
	return func(m *editorModel) tea.Cmd {
		moveByMotion(m, motionSpec{kind: kind, char: ch}, max(1, m.countPrefix), m.hasCountPrefix)
		return nil
	}
}

func beginFindMotion(kind string) Command {
	return func(m *editorModel) tea.Cmd {
		m.pendingFind = &pendingFindState{
			key:   kind,
			count: max(1, m.countPrefix),
		}
		m.keySequence = nil
		m.resetCountPrefix()
		return nil
	}
}

func repeatLastFind(reverse bool) Command {
	return func(m *editorModel) tea.Cmd {
		if m.lastFind == nil {
			return nil
		}
		key := m.lastFind.key
		if reverse {
			key = reverseFindKey(key)
		}
		moveByMotion(m, motionSpec{
			kind: kindForFindKey(key),
			key:  key,
			char: m.lastFind.char,
		}, max(1, m.countPrefix), m.hasCountPrefix)
		return nil
	}
}

func reverseFindKey(key string) string {
	switch key {
	case "f":
		return "F"
	case "F":
		return "f"
	case "t":
		return "T"
	case "T":
		return "t"
	default:
		return key
	}
}

func kindForFindKey(key string) string {
	switch key {
	case "f", "F":
		return "find"
	case "t", "T":
		return "till"
	default:
		return ""
	}
}

func (m *editorModel) handlePendingFind(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.pendingFind = nil
		m.resetCountPrefix()
		return m, nil
	}

	if msg.Type != tea.KeyRunes || len(msg.Runes) != 1 || msg.Alt {
		m.pendingFind = nil
		m.resetCountPrefix()
		return m, nil
	}

	pending := m.pendingFind
	m.pendingFind = nil
	moveByMotion(m, motionSpec{
		kind: kindForFindKey(pending.key),
		key:  pending.key,
		char: byte(msg.Runes[0]),
	}, pending.count, pending.count > 1)
	return m, nil
}

func moveByMotion(model *editorModel, motion motionSpec, count int, countProvided bool) bool {
	target, ok := resolveMotionCursor(model, motion, count, countProvided)
	if !ok {
		return false
	}
	model.cursor = target
	model.desiredCol = model.cursor.Col
	if model.mode == ModeInsert {
		model.noteInsertAction(insertActionMotion)
	}
	model.ensureCursorVisible()
	return true
}

func resolveMotionCursor(model *editorModel, motion motionSpec, count int, countProvided bool) (Cursor, bool) {
	count = max(1, count)

	switch motion.kind {
	case "h":
		tmp := *model
		tmp.countPrefix = count
		moveCursorLeft(&tmp)
		return tmp.cursor, tmp.cursor != model.cursor
	case "l":
		tmp := *model
		tmp.countPrefix = count
		moveCursorRight(&tmp)
		return tmp.cursor, tmp.cursor != model.cursor
	case "j":
		tmp := *model
		tmp.countPrefix = count
		moveCursorDown(&tmp)
		return tmp.cursor, tmp.cursor != model.cursor
	case "k":
		tmp := *model
		tmp.countPrefix = count
		moveCursorUp(&tmp)
		return tmp.cursor, tmp.cursor != model.cursor
	case "0":
		return newCursor(model.cursor.Row, 0), model.cursor.Col != 0
	case "^":
		tmp := *model
		moveToFirstNonWhitespace(&tmp)
		return tmp.cursor, tmp.cursor != model.cursor
	case "$":
		return newCursor(model.cursor.Row, max(0, model.buffer.lineLength(model.cursor.Row)-1)), model.buffer.lineLength(model.cursor.Row) > 0 && model.cursor.Col != model.buffer.lineLength(model.cursor.Row)-1
	case "g_":
		return newCursor(model.cursor.Row, lastNonWhitespaceColumn(model.buffer.Line(model.cursor.Row))), model.cursor.Col != lastNonWhitespaceColumn(model.buffer.Line(model.cursor.Row))
	case "gg":
		row := 0
		if countProvided {
			row = min(model.buffer.lineCount()-1, count-1)
		}
		return newCursor(row, firstNonWhitespaceColumn(model.buffer.Line(row))), row != model.cursor.Row || firstNonWhitespaceColumn(model.buffer.Line(row)) != model.cursor.Col
	case "G":
		row := model.buffer.lineCount() - 1
		if countProvided {
			row = min(model.buffer.lineCount()-1, max(0, count-1))
		}
		return newCursor(row, firstNonWhitespaceColumn(model.buffer.Line(row))), row != model.cursor.Row || firstNonWhitespaceColumn(model.buffer.Line(row)) != model.cursor.Col
	case "w":
		return cursorForTextOffset(model, nextWordStartOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, false)), true
	case "W":
		return cursorForTextOffset(model, nextWordStartOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, true)), true
	case "b":
		return cursorForTextOffset(model, prevWordStartOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, false)), true
	case "B":
		return cursorForTextOffset(model, prevWordStartOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, true)), true
	case "e":
		return cursorForTextOffset(model, nextWordEndOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, false)), true
	case "E":
		return cursorForTextOffset(model, nextWordEndOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, true)), true
	case "ge":
		return cursorForTextOffset(model, prevWordEndOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, false)), true
	case "gE":
		return cursorForTextOffset(model, prevWordEndOffset(model.buffer.text(), cursorToTextOffset(model.buffer, model.cursor), count, true)), true
	case "{":
		return paragraphStartCursor(model, count, false)
	case "}":
		return paragraphStartCursor(model, count, true)
	case "%":
		return matchingBracketCursor(model)
	case "find", "till":
		return findMotionCursor(model, motion, count)
	case "H":
		return screenCursor(model, "H", count, countProvided)
	case "M":
		return screenCursor(model, "M", count, countProvided)
	case "L":
		return screenCursor(model, "L", count, countProvided)
	default:
		return Cursor{}, false
	}
}

func screenCursor(model *editorModel, kind string, count int, countProvided bool) (Cursor, bool) {
	top := model.viewport.YOffset
	height := max(1, model.viewport.Height)
	bottom := min(model.buffer.lineCount()-1, top+height-1)
	row := top
	switch kind {
	case "H":
		if countProvided {
			row = min(bottom, top+count-1)
		}
	case "M":
		row = top + (bottom-top)/2
	case "L":
		row = bottom
		if countProvided {
			row = max(top, bottom-count+1)
		}
	}
	row = max(0, min(model.buffer.lineCount()-1, row))
	col := firstNonWhitespaceColumn(model.buffer.Line(row))
	return newCursor(row, col), row != model.cursor.Row || col != model.cursor.Col
}

func paragraphStartCursor(model *editorModel, count int, forward bool) (Cursor, bool) {
	row := model.cursor.Row
	for range count {
		if forward {
			row = nextParagraphRow(model.buffer.lines, row)
		} else {
			row = prevParagraphRow(model.buffer.lines, row)
		}
	}
	col := firstNonWhitespaceColumn(model.buffer.Line(row))
	return newCursor(row, col), row != model.cursor.Row || col != model.cursor.Col
}

func nextParagraphRow(lines []string, row int) int {
	i := row + 1
	for i < len(lines) && !isBlankLine(lines[i]) {
		i++
	}
	for i < len(lines) && isBlankLine(lines[i]) {
		i++
	}
	if i >= len(lines) {
		return len(lines) - 1
	}
	return i
}

func prevParagraphRow(lines []string, row int) int {
	i := row - 1
	for i >= 0 && isBlankLine(lines[i]) {
		i--
	}
	for i >= 0 && !isBlankLine(lines[i]) {
		i--
	}
	i++
	if i < 0 {
		return 0
	}
	return i
}

func isBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

func matchingBracketCursor(model *editorModel) (Cursor, bool) {
	text := model.buffer.text()
	offset := cursorToTextOffset(model.buffer, model.cursor)
	if offset < 0 || offset >= len(text) {
		return Cursor{}, false
	}

	ch := text[offset]
	switch ch {
	case '(', '[', '{':
		match, ok := findMatchingForward(text, offset, ch, matchingCloseBracket(ch))
		if !ok {
			return Cursor{}, false
		}
		return cursorForTextOffset(model, match), true
	case ')', ']', '}':
		match, ok := findMatchingBackward(text, offset, matchingOpenBracket(ch), ch)
		if !ok {
			return Cursor{}, false
		}
		return cursorForTextOffset(model, match), true
	default:
		open, close, ok := enclosingBracketPair(model)
		if !ok {
			return Cursor{}, false
		}
		if model.cursor.Col-open <= close-model.cursor.Col {
			return newCursor(model.cursor.Row, open), true
		}
		return newCursor(model.cursor.Row, close), true
	}
}

func findMatchingForward(text string, start int, open, close byte) (int, bool) {
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func findMatchingBackward(text string, start int, open, close byte) (int, bool) {
	depth := 0
	for i := start; i >= 0; i-- {
		switch text[i] {
		case close:
			depth++
		case open:
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func matchingCloseBracket(open byte) byte {
	switch open {
	case '(':
		return ')'
	case '[':
		return ']'
	case '{':
		return '}'
	default:
		return 0
	}
}

func matchingOpenBracket(close byte) byte {
	switch close {
	case ')':
		return '('
	case ']':
		return '['
	case '}':
		return '{'
	default:
		return 0
	}
}

func findMotionCursor(model *editorModel, motion motionSpec, count int) (Cursor, bool) {
	line := model.buffer.Line(model.cursor.Row)
	col, ok := findInLine(line, model.cursor.Col, motion.char, count, motion.key)
	if !ok {
		return Cursor{}, false
	}

	target := col
	switch motion.key {
	case "t":
		target = col - 1
	case "T":
		target = col + 1
	}

	if target < 0 || target >= len(line) {
		return Cursor{}, false
	}

	model.lastFind = &findMotionState{key: motion.key, char: motion.char}
	return newCursor(model.cursor.Row, target), true
}

func findInLine(line string, startCol int, ch byte, count int, key string) (int, bool) {
	count = max(1, count)
	switch key {
	case "f", "t":
		for i := startCol + 1; i < len(line); i++ {
			if line[i] != ch {
				continue
			}
			count--
			if count == 0 {
				return i, true
			}
		}
	case "F", "T":
		for i := startCol - 1; i >= 0; i-- {
			if line[i] != ch {
				continue
			}
			count--
			if count == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func parseOperatorMotion(tokens []string) (motionSpec, bool, bool) {
	if len(tokens) == 0 {
		return motionSpec{}, true, false
	}

	count := 0
	i := 0
	for i < len(tokens) && len(tokens[i]) == 1 && tokens[i] >= "0" && tokens[i] <= "9" && (tokens[i] != "0" || count > 0) {
		count = count*10 + int(tokens[i][0]-'0')
		i++
	}
	if i == len(tokens) {
		return motionSpec{}, true, false
	}
	if count == 0 {
		count = 1
	}

	head := tokens[i]
	rest := tokens[i:]
	if len(rest) == 1 {
		switch head {
		case "d", "c", "y":
			return motionSpec{kind: "same-line", count: count}, false, true
		case "h", "j", "k", "l", "w", "W", "b", "B", "e", "E", "0", "^", "$", "%", "{", "}":
			return motionSpec{kind: head, count: count}, false, true
		case "G":
			return motionSpec{kind: "G", count: count}, false, true
		case "H", "M", "L":
			return motionSpec{kind: head, count: count}, false, true
		case "f", "F", "t", "T":
			return motionSpec{}, true, false
		case ";", ",":
			return motionSpec{kind: head, count: count}, false, true
		case "i", "a", "g":
			return motionSpec{}, true, false
		}
	}

	switch head {
	case "g":
		if len(rest) == 2 {
			switch rest[1] {
			case "g":
				return motionSpec{kind: "gg", count: count}, false, true
			case "e":
				return motionSpec{kind: "ge", count: count}, false, true
			case "E":
				return motionSpec{kind: "gE", count: count}, false, true
			case "_":
				return motionSpec{kind: "g_", count: count}, false, true
			}
		}
	case "f", "F", "t", "T":
		if len(rest) == 2 && len(rest[1]) == 1 {
			return motionSpec{
				kind:  kindForFindKey(head),
				key:   head,
				char:  rest[1][0],
				count: count,
			}, false, true
		}
	case "i", "a":
		if len(rest) == 2 && len(rest[1]) == 1 {
			return motionSpec{
				kind:   "text-object",
				around: head == "a",
				object: rest[1][0],
				count:  count,
			}, false, true
		}
	case ";", ",":
		return motionSpec{kind: head, count: count}, false, true
	}

	return motionSpec{}, false, false
}

func executeOperatorMotion(model *editorModel, operator string, motion motionSpec) tea.Cmd {
	count := max(1, model.countPrefix) * max(1, motion.count)

	if operator == "d" && motion.kind == "same-line" {
		model.countPrefix = count
		return deleteLineWithCount(model)
	}
	if operator == "y" && motion.kind == "same-line" {
		model.countPrefix = count
		return yankLinesWithCount(model)
	}
	if operator == "c" && motion.kind == "same-line" {
		model.countPrefix = count
		return changeLinesWithCount(model)
	}

	if motion.kind == "text-object" {
		return applyTextObjectOperation(model, operator, motion)
	}

	if motion.kind == ";" || motion.kind == "," {
		if model.lastFind == nil {
			return nil
		}
		key := model.lastFind.key
		if motion.kind == "," {
			key = reverseFindKey(key)
		}
		motion.kind = kindForFindKey(key)
		motion.key = key
		motion.char = model.lastFind.char
	}

	if isLinewiseOperatorMotion(motion.kind) {
		target, ok := resolveMotionCursor(model, motion, count, motion.count > 1 || model.hasCountPrefix)
		if !ok {
			return nil
		}
		return applyLinewiseOperator(model, operator, model.cursor.Row, target.Row)
	}

	target, ok := resolveMotionCursor(model, motion, count, motion.count > 1 || model.hasCountPrefix)
	if !ok {
		return nil
	}
	return applyCharwiseOperator(model, operator, motion, target)
}

func isLinewiseOperatorMotion(kind string) bool {
	switch kind {
	case "j", "k", "gg", "G", "{", "}", "H", "M", "L":
		return true
	default:
		return false
	}
}

func applyLinewiseOperator(model *editorModel, operator string, startRow, endRow int) tea.Cmd {
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}

	count := endRow - startRow + 1
	model.cursor.Row = startRow
	model.cursor.Col = 0
	model.countPrefix = count
	switch operator {
	case "d":
		return deleteLineWithCount(model)
	case "y":
		return yankLinesWithCount(model)
	case "c":
		return changeLinesWithCount(model)
	default:
		return nil
	}
}

func applyCharwiseOperator(model *editorModel, operator string, motion motionSpec, target Cursor) tea.Cmd {
	start, end, ok := operatorCharRange(model, motion, target)
	if !ok {
		return nil
	}

	switch operator {
	case "d":
		model.buffer.saveUndoState(model.cursor)
		model.yankBuffer = model.buffer.deleteRange(start, end)
		writeClipboardText(model.yankBuffer)
		model.cursor = start
		model.ensureCursorVisible()
		return nil
	case "y":
		model.yankBuffer = model.buffer.getRange(start, end)
		writeClipboardText(model.yankBuffer)
		return nil
	case "c":
		model.buffer.saveUndoState(model.cursor)
		model.yankBuffer = model.buffer.deleteRange(start, end)
		writeClipboardText(model.yankBuffer)
		model.cursor = start
		model.ensureCursorVisible()
		return switchMode(model, ModeInsert)
	default:
		return nil
	}
}

func operatorCharRange(model *editorModel, motion motionSpec, target Cursor) (Cursor, Cursor, bool) {
	currentOff := cursorToTextOffset(model.buffer, model.cursor)
	targetOff := cursorToTextOffset(model.buffer, target)

	if currentOff == targetOff && motion.kind != "find" && motion.kind != "till" {
		return Cursor{}, Cursor{}, false
	}

	if targetOff > currentOff {
		endOff := targetOff
		if isExclusiveForwardMotion(motion) {
			endOff = prevTextOffset(model.buffer, targetOff)
		}
		if endOff < currentOff {
			return Cursor{}, Cursor{}, false
		}
		return model.cursor, cursorForTextOffset(model, endOff), true
	}

	endOff := prevTextOffset(model.buffer, currentOff)
	if endOff < 0 {
		return Cursor{}, Cursor{}, false
	}

	startOff := targetOff
	if isExclusiveBackwardMotion(motion) {
		startOff = nextTextOffset(model.buffer, targetOff)
	}
	if startOff > endOff {
		return Cursor{}, Cursor{}, false
	}
	return cursorForTextOffset(model, startOff), cursorForTextOffset(model, endOff), true
}

func isExclusiveForwardMotion(motion motionSpec) bool {
	switch motion.kind {
	case "l", "w", "W":
		return true
	case "find":
		return motion.key == "t"
	case "till":
		return true
	default:
		return false
	}
}

func isExclusiveBackwardMotion(motion motionSpec) bool {
	switch motion.kind {
	case "h":
		return false
	case "find":
		return motion.key == "T"
	case "till":
		return true
	default:
		return false
	}
}

func applyTextObjectOperation(model *editorModel, operator string, motion motionSpec) tea.Cmd {
	start, end, ok := textObjectBoundary(model, motion)
	if !ok {
		return nil
	}

	selectedText := model.buffer.getRange(start, end)
	switch operator {
	case "y":
		model.yankBuffer = selectedText
		writeClipboardText(model.yankBuffer)
		return nil
	case "d", "c":
		model.buffer.saveUndoState(model.cursor)
		model.yankBuffer = selectedText
		writeClipboardText(model.yankBuffer)
		model.buffer.deleteRange(start, end)
		model.cursor = start
		if operator == "c" {
			return switchMode(model, ModeInsert)
		}
		return nil
	default:
		return nil
	}
}

func textObjectBoundary(model *editorModel, motion motionSpec) (Cursor, Cursor, bool) {
	switch motion.object {
	case 'w':
		if motion.around {
			start, end, ok := getAroundWordBoundary(model)
			return Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end}, ok
		}
		start, end := getWordBoundary(model)
		if start == end {
			return Cursor{}, Cursor{}, false
		}
		return Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end - 1}, true
	case 'W':
		if motion.around {
			start, end, ok := getAroundBigWordBoundary(model)
			return Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end}, ok
		}
		start, end := getBigWordBoundary(model)
		if start == end {
			return Cursor{}, Cursor{}, false
		}
		return Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end - 1}, true
	case '(', 'b':
		return delimitedBoundary(model, '(', ')', motion.around)
	case '[', ']':
		return delimitedBoundary(model, '[', ']', motion.around)
	case '{', 'B', '}':
		return delimitedBoundary(model, '{', '}', motion.around)
	case '"', '\'':
		start, end, ok, empty := getQuoteBoundary(model, motion.object, motion.around)
		if !ok || empty {
			return Cursor{}, Cursor{}, false
		}
		return Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end}, true
	default:
		return Cursor{}, Cursor{}, false
	}
}

func delimitedBoundary(model *editorModel, open, close byte, around bool) (Cursor, Cursor, bool) {
	start, end, ok := getDelimitedBoundary(model, open, close, around)
	if !ok {
		return Cursor{}, Cursor{}, false
	}
	return Cursor{Row: model.cursor.Row, Col: start}, Cursor{Row: model.cursor.Row, Col: end}, true
}

func getDelimitedBoundary(model *editorModel, open, close byte, around bool) (int, int, bool) {
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
		case open:
			stack = append(stack, i)
		case close:
			if len(stack) == 0 {
				continue
			}
			match := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			pairs = append(pairs, pair{open: match, close: i})
		}
	}

	cursorCol := model.cursor.Col
	bestOpen, bestClose := -1, -1
	bestSpan := 0
	for _, pair := range pairs {
		if cursorCol < pair.open || cursorCol > pair.close {
			continue
		}
		span := pair.close - pair.open
		if bestOpen == -1 || span < bestSpan {
			bestOpen = pair.open
			bestClose = pair.close
			bestSpan = span
		}
	}
	if bestOpen == -1 {
		return 0, 0, false
	}
	if around {
		return bestOpen, bestClose, true
	}
	if bestOpen+1 > bestClose-1 {
		return 0, 0, false
	}
	return bestOpen + 1, bestClose - 1, true
}

func getBigWordBoundary(model *editorModel) (int, int) {
	return getClassBoundary(model.buffer.Line(model.cursor.Row), model.cursor.Col, true)
}

func getAroundBigWordBoundary(model *editorModel) (int, int, bool) {
	line := model.buffer.Line(model.cursor.Row)
	if len(line) == 0 {
		return 0, 0, false
	}
	start, end := getBigWordBoundary(model)
	if start == end {
		return 0, 0, false
	}
	for start > 0 && isWhitespace(line[start-1]) {
		start--
	}
	for end < len(line) && isWhitespace(line[end]) {
		end++
	}
	return start, end - 1, true
}

func getClassBoundary(line string, col int, big bool) (int, int) {
	if len(line) == 0 {
		return 0, 0
	}
	col = max(0, min(len(line)-1, col))
	class := wordClass(line[col], big)
	start := col
	for start > 0 && wordClass(line[start-1], big) == class {
		start--
	}
	end := col
	for end+1 < len(line) && wordClass(line[end+1], big) == class {
		end++
	}
	return start, end + 1
}

func enclosingBracketPair(model *editorModel) (int, int, bool) {
	bestOpen, bestClose := -1, -1
	bestSpan := 0
	for _, pair := range []struct {
		open  byte
		close byte
	}{{'(', ')'}, {'[', ']'}, {'{', '}'}} {
		open, close, ok := getDelimitedBoundary(model, pair.open, pair.close, true)
		if !ok {
			continue
		}
		span := close - open
		if bestOpen == -1 || span < bestSpan {
			bestOpen, bestClose, bestSpan = open, close, span
		}
	}
	return bestOpen, bestClose, bestOpen != -1
}

func firstNonWhitespaceColumn(line string) int {
	for i := 0; i < len(line); i++ {
		if !isWhitespace(line[i]) {
			return i
		}
	}
	return 0
}

func lastNonWhitespaceColumn(line string) int {
	for i := len(line) - 1; i >= 0; i-- {
		if !isWhitespace(line[i]) {
			return i
		}
	}
	return 0
}

func cursorToTextOffset(buf *buffer, cursor Cursor) int {
	offset := 0
	for row := 0; row < cursor.Row; row++ {
		offset += len(buf.Line(row)) + 1
	}
	return offset + cursor.Col
}

func cursorForTextOffset(model *editorModel, offset int) Cursor {
	if offset < 0 {
		return newCursor(0, 0)
	}
	for row := 0; row < model.buffer.lineCount(); row++ {
		lineLen := model.buffer.lineLength(row)
		if offset < lineLen {
			return newCursor(row, offset)
		}
		if offset == lineLen {
			if row == model.buffer.lineCount()-1 {
				if lineLen == 0 {
					return newCursor(row, 0)
				}
				return newCursor(row, lineLen-1)
			}
			return newCursor(row+1, 0)
		}
		offset -= lineLen + 1
	}
	lastRow := model.buffer.lineCount() - 1
	lastLen := model.buffer.lineLength(lastRow)
	return newCursor(lastRow, max(0, lastLen-1))
}

func prevTextOffset(buf *buffer, offset int) int {
	if offset <= 0 {
		return -1
	}
	return offset - 1
}

func nextTextOffset(buf *buffer, offset int) int {
	text := buf.text()
	if offset >= len(text)-1 {
		return len(text) - 1
	}
	return offset + 1
}

func nextWordStartOffset(text string, offset, count int, big bool) int {
	offset = clampOffset(text, offset)
	for range count {
		i := offset
		if !isWhitespace(text[i]) {
			class := wordClass(text[i], big)
			for i < len(text) && wordClass(text[i], big) == class {
				i++
			}
		}
		for i < len(text) && isWhitespace(text[i]) {
			i++
		}
		if i >= len(text) {
			return len(text) - 1
		}
		offset = i
	}
	return offset
}

func prevWordStartOffset(text string, offset, count int, big bool) int {
	offset = clampOffset(text, offset)
	for range count {
		i := max(0, offset-1)
		for i > 0 && isWhitespace(text[i]) {
			i--
		}
		class := wordClass(text[i], big)
		for i > 0 && wordClass(text[i-1], big) == class {
			i--
		}
		offset = i
	}
	return offset
}

func nextWordEndOffset(text string, offset, count int, big bool) int {
	offset = clampOffset(text, offset)
	for range count {
		i := offset
		if isWhitespace(text[i]) {
			for i < len(text) && isWhitespace(text[i]) {
				i++
			}
			if i >= len(text) {
				return len(text) - 1
			}
		} else {
			class := wordClass(text[i], big)
			if i+1 < len(text) && wordClass(text[i+1], big) == class {
				for i+1 < len(text) && wordClass(text[i+1], big) == class {
					i++
				}
				offset = i
				continue
			}
			i++
			for i < len(text) && isWhitespace(text[i]) {
				i++
			}
			if i >= len(text) {
				return len(text) - 1
			}
		}

		class := wordClass(text[i], big)
		for i+1 < len(text) && wordClass(text[i+1], big) == class {
			i++
		}
		offset = i
	}
	return offset
}

func prevWordEndOffset(text string, offset, count int, big bool) int {
	offset = clampOffset(text, offset)
	for range count {
		i := max(0, offset-1)
		for i > 0 && isWhitespace(text[i]) {
			i--
		}
		class := wordClass(text[i], big)
		for i+1 < len(text) && wordClass(text[i+1], big) == class {
			i++
		}
		offset = i
		if i > 0 {
			offset = i
		}
	}
	return offset
}

func clampOffset(text string, offset int) int {
	if len(text) == 0 {
		return 0
	}
	if offset < 0 {
		return 0
	}
	if offset >= len(text) {
		return len(text) - 1
	}
	return offset
}

func wordClass(ch byte, big bool) int {
	if isWhitespace(ch) {
		return 0
	}
	if big {
		return 1
	}
	if isKeywordChar(ch) {
		return 1
	}
	return 2
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isKeywordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}
