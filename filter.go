package command

import (
	"sort"
	"strings"
	"unicode"
)

// candidate é um comando já pontuado, pronto para a ordenação.
type candidate struct {
	command  Command
	score    int
	titleKey string
}

// filterCommands aplica o filtro e devolve os comandos ordenados.
//
// Quando a query está vazia (ou contém apenas espaços), todos os comandos são
// devolvidos com score zero; nesse caso a ordenação acontece apenas pelo
// título. A função nunca itera mapas, portanto a saída é determinística.
func filterCommands(cmds []Command, query string) []Command {
	pattern := foldString(strings.TrimSpace(query))
	candidates := make([]candidate, 0, len(cmds))

	for _, cmd := range cmds {
		score := 0
		if pattern != "" {
			var ok bool
			score, ok = commandScore(pattern, cmd)
			if !ok {
				continue
			}
		}
		candidates = append(candidates, candidate{
			command:  cloneCommand(cmd),
			score:    score,
			titleKey: foldString(cmd.Title),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].titleKey != candidates[j].titleKey {
			return candidates[i].titleKey < candidates[j].titleKey
		}
		return false
	})

	out := make([]Command, len(candidates))
	for i, c := range candidates {
		out[i] = c.command
	}
	return out
}

// commandScore calcula o score de um comando para a query já folded.
//
// O título tem peso 3, o grupo peso 2 e a descrição peso 1. Um bônus de 15 é
// somado quando a query casa com o título e também com um campo secundário.
func commandScore(pattern string, cmd Command) (int, bool) {
	total := 0
	matched := false

	titleScore, titleOK := matchScore(pattern, cmd.Title)
	if titleOK {
		total += titleScore * 3
		matched = true
	}
	groupScore, groupOK := matchScore(pattern, cmd.Group)
	if groupOK {
		total += groupScore * 2
		matched = true
	}
	descriptionScore, descriptionOK := matchScore(pattern, cmd.Description)
	if descriptionOK {
		total += descriptionScore
		matched = true
	}
	if titleOK && (groupOK || descriptionOK) {
		total += 15
	}

	return total, matched
}

// matchScore verifica se pattern é subsequência de text e devolve um score.
//
// pattern deve estar previamente folded; text é folded aqui. O score cresce
// quando o casamento é exato, é um prefixo ou começa em uma fronteira de
// palavra, quando os caracteres são consecutivos e quando o primeiro
// casamento aparece cedo. Lacunas descontam no máximo 30 pontos, de modo que
// o score nunca fica negativo.
func matchScore(pattern, text string) (int, bool) {
	if pattern == "" {
		return 0, true
	}

	p := []rune(pattern)
	t := []rune(foldString(text))
	if len(p) > len(t) {
		return 0, false
	}

	positions := make([]int, 0, len(p))
	index := 0
	for _, want := range p {
		found := -1
		for index < len(t) {
			if t[index] == want {
				found = index
				index++
				break
			}
			index++
		}
		if found < 0 {
			return 0, false
		}
		positions = append(positions, found)
	}

	score := 100
	switch {
	case len(p) == len(t) && equalRunes(p, t):
		score += 700
	case len(p) < len(t) && equalRunes(p, t[:len(p)]):
		score += 400
	case isWordBoundary(t, positions[0]):
		score += 200
	}

	if positions[0] < 40 {
		score += 40 - positions[0]
	}

	gaps := positions[len(positions)-1] - positions[0] - (len(positions) - 1)
	if gaps > 30 {
		gaps = 30
	}
	score -= gaps

	for i := 1; i < len(positions); i++ {
		switch {
		case positions[i] == positions[i-1]+1:
			score += 15
		case isWordBoundary(t, positions[i]):
			score += 8
		}
	}

	return score, true
}

// equalRunes compara duas fatias de runas.
func equalRunes(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isWordBoundary informa se a posição i começa uma palavra.
func isWordBoundary(text []rune, i int) bool {
	if i <= 0 {
		return true
	}
	switch text[i-1] {
	case ' ', '\t', '-', '_', '.', '/', ':', '\\', '(', '[':
		return true
	default:
		return false
	}
}

// foldString converte o texto para minúsculas e remove acentos latinos.
//
// A transformação preserva a quantidade de runas, o que mantém os índices
// usados pelo cálculo de score. Apenas letras latinas comuns são mapeadas;
// outros alfabetos passam por unicode.ToLower.
func foldString(s string) string {
	return strings.Map(func(r rune) rune {
		r = unicode.ToLower(r)
		switch r {
		case 'à', 'á', 'â', 'ã', 'ä', 'å', 'ā', 'ă', 'ą':
			return 'a'
		case 'ç', 'ć', 'ĉ', 'ċ', 'č':
			return 'c'
		case 'ď', 'đ':
			return 'd'
		case 'è', 'é', 'ê', 'ë', 'ē', 'ĕ', 'ė', 'ę', 'ě':
			return 'e'
		case 'ĝ', 'ğ', 'ġ', 'ģ':
			return 'g'
		case 'ĥ', 'ħ':
			return 'h'
		case 'ì', 'í', 'î', 'ï', 'ĩ', 'ī', 'ĭ', 'į', 'ı':
			return 'i'
		case 'ĵ':
			return 'j'
		case 'ķ':
			return 'k'
		case 'ĺ', 'ļ', 'ľ', 'ł':
			return 'l'
		case 'ñ', 'ń', 'ņ', 'ň':
			return 'n'
		case 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'ō', 'ŏ', 'ő':
			return 'o'
		case 'ŕ', 'ŗ', 'ř':
			return 'r'
		case 'ś', 'ŝ', 'ş', 'š':
			return 's'
		case 'ţ', 'ť', 'ŧ':
			return 't'
		case 'ù', 'ú', 'û', 'ü', 'ũ', 'ū', 'ŭ', 'ů', 'ű', 'ų':
			return 'u'
		case 'ŵ':
			return 'w'
		case 'ý', 'ÿ', 'ŷ':
			return 'y'
		case 'ź', 'ż', 'ž':
			return 'z'
		default:
			return r
		}
	}, s)
}
