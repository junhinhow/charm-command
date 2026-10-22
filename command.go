// Package command fornece uma palette de comandos (estilo Ctrl+P) para
// aplicações Bubble Tea em terminal.
//
// A palette mantém uma lista de comandos, filtra por subsequência
// (case-insensitive e sem acentos) sobre título, descrição e grupo, e
// renderiza um overlay centralizado com navegação por teclado. O pacote é
// independente de outras bibliotecas do Charm Suite: o filtro é implementado
// internamente.
//
// A navegação é clampada: a seleção não dá a volta ao chegar ao primeiro ou
// ao último item. A tecla enter executa o comando selecionado e fecha a
// palette; esc fecha e limpa a busca. A saída de [Palette.View] tem exatamente
// a altura configurada e nunca ultrapassa a largura configurada.
//
// Cópias de [Palette] compartilham o mesmo estado interno. Isso permite que
// [Palette.Update], que tem receptor por valor para satisfazer [tea.Model],
// altere o estado observado pelo ponteiro devolvido por [New].
package command

import (
	"strings"

	"charm.land/bubbletea/v2"
)

const (
	// defaultWidth é a largura inicial da área de desenho.
	defaultWidth = 80
	// defaultHeight é a altura inicial da área de desenho.
	defaultHeight = 24
	// defaultMaxVisible é a quantidade máxima de itens visíveis.
	defaultMaxVisible = 8
	// maxBoxWidth é a largura máxima do box, mesmo em áreas mais largas.
	maxBoxWidth = 64
	// defaultTitle é o título inicial da palette.
	defaultTitle = "Commands"
	// defaultPlaceholder é o texto exibido quando a busca está vazia.
	defaultPlaceholder = "Type a command…"
)

// Command é uma ação oferecida na palette.
type Command struct {
	// ID identifica o comando de forma única; usado por Remove e Add.
	ID string
	// Title é o texto principal exibido e filtrado.
	Title string
	// Description é o texto secundário exibido e filtrado.
	Description string
	// Group é o nome do grupo exibido como cabeçalho.
	Group string
	// Keys são os atalhos exibidos, por exemplo []string{"ctrl+s"}.
	Keys []string
	// Run é a ação executada no enter; pode ser nil. Ela devolve um comando
	// do Bubble Tea, que é executado pela aplicação. Para encerrar o
	// programa, use func() tea.Cmd { return tea.Quit }.
	Run func() tea.Cmd
}

// paletteState concentra o estado mutável compartilhado pelas cópias de
// Palette. Os campos não são exportados para preservar a API da biblioteca.
type paletteState struct {
	commands    []Command
	query       string
	cursor      int
	open        bool
	width       int
	height      int
	title       string
	placeholder string
	maxVisible  int
	theme       Theme
}

// Palette é o modelo Bubble Tea da command palette.
//
// O valor zero é utilizável: a primeira chamada a um método de mutação
// inicializa o estado com os valores padrão. Prefira [New] para já informar
// os comandos iniciais.
type Palette struct {
	state *paletteState
}

// newState devolve um estado com os valores padrão e nenhum comando.
func newState() *paletteState {
	return &paletteState{
		width:       defaultWidth,
		height:      defaultHeight,
		title:       defaultTitle,
		placeholder: defaultPlaceholder,
		maxVisible:  defaultMaxVisible,
		theme:       DefaultTheme(),
	}
}

// ensureState inicializa o estado do valor zero, se necessário.
func (p *Palette) ensureState() *paletteState {
	if p.state == nil {
		p.state = newState()
	}
	return p.state
}

// New cria uma palette com os comandos informados.
//
// A ordem inicial dos comandos é preservada, mas [Palette.Filtered] sempre
// devolve os resultados ordenados por score e depois por título. IDs não
// vazios duplicados são substituídos pela última ocorrência, mantendo a
// posição da primeira. Comandos com ID vazio são mantidos separadamente.
func New(commands ...Command) *Palette {
	p := &Palette{state: newState()}
	p.SetCommands(commands...)
	return p
}

// Option configura uma palette durante a criação.
type Option func(*Palette)

// WithMaxVisible define a quantidade máxima de itens visíveis por página.
//
// Valores menores que 1 são tratados como 1.
func WithMaxVisible(n int) Option {
	return func(p *Palette) {
		if n < 1 {
			n = 1
		}
		p.ensureState().maxVisible = n
	}
}

// WithTitle define o título exibido no topo da palette.
//
// Uma string vazia esconde o título.
func WithTitle(s string) Option {
	return func(p *Palette) {
		p.ensureState().title = s
	}
}

// WithTheme substitui o tema padrão pelo tema informado.
//
// Para manter todos os estilos padrão e alterar apenas alguns campos, parta
// de [DefaultTheme] e modifique a cópia.
func WithTheme(t Theme) Option {
	return func(p *Palette) {
		p.ensureState().theme = t
	}
}

// SetCommands substitui a lista de comandos e reinicia a seleção.
//
// IDs não vazios duplicados são substituídos pela última ocorrência, mantendo
// a posição da primeira. Comandos com ID vazio são mantidos separadamente. A
// busca atual é mantida.
func (p *Palette) SetCommands(cmds ...Command) {
	st := p.ensureState()
	st.commands = normalizeCommands(cmds)
	st.cursor = 0
}

