package markdown

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Theme is how the renderer draws: the UI's colours and glyphs, so this
// package stays free of UI code.
type Theme struct {
	Dark                                         bool
	Text, Subtle, Muted, Accent, Accent2, Rule   color.Color
	Surface, SurfaceHi, Green, Red, Yellow, Blue color.Color

	Bullets           []string // by depth
	CheckOn, CheckOff string
	Quote             string // the bar before quoted lines

	// Heading, when set, may draw a heading as an image: it gets the level,
	// the plain text and the width, and returns the lines to show, or nil
	// to draw the heading as text.
	Heading func(level int, text string, width int) []string
}

var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

// RenderTheme renders markdown text (no images: Split them out first) to
// styled lines for a column of width cells.
func RenderTheme(src string, width int, th Theme) string {
	width = max(width, 16)
	source := []byte(src)
	doc := md.Parser().Parse(text.NewReader(source))
	r := &renderer{src: source, th: th}
	lines := r.blocks(doc, width, 0)
	return strings.Join(lines, "\n")
}

type renderer struct {
	src []byte
	th  Theme
}

func (r *renderer) fg(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

// blocks renders a container's children, a blank line between blocks
// (none between the items of a tight list's paragraphs).
func (r *renderer) blocks(n ast.Node, width, depth int) []string {
	var out []string
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		lines := r.block(c, width, depth)
		if len(lines) == 0 {
			continue
		}
		if len(out) > 0 && !tightSibling(c) {
			out = append(out, "")
		}
		out = append(out, lines...)
	}
	return out
}

// tightSibling is true for a block inside a tight list item, which follows
// the previous one without a blank line.
func tightSibling(n ast.Node) bool {
	if _, ok := n.Parent().(*ast.ListItem); ok {
		if l, ok := n.Parent().Parent().(*ast.List); ok {
			return l.IsTight
		}
	}
	return false
}

func (r *renderer) block(n ast.Node, width, depth int) []string {
	switch n := n.(type) {
	case *ast.Heading:
		return r.heading(n, width)
	case *ast.Paragraph, *ast.TextBlock:
		return wrap(r.inline(n, inlineStyle{}), width)
	case *ast.List:
		return r.list(n, width, depth)
	case *ast.FencedCodeBlock:
		return r.code(string(n.Language(r.src)), r.lines(n), width)
	case *ast.CodeBlock:
		return r.code("", r.lines(n), width)
	case *ast.Blockquote:
		bar := r.fg(r.th.Accent2).Render(r.th.Quote) + " "
		inner := r.blocks(n, width-2, depth)
		for i, l := range inner {
			inner[i] = bar + l
		}
		return inner
	case *ast.ThematicBreak:
		return []string{r.fg(r.th.Rule).Render(strings.Repeat("─", width))}
	case *ast.HTMLBlock:
		var out []string
		for _, l := range r.lines(n) {
			out = append(out, r.fg(r.th.Muted).Render(ansi.Truncate(l, width, "…")))
		}
		return out
	case *extast.Table:
		return r.table(n, width)
	}
	// Anything else: its children.
	return r.blocks(n, width, depth)
}

func (r *renderer) lines(n ast.Node) []string {
	var out []string
	ls := n.Lines()
	for i := 0; i < ls.Len(); i++ {
		seg := ls.At(i)
		out = append(out, strings.TrimRight(string(seg.Value(r.src)), "\n"))
	}
	return out
}

// heading draws the text in the accent colour with a rule under it for
// levels 1 and 2. Theme.Heading can draw levels 1 and 2 bigger, as kitty
// images (Rex supports neither kitty's text sizing, OSC 66, nor VT100
// double-height lines; tested on the iOS client).
func (r *renderer) heading(n *ast.Heading, width int) []string {
	st := inlineStyle{bold: true}
	switch n.Level {
	case 1:
		st.fg = r.th.Accent
	case 2:
		st.fg = r.th.Accent2
	case 3:
		st.fg = r.th.Text
	default:
		st.fg = r.th.Subtle
	}
	var out []string
	if r.th.Heading != nil && n.Level <= 2 {
		out = r.th.Heading(n.Level, plainText(n, r.src), width)
	}
	if out == nil {
		out = wrap(r.inline(n, st), width)
	}
	switch n.Level {
	case 1:
		out = append(out, r.fg(r.th.Accent).Render(strings.Repeat("━", width)))
	case 2:
		out = append(out, r.fg(r.th.Rule).Render(strings.Repeat("─", width)))
	}
	return out
}

