package testes

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"Sistema-de-caronas-compartilhadas/servicos/usuarios"
)

type resultadoCriacaoConta struct {
	ID          int
	Nome        string
	Tipo        string
	Sucesso     bool
	ErroConexao error
	ErroCriacao error
}

func criarContas(t *testing.T, endereco string, totalContas int) []resultadoCriacaoConta {
	var wg sync.WaitGroup
	var mu sync.Mutex
	resultados := make([]resultadoCriacaoConta, 0, totalContas)

	logEtapa(t, "CADASTRO", "iniciando (%d)", totalContas)

	restaurar := silenciarStdout(t)

	for i := 1; i <= totalContas; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tipo := protocolo.PerfilPassageiro
			if id%2 == 0 {
				tipo = protocolo.PerfilMotorista
			}
			nome := "usuario_" + strconv.Itoa(id)

			cliente, err := conexao.ConectarTCP(endereco)
			if err != nil {
				mu.Lock()
				resultados = append(resultados, resultadoCriacaoConta{
					ID: id, Nome: nome, Tipo: tipo, ErroConexao: err,
				})
				mu.Unlock()
				return
			}
			defer cliente.Fechar()

			sucesso, err := usuarios.CadastrarCliente(cliente, nome, "senha123", tipo)

			mu.Lock()
			resultados = append(resultados, resultadoCriacaoConta{
				ID: id, Nome: nome, Tipo: tipo, Sucesso: sucesso, ErroCriacao: err,
			})
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	restaurar()

	logEtapa(t, "CADASTRO", "concluido")
	return resultados
}

func TestCriacaoContas(t *testing.T) {
	porta := obterPortaLivre(t)
	enderecoTeste := fmt.Sprintf("127.0.0.1:%d", porta)

	cmdServidor := iniciarServidorRealModular(t, enderecoTeste)

	defer func() {
		pararServidorReal(cmdServidor)
		limparAmbientePersistencia()
	}()

	const totalContas = 100
	esperadoPorTipo := totalContas / 2

	resultados := criarContas(t, enderecoTeste, totalContas)

	var passageiros, motoristas, falhas, falhaConexao int
	for _, r := range resultados {
		switch {
		case r.ErroConexao != nil:
			falhaConexao++
		case r.Sucesso && r.Tipo == protocolo.PerfilPassageiro:
			passageiros++
		case r.Sucesso && r.Tipo == protocolo.PerfilMotorista:
			motoristas++
		default:
			falhas++
		}
	}

	conclusao := "[APROVADO] todas as contas foram criadas sem conflito"
	if falhaConexao > 0 {
		conclusao = "[REPROVADO] houve falha de infraestrutura do teste (conexão)"
	} else if passageiros != esperadoPorTipo || motoristas != esperadoPorTipo || falhas > 0 {
		conclusao = "[REPROVADO] nem todas as contas esperadas foram criadas"
	}

	imprimirVisaoGeral(t,
		"VISÃO GERAL: Cadastro em massa de contas",
		[]string{
			fmt.Sprintf("Iniciado cadastro de %d contas ao mesmo tempo", totalContas),
			fmt.Sprintf("Metade como passageiro (%d), metade como motorista (%d), todos com nomes únicos", esperadoPorTipo, esperadoPorTipo),
		},
		[]itemResumo{
			{Rotulo: "passageiros cadastrados", Valor: passageiros, Ok: passageiros == esperadoPorTipo},
			{Rotulo: "motoristas cadastrados", Valor: motoristas, Ok: motoristas == esperadoPorTipo},
			{Rotulo: "falhas de cadastro (negócio)", Valor: falhas, Ok: falhas == 0},
			{Rotulo: "falhas de conexão (infra do teste)", Valor: falhaConexao, Ok: falhaConexao == 0},
		},
		conclusao,
	)

	if falhaConexao > 0 {
		t.Fatalf("[FALHA] %d conexões falharam.", falhaConexao)
	}
	if passageiros != esperadoPorTipo {
		t.Errorf("Esperava %d passageiros, obteve %d", esperadoPorTipo, passageiros)
	}
	if motoristas != esperadoPorTipo {
		t.Errorf("Esperava %d motoristas, obteve %d", esperadoPorTipo, motoristas)
	}
	if falhas > 0 {
		t.Errorf("Houve %d falhas de criação inesperadas", falhas)
	}
}

func TestCriacaoContas_NomeDuplicado(t *testing.T) {
	porta := obterPortaLivre(t)
	enderecoTeste := fmt.Sprintf("127.0.0.1:%d", porta)

	cmdServidor := iniciarServidorRealModular(t, enderecoTeste)

	defer func() {
		pararServidorReal(cmdServidor)
		limparAmbientePersistencia()
	}()

	const totalTentativas = 20
	const nomeDisputado = "usuario_duplicado"

	logEtapa(t, "CADASTRO_DUPLICADO", "iniciando (%d)", totalTentativas)

	var wg sync.WaitGroup
	var mu sync.Mutex
	sucessos := 0

	restaurar := silenciarStdout(t)
	for i := 1; i <= totalTentativas; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cliente, err := conexao.ConectarTCP(enderecoTeste)
			if err != nil {
				return
			}
			defer cliente.Fechar()

			sucesso, _ := usuarios.CadastrarCliente(cliente, nomeDisputado, "senha123", protocolo.PerfilPassageiro)
			if sucesso {
				mu.Lock()
				sucessos++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	restaurar()

	logEtapa(t, "CADASTRO_DUPLICADO", "concluido")

	falhasEsperadas := totalTentativas - sucessos

	conclusao := "[APROVADO] nome duplicado tratado corretamente sob concorrência"
	if sucessos != 1 {
		conclusao = fmt.Sprintf("[REPROVADO] esperava 1 sucesso, obteve %d (risco de corrida na checagem de duplicidade)", sucessos)
	}

	imprimirVisaoGeral(t,
		"VISÃO GERAL: Disputa pelo mesmo nome de usuário",
		[]string{
			fmt.Sprintf("Iniciadas %d tentativas simultâneas de cadastro com o MESMO nome ('%s')", totalTentativas, nomeDisputado),
			"Esperado: só a primeira tentativa vence, as demais são recusadas por duplicidade",
		},
		[]itemResumo{
			{Rotulo: "cadastro aceito", Valor: sucessos, Ok: sucessos == 1},
			{Rotulo: "recusados por nome duplicado", Valor: falhasEsperadas, Ok: sucessos == 1},
		},
		conclusao,
	)

	if sucessos != 1 {
		t.Fatalf("[FALHA] esperava 1 cadastro bem-sucedido, obteve %d.", sucessos)
	}
}
