package command

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// semANSI remove os códigos de escape ANSI do texto.
func semANSI(s string) string {
	return ansi.Strip(s)
}

// larguraMaxima devolve a maior largura de exibição entre as linhas.
func larguraMaxima(s string) int {
	maior := 0
	for _, linha := range strings.Split(s, "\n") {
		if w := lipgloss.Width(linha); w > maior {
			maior = w
		}
	}
	return maior
}

// tecla monta uma mensagem de tecla imprimível.
func tecla(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: text, Code: []rune(text)[0]}
}

// teclaCtrl monta uma mensagem de tecla com Ctrl.
func teclaCtrl(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl}
}

// comandoSimples cria um comando de teste.
func comandoSimples(id, titulo string) Command {
	return Command{ID: id, Title: titulo}
}

// TestImplementsTeaModel garante que valor e ponteiro satisfazem tea.Model.
func TestImplementsTeaModel(t *testing.T) {
	var _ tea.Model = Palette{}
	var _ tea.Model = (*Palette)(nil)
}

// TestNew e TestCommands verificam a construção e a cópia defensiva.
func TestNew(t *testing.T) {
	p := New(
		comandoSimples("a", "Abrir"),
		comandoSimples("b", "Salvar"),
	)
	if p.IsOpen() {
		t.Fatal("palette nova deveria estar fechada")
	}
	if p.Query() != "" {
		t.Fatal("palette nova deveria ter query vazia")
	}
	if got := len(p.Commands()); got != 2 {
		t.Fatalf("esperados 2 comandos, obteve %d", got)
	}

	cmds := p.Commands()
	cmds[0].Title = "alterado"
	cmds[0].Keys = append(cmds[0].Keys, "x")
	if got := p.Commands()[0].Title; got != "Abrir" {
		t.Errorf("Commands devolveu fatia interna: %q", got)
	}
}

// TestOpenCloseToggle cobre o ciclo de vida e a View vazia.
func TestOpenCloseToggle(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	if got := p.View().Content; got != "" {
		t.Fatalf("View fechada deveria ser vazia, obteve %q", got)
	}
	if cmd := p.Open(); cmd != nil {
		t.Error("Open deveria devolver nil")
	}
	if !p.IsOpen() {
		t.Fatal("palette deveria estar aberta")
	}
	if got := p.View().Content; got == "" {
		t.Fatal("View aberta não deveria ser vazia")
	}

	p.Close()
	if p.IsOpen() {
		t.Fatal("palette deveria estar fechada")
	}
	if got := p.View().Content; got != "" {
		t.Fatalf("View fechada deveria ser vazia, obteve %q", got)
	}

	if cmd := p.Toggle(); cmd != nil {
		t.Error("Toggle deveria devolver nil")
	}
	if !p.IsOpen() {
		t.Fatal("Toggle deveria abrir")
	}
	p.Toggle()
	if p.IsOpen() {
		t.Fatal("Toggle deveria fechar")
	}
}

// TestSetQueryFilterTitle cobre busca por título.
func TestSetQueryFilterTitle(t *testing.T) {
	p := New(
		comandoSimples("open", "Abrir arquivo"),
		comandoSimples("save", "Salvar arquivo"),
		comandoSimples("quit", "Sair"),
	)
	p.SetQuery("salvar")
	got := p.Filtered()
	if len(got) != 1 || got[0].ID != "save" {
		t.Fatalf("filtro por título falhou: %+v", got)
	}
}

// TestFilterDescription cobre busca por descrição.
func TestFilterDescription(t *testing.T) {
	p := New(
		Command{ID: "save", Title: "Salvar", Description: "Grava o arquivo"},
		Command{ID: "open", Title: "Abrir", Description: "Lê o arquivo"},
	)
	p.SetQuery("grava")
	got := p.Filtered()
	if len(got) != 1 || got[0].ID != "save" {
		t.Fatalf("filtro por descrição falhou: %+v", got)
	}
}

