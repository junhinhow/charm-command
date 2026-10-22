package command_test

import (
	"fmt"

	"charm.land/bubbletea/v2"
	"github.com/junhinhow/charm-command"
)

// ExamplePalette_Filtered mostra como consultar os comandos filtrados sem
// renderizar a palette.
func ExamplePalette_Filtered() {
	p := command.New(
		command.Command{ID: "open", Title: "Abrir arquivo", Group: "Arquivo"},
		command.Command{ID: "save", Title: "Salvar arquivo", Group: "Arquivo"},
		command.Command{ID: "quit", Title: "Sair", Group: "Sistema"},
	)
	p.SetQuery("arquivo")

	for _, cmd := range p.Filtered() {
		fmt.Println(cmd.Title)
	}

	// Output:
	// Abrir arquivo
	// Salvar arquivo
}

// ExampleNew mostra a integração mínima no Update de um modelo Bubble Tea.
func ExampleNew() {
	p := command.New(
		command.Command{
			ID:    "save",
			Title: "Salvar",
			Keys:  []string{"ctrl+s"},
			Run: func() tea.Cmd {
				return nil
			},
		},
	)
	p.SetSize(80, 24)
	p.Open()

	model, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	fmt.Println(model.(*command.Palette).IsOpen())
	fmt.Printf("%q\n", model.(*command.Palette).Query())

	// Output:
	// false
	// ""
}