// list draws items with a bullet by depth, a number, or a checkbox; the
// item's lines hang under the first.
func (r *renderer) list(n *ast.List, width, depth int) []string {
	var out []string
	num := n.Start
	if num == 0 {
		num = 1
	}
	// Ordered markers line up on the widest number.
	numW := len(strconv.Itoa(num + n.ChildCount() - 1))
	first := true
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		var marker string
		if n.IsOrdered() {
			s := strconv.Itoa(num) + "."
			marker = r.fg(r.th.Subtle).Render(strings.Repeat(" ", numW+1-len(s)) + s)
			num++
		} else {
			marker = r.fg(r.th.Accent).Render(r.th.Bullets[min(depth, len(r.th.Bullets)-1)])
		}
		if box, ok := taskBox(item); ok {
			if box.IsChecked {
				marker = r.fg(r.th.Green).Render(r.th.CheckOn)
			} else {
				marker = r.fg(r.th.Muted).Render(r.th.CheckOff)
			}
		}
		pad := strings.Repeat(" ", ansi.StringWidth(marker)+1)
		body := r.blocks(item, width-len(pad), depth+1)
		if !first && !n.IsTight {
			out = append(out, "")
		}
		first = false
		for i, l := range body {
			switch {
			case i == 0:
				out = append(out, marker+" "+l)
			case l == "":
				out = append(out, "")
			default:
				out = append(out, pad+l)
			}
		}
		if len(body) == 0 {
			out = append(out, marker)
		}
	}
	return out
}

// taskBox finds a GFM task checkbox at the start of a list item.
func taskBox(item ast.Node) (*extast.TaskCheckBox, bool) {
	c := item.FirstChild()
	if c == nil || c.FirstChild() == nil {
		return nil, false
	}
	box, ok := c.FirstChild().(*extast.TaskCheckBox)
	return box, ok
}

// code draws a shaded panel with the language at the top right and the
// lines highlighted by Chroma. Long lines wrap.
func (r *renderer) code(lang string, lines []string, width int) []string {
	bg := r.th.Surface
	panel := lipgloss.NewStyle().Background(bg)
	inner := max(width-2, 8)
	var out []string
	label := ""
	if lang != "" {
		label = r.fg(r.th.Muted).Background(bg).Render(lang + " ")
	}
	out = append(out, panel.Render(strings.Repeat(" ", width-ansi.StringWidth(label)))+label)
	for _, l := range highlight(lang, strings.Join(lines, "\n"), r.th, bg) {
		for _, part := range strings.Split(ansi.Hardwrap(l, inner, true), "\n") {
			fill := inner - ansi.StringWidth(part)
			out = append(out, panel.Render(" ")+part+panel.Render(strings.Repeat(" ", max(fill, 0)+1)))
		}
	}
	out = append(out, panel.Render(strings.Repeat(" ", width)))
	return out
}

