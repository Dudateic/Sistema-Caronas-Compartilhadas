package testes

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/caronas"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/reservas"
)

type resultadoPassageiro struct {
	ID          int
	Nome        string
	Sucesso     bool
	ErroConexao error
	ErroReserva string
	Latencia    time.Duration
}

func limparAmbientePersistencia() {
	arquivosLixo := []string{
		"../caronas.json", "../reservas.json", "../usuarios.json", "../notificacoes.json",
		"caronas.json", "reservas.json", "usuarios.json", "notificacoes.json",
		"../dados/caronas.json", "../dados/reservas.json", "../dados/usuarios.json",
	}
	for _, arq := range arquivosLixo {
		_ = os.Remove(arq)
	}
}

func obterPortaLivre(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Falha ao obter porta livre: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func esperarServidorPronto(endereco string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", endereco, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("servidor não respondeu em %s", timeout)
}

func iniciarServidorRealModular(t *testing.T, endereco string) *exec.Cmd {
	limparAmbientePersistencia()

	cmd := exec.Command("go", "run", "aplicacao/servidor/main.go", endereco)
	cmd.Dir = ".."
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logEtapa(t, "SERVIDOR", "iniciando")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Falha ao iniciar o seu servidor real: %v", err)
	}

	if err := esperarServidorPronto(endereco, 15*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("Servidor não ficou pronto a tempo: %v", err)
	}
	logEtapa(t, "SERVIDOR", "concluido")

	return cmd
}

func pararServidorReal(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}

func prepararCaronaInicial(t *testing.T, endereco string) int {
	logEtapa(t, "CARONA", "iniciando")

	cliMotorista, err := conexao.ConectarTCP(endereco)
	if err != nil {
		t.Fatalf("Falha ao conectar cliente motorista: %v", err)
	}
	defer cliMotorista.Fechar()

	restaurar := silenciarStdout(t)
	_, err = caronas.PublicarCarona(cliMotorista, "Piloto_Teste", []string{"Salvador", "Feira"}, "2026-10-10", "14:00", 50, 20.0)
	restaurar()

	if err != nil {
		t.Fatalf("Falha ao publicar carona: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	logEtapa(t, "CARONA", "concluido (id=1, vagas=50)")
	return 1
}

func executarDisparosConcorrentes(t *testing.T, endereco string, caronaID int, totalPassageiros int) []resultadoPassageiro {
	var wg sync.WaitGroup
	var mu sync.Mutex
	resultados := make([]resultadoPassageiro, 0, totalPassageiros)

	logEtapa(t, "RESERVAS", "iniciando (%d)", totalPassageiros)

	restaurar := silenciarStdout(t)

	for i := 1; i <= totalPassageiros; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nome := "Passageiro_" + strconv.Itoa(id)
			inicio := time.Now()

			cliPassageiro, err := conexao.ConectarTCP(endereco)
			if err != nil {
				mu.Lock()
				resultados = append(resultados, resultadoPassageiro{
					ID: id, Nome: nome, Sucesso: false,
					ErroConexao: err, Latencia: time.Since(inicio),
				})
				mu.Unlock()
				return
			}
			defer cliPassageiro.Fechar()

			itinerario := protocolo.Itinerario{
				Data: "2026-10-10",
				Trechos: []protocolo.TrechoItinerario{
					{CaronaID: caronaID, Origem: "Salvador", Destino: "Feira", Preco: 20.0},
				},
			}

			sucesso, msg, _ := reservas.ReservarItinerario(cliPassageiro, nome, itinerario)
			latencia := time.Since(inicio)
			msgTexto := "código " + strconv.Itoa(msg)

			mu.Lock()
			resultados = append(resultados, resultadoPassageiro{
				ID: id, Nome: nome, Sucesso: sucesso,
				ErroReserva: msgTexto, Latencia: latencia,
			})
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	restaurar()

	logEtapa(t, "RESERVAS", "concluido")
	return resultados
}

func TestConcorrencia(t *testing.T) {
	porta := obterPortaLivre(t)
	enderecoTeste := fmt.Sprintf("127.0.0.1:%d", porta)

	cmdServidor := iniciarServidorRealModular(t, enderecoTeste)

	defer func() {
		pararServidorReal(cmdServidor)
		limparAmbientePersistencia()
	}()

	const totalVagas = 50
	caronaID := prepararCaronaInicial(t, enderecoTeste)

	const totalPassageiros = 200
	resultados := executarDisparosConcorrentes(t, enderecoTeste, caronaID, totalPassageiros)

	var vendidas, negadas, falhaConexao int
	for _, r := range resultados {
		switch {
		case r.ErroConexao != nil:
			falhaConexao++
		case r.Sucesso:
			vendidas++
		default:
			negadas++
		}
	}

	conclusao := "[APROVADO] sem overbooking e sem travamento"
	if falhaConexao > 0 {
		conclusao = "[REPROVADO] houve falha de infraestrutura do teste (conexão)"
	} else if vendidas > totalVagas {
		conclusao = fmt.Sprintf("[REPROVADO] overbooking! vendeu %d vagas, só existiam %d", vendidas, totalVagas)
	} else if vendidas < totalVagas {
		conclusao = fmt.Sprintf("[REPROVADO] servidor bloqueou passageiros, vendeu só %d de %d vagas", vendidas, totalVagas)
	}

	imprimirVisaoGeral(t,
		"VISÃO GERAL: Disputa de vagas em carona (concorrência)",
		[]string{
			fmt.Sprintf("Carona publicada com %d vagas (Salvador -> Feira de Santana)", totalVagas),
			fmt.Sprintf("Disparados %d passageiros simultâneos tentando reservar a mesma carona", totalPassageiros),
		},
		[]itemResumo{
			{Rotulo: "vagas vendidas", Valor: vendidas, Ok: vendidas == totalVagas},
			{Rotulo: "reservas negadas (vagas esgotadas)", Valor: negadas, Ok: true},
			{Rotulo: "falhas de conexão (infra do teste)", Valor: falhaConexao, Ok: falhaConexao == 0},
		},
		conclusao,
	)

	if falhaConexao > 0 {
		t.Fatalf("[FALHA] %d conexões falharam antes de chegar ao servidor.", falhaConexao)
	}
	if vendidas > totalVagas {
		t.Fatalf("[FALHA] Overbooking: vendeu %d vagas, carro só tem %d.", vendidas, totalVagas)
	}
	if vendidas < totalVagas {
		t.Fatalf("[FALHA] Servidor bloqueou passageiros. Apenas %d compraram (negadas: %d).", vendidas, negadas)
	}
}
