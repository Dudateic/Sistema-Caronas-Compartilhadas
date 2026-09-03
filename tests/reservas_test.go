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
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/reservas"
)

// Inicia um servidor TCP real em porta dinamica despachando as acoes de reservas
func iniciarServidorReservas(t *testing.T) (net.Listener, string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Erro ao abrir listener de teste: %v", err)
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
					case protocolo.TipoBuscarItinerariosReq:
						reservas.ProcessarBuscarItinerarios(c, linha)
					case protocolo.TipoReservarItinerarioReq:
						reservas.ProcessarReservarItinerario(c, linha)
					case protocolo.TipoConsultarReservasReq:
						reservas.ProcessarConsultarReservas(c, linha)
					case protocolo.TipoCancelarReservaReq:
						reservas.ProcessarCancelarReserva(c, linha)
					}
				}
			}(conn)
		}
	}()

	return listener, listener.Addr().String()
}

// Configura o cenario inicial de caronas para os testes
// Configura o cenario inicial de caronas para os testes sem acoplamento de tipos de trecho
func resetarDadosDeTeste() {
	caronasJSON := `[
		{
			"id": 1,
			"motorista": "carlos",
			"data": "2026-10-15",
			"horario": "08:00",
			"trechos": [
				{
					"origem": "Salvador",
					"destino": "Feira de Santana",
					"preco": 35.0,
					"assentos_livres": 1,
					"passageiros": []
				}
			]
		},
		{
			"id": 2,
			"motorista": "ana",
			"data": "2026-10-15",
			"horario": "10:30",
			"trechos": [
				{
					"origem": "Feira de Santana",
					"destino": "Serrinha",
					"preco": 25.0,
					"assentos_livres": 2,
					"passageiros": []
				}
			]
		}
	]`

	caronas.MutexCaronas.Lock()
	_ = json.Unmarshal([]byte(caronasJSON), &caronas.CaronasRegistradas)
	caronas.MutexCaronas.Unlock()

	reservas.MutexReservas.Lock()
	reservas.ReservasRegistradas = []protocolo.ReservaDetalhada{}
	reservas.ProximoReservaID = 1
	reservas.MutexReservas.Unlock()
}

func TestFluxoCompletoReservas(t *testing.T) {
	resetarDadosDeTeste()

	listener, endereco := iniciarServidorReservas(t)
	defer listener.Close()

	cliente, err := conexao.ConectarTCP(endereco, 2*time.Second)
	if err != nil {
		t.Fatalf("Falha ao conectar no servidor de teste: %v", err)
	}
	defer cliente.Fechar()

	// -------------------------------------------------------------
	// 1. Teste de Busca por DFS (Conexao: Salvador -> Serrinha)
	// -------------------------------------------------------------
	itinerarios, err := reservas.BuscarItinerarios(cliente, "Salvador", "Serrinha", "2026-10-15")
	if err != nil {
		t.Fatalf("Erro ao buscar itinerarios: %v", err)
	}

	if len(itinerarios) != 1 {
		t.Fatalf("Esperava encontrar 1 itinerario composto, obteve %d", len(itinerarios))
	}

	if len(itinerarios[0].Trechos) != 2 {
		t.Fatalf("Itinerario composto deveria ter 2 trechos, tem %d", len(itinerarios[0].Trechos))
	}

	if itinerarios[0].PrecoTotal != 60.0 {
		t.Fatalf("Preco total esperado R$ 60.00, obteve R$ %.2f", itinerarios[0].PrecoTotal)
	}

	// -------------------------------------------------------------
	// 2. Reserva Atomica dos 2 trechos
	// -------------------------------------------------------------
	sucesso, idReserva, err := reservas.ReservarItinerario(cliente, "passageiro_maria", itinerarios[0])
	if err != nil || !sucesso {
		t.Fatalf("Falha ao reservar itinerario: sucesso=%v, err=%v", sucesso, err)
	}

	if idReserva != 1 {
		t.Errorf("Esperava ID de reserva 1, obteve %d", idReserva)
	}

	// Verifica se a vaga de Salvador -> Feira zerou na memoria
	caronas.MutexCaronas.Lock()
	vagasTrecho1 := caronas.CaronasRegistradas[0].Trechos[0].AssentosLivres
	caronas.MutexCaronas.Unlock()

	if vagasTrecho1 != 0 {
		t.Fatalf("Esperava 0 vagas restantes no trecho 1, tem %d", vagasTrecho1)
	}

	// -------------------------------------------------------------
	// 3. Teste de Falha por Overbooking (Tentativa sem vaga no trecho 1)
	// -------------------------------------------------------------
	sucessoSegundaReserva, _, _ := reservas.ReservarItinerario(cliente, "outro_passageiro", itinerarios[0])
	if sucessoSegundaReserva {
		t.Fatalf("Erro: o servidor permitiu reserva em trecho esgotado!")
	}

	// -------------------------------------------------------------
	// 4. Consulta de Minhas Reservas
	// -------------------------------------------------------------
	minhas, err := reservas.ConsultarMinhasReservas(cliente, "passageiro_maria")
	if err != nil {
		t.Fatalf("Erro ao consultar reservas: %v", err)
	}

	if len(minhas) != 1 || minhas[0].ID != idReserva {
		t.Fatalf("Esperava 1 reserva vinculada a maria com ID %d", idReserva)
	}

	// -------------------------------------------------------------
	// 5. Cancelamento e Devolucao de Assentos
	// -------------------------------------------------------------
	cancelou, err := reservas.CancelarMinhaReserva(cliente, idReserva, "passageiro_maria")
	if err != nil || !cancelou {
		t.Fatalf("Erro ao cancelar reserva: cancelou=%v, err=%v", cancelou, err)
	}

	// Checa se o assento foi devolvido
	caronas.MutexCaronas.Lock()
	vagasRestauradas := caronas.CaronasRegistradas[0].Trechos[0].AssentosLivres
	passageirosRestantes := len(caronas.CaronasRegistradas[0].Trechos[0].Passageiros)
	caronas.MutexCaronas.Unlock()

	if vagasRestauradas != 1 {
		t.Errorf("Esperava restauracao de 1 vaga apos cancelamento, obteve %d", vagasRestauradas)
	}

	if passageirosRestantes != 0 {
		t.Errorf("Passageiro deveria ter sido removido do trecho, restam %d", passageirosRestantes)
	}
}