// TestFilterGroup cobre busca por grupo.
func TestFilterGroup(t *testing.T) {
	p := New(
		Command{ID: "save", Title: "Salvar", Group: "Arquivo"},
		Command{ID: "settings", Title: "Configurações", Group: "Sistema"},
	)
	p.SetQuery("sistema")
	got := p.Filtered()
	if len(got) != 1 || got[0].ID != "settings" {
		t.Fatalf("filtro por grupo falhou: %+v", got)
	}
}

// TestFilterCaseAndAccent cobre case-insensitive e remoção de acentos nos
// dois sentidos.
func TestFilterCaseAndAccent(t *testing.T) {
	p := New(
		comandoSimples("acao", "Ação rápida"),
		comandoSimples("config", "Configurações"),
	)

	p.SetQuery("ACAO")
	if got := p.Filtered(); len(got) != 1 || got[0].ID != "acao" {
		t.Fatalf("query sem acento deveria casar com título acentuado: %+v", got)
	}

	p.SetQuery("ação")
	if got := p.Filtered(); len(got) != 1 || got[0].ID != "acao" {
		t.Fatalf("query acentuada deveria casar: %+v", got)
	}

	p.SetQuery("CONFIGURACOES")
	if got := p.Filtered(); len(got) != 1 || got[0].ID != "config" {
		t.Fatalf("query sem acento deveria casar com Configurações: %+v", got)
	}

	p.SetQuery("configurações")
	if got := p.Filtered(); len(got) != 1 || got[0].ID != "config" {
		t.Fatalf("query acentuada deveria casar com Configurações: %+v", got)
	}
}

// TestFoldString cobre a redução de acentos e o mapeamento de letras latinas
// comuns.
func TestFoldString(t *testing.T) {
	if got := foldString("AÇÃO Café Æsop"); got != "acao cafe asop" {
		t.Fatalf("foldString = %q", got)
	}
}

// TestFilterOrdering verifica score e desempate por título.
func TestFilterOrdering(t *testing.T) {
	p := New(
		comandoSimples("prefix", "Abrir arquivo"),
		comandoSimples("exact", "Abrir"),
		comandoSimples("other", "Barra"),
	)
	p.SetQuery("abrir")
	got := p.Filtered()
	if len(got) != 2 {
		t.Fatalf("esperados 2 resultados, obteve %d", len(got))
	}
	if got[0].ID != "exact" {
		t.Fatalf("casamento exato deveria vir primeiro: %+v", got)
	}
	if got[1].ID != "prefix" {
		t.Fatalf("prefixo deveria vir depois: %+v", got)
	}

	// Query vazia devolve todos, ordenados por título folded.
	p.SetQuery("")
	got = p.Filtered()
	if len(got) != 3 {
		t.Fatalf("esperados 3 comandos, obteve %d", len(got))
	}
	want := []string{"exact", "prefix", "other"}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("ordem vazia[%d] = %q, esperava %q", i, got[i].ID, id)
		}
	}
}

// TestFilterNoResults cobre query sem correspondência.
func TestFilterNoResults(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	p.SetQuery("zzz")
	if got := p.Filtered(); len(got) != 0 {
		t.Fatalf("esperado resultado vazio, obteve %+v", got)
	}
	if _, ok := p.Selected(); ok {
		t.Error("Selected deveria devolver false sem resultados")
	}
}

