package testes

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func logEtapa(t *testing.T, etapa string, msg string, args ...any) {
	if len(args) > 0 {
		t.Logf("[%s] "+msg, append([]any{etapa}, args...)...)
	} else {
		t.Logf("[%s] %s", etapa, msg)
	}
}

// silenciarStdout redireciona temporariamente os.Stdout para "/dev/null",
// pra não imprimir os fmt.Printf soltos dentro dos pacotes de serviço
func silenciarStdout(t *testing.T) func() {
	original := os.Stdout
	nulo, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Logf("[aviso] não foi possível silenciar stdout: %v", err)
		return func() {}
	}
	os.Stdout = nulo
	return func() {
		os.Stdout = original
		_ = nulo.Close()
	}
}

type itemResumo struct {
	Rotulo string
	Valor  int
	Ok     bool
}

func imprimirVisaoGeral(t *testing.T, titulo string, contexto []string, itens []itemResumo, conclusao string) {
	var sb strings.Builder

	linha := strings.Repeat("", 60)
	sb.WriteString("\n" + linha + "\n")
	sb.WriteString(" " + titulo + "\n")
	sb.WriteString(linha + "\n")

	for _, c := range contexto {
		sb.WriteString(" " + c + "\n")
	}

	sb.WriteString("\n Resultado:\n")
	for _, item := range itens {
		marca := "✘"
		if item.Ok {
			marca = "✔"
		}
		sb.WriteString(fmt.Sprintf("   %s %-4d %s\n", marca, item.Valor, item.Rotulo))
	}

	sb.WriteString("\n Conclusão: " + conclusao + "\n")
	sb.WriteString(linha)

	t.Log(sb.String())
}
