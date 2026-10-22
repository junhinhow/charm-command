package command

import (
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// rowKind identifica o tipo de uma linha da lista.
type rowKind int

const (
	// rowGroup é um cabeçalho de grupo.
	rowGroup rowKind = iota
	// rowCommand é uma linha de comando.
	rowCommand
)

// listRow é uma linha já expandida da lista, incluindo cabeçalhos de grupo.
type listRow struct {
	kind  rowKind
	group string
	index int
}

// View implementa [tea.Model] e devolve o overlay da palette.
//
// Com a palette fechada, ou com largura/altura zero, o conteúdo é vazio.
// Quando aberta, o resultado tem exatamente a altura configurada por
// [Palette.SetSize] e nenhuma linha ultrapassa a largura configurada. O box é
// centralizado e o restante da área é preenchido com espaços.
func (p Palette) View() tea.View {
	if p.state == nil || !p.state.open {
		return tea.View{}
	}
	return tea.NewView(p.overlay())
}

// overlay monta a tela completa com o box centralizado.
func (p *Palette) overlay() string {
	st := p.ensureState()
	if !st.open || st.width <= 0 || st.height <= 0 {
		return ""
	}

	box := st.renderBox(st.width, st.height)
	if box == "" {
		return ""
	}

	lines := strings.Split(box, "\n")
	if len(lines) > st.height {
		lines = lines[:st.height]
	}

	boxWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > boxWidth {
			boxWidth = w
		}
	}
	leftPad := 0
	if st.width > boxWidth {
		leftPad = (st.width - boxWidth) / 2
	}

	blank := strings.Repeat(" ", st.width)
	topPad := (st.height - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}
	bottomPad := st.height - len(lines) - topPad
	if bottomPad < 0 {
		bottomPad = 0
	}

	out := make([]string, 0, st.height)
	for i := 0; i < topPad; i++ {
		out = append(out, blank)
	}
	for _, line := range lines {
		line = ansi.Truncate(line, st.width, "")
		available := st.width - leftPad
		if available < 0 {
			available = 0
		}
		rightPad := available - lipgloss.Width(line)
		if rightPad < 0 {
			rightPad = 0
		}
		out = append(out, strings.Repeat(" ", leftPad)+line+strings.Repeat(" ", rightPad))
	}
	for i := 0; i < bottomPad; i++ {
		out = append(out, blank)
	}

	return strings.Join(out, "\n")
}

// renderBox desenha a moldura e o conteúdo, respeitando as dimensões máximas.
//
// Quando a área é pequena demais para a moldura, o padding e depois a própria
// borda são descartados, em vez de estourar a largura ou a altura.
func (st *paletteState) renderBox(maxWidth, maxHeight int) string {
	if maxWidth <= 0 || maxHeight <= 0 {
		return ""
	}
	if maxWidth > maxBoxWidth {
		maxWidth = maxBoxWidth
	}

	border := st.theme.Border.Margin(0)
	frameWidth := border.GetHorizontalFrameSize()
	if maxWidth-frameWidth < 1 {
		border = border.Padding(0)
		frameWidth = border.GetHorizontalFrameSize()
	}
	if maxWidth-frameWidth < 1 {
		border = lipgloss.NewStyle()
		frameWidth = 0
	}

	frameHeight := border.GetVerticalFrameSize()
	if maxHeight-frameHeight < 1 {
		border = lipgloss.NewStyle()
		frameWidth = 0
		frameHeight = 0
	}

	contentWidth := maxWidth - frameWidth
	if contentWidth < 1 {
		contentWidth = 1
	}
	interiorHeight := maxHeight - frameHeight
	if interiorHeight < 1 {
		interiorHeight = 1
	}

	lines := st.contentLines(contentWidth, interiorHeight)
	if len(lines) == 0 {
		return ""
	}

	box := border.Render(strings.Join(lines, "\n"))
	return clampBlock(box, maxWidth, maxHeight)
}

