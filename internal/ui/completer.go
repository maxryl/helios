package ui

import (
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"helios/internal/db"
)

const maxCompletions = 10

type suggestContext int

const (
	suggestAll     suggestContext = iota
	suggestTables
	suggestColumns
)

var tableKeywords = map[string]bool{
	"FROM": true, "JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true,
	"FULL": true, "CROSS": true, "INTO": true, "UPDATE": true, "TABLE": true,
	"DELETE": true,
}

var columnKeywords = map[string]bool{
	"SELECT": true, "WHERE": true, "ON": true, "SET": true,
	"AND": true, "OR": true, "BY": true, "HAVING": true,
	"ORDER": true, "GROUP": true, "DISTINCT": true,
}

func wordBoundary(r rune) bool {
	return unicode.IsSpace(r) || r == ',' || r == ';' || r == '(' || r == ')' || r == '='
}

// cursorOffset returns the rune offset of the cursor in the text.
func cursorOffset(text string, row, col int) int {
	lines := strings.Split(text, "\n")
	offset := 0
	for i := 0; i < row && i < len(lines); i++ {
		offset += len([]rune(lines[i])) + 1 // +1 for newline
	}
	if row < len(lines) {
		lineRunes := len([]rune(lines[row]))
		if col > lineRunes {
			col = lineRunes
		}
		offset += col
	}
	return offset
}

// textUpToCursor returns the portion of the text before the cursor.
func textUpToCursor(text string, row, col int) string {
	runes := []rune(text)
	off := cursorOffset(text, row, col)
	if off > len(runes) {
		off = len(runes)
	}
	return string(runes[:off])
}

func extractCurrentWord(text string) string {
	runes := []rune(text)
	i := len(runes) - 1
	for i >= 0 && !wordBoundary(runes[i]) {
		i--
	}
	return string(runes[i+1:])
}

func detectContext(text string) suggestContext {
	runes := []rune(text)
	i := len(runes) - 1
	for i >= 0 && !wordBoundary(runes[i]) {
		i--
	}
	for i >= 0 {
		for i >= 0 && wordBoundary(runes[i]) {
			i--
		}
		if i < 0 {
			break
		}
		end := i + 1
		for i >= 0 && !wordBoundary(runes[i]) {
			i--
		}
		word := strings.ToUpper(string(runes[i+1 : end]))
		if tableKeywords[word] {
			return suggestTables
		}
		if columnKeywords[word] {
			return suggestColumns
		}
	}
	return suggestAll
}

// tableRef is a table referenced in the query, optionally with an alias.
type tableRef struct {
	table string // actual table name
	alias string // alias (empty if none)
}

// parseTableRefs extracts table names and aliases from FROM and JOIN clauses.
// Handles patterns like:
//
//	FROM tr_company
//	FROM tr_company c
//	FROM tr_company AS c
//	JOIN tr_stock s ON ...
//	JOIN tr_stock AS s ON ...
func parseTableRefs(sql string) []tableRef {
	tokens := tokenize(sql)
	var refs []tableRef

	for i := 0; i < len(tokens); i++ {
		upper := strings.ToUpper(tokens[i])
		if !tableKeywords[upper] {
			continue
		}
		// The next token after a table keyword is the table name.
		i++
		if i >= len(tokens) {
			break
		}
		tableName := tokens[i]

		// Check for alias: the token after table name could be AS or a plain identifier.
		alias := ""
		if i+1 < len(tokens) {
			next := strings.ToUpper(tokens[i+1])
			if next == "AS" {
				// tableName AS alias
				if i+2 < len(tokens) {
					alias = tokens[i+2]
					i += 2
				}
			} else if !isKeyword(next) && next != "ON" && next != "WHERE" &&
				next != "SET" && next != "(" && next != ")" &&
				next != "," && next != ";" {
				// tableName alias (implicit)
				alias = tokens[i+1]
				i++
			}
		}

		refs = append(refs, tableRef{table: tableName, alias: alias})
	}

	return refs
}

// tokenize splits SQL into whitespace-delimited tokens, keeping punctuation
// as separate tokens.
func tokenize(sql string) []string {
	var tokens []string
	runes := []rune(sql)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		// Single-char punctuation tokens.
		if r == ',' || r == ';' || r == '(' || r == ')' || r == '=' || r == '.' || r == '*' {
			tokens = append(tokens, string(r))
			i++
			continue
		}
		// String literals — skip over them.
		if r == '\'' {
			i++
			for i < len(runes) && runes[i] != '\'' {
				i++
			}
			i++ // skip closing quote
			continue
		}
		// Word token.
		start := i
		for i < len(runes) && !unicode.IsSpace(runes[i]) &&
			runes[i] != ',' && runes[i] != ';' && runes[i] != '(' &&
			runes[i] != ')' && runes[i] != '=' && runes[i] != '.' &&
			runes[i] != '*' && runes[i] != '\'' {
			i++
		}
		if i > start {
			tokens = append(tokens, string(runes[start:i]))
		}
	}
	return tokens
}

