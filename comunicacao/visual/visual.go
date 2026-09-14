package visual

import (
	"fmt"
	"strings"
)

const (
	Reset       = "\033[0m"
	Negrito     = "\033[1m"
	Vermelho    = "\033[38;5;203m"
	Verde       = "\033[38;5;48m"
	Amarelo     = "\033[38;5;221m"
	VerdeSuave  = "\033[38;5;114m"
	CinzaClaro  = "\033[38;5;250m"
	CinzaBorda  = "\033[38;5;240m"
	TextoBranco = "\033[38;5;253m"
)

// ExibirCabecalho exibe um banner com cantos arredondados
func ExibirCabecalho(titulo string) {
	fmt.Println()
	runasTitulo := []rune(strings.ToUpper(titulo))
	tamanhoTitulo := len(runasTitulo)

	larguraCaixa := tamanhoTitulo + 8
	if larguraCaixa < 52 {
		larguraCaixa = 52
	}

	bordaTopo := "╭" + strings.Repeat("─", larguraCaixa) + "╮"
	fmt.Println(CinzaClaro + Negrito + bordaTopo + Reset)

	espacosEsq := (larguraCaixa - tamanhoTitulo) / 2
	espacosDir := larguraCaixa - tamanhoTitulo - espacosEsq
	linhaTexto := "│" + strings.Repeat(" ", espacosEsq) + string(runasTitulo) + strings.Repeat(" ", espacosDir) + "│"
	fmt.Println(CinzaClaro + Negrito + linhaTexto + Reset)

	bordaBaixo := "╰" + strings.Repeat("─", larguraCaixa) + "╯"
	fmt.Println(CinzaClaro + Negrito + bordaBaixo + Reset)
	fmt.Println()
}

// MensagemSucesso padroniza avisos de sucesso
func MensagemSucesso(msg string) {
	fmt.Println(Verde + Negrito + " [SUCESSO] " + Reset + TextoBranco + msg + Reset)
}

// MensagemErro padroniza avisos de erro
func MensagemErro(msg string) {
	fmt.Println(Vermelho + Negrito + " ✖ [ERRO] " + Reset + TextoBranco + msg + Reset)
}

// MensagemAviso padroniza alertas
func MensagemAviso(msg string) {
	fmt.Println(Amarelo + Negrito + " ⚠ [AVISO] " + Reset + TextoBranco + msg + Reset)
}

// LinhaDivisoria imprime uma linha pontilhada
func LinhaDivisoria() {
	fmt.Println(CinzaBorda + "┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄" + Reset)
}

// TabelaCabecalho exibe o cabeçalho da tabela
func TabelaCabecalho(colunas []string, larguras []int) {
	linhaBordaEstilo(larguras, "top")
	var sb strings.Builder
	sb.WriteString(CinzaBorda + "│" + Reset)
	for i, col := range colunas {
		colTruncada := truncarTexto(col, larguras[i])
		formato := fmt.Sprintf(" %s%%-%ds%s %s│%s", VerdeSuave+Negrito, larguras[i], Reset, CinzaBorda, Reset)
		sb.WriteString(fmt.Sprintf(formato, colTruncada))
	}
	fmt.Println(sb.String())
	linhaBordaEstilo(larguras, "middle")
}

// TabelaLinha exibe uma linha de dados com espaçamento
func TabelaLinha(valores []string, larguras []int) {
	var sb strings.Builder
	sb.WriteString(CinzaBorda + "│" + Reset)
	for i, val := range valores {
		valFormatado := truncarTexto(val, larguras[i])
		formato := fmt.Sprintf(" %s%%-%ds%s %s│%s", TextoBranco, larguras[i], Reset, CinzaBorda, Reset)
		sb.WriteString(fmt.Sprintf(formato, valFormatado))
	}
	fmt.Println(sb.String())
}

// TabelaRodape fecha a tabela
func TabelaRodape(larguras []int) {
	linhaBordaEstilo(larguras, "bottom")
}

// Função auxiliar para desenhar as bordas da tabela
func linhaBordaEstilo(larguras []int, posicao string) {
	var esq, meio, dir string
	switch posicao {
	case "top":
		esq, meio, dir = "┌", "┬", "┐"
	case "middle":
		esq, meio, dir = "├", "┼", "┤"
	case "bottom":
		esq, meio, dir = "└", "┴", "┘"
	}

	var sb strings.Builder
	sb.WriteString(esq)
	for i, l := range larguras {
		sb.WriteString(strings.Repeat("─", l+2))
		if i < len(larguras)-1 {
			sb.WriteString(meio)
		}
	}
	sb.WriteString(dir)
	fmt.Println(CinzaBorda + sb.String() + Reset)
}

// Função auxiliar para truncar textos
func truncarTexto(texto string, larguraMaxima int) string {
	runas := []rune(texto)
	if len(runas) <= larguraMaxima {
		return texto
	}

	if larguraMaxima > 3 {
		return string(runas[:larguraMaxima-3]) + "..."
	}

	if larguraMaxima > 0 && len(runas) >= larguraMaxima {
		return string(runas[:larguraMaxima])
	}

	return ""
}