// TestNavigationClamped cobre setas, ctrl+p/ctrl+n, home/end e paginação.
func TestNavigationClamped(t *testing.T) {
	p := New(
		comandoSimples("a", "A"),
		comandoSimples("b", "B"),
		comandoSimples("c", "C"),
	)
	p.SetSize(40, 10)
	p.Open()

	selectedID := func() string {
		cmd, ok := p.Selected()
		if !ok {
			t.Fatal("Selected devolveu false")
		}
		return cmd.ID
	}

	if got := selectedID(); got != "a" {
		t.Fatalf("seleção inicial = %q", got)
	}
	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyDown},
		{Code: tea.KeyDown},
		{Code: tea.KeyDown},
		{Code: tea.KeyDown},
	} {
		p.Update(msg)
	}
	if got := selectedID(); got != "c" {
		t.Fatalf("fim deveria clampar em c, obteve %q", got)
	}
	for i := 0; i < 4; i++ {
		p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if got := selectedID(); got != "a" {
		t.Fatalf("início deveria clampar em a, obteve %q", got)
	}

	p.Update(teclaCtrl('n'))
	if got := selectedID(); got != "b" {
		t.Fatalf("ctrl+n = %q", got)
	}
	p.Update(teclaCtrl('p'))
	if got := selectedID(); got != "a" {
		t.Fatalf("ctrl+p = %q", got)
	}

	p.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if got := selectedID(); got != "c" {
		t.Fatalf("end = %q", got)
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if got := selectedID(); got != "a" {
		t.Fatalf("home = %q", got)
	}

	p.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if got := selectedID(); got != "c" {
		t.Fatalf("pgdown deveria ir para o fim, obteve %q", got)
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if got := selectedID(); got != "a" {
		t.Fatalf("pgup deveria voltar ao início, obteve %q", got)
	}
}

// TestNavigationResetsOnQuery cobre o reinício da seleção ao filtrar.
func TestNavigationResetsOnQuery(t *testing.T) {
	p := New(
		comandoSimples("a", "Alfa"),
		comandoSimples("b", "Beta"),
		comandoSimples("c", "Gama"),
	)
	p.Open()
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p.SetQuery("b")
	cmd, ok := p.Selected()
	if !ok || cmd.ID != "b" {
		t.Fatalf("query deveria reiniciar a seleção em Beta, obteve %+v ok=%v", cmd, ok)
	}
}

// TestEnterRunsCommand verifica que enter executa o comando certo e fecha.
func TestEnterRunsCommand(t *testing.T) {
	var contadorA, contadorB int
	p := New(
		Command{ID: "a", Title: "Alfa", Run: func() tea.Cmd {
			contadorA++
			return nil
		}},
		Command{ID: "b", Title: "Beta", Run: func() tea.Cmd {
			contadorB++
			return func() tea.Msg { return "executou" }
		}},
	)
	p.SetSize(40, 10)
	p.Open()
	p.SetQuery("beta")

	model, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if _, ok := model.(*Palette); !ok {
		t.Fatalf("Update devolveu %T, esperava *Palette", model)
	}
	if cmd == nil {
		t.Fatal("enter deveria devolver o comando do Run")
	}
	if msg := cmd(); msg != "executou" {
		t.Errorf("Run devolveu mensagem inesperada: %v", msg)
	}
	if contadorA != 0 || contadorB != 1 {
		t.Fatalf("comando errado executado: A=%d B=%d", contadorA, contadorB)
	}
	if p.IsOpen() {
		t.Fatal("enter deveria fechar a palette")
	}
}

// TestEnterWithoutRun cobre comando sem ação.
func TestEnterWithoutRun(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	p.Open()
	if _, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatalf("comando sem Run deveria devolver nil, obteve %v", cmd)
	}
	if p.IsOpen() {
		t.Fatal("enter deveria fechar mesmo sem Run")
	}
}

// TestEnterWithoutSelection cobre enter com filtro vazio.
func TestEnterWithoutSelection(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	p.Open()
	p.SetQuery("zzz")
	if _, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatalf("esperado nil, obteve %v", cmd)
	}
	if p.IsOpen() {
		t.Fatal("palette deveria fechar")
	}
}

// TestEscClearsQuery cobre esc e edição básica da query.
func TestEscClearsQuery(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	p.Open()

	_, _ = p.Update(tecla("a"))
	p.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := p.Query(); got != "" {
		t.Fatalf("backspace deixou %q", got)
	}

	_, _ = p.Update(tecla("b"))
	_, _ = p.Update(tecla("c"))
	if got := p.Query(); got != "bc" {
		t.Fatalf("query = %q, esperava bc", got)
	}

	p.Update(teclaCtrl('u'))
	if got := p.Query(); got != "" {
		t.Fatalf("ctrl+u deixou %q", got)
	}

	_, _ = p.Update(tecla("x"))
	p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if p.IsOpen() {
		t.Fatal("esc deveria fechar")
	}
	if got := p.Query(); got != "" {
		t.Fatalf("esc deveria limpar a query, obteve %q", got)
	}
}