var allKeywords = func() map[string]bool {
	m := make(map[string]bool)
	for k := range tableKeywords {
		m[k] = true
	}
	for k := range columnKeywords {
		m[k] = true
	}
	for _, kw := range []string{
		"AS", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "ILIKE", "IS",
		"NULL", "TRUE", "FALSE", "LIMIT", "OFFSET", "INSERT", "VALUES",
		"CREATE", "ALTER", "DROP", "INDEX", "VIEW", "TRIGGER", "FUNCTION",
		"SCHEMA", "DATABASE", "BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT",
		"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE", "CHECK",
		"DEFAULT", "CONSTRAINT", "CASCADE", "RESTRICT", "RETURNING",
		"WITH", "RECURSIVE", "UNION", "ALL", "INTERSECT", "EXCEPT",
		"EXPLAIN", "ANALYZE", "ASC", "DESC", "CASE", "WHEN", "THEN",
		"ELSE", "END", "NATURAL", "USING",
	} {
		m[kw] = true
	}
	return m
}()

func isKeyword(s string) bool {
	return allKeywords[strings.ToUpper(s)]
}

// Completer shows autocomplete suggestions in an inline list widget placed
// between the editor and results pane. It never uses popups or overlays,
// so it cannot steal keyboard focus from the editor.
type Completer struct {
	schema      *db.SchemaCache
	editor      *widget.Entry
	list        *widget.List
	bg          *canvas.Rectangle
	holder      *fyne.Container
	suggestions []string
	currentWord string
	selected    int
	inserting   bool
	visible     bool
}

// NewCompleter creates a Completer.
func NewCompleter(editor *widget.Entry, schema *db.SchemaCache) *Completer {
	c := &Completer{
		schema:   schema,
		editor:   editor,
		selected: -1,
	}

	c.list = widget.NewList(
		func() int { return len(c.suggestions) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(c.suggestions) {
				obj.(*widget.Label).SetText(c.suggestions[id])
			}
		},
	)

	// OnSelected only tracks the highlight — insertion happens on Enter
	// via AcceptSelected. This lets the user browse with arrow keys
	// without committing.
	c.list.OnSelected = func(id widget.ListItemID) {
		c.selected = id
	}

	c.bg = canvas.NewRectangle(theme.OverlayBackgroundColor())
	c.holder = container.NewStack()

	return c
}

// Widget returns the container to embed in the terminal layout.
func (c *Completer) Widget() *fyne.Container {
	return c.holder
}

// OnTextChanged is called from the editor's OnChanged callback.
func (c *Completer) OnTextChanged(fullText string) {
	if c.inserting {
		return
	}

	// Use only the text up to the cursor position so that suggestions
	// are context-aware regardless of where the cursor is in the query.
	text := textUpToCursor(fullText, c.editor.CursorRow, c.editor.CursorColumn)
	word := extractCurrentWord(text)

	if len(word) < 2 {
		c.Dismiss()
		return
	}

	if word == c.currentWord && c.visible {
		return
	}

	// Pass the full text for table parsing (FROM/JOIN may be after cursor),
	// but use text-up-to-cursor for context detection.
	suggestions := c.computeSuggestions(fullText, text, word)

	if len(suggestions) == 0 {
		c.Dismiss()
		return
	}

	if len(suggestions) > maxCompletions {
		suggestions = suggestions[:maxCompletions]
	}

	c.suggestions = suggestions
	c.currentWord = word

	// Clear any stale selection in the list widget before updating.
	c.list.UnselectAll()

	if !c.visible {
		c.show()
	} else {
		c.list.Refresh()
	}

	// Always select the first item.
	c.selected = 0
	c.list.Select(0)
}

// computeSuggestions determines what to suggest based on SQL context.
// fullText is the entire editor content (used for parsing FROM/JOIN tables).
// textAtCursor is the text up to the cursor (used for context detection).
func (c *Completer) computeSuggestions(fullText, textAtCursor, word string) []string {
	// Dot notation: alias.col or table.col — resolve to specific table.
	if dotIdx := strings.LastIndex(word, "."); dotIdx >= 0 {
		qualifier := word[:dotIdx]
		colPrefix := word[dotIdx+1:]
		return c.suggestColumnsFor(fullText, qualifier, colPrefix)
	}

	ctx := detectContext(textAtCursor)
	switch ctx {
	case suggestTables:
		return c.schema.SuggestTables(word)
	case suggestColumns:
		return c.suggestColumnsFromQuery(fullText, word)
	default:
		return nil
	}
}

// suggestColumnsFor handles alias.prefix or table.prefix — resolves the
// qualifier to a real table name (via alias or direct match) and returns
// matching columns.
func (c *Completer) suggestColumnsFor(text, qualifier, colPrefix string) []string {
	tableName := c.resolveTableName(text, qualifier)
	if tableName == "" {
		return nil
	}

	allCols := c.schema.SuggestColumns(tableName)
	if colPrefix == "" {
		// Just typed "alias." — show all columns with the qualifier prefix.
		result := make([]string, len(allCols))
		for i, col := range allCols {
			result[i] = qualifier + "." + col
		}
		return result
	}

	upper := strings.ToUpper(colPrefix)
	var matches []string
	for _, col := range allCols {
		if strings.HasPrefix(strings.ToUpper(col), upper) {
			matches = append(matches, qualifier+"."+col)
		}
	}
	return matches
}

