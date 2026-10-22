# charm-command

A command palette (Ctrl+P style) for [Bubble Tea](https://github.com/charmbracelet/bubbletea) terminal applications.

`charm-command` keeps a list of commands, filters them by subsequence over the
title, description and group, and renders a centered overlay with keyboard
navigation. It is part of the Charm Suite and has no dependency on the other
libraries in the suite: the matcher is implemented internally.

> Not to be confused with [`charm-palette`](https://github.com/junhinhow/charm-palette),
> which is a color palette renderer.

## Install

```bash
go get github.com/junhinhow/charm-command
```

## Usage

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
			Title:       "Save file",
			Description: "Write the current buffer to disk",
			Group:       "File",
			Keys:        []string{"ctrl+s"},
			Run: func() tea.Cmd {
				return func() tea.Msg { return savedMsg{} }
			},
		},
		command.Command{
			ID:    "quit",
			Title: "Quit",
			Group: "App",
			Run: func() tea.Cmd {
				return tea.Quit
			},
		},
	)
	p.SetSize(80, 24) // replace with the terminal size
	return model{palette: p, body: "Press ctrl+p to open the command palette."}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.palette.SetSize(msg.Width, msg.Height)

	case tea.KeyPressMsg:
		// Handle global keys before forwarding anything to the palette.
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
		m.body = "File saved."
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

The palette is a normal `tea.Model`. `Update` has a value receiver to satisfy
the interface, but all copies of a `Palette` share the same internal state, so
the pointer returned by `New` always observes the latest query and selection.

### Overlay composition

`SetSize` defines the whole drawing area. `View` returns exactly that many
lines, centers the box, and fills the remaining area with spaces. When the
palette is open, render it instead of the background content, or compose the
two with `lipgloss` layers.

## Keys

| Key | Action |
| --- | --- |
| `↑`, `ctrl+p` | previous item (clamped) |
| `↓`, `ctrl+n` | next item (clamped) |
| `pgup`, `pgdown` | previous / next page |
| `home`, `end` | first / last item |
| `enter` | run the selected command and close |
| `esc` | close and clear the query |
| `backspace` | delete the last query character |
| `ctrl+u` | clear the query |
| printable text | append to the query |

Navigation is clamped, not circular: the selection stops at the first and last
items.

## API

| Symbol | Description |
| --- | --- |
| `Command` | A command: `ID`, `Title`, `Description`, `Group`, `Keys`, `Run`. |
| `New(commands ...Command) *Palette` | Creates a palette with the given commands. |
| `(*Palette) SetCommands`, `Add`, `Remove`, `Commands` | Manage the command list. Non-empty duplicate IDs replace the existing entry. |
| `(*Palette) Open`, `Close`, `Toggle`, `IsOpen` | Control visibility. |
| `(*Palette) SetQuery`, `Query`, `Filtered`, `Selected` | Inspect and change the filter and selection. |
| `(*Palette) SetSize`, `SetPlaceholder` | Configure the overlay area and the empty-query hint. |
| `(Palette) Init`, `Update`, `View` | Bubble Tea model implementation. |
| `Option`, `WithMaxVisible`, `WithTitle`, `WithTheme` | Construction options. |
| `Theme`, `DefaultTheme()` | Lipgloss styles used by the renderer. |

## Design notes

- **Filtering.** The matcher is a case-insensitive, accent-insensitive
  subsequence search over `Title`, `Description` and `Group`. Results are
  sorted by score (title weight 3, group weight 2, description weight 1) and
  then by folded title. Sorting is stable, so commands with identical scores
  and titles keep their insertion order. An empty query returns every command
  sorted by title.
- **No `charm-fuzzy`.** The matcher is deliberately self-contained. It does
  not support typo tolerance or match highlighting.
- **Accents.** Precomposed Latin-1 and Latin Extended-A letters are folded to
  their ASCII base. Combining marks are not normalized; a query typed with
  decomposed accents may not match a precomposed title.
- **Groups.** A group header is printed whenever the group differs from the
  previous visible command. Because results are sorted by score and title,
  groups may interleave.
- **Query lifecycle.** `enter` closes but keeps the query; `esc` closes and
  clears it. `Close` and `Toggle` do not clear the query.
- **Sizing.** The box is at most 64 columns wide. Descriptions are hidden below
  48 columns, shortcuts below 30, and the border and padding are dropped when
  the area is too small. A zero width or height produces an empty view.
- **Defaults.** A new palette is 80x24 with up to 8 visible items, title
  `Commands` and placeholder `Type a command…`.
- **Determinism.** Rendering never iterates maps. The tests assert that 100
  consecutive renders are byte-identical.
- **Scope.** No mouse support, no asynchronous command loading, and no paste
  handling (`tea.PasteMsg` is ignored). Add those in the host application if
  needed.

## License

MIT