// highlight colours code by its language (guessed when unnamed), one
// string per line, on bg.
func highlight(lang, code string, th Theme, bg color.Color) []string {
	lexer := lexers.Get(lang)
	if lexer == nil && lang == "" {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	name := "catppuccin-mocha"
	if !th.Dark {
		name = "catppuccin-latte"
	}
	style := chromastyles.Get(name)
	it, err := lexer.Tokenise(nil, code)
	plain := lipgloss.NewStyle().Foreground(th.Text).Background(bg)
	if err != nil {
		var out []string
		for _, l := range strings.Split(code, "\n") {
			out = append(out, plain.Render(l))
		}
		return out
	}
	var out []string
	var line strings.Builder
	for _, tok := range it.Tokens() {
		e := style.Get(tok.Type)
		st := plain
		if e.Colour.IsSet() {
			st = st.Foreground(lipgloss.Color(e.Colour.String()))
		}
		if e.Bold == chroma.Yes {
			st = st.Bold(true)
		}
		if e.Italic == chroma.Yes {
			st = st.Italic(true)
		}
		parts := strings.Split(tok.Value, "\n")
		for i, p := range parts {
			if i > 0 {
				out = append(out, line.String())
				line.Reset()
			}
			if p != "" {
				line.WriteString(st.Render(strings.ReplaceAll(p, "\t", "    ")))
			}
		}
	}
	if line.Len() > 0 {
		out = append(out, line.String())
	}
	return out
}

// table draws a rounded grid when it fits, and a stack of cards (one per
// row, "header: value") when it doesn't.
func (r *renderer) table(n *extast.Table, width int) []string {
	var header []string
	var rows [][]string
	var aligns []extast.Alignment
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for c := row.FirstChild(); c != nil; c = c.NextSibling() {
			st := inlineStyle{}
			if _, ok := row.(*extast.TableHeader); ok {
				st.bold, st.fg = true, r.th.Text
				if tc, ok := c.(*extast.TableCell); ok {
					aligns = append(aligns, tc.Alignment)
				}
			}
			cells = append(cells, r.inline(c, st))
		}
		if _, ok := row.(*extast.TableHeader); ok {
			header = cells
		} else {
			rows = append(rows, cells)
		}
	}
	cols := len(header)
	widths := make([]int, cols)
	for _, row := range append([][]string{header}, rows...) {
		for i := 0; i < cols && i < len(row); i++ {
			widths[i] = max(widths[i], ansi.StringWidth(row[i]))
		}
	}
	total := 1
	for _, w := range widths {
		total += w + 3
	}
	border := r.fg(r.th.Rule)
	if total <= width {
		line := func(l, m, rr string) string {
			var b strings.Builder
			b.WriteString(l)
			for i, w := range widths {
				if i > 0 {
					b.WriteString(m)
				}
				b.WriteString(strings.Repeat("─", w+2))
			}
			return border.Render(b.String() + rr)
		}
		cellsLine := func(cells []string) string {
			var b strings.Builder
			b.WriteString(border.Render("│"))
			for i, w := range widths {
				c := ""
				if i < len(cells) {
					c = cells[i]
				}
				pad := w - ansi.StringWidth(c)
				switch {
				case i < len(aligns) && aligns[i] == extast.AlignRight:
					c = strings.Repeat(" ", pad) + c
				case i < len(aligns) && aligns[i] == extast.AlignCenter:
					c = strings.Repeat(" ", pad/2) + c + strings.Repeat(" ", pad-pad/2)
				default:
					c += strings.Repeat(" ", pad)
				}
				b.WriteString(" " + c + " " + border.Render("│"))
			}
			return b.String()
		}
		out := []string{line("╭", "┬", "╮"), cellsLine(header), line("├", "┼", "┤")}
		for _, row := range rows {
			out = append(out, cellsLine(row))
		}
		return append(out, line("╰", "┴", "╯"))
	}
	// Cards: each row's cells under their headers, with a bar on the left.
	var out []string
	bar := r.fg(r.th.Accent2).Render(r.th.Quote) + " "
	for k, row := range rows {
		if k > 0 {
			out = append(out, "")
		}
		for i, c := range row {
			if i < len(header) {
				for _, l := range wrap(r.fg(r.th.Muted).Render(ansi.Strip(header[i])), width-2) {
					out = append(out, bar+l)
				}
			}
			for _, l := range wrap(c, width-4) {
				out = append(out, bar+"  "+l)
			}
		}
	}
	return out
}

// inlineStyle is the style in effect for a run of inline text.
type inlineStyle struct {
	bold, italic, strike, underline bool
	code                            bool
	fg                              color.Color
}