// TestUpdateIgnoresClosed e TestUpdateIgnoresOtherMsgs cobrem mensagens
// irrelevantes.
func TestUpdateIgnoresClosed(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	_, _ = p.Update(tecla("a"))
	if got := p.Query(); got != "" {
		t.Fatalf("palette fechada não deveria aceitar texto, obteve %q", got)
	}
	model, cmd := p.Update(struct{}{})
	if _, ok := model.(*Palette); !ok {
		t.Fatalf("Update devolveu %T", model)
	}
	if cmd != nil {
		t.Errorf("mensagem desconhecida deveria devolver nil, obteve %v", cmd)
	}
}

// TestAddRemoveSetCommands cobre a gestão da lista.
func TestAddRemoveSetCommands(t *testing.T) {
	p := New(comandoSimples("a", "A"), comandoSimples("b", "B"))
	p.Add(comandoSimples("c", "C"))
	if got := len(p.Commands()); got != 3 {
		t.Fatalf("Add: esperados 3, obteve %d", got)
	}

	p.Add(comandoSimples("a", "A2"))
	cmds := p.Commands()
	if len(cmds) != 3 {
		t.Fatalf("duplicata não deveria crescer a lista: %d", len(cmds))
	}
	if cmds[0].ID != "a" || cmds[0].Title != "A2" {
		t.Fatalf("duplicata deveria substituir na mesma posição: %+v", cmds)
	}

	if !p.Remove("a") {
		t.Fatal("Remove deveria devolver true")
	}
	if p.Remove("a") {
		t.Fatal("Remove de ID inexistente deveria devolver false")
	}
	if got := len(p.Commands()); got != 2 {
		t.Fatalf("após Remove: esperados 2, obteve %d", got)
	}

	p.SetCommands(
		comandoSimples("x", "X"),
		comandoSimples("x", "X2"),
		comandoSimples("y", "Y"),
	)
	cmds = p.Commands()
	if len(cmds) != 2 || cmds[0].ID != "x" || cmds[0].Title != "X2" {
		t.Fatalf("SetCommands com duplicata: %+v", cmds)
	}

	p.SetCommands(Command{Title: "Sem ID 1"}, Command{Title: "Sem ID 2"})
	if got := len(p.Commands()); got != 2 {
		t.Fatalf("comandos sem ID não deveriam ser deduplicados: %d", got)
	}
}

// TestCustomTheme garante que um tema mínimo também renderiza dentro dos
// limites.
func TestCustomTheme(t *testing.T) {
	p := New(
		Command{ID: "a", Title: "Abrir", Description: "Descrição", Group: "Grupo", Keys: []string{"ctrl+o"}},
	)
	WithTheme(Theme{})(p)
	p.SetSize(40, 10)
	p.Open()
	view := p.View().Content
	if len(strings.Split(view, "\n")) != 10 {
		t.Fatalf("esperadas 10 linhas, obteve %d", len(strings.Split(view, "\n")))
	}
	if w := larguraMaxima(view); w > 40 {
		t.Fatalf("largura %d excede 40", w)
	}
}

// TestRemoveClampsCursor cobre a seleção após remoções.
func TestRemoveClampsCursor(t *testing.T) {
	p := New(
		comandoSimples("a", "A"),
		comandoSimples("b", "B"),
		comandoSimples("c", "C"),
	)
	p.Open()
	p.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if !p.Remove("c") {
		t.Fatal("Remove falhou")
	}
	cmd, ok := p.Selected()
	if !ok || cmd.ID != "b" {
		t.Fatalf("cursor deveria ser limitado a b, obteve %+v ok=%v", cmd, ok)
	}
}