// Commands devolve uma cópia dos comandos na ordem de inserção.
//
// Alterar a fatia devolvida não afeta a palette; as fatias de Keys também são
// copiadas.
func (p *Palette) Commands() []Command {
	st := p.ensureState()
	return cloneCommands(st.commands)
}

// Add acrescenta comandos à palette.
//
// Se o ID não vazio já existir, o comando anterior é substituído na mesma
// posição. Comandos com ID vazio são sempre acrescentados. A seleção é apenas
// limitada ao novo tamanho.
func (p *Palette) Add(cmds ...Command) {
	st := p.ensureState()
	for _, cmd := range cmds {
		cmd = cloneCommand(cmd)
		replaced := false
		if cmd.ID != "" {
			for i := range st.commands {
				if st.commands[i].ID == cmd.ID {
					st.commands[i] = cmd
					replaced = true
					break
				}
			}
		}
		if !replaced {
			st.commands = append(st.commands, cmd)
		}
	}
	st.clampCursor()
}

// Remove remove o comando com o ID informado.
//
// Devolve true quando algum comando foi removido. A seleção é limitada ao
// novo tamanho da lista.
func (p *Palette) Remove(id string) bool {
	st := p.ensureState()
	for i := range st.commands {
		if st.commands[i].ID != id {
			continue
		}
		st.commands = append(st.commands[:i], st.commands[i+1:]...)
		st.clampCursor()
		return true
	}
	return false
}

// Open abre a palette e devolve o comando do Bubble Tea associado.
//
// O comando devolvido é sempre nil; a assinatura existe para permitir o uso
// direto em retornos de Update.
func (p *Palette) Open() tea.Cmd {
	p.ensureState().open = true
	return nil
}

// Close fecha a palette sem alterar a busca nem a seleção.
func (p *Palette) Close() {
	p.ensureState().open = false
}

// Toggle alterna entre aberta e fechada e devolve o comando associado.
//
// O comando devolvido é sempre nil.
func (p *Palette) Toggle() tea.Cmd {
	st := p.ensureState()
	st.open = !st.open
	return nil
}

// IsOpen informa se a palette está aberta.
func (p *Palette) IsOpen() bool {
	return p.ensureState().open
}

// SetQuery substitui o texto da busca e reinicia a seleção no primeiro item.
//
// A query é preservada como informada; espaços nas extremidades são ignorados
// apenas durante o filtro.
func (p *Palette) SetQuery(q string) {
	st := p.ensureState()
	if st.query == q {
		return
	}
	st.query = q
	st.cursor = 0
}

// Query devolve o texto atual da busca.
func (p *Palette) Query() string {
	return p.ensureState().query
}

// Filtered devolve os comandos que correspondem à busca atual.
//
// O filtro é uma subsequência case-insensitive e sem acentos sobre título,
// descrição e grupo. O resultado é ordenado por score decrescente e, em caso
// de empate, por título (folded) em ordem crescente. A ordenação é estável,
// portanto comandos com o mesmo score e título mantêm a ordem de inserção.
func (p *Palette) Filtered() []Command {
	st := p.ensureState()
	return filterCommands(st.commands, st.query)
}

// Selected devolve o comando atualmente selecionado.
//
// O segundo valor é false quando a lista filtrada está vazia.
func (p *Palette) Selected() (Command, bool) {
	st := p.ensureState()
	filtered := filterCommands(st.commands, st.query)
	if st.cursor < 0 || st.cursor >= len(filtered) {
		return Command{}, false
	}
	return filtered[st.cursor], true
}

// SetSize define a área onde o overlay será desenhado.
//
// Valores negativos são tratados como zero. Com largura ou altura zero,
// [Palette.View] devolve conteúdo vazio.
func (p *Palette) SetSize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	st := p.ensureState()
	st.width = width
	st.height = height
}

// SetPlaceholder define o texto exibido quando a busca está vazia.
func (p *Palette) SetPlaceholder(s string) {
	p.ensureState().placeholder = s
}

// clampCursor garante que o cursor esteja dentro da lista filtrada atual.
func (st *paletteState) clampCursor() {
	n := len(filterCommands(st.commands, st.query))
	if n == 0 {
		st.cursor = 0
		return
	}
	if st.cursor < 0 {
		st.cursor = 0
	}
	if st.cursor >= n {
		st.cursor = n - 1
	}
}

// normalizeCommands copia os comandos e substitui IDs não vazios duplicados
// pela última ocorrência, preservando a posição da primeira.
func normalizeCommands(cmds []Command) []Command {
	out := make([]Command, 0, len(cmds))
	for _, cmd := range cmds {
		cmd = cloneCommand(cmd)
		replaced := false
		if cmd.ID != "" {
			for i := range out {
				if out[i].ID == cmd.ID {
					out[i] = cmd
					replaced = true
					break
				}
			}
		}
		if !replaced {
			out = append(out, cmd)
		}
	}
	return out
}

// cloneCommands devolve uma cópia profunda suficiente dos comandos.
func cloneCommands(cmds []Command) []Command {
	out := make([]Command, len(cmds))
	for i, cmd := range cmds {
		out[i] = cloneCommand(cmd)
	}
	return out
}

// cloneCommand copia o comando e sua fatia de atalhos.
func cloneCommand(cmd Command) Command {
	if cmd.Keys != nil {
		cmd.Keys = append([]string(nil), cmd.Keys...)
	}
	return cmd
}

// joinKeys devolve os atalhos separados por espaço.
func joinKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return strings.Join(keys, " ")
}