// resolveTableName maps a qualifier (could be a table name or alias) to the
// actual table name by parsing the query's FROM/JOIN clauses.
func (c *Completer) resolveTableName(text, qualifier string) string {
	refs := parseTableRefs(text)
	lq := strings.ToLower(qualifier)

	// First check aliases.
	for _, ref := range refs {
		if ref.alias != "" && strings.ToLower(ref.alias) == lq {
			return ref.table
		}
	}
	// Then check direct table names.
	for _, ref := range refs {
		if strings.ToLower(ref.table) == lq {
			return ref.table
		}
	}
	// Fall back to schema lookup (maybe it's a table not yet in the query).
	cols := c.schema.SuggestColumns(qualifier)
	if cols != nil {
		return qualifier
	}
	return ""
}

// suggestColumnsFromQuery suggests columns from all tables referenced in the
// query (FROM + JOINs), filtered by the given prefix.
func (c *Completer) suggestColumnsFromQuery(text, prefix string) []string {
	refs := parseTableRefs(text)

	if len(refs) == 0 {
		// No tables in query yet — suggest from all tables.
		return c.schema.SuggestAllColumns(prefix)
	}

	upper := strings.ToUpper(prefix)
	seen := make(map[string]bool)
	var matches []string

	for _, ref := range refs {
		cols := c.schema.SuggestColumns(ref.table)
		for _, col := range cols {
			if strings.HasPrefix(strings.ToUpper(col), upper) && !seen[col] {
				seen[col] = true
				matches = append(matches, col)
			}
		}
	}

	return matches
}

func (c *Completer) insertCompletion(completion, currentWord string) {
	c.inserting = true
	defer func() { c.inserting = false }()

	text := c.editor.Text
	runes := []rune(text)
	off := cursorOffset(text, c.editor.CursorRow, c.editor.CursorColumn)
	if off > len(runes) {
		off = len(runes)
	}

	// The current word ends at the cursor. Find where it starts.
	wordStart := off - len([]rune(currentWord))
	if wordStart < 0 {
		wordStart = 0
	}

	// Replace the word at cursor with the completion.
	newText := string(runes[:wordStart]) + completion + string(runes[off:])
	c.editor.SetText(newText)

	// Move cursor to end of inserted completion.
	insertEnd := wordStart + len([]rune(completion))
	newLines := strings.Split(newText, "\n")
	row, remaining := 0, insertEnd
	for row < len(newLines) {
		lineLen := len([]rune(newLines[row]))
		if remaining <= lineLen {
			break
		}
		remaining -= lineLen + 1 // +1 for newline
		row++
	}
	c.editor.CursorRow = row
	c.editor.CursorColumn = remaining
	c.editor.Refresh()

	c.Dismiss()
}

// SelectNext moves the selection down one item.
func (c *Completer) SelectNext() {
	if len(c.suggestions) == 0 {
		return
	}
	c.selected++
	if c.selected >= len(c.suggestions) {
		c.selected = 0
	}
	c.list.Select(c.selected)
}

// SelectPrevious moves the selection up one item.
func (c *Completer) SelectPrevious() {
	if len(c.suggestions) == 0 {
		return
	}
	c.selected--
	if c.selected < 0 {
		c.selected = len(c.suggestions) - 1
	}
	c.list.Select(c.selected)
}

// AcceptSelected inserts the currently selected suggestion.
func (c *Completer) AcceptSelected() bool {
	if c.selected < 0 || c.selected >= len(c.suggestions) {
		return false
	}
	c.insertCompletion(c.suggestions[c.selected], c.currentWord)
	return true
}

// Dismiss hides the suggestion list.
func (c *Completer) Dismiss() {
	if !c.visible {
		return
	}
	c.holder.Objects = nil
	c.holder.Refresh()
	c.visible = false
	c.currentWord = ""
	c.suggestions = nil
	c.selected = -1
}

func (c *Completer) show() {
	c.list.Refresh()

	rows := len(c.suggestions)
	if rows > 6 {
		rows = 6
	}
	height := float32(rows)*theme.TextSize()*2.2 + theme.Padding()*2
	if height < 60 {
		height = 60
	}

	c.bg.Resize(fyne.NewSize(0, height))
	listWithBg := container.NewStack(c.bg, c.list)
	sized := container.New(&fixedHeightLayout{height: height}, listWithBg)

	c.holder.Objects = []fyne.CanvasObject{sized}
	c.holder.Refresh()
	c.visible = true
}

type fixedHeightLayout struct {
	height float32
}

func (l *fixedHeightLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, l.height)
}

func (l *fixedHeightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(size.Width, l.height))
	}
}
