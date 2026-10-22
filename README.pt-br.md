# charm-command

Uma command palette (estilo Ctrl+P) para aplicações de terminal feitas com
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

`charm-command` mantém uma lista de comandos, filtra por subsequência sobre
título, descrição e grupo, e renderiza um overlay centralizado com navegação
por teclado. Faz parte do Charm Suite e não depende das outras bibliotecas da
suíte: o filtro é implementado internamente.

> Não confundir com [`charm-palette`](https://github.com/junhinhow/charm-palette),
> que é um renderizador de paletas de cores.

## Instalação

```bash
go get github.com/junhinhow/charm-command
```

## Uso

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/junhinhow/charm-command"
)

type savedMsg struct{}

type model struct {
	palette *command.Palette
	body    string
}

func newModel() model {
	p := command.New(
		command.Command{
			ID:          "save",
			Title:       "Salvar arquivo",
			Description: "Grava o buffer atual em disco",
			Group:       "Arquivo",
			Keys:        []string{"ctrl+s"},
			Run: func() tea.Cmd {
				return func() tea.Msg { return savedMsg{} }
			},
		},
		command.Command{
			ID:    "quit",
			Title: "Sair",
			Group: "Aplicativo",
			Run: func() tea.Cmd {
				return tea.Quit
			},
		},
	)
	p.SetSize(80, 24) // substitua pelo tamanho do terminal
	return model{palette: p, body: "Pressione ctrl+p para abrir a command palette."}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.palette.SetSize(msg.Width, msg.Height)

	case tea.KeyPressMsg:
		// Trate teclas globais antes de encaminhar qualquer coisa à palette.
		if msg.Keystroke() == "ctrl+c" {
			return m, tea.Quit
		}
		if !m.palette.IsOpen() && msg.Keystroke() == "ctrl+p" {
			return m, m.palette.Open()
		}
		if m.palette.IsOpen() {
			updated, cmd := m.palette.Update(msg)
			m.palette = updated.(*command.Palette)
			return m, cmd
		}

	case savedMsg:
		m.body = "Arquivo salvo."
	}

	return m, nil
}

func (m model) View() tea.View {
	if m.palette.IsOpen() {
		return m.palette.View()
	}
	return tea.NewView(m.body)
}

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

A palette é um `tea.Model` comum. `Update` usa receptor por valor para
satisfazer a interface, mas todas as cópias de uma `Palette` compartilham o
mesmo estado interno; assim, o ponteiro devolvido por `New` sempre observa a
busca e a seleção mais recentes.

### Composição do overlay

`SetSize` define a área total de desenho. `View` devolve exatamente essa
quantidade de linhas, centraliza o box e preenche o restante com espaços.
Quando a palette está aberta, renderize-a no lugar do conteúdo de fundo ou
combine os dois com as camadas do `lipgloss`.

## Teclas

| Tecla | Ação |
| --- | --- |
| `↑`, `ctrl+p` | item anterior (clampado) |
| `↓`, `ctrl+n` | próximo item (clampado) |
| `pgup`, `pgdown` | página anterior / próxima |
| `home`, `end` | primeiro / último item |
| `enter` | executa o comando selecionado e fecha |
| `esc` | fecha e limpa a busca |
| `backspace` | apaga o último caractere da busca |
| `ctrl+u` | limpa a busca |
| texto imprimível | acrescenta à busca |

A navegação é clampada, não circular: a seleção para no primeiro e no último
item.

## API

| Símbolo | Descrição |
| --- | --- |
| `Command` | Um comando: `ID`, `Title`, `Description`, `Group`, `Keys`, `Run`. |
| `New(commands ...Command) *Palette` | Cria uma palette com os comandos informados. |
| `(*Palette) SetCommands`, `Add`, `Remove`, `Commands` | Gerenciam a lista de comandos. IDs não vazios duplicados substituem a entrada existente. |
| `(*Palette) Open`, `Close`, `Toggle`, `IsOpen` | Controlam a visibilidade. |
| `(*Palette) SetQuery`, `Query`, `Filtered`, `Selected` | Inspecionam e alteram o filtro e a seleção. |
| `(*Palette) SetSize`, `SetPlaceholder` | Configuram a área do overlay e o texto de busca vazia. |
| `(Palette) Init`, `Update`, `View` | Implementação do modelo Bubble Tea. |
| `Option`, `WithMaxVisible`, `WithTitle`, `WithTheme` | Opções de construção. |
| `Theme`, `DefaultTheme()` | Estilos lipgloss usados pelo renderizador. |

## Notas de projeto

- **Filtro.** O matcher é uma busca por subsequência case-insensitive e sem
  acentos sobre `Title`, `Description` e `Group`. Os resultados são ordenados
  por score (peso 3 para título, 2 para grupo e 1 para descrição) e depois pelo
  título folded. A ordenação é estável: comandos com score e título idênticos
  mantêm a ordem de inserção. Uma busca vazia devolve todos os comandos
  ordenados por título.
- **Sem `charm-fuzzy`.** O matcher é deliberadamente autocontido. Não há
  tolerância a erros de digitação nem destaque dos trechos casados.
- **Acentos.** Letras latinas precompostas de Latin-1 e Latin Extended-A são
  reduzidas à base ASCII. Marcas combinantes não são normalizadas; uma busca
  digitada com acentos decompostos pode não casar com um título precomposto.
- **Grupos.** Um cabeçalho de grupo é impresso sempre que o grupo difere do
  comando visível anterior. Como os resultados são ordenados por score e
  título, grupos podem se intercalar.
- **Ciclo da busca.** `enter` fecha mas mantém a busca; `esc` fecha e limpa.
  `Close` e `Toggle` não limpam a busca.
- **Dimensionamento.** O box tem no máximo 64 colunas. Descrições são
  escondidas abaixo de 48 colunas, atalhos abaixo de 30, e a borda e o padding
  são descartados quando a área é pequena demais. Largura ou altura zero
  produzem uma view vazia.
- **Padrões.** Uma palette nova tem 80x24, até 8 itens visíveis, título
  `Commands` e placeholder `Type a command…`.
- **Determinismo.** A renderização nunca itera mapas. Os testes garantem que
  100 renders consecutivos são idênticos byte a byte.
- **Escopo.** Sem suporte a mouse, sem carregamento assíncrono de comandos e
  sem tratamento de colagem (`tea.PasteMsg` é ignorado). Adicione esses
  comportamentos na aplicação hospedeira, se necessário.

## Licença

MIT
