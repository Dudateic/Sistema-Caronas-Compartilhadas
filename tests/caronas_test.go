package testes

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/caronas"
)

// Inicia um servidor TCP real em porta dinamica para o servico de caronas
func iniciarServidorCaronas(t *testing.T) (net.Listener, string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Erro ao abrir porta para teste de caronas: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()
				leitor := bufio.NewReader(c)

				for {
					linha, err := leitor.ReadString('\n')
					if err != nil {
						return
					}

					var base protocolo.MensagemBase
					if err := json.Unmarshal([]byte(linha), &base); err != nil {
						continue
					}

					switch base.Tipo {
					case protocolo.TipoPublicarCaronaReq:
						caronas.ProcessarPublicarCarona(c, linha)
					case protocolo.TipoConsultarCaronasReq:
						caronas.ProcessarConsultarCaronas(c, linha)
					case protocolo.TipoCancelarCaronaReq:
						caronas.ProcessarCancelarCarona(c, linha)
					}
				}
			}(conn)
		}
	}()

	return listener, listener.Addr().String()
}

func TestFluxoGerenciamentoCaronas(t *testing.T) {
	// Limpa o estado em memoria antes do teste
	caronas.MutexCaronas.Lock()
	caronas.CaronasRegistradas = []protocolo.CaronaDetalhada{}
	caronas.ProximoCaronaID = 1
	caronas.MutexCaronas.Unlock()

	listener, endereco := iniciarServidorCaronas(t)
	defer listener.Close()

	cliente, err := conexao.ConectarTCP(endereco, 2*time.Second)
	if err != nil {
		t.Fatalf("Erro ao conectar cliente: %v", err)
	}
	defer cliente.Fechar()

	motorista := "carlos_motorista"
	rota := []string{"Salvador", "Feira de Santana", "Serrinha"}

	// Validacao de publicacao invalida (rota insuficiente)
	idInvalido, _ := caronas.PublicarCarona(cliente, motorista, []string{"Salvador"}, "2026-10-20", "08:00", 3, 20.0)
	if idInvalido != 0 {
		t.Fatalf("Erro: publicou carona com apenas 1 cidade na rota!")
	}

	// Publicacao com sucesso
	caronaID, err := caronas.PublicarCarona(cliente, motorista, rota, "2026-10-20", "08:00", 3, 25.0)
	if err != nil || caronaID == 0 {
		t.Fatalf("Falha ao publicar carona valida: id=%d, err=%v", caronaID, err)
	}

	// Inspeciona se gerou os 2 trechos consecutivos corretamente
	caronas.MutexCaronas.Lock()
	if len(caronas.CaronasRegistradas) != 1 {
		t.Fatalf("Esperava 1 carona registrada, encontrou %d", len(caronas.CaronasRegistradas))
	}
	trechos := caronas.CaronasRegistradas[0].Trechos
	caronas.MutexCaronas.Unlock()

	if len(trechos) != 2 {
		t.Fatalf("Esperava 2 trechos gerados para 3 cidades, obteve %d", len(trechos))
	}
	if trechos[0].Origem != "Salvador" || trechos[0].Destino != "Feira de Santana" {
		t.Errorf("Trecho 1 incorreto: %s -> %s", trechos[0].Origem, trechos[0].Destino)
	}
	if trechos[1].Origem != "Feira de Santana" || trechos[1].Destino != "Serrinha" {
		t.Errorf("Trecho 2 incorreto: %s -> %s", trechos[1].Origem, trechos[1].Destino)
	}

	// Consulta de caronas do motorista
	lista, err := caronas.ConsultarCaronas(cliente, motorista)
	if err != nil {
		t.Fatalf("Erro ao consultar caronas: %v", err)
	}
	if len(lista) != 1 || lista[0].ID != caronaID {
		t.Fatalf("Esperava encontrar a carona ID %d, obteve %d itens", caronaID, len(lista))
	}

	// Consulta com outro motorista deve vir vazia
	listaVazia, _ := caronas.ConsultarCaronas(cliente, "outro_motorista")
	if len(listaVazia) != 0 {
		t.Fatalf("Erro: motorista sem caronas obteve resultados!")
	}

	// Bloqueio de cancelamento por terceiro
	cancelouInvasor, _ := caronas.CancelarCarona(cliente, caronaID, "invasor")
	if cancelouInvasor {
		t.Fatalf("Erro: permitiu que um usuario cancelasse a carona de outro motorista!")
	}

	// Cancelamento legitimo pelo dono
	cancelou, err := caronas.CancelarCarona(cliente, caronaID, motorista)
	if err != nil || !cancelou {
		t.Fatalf("Falha ao cancelar carona pelo dono: cancelou=%v, err=%v", cancelou, err)
	}

	caronas.MutexCaronas.Lock()
	totalRestante := len(caronas.CaronasRegistradas)
	caronas.MutexCaronas.Unlock()

	if totalRestante != 0 {
		t.Errorf("Carona deveria ter sido removida da memoria, restam %d", totalRestante)
	}
}