// clampBlock garante que um bloco não ultrapasse a largura nem a altura.
func clampBlock(block string, width, height int) string {
	lines := strings.Split(block, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = ansi.Truncate(line, width, "")
		}
	}
	return strings.Join(lines, "\n")
}

// contentLines monta as linhas internas do box.
func (st *paletteState) contentLines(width, height int) []string {
	filtered := filterCommands(st.commands, st.query)
	base := lipgloss.NewStyle().Inherit(st.theme.Background)

	showTitle := st.title != "" && width >= 10 && height >= 5
	showSeparator := height >= 4

	capacity := height - 1 - boolToInt(showTitle) - boolToInt(showSeparator)
	if capacity < 1 {
		showSeparator = false
		capacity = height - 1 - boolToInt(showTitle)
	}
	if capacity < 1 {
		showTitle = false
		capacity = height - 1
	}
	if capacity < 0 {
		capacity = 0
	}
	if capacity > st.maxVisible {
		capacity = st.maxVisible
	}

	lines := make([]string, 0, height)
	if showTitle {
		lines = append(lines, st.renderTitle(width, base))
	}
	lines = append(lines, st.renderQuery(width, base))
	if showSeparator {
		lines = append(lines, st.renderSeparator(width, base))
	}

	if capacity > 0 {
		rows := st.buildRows(filtered)
		if len(rows) == 0 {
			lines = append(lines, st.renderEmpty(width, base))
		} else {
			start, end := st.viewport(rows, capacity)
			for _, row := range rows[start:end] {
				if row.kind == rowGroup {
					lines = append(lines, st.renderGroup(width, row.group, base))
					continue
				}
				selected := row.index == st.cursor
				lines = append(lines, st.renderItem(width, filtered[row.index], selected, base))
			}
		}
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}

// renderTitle desenha a linha de título.
func (st *paletteState) renderTitle(width int, base lipgloss.Style) string {
	style := st.theme.Title.Inherit(base)
	return fitLine(style.Render(truncateText(st.title, width)), width, base)
}

// renderQuery desenha a linha de busca, mostrando o placeholder quando vazio.
func (st *paletteState) renderQuery(width int, base lipgloss.Style) string {
	queryStyle := st.theme.Query.Inherit(base)
	textStyle := queryStyle
	prompt := "> "
	if width < 3 {
		prompt = ""
	}

	text := st.query
	if text == "" {
		text = st.placeholder
		textStyle = st.theme.Description.Inherit(base)
	}

	available := width - lipgloss.Width(prompt)
	if available < 0 {
		available = 0
	}
	if available == 0 {
		text = ""
	} else if st.query == "" {
		text = truncateText(text, available)
	} else if textWidth := lipgloss.Width(text); textWidth > available {
		text = ansi.TruncateLeft(text, textWidth-available+1, "…")
	}

	line := queryStyle.Render(prompt) + textStyle.Render(text)
	return fitLine(line, width, base)
}

// renderSeparator desenha a linha divisória entre a busca e a lista.
func (st *paletteState) renderSeparator(width int, base lipgloss.Style) string {
	style := lipgloss.NewStyle()
	if fg := st.theme.Border.GetBorderTopForeground(); fg != nil {
		style = style.Foreground(fg)
	} else {
		style = st.theme.Description
	}
	style = style.Inherit(base)
	return style.Render(strings.Repeat("─", width))
}

// renderGroup desenha um cabeçalho de grupo.
func (st *paletteState) renderGroup(width int, group string, base lipgloss.Style) string {
	style := st.theme.Group.Inherit(base)
	text := truncateText("  "+group, width)
	return fitLine(style.Render(text), width, base)
}

// renderEmpty desenha a mensagem exibida quando o filtro não encontra nada.
func (st *paletteState) renderEmpty(width int, base lipgloss.Style) string {
	style := st.theme.Description.Inherit(base)
	return fitLine(style.Render(truncateText("No commands found.", width)), width, base)
}

// renderItem desenha uma linha de comando.
func (st *paletteState) renderItem(width int, cmd Command, selected bool, base lipgloss.Style) string {
	prefix := "  "
	if selected {
		prefix = "▸ "
	}
	if width < 2 {
		prefix = ""
	}

	available := width - lipgloss.Width(prefix)
	if available < 1 {
		available = 1
	}

	label := cmd.Title
	if label == "" {
		label = cmd.ID
	}

	keys := joinKeys(cmd.Keys)
	keysWidth := lipgloss.Width(keys)
	showKeys := keys != "" && width >= 30 && keysWidth+2 <= available/2

	contentWidth := available
	if showKeys {
		contentWidth -= keysWidth + 2
	}
	if contentWidth < 1 {
		contentWidth = 1
		showKeys = false
	}

	title := truncateText(label, contentWidth)
	remaining := contentWidth - lipgloss.Width(title)
	description := ""
	if cmd.Description != "" && width >= 48 && remaining >= 8 {
		description = "  " + truncateText(cmd.Description, remaining-2)
	}

	if selected {
		body := prefix + title + description
		if showKeys {
			gap := width - lipgloss.Width(body) - keysWidth
			if gap < 1 {
				gap = 1
			}
			body += strings.Repeat(" ", gap) + keys
		}
		body = padPlain(body, width)
		style := st.theme.SelectedItem.Inherit(base)
		return clampLine(style.Render(body), width)
	}

	itemStyle := st.theme.Item.Inherit(base)
	line := itemStyle.Render(prefix) + itemStyle.Render(title)
	if description != "" {
		line += st.theme.Description.Inherit(base).Render(description)
	}
	if showKeys {
		gap := width - lipgloss.Width(line) - keysWidth
		if gap < 1 {
			gap = 1
		}
		line += base.Render(strings.Repeat(" ", gap))
		line += st.theme.Shortcut.Inherit(base).Render(keys)
	}

	return fitLine(line, width, base)
}

// buildRows expande os comandos filtrados em linhas, inserindo um cabeçalho
// sempre que o grupo muda em relação ao item anterior.
func (st *paletteState) buildRows(filtered []Command) []listRow {
	rows := make([]listRow, 0, len(filtered)+4)
	lastGroup := ""
	for i, cmd := range filtered {
		if cmd.Group != "" && cmd.Group != lastGroup {
			rows = append(rows, listRow{kind: rowGroup, group: cmd.Group})
		}
		rows = append(rows, listRow{kind: rowCommand, index: i})
		lastGroup = cmd.Group
	}
	return rows
}

// viewport calcula a janela de linhas que mantém o item selecionado visível.
func (st *paletteState) viewport(rows []listRow, capacity int) (int, int) {
	if capacity <= 0 || len(rows) == 0 {
		return 0, 0
	}

	cursorRow := 0
	for i, row := range rows {
		if row.kind == rowCommand && row.index == st.cursor {
			cursorRow = i
			break
		}
	}

	start := 0
	if cursorRow >= capacity {
		start = cursorRow - capacity + 1
	}
	if maxStart := len(rows) - capacity; start > maxStart && maxStart >= 0 {
		start = maxStart
	}
	if start < 0 {
		start = 0
	}

	end := start + capacity
	if end > len(rows) {
		end = len(rows)
	}
	return start, end
}

// truncateText corta o texto em max colunas, acrescentando reticências.
func truncateText(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	return ansi.Truncate(s, max, "…")
}

// fitLine garante exatamente width colunas, truncando ou preenchendo com
// espaços estilizados pela base.
func fitLine(line string, width int, base lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "")
	}
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += base.Render(strings.Repeat(" ", pad))
	}
	return line
}

// padPlain completa a string com espaços simples até width colunas.
func padPlain(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// clampLine trunca uma linha estilizada para no máximo width colunas.
func clampLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) > width {
		return ansi.Truncate(line, width, "")
	}
	return line
}

// boolToInt converte um booleano em 1 ou 0.
func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