// TestEmptyList cobre navegação e renderização sem comandos.
func TestEmptyList(t *testing.T) {
	p := New()
	p.SetSize(40, 10)
	p.Open()
	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyDown},
		{Code: tea.KeyUp},
		{Code: tea.KeyPgDown},
		{Code: tea.KeyHome},
		{Code: tea.KeyEnd},
	} {
		p.Update(msg)
	}
	if _, ok := p.Selected(); ok {
		t.Error("Selected deveria devolver false")
	}
	view := semANSI(p.View().Content)
	if !strings.Contains(view, "No commands found.") {
		t.Errorf("View deveria informar lista vazia:\n%s", view)
	}
}

// TestZeroValuePalette cobre o uso do valor zero.
func TestZeroValuePalette(t *testing.T) {
	var p Palette
	p.Open()
	p.SetSize(40, 10)
	p.Add(comandoSimples("a", "Abrir"))
	if !p.IsOpen() {
		t.Fatal("palette zero deveria abrir")
	}
	if got := p.View().Content; got == "" {
		t.Fatal("View deveria renderizar")
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if p.IsOpen() {
		t.Fatal("esc deveria fechar")
	}
}

// TestWithOptions cobre as opções públicas.
func TestWithOptions(t *testing.T) {
	p := New(
		comandoSimples("a", "Alfa"),
		comandoSimples("b", "Beta"),
		comandoSimples("c", "Gama"),
		comandoSimples("d", "Delta"),
	)
	WithMaxVisible(2)(p)
	WithTitle("Minha Palette")(p)
	WithTheme(DefaultTheme())(p)
	p.SetSize(60, 12)
	p.Open()
	view := semANSI(p.View().Content)
	if !strings.Contains(view, "Minha Palette") {
		t.Errorf("título não apareceu:\n%s", view)
	}
	if !strings.Contains(view, "Alfa") || !strings.Contains(view, "Beta") {
		t.Errorf("dois primeiros itens deveriam aparecer:\n%s", view)
	}
	if strings.Contains(view, "Gama") || strings.Contains(view, "Delta") {
		t.Errorf("itens além de WithMaxVisible(2) não deveriam aparecer:\n%s", view)
	}
}

// TestPlaceholder cobre o texto de placeholder.
func TestPlaceholder(t *testing.T) {
	p := New(comandoSimples("a", "Abrir"))
	p.SetSize(60, 12)
	p.SetPlaceholder("Buscar comandos…")
	p.Open()
	view := semANSI(p.View().Content)
	if !strings.Contains(view, "Buscar comandos…") {
		t.Errorf("placeholder não apareceu:\n%s", view)
	}
}

// TestGroups cobre cabeçalhos de grupo apenas quando o grupo muda.
func TestGroups(t *testing.T) {
	p := New(
		Command{ID: "a", Title: "Alfa", Group: "Arquivo"},
		Command{ID: "b", Title: "Beta", Group: "Arquivo"},
		Command{ID: "c", Title: "Gama", Group: "Sistema"},
	)
	p.SetSize(60, 14)
	p.Open()
	view := semANSI(p.View().Content)
	if got := strings.Count(view, "Arquivo"); got != 1 {
		t.Errorf("cabeçalho Arquivo deveria aparecer 1 vez, apareceu %d:\n%s", got, view)
	}
	if got := strings.Count(view, "Sistema"); got != 1 {
		t.Errorf("cabeçalho Sistema deveria aparecer 1 vez, apareceu %d:\n%s", got, view)
	}

	p.SetCommands(
		Command{ID: "a", Title: "Alfa", Group: "Arquivo"},
		Command{ID: "b", Title: "Beta", Group: "Sistema"},
		Command{ID: "c", Title: "Gama", Group: "Arquivo"},
	)
	view = semANSI(p.View().Content)
	if got := strings.Count(view, "Arquivo"); got != 2 {
		t.Errorf("Arquivo deveria repetir ao mudar de grupo, apareceu %d:\n%s", got, view)
	}
}

// TestViewOverlayDimensions cobre as dimensões exigidas do overlay.
func TestViewOverlayDimensions(t *testing.T) {
	cmds := []Command{
		{ID: "save", Title: "Salvar", Description: "Grava o arquivo", Group: "Arquivo", Keys: []string{"ctrl+s"}},
		{ID: "open", Title: "Abrir", Description: "Abre o arquivo", Group: "Arquivo", Keys: []string{"ctrl+o"}},
		{ID: "quit", Title: "Sair", Description: "Encerra o app", Group: "Sistema"},
	}

	p := New(cmds...)
	p.Open()
	p.SetSize(80, 24)
	view := p.View().Content
	linhas := strings.Split(view, "\n")
	if len(linhas) != 24 {
		t.Fatalf("80x24: esperadas 24 linhas, obteve %d", len(linhas))
	}
	if w := larguraMaxima(view); w > 80 {
		t.Fatalf("80x24: largura %d excede 80", w)
	}

	p.SetSize(10, 5)
	view = p.View().Content
	linhas = strings.Split(view, "\n")
	if len(linhas) != 5 {
		t.Fatalf("10x5: esperadas 5 linhas, obteve %d", len(linhas))
	}
	if w := larguraMaxima(view); w > 10 {
		t.Fatalf("10x5: largura %d excede 10", w)
	}
	limpo := semANSI(view)
	if strings.Contains(limpo, "ctrl+s") {
		t.Errorf("atalho deveria ser escondido em 10x5:\n%s", limpo)
	}
	if strings.Contains(limpo, "Grava o arquivo") {
		t.Errorf("descrição deveria ser escondida em 10x5:\n%s", limpo)
	}
}

// TestViewTinySizes garante que larguras e alturas minúsculas não panicam.
func TestViewTinySizes(t *testing.T) {
	p := New(
		Command{ID: "a", Title: "Abrir", Description: "Descrição longa", Group: "Grupo", Keys: []string{"ctrl+o"}},
	)
	p.Open()
	for _, size := range [][2]int{
		{0, 0}, {0, 5}, {5, 0}, {-1, -1},
		{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {10, 5}, {20, 3},
	} {
		p.SetSize(size[0], size[1])
		view := p.View().Content
		if size[0] <= 0 || size[1] <= 0 {
			if view != "" {
				t.Errorf("%dx%d deveria devolver vazio", size[0], size[1])
			}
			continue
		}
		linhas := strings.Split(view, "\n")
		if len(linhas) != size[1] {
			t.Errorf("%dx%d: esperadas %d linhas, obteve %d", size[0], size[1], size[1], len(linhas))
		}
		for i, linha := range linhas {
			if w := lipgloss.Width(linha); w != size[0] {
				t.Errorf("%dx%d: linha %d tem largura %d", size[0], size[1], i, w)
			}
		}
	}
}

// TestViewDeterministic garante 100 renders idênticos.
func TestViewDeterministic(t *testing.T) {
	p := New(
		Command{ID: "a", Title: "Alfa", Description: "Primeiro", Group: "Um"},
		Command{ID: "b", Title: "Beta", Description: "Segundo", Group: "Dois"},
		Command{ID: "c", Title: "Gama", Description: "Terceiro", Group: "Um"},
		Command{ID: "d", Title: "Delta", Description: "Quarto", Group: "Dois"},
	)
	p.SetSize(80, 24)
	p.Open()
	p.SetQuery("a")
	primeiro := p.View().Content
	for i := 0; i < 100; i++ {
		if got := p.View().Content; got != primeiro {
			t.Fatalf("render %d divergiu", i)
		}
	}
}

// TestScrollingKeepsSelectionVisible cobre a janela de itens.
func TestScrollingKeepsSelectionVisible(t *testing.T) {
	cmds := make([]Command, 20)
	for i := range cmds {
		cmds[i] = comandoSimples(fmt.Sprintf("c%02d", i), fmt.Sprintf("Comando %02d", i))
	}
	p := New(cmds...)
	WithMaxVisible(3)(p)
	p.SetSize(60, 12)
	p.Open()
	p.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	view := semANSI(p.View().Content)
	if !strings.Contains(view, "Comando 19") {
		t.Errorf("último item deveria estar visível:\n%s", view)
	}
	if strings.Count(view, "Comando") != 3 {
		t.Errorf("WithMaxVisible(3) deveria exibir 3 itens:\n%s", view)
	}
}
