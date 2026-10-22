package command

import (
	"charm.land/bubbletea/v2"
)

// Init implementa [tea.Model].
//
// A palette não executa nenhuma ação ao ser inicializada.
func (p Palette) Init() tea.Cmd {
	return nil
}

// Update implementa [tea.Model] e processa as mensagens recebidas.
//
// Somente mensagens de tecla são tratadas, e apenas enquanto a palette está
// aberta; qualquer outra mensagem é ignorada. O modelo devolvido é sempre um
// *Palette que compartilha o estado do receptor, portanto a aplicação pode
// continuar usando o ponteiro original.
//
// Teclas tratadas:
//
//	↑ / ctrl+p    item anterior (clampado)
//	↓ / ctrl+n    próximo item (clampado)
//	pgup          página anterior
//	pgdown        próxima página
//	home / end    primeiro / último item
//	enter         executa Run do item selecionado e fecha
//	esc           fecha e limpa a busca
//	backspace     apaga o último caractere da busca
//	ctrl+u        limpa a busca
//	texto         acrescenta à busca
//
// A paginação usa [Palette.WithMaxVisible] como tamanho de página. Enter não
// limpa a busca; apenas esc o faz.
func (p Palette) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	st := p.ensureState()
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok || !st.open {
		return &p, nil
	}
	return &p, p.handleKey(keyMsg)
}

// handleKey processa uma tecla com a palette aberta.
func (p *Palette) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	st := p.ensureState()
	key := msg.Key()

	switch key.Code {
	case tea.KeyEscape:
		st.open = false
		st.query = ""
		st.cursor = 0
		return nil
	case tea.KeyEnter, tea.KeyKpEnter:
		selected, ok := p.Selected()
		st.open = false
		if ok && selected.Run != nil {
			return selected.Run()
		}
		return nil
	}

	ctrl := key.Mod.Contains(tea.ModCtrl)
	if ctrl {
		switch key.Code {
		case 'p', 'P':
			p.move(-1)
			return nil
		case 'n', 'N':
			p.move(1)
			return nil
		case 'u', 'U':
			p.resetQuery()
			return nil
		}
	}

	switch key.Code {
	case tea.KeyUp:
		p.move(-1)
		return nil
	case tea.KeyDown:
		p.move(1)
		return nil
	case tea.KeyPgUp:
		p.page(-1)
		return nil
	case tea.KeyPgDown:
		p.page(1)
		return nil
	case tea.KeyHome:
		st.cursor = 0
		return nil
	case tea.KeyEnd:
		p.moveToEnd()
		return nil
	case tea.KeyBackspace:
		p.backspace()
		return nil
	}

	if key.Text != "" && !ctrl && !key.Mod.Contains(tea.ModAlt) {
		st.query += key.Text
		st.cursor = 0
	}

	return nil
}

// move desloca a seleção em delta itens, clampando nas extremidades.
func (p *Palette) move(delta int) {
	st := p.ensureState()
	n := len(filterCommands(st.commands, st.query))
	if n == 0 {
		st.cursor = 0
		return
	}
	next := st.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= n {
		next = n - 1
	}
	st.cursor = next
}

// page desloca a seleção em uma página, clampando nas extremidades.
func (p *Palette) page(direction int) {
	step := p.ensureState().maxVisible
	if step < 1 {
		step = 1
	}
	p.move(direction * step)
}

// moveToEnd seleciona o último item filtrado.
func (p *Palette) moveToEnd() {
	st := p.ensureState()
	n := len(filterCommands(st.commands, st.query))
	if n == 0 {
		st.cursor = 0
		return
	}
	st.cursor = n - 1
}

// resetQuery limpa a busca e reinicia a seleção.
func (p *Palette) resetQuery() {
	st := p.ensureState()
	st.query = ""
	st.cursor = 0
}

// backspace apaga o último caractere da busca e reinicia a seleção.
func (p *Palette) backspace() {
	st := p.ensureState()
	runes := []rune(st.query)
	if len(runes) == 0 {
		return
	}
	st.query = string(runes[:len(runes)-1])
	st.cursor = 0
}