func (r *renderer) render(s string, st inlineStyle) string {
	ls := lipgloss.NewStyle()
	if st.fg != nil {
		ls = ls.Foreground(st.fg)
	} else {
		ls = ls.Foreground(r.th.Text)
	}
	ls = ls.Bold(st.bold).Italic(st.italic).Strikethrough(st.strike).Underline(st.underline)
	if st.code {
		ls = ls.Background(r.th.SurfaceHi)
	}
	return ls.Render(s)
}

// inline renders a block's inline content.
func (r *renderer) inline(n ast.Node, st inlineStyle) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b.WriteString(r.inlineNode(c, st))
	}
	return b.String()
}

func (r *renderer) inlineNode(n ast.Node, st inlineStyle) string {
	switch n := n.(type) {
	case *ast.Text:
		s := r.render(textValue(n, r.src), st)
		switch {
		case n.HardLineBreak():
			s += "\n"
		case n.SoftLineBreak():
			s += " "
		}
		return s
	case *ast.String:
		return r.render(string(n.Value), st)
	case *ast.CodeSpan:
		var b strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				b.Write(t.Segment.Value(r.src))
			}
		}
		cs := st
		cs.code, cs.bold = true, false
		if cs.fg == nil {
			cs.fg = r.th.Accent
		}
		return r.render(" "+b.String()+" ", cs)
	case *ast.Emphasis:
		es := st
		if n.Level >= 2 {
			es.bold = true
		} else {
			es.italic = true
		}
		return r.inline(n, es)
	case *extast.Strikethrough:
		ss := st
		ss.strike = true
		ss.fg = r.th.Muted
		return r.inline(n, ss)
	case *ast.Link:
		ls := st
		ls.underline, ls.fg = true, r.th.Blue
		return ansi.SetHyperlink(string(n.Destination)) + r.inline(n, ls) + ansi.ResetHyperlink()
	case *ast.AutoLink:
		url := string(n.URL(r.src))
		ls := st
		ls.underline, ls.fg = true, r.th.Blue
		return ansi.SetHyperlink(url) + r.render(string(n.Label(r.src)), ls) + ansi.ResetHyperlink()
	case *ast.Image:
		ms := st
		ms.fg = r.th.Muted
		return r.render("["+string(n.Text(r.src))+"]", ms)
	case *ast.RawHTML:
		var b strings.Builder
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			b.Write(seg.Value(r.src))
		}
		ms := st
		ms.fg = r.th.Muted
		return r.render(b.String(), ms)
	case *extast.TaskCheckBox:
		return "" // drawn as the list marker
	}
	return r.inline(n, st)
}

// wrap breaks styled text into lines of width cells, keeping hard breaks.
func wrap(s string, width int) []string {
	var out []string
	for _, para := range strings.Split(strings.TrimRight(s, " "), "\n") {
		w := ansi.Wrap(para, max(width, 8), " -")
		for _, l := range strings.Split(w, "\n") {
			out = append(out, strings.TrimRight(l, " "))
		}
	}
	return out
}

// textValue is a text node's text with backslash escapes and entities
// resolved, as goldmark's HTML writer does: `\~7k` is `~7k`, `&amp;` is
// `&`. Raw text (code) keeps them.
func textValue(t *ast.Text, src []byte) string {
	v := t.Segment.Value(src)
	if t.IsRaw() {
		return string(v)
	}
	v = util.UnescapePunctuations(v)
	v = util.ResolveNumericReferences(v)
	v = util.ResolveEntityNames(v)
	return string(v)
}

// plainText is a node's text without markup.
func plainText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.WriteString(textValue(t, src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

// Heading is a level 1 or 2 heading's plain text.
type Heading struct {
	Level int
	Text  string
}

// Headings lists the level 1 and 2 headings of markdown text, so the UI
// can prepare their images before drawing.
func Headings(src string) []Heading {
	source := []byte(src)
	doc := md.Parser().Parse(text.NewReader(source))
	var out []Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering && h.Level <= 2 {
			out = append(out, Heading{Level: h.Level, Text: plainText(h, source)})
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return out
}
