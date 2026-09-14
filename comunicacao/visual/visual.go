package visual

import (
	"fmt"
	"strings"
)

const (
	Reset    = "\033[0m"
	Negrito  = "\033[1m"
	Vermelho = "\033[31m"
	Verde    = "\033[32m"
	Amarelo  = "\033[33m"
	Azul     = "\033[34m"
	Ciano    = "\033[36m"
	Cinza    = "\033[90m"
)

// ExibirCabecalho padroniza a exibição de títulos de telas
func ExibirCabecalho(titulo string) {
	fmt.Println()
	fmt.Println(Ciano + Negrito + "==================================================" + Reset)
	espacos := (50 - len(titulo)) / 2
	if espacos < 0 {
		espacos = 0
	}
	padding := strings.Repeat(" ", espacos)
	fmt.Println(Ciano + Negrito + padding + strings.ToUpper(titulo) + Reset)
	fmt.Println(Ciano + Negrito + "==================================================" + Reset)
	fmt.Println()
}

// MensagemSucesso padroniza avisos de sucesso
func MensagemSucesso(msg string) {
	fmt.Println(Verde + "[SUCESSO] " + msg + Reset)
}

// MensagemErro padroniza avisos de erro
func MensagemErro(msg string) {
	fmt.Println(Vermelho + "[ERRO] " + msg + Reset)
}

// MensagemAviso padroniza alertas
func MensagemAviso(msg string) {
	fmt.Println(Amarelo + "[AVISO] " + msg + Reset)
}

// LinhaDivisoria imprime uma linha simples para separar blocos
func LinhaDivisoria() {
	fmt.Println(Cinza + "--------------------------------------------------" + Reset)
}

// TabelaCabecalho exibe o cabeçalho de uma tabela formatada com colunas alinhadas
func TabelaCabecalho(colunas []string, larguras []int) {
	linhaBorda(larguras)
	var sb strings.Builder
	sb.WriteString("|")
	for i, col := range colunas {
		formato := fmt.Sprintf(" %%-%ds |", larguras[i])
		sb.WriteString(fmt.Sprintf(formato, col))
	}
	fmt.Println(Ciano + Negrito + sb.String() + Reset)
	linhaBorda(larguras)
}

// TabelaLinha exibe uma linha de dados dentro da tabela formatada
func TabelaLinha(valores []string, larguras []int) {
	var sb strings.Builder
	sb.WriteString("|")
	for i, val := range valores {
		// Trunca o texto se for maior que a largura da coluna para não quebrar o layout
		valFormatado := val
		if len(val) > larguras[i] {
			valFormatado = val[:larguras[i]-3] + "..."
		}
		formato := fmt.Sprintf(" %%-%ds |", larguras[i])
		sb.WriteString(fmt.Sprintf(formato, valFormatado))
	}
	fmt.Println(sb.String())
}

// TabelaRodape fecha a estrutura da tabela
func TabelaRodape(larguras []int) {
	linhaBorda(larguras)
}

// Função auxiliar interna para desenhar as bordas horizontais da tabela
func linhaBorda(larguras []int) {
	var sb strings.Builder
	sb.WriteString("+")
	for _, l := range larguras {
		sb.WriteString(strings.Repeat("-", l+2) + "+")
	}
	fmt.Println(Cinza + sb.String() + Reset)
}
