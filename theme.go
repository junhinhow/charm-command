package command

import "charm.land/lipgloss/v2"

// Theme agrupa os estilos usados para desenhar a palette.
//
// Os estilos são aplicados por segmento (título, query, item, descrição,
// grupo e atalho). O estilo [Theme.Background] define a base do corpo do box:
// suas cores e atributos são herdados pelos demais segmentos, enquanto
// padding, margem e borda são ignorados nessa herança. Use [DefaultTheme]
// como ponto de partida e altere apenas o que for necessário.
type Theme struct {
	// Background estilo base do corpo do box (cor de fundo e atributos).
	Background lipgloss.Style
	// Border estilo da moldura do box, incluindo padding opcional.
	Border lipgloss.Style
	// Title estilo do título exibido no topo do box.
	Title lipgloss.Style
	// Query estilo do texto digitado na busca.
	Query lipgloss.Style
	// Item estilo dos itens não selecionados.
	Item lipgloss.Style
	// SelectedItem estilo do item selecionado; o fundo cobre a linha inteira.
	SelectedItem lipgloss.Style
	// Description estilo das descrições dos comandos.
	Description lipgloss.Style
	// Group estilo dos cabeçalhos de grupo.
	Group lipgloss.Style
	// Shortcut estilo dos atalhos exibidos à direita dos itens.
	Shortcut lipgloss.Style
}

// DefaultTheme devolve o tema escuro usado por padrão.
//
// As cores foram escolhidas para manter contraste sobre o fundo #1E1E2E.
// O tema é determinístico: não consulta o terminal nem variáveis de ambiente.
func DefaultTheme() Theme {
	return Theme{
		Background: lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E2E")).
			Foreground(lipgloss.Color("#CDD6F4")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")),
		Query: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89B4FA")),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")),
		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")),
		Description: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")),
		Group: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F9E2AF")),
		Shortcut: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94E2D5")),
	}
}
