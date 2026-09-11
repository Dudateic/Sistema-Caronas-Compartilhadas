/**
 * Pacote responsavel pelas buscas de itinerarios (DFS) e realizacao de reservas.
 *
 * @author Maria Eduarda
 */
package reservas

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/caronas"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/persistencia"
)

var (
	ReservasRegistradas []protocolo.ReservaDetalhada
	ProximoReservaID    = 1
	MutexReservas       sync.Mutex
)

const ArquivoReservas = "reservas.json"

var NotificacoesRegistradas []protocolo.Notificacao

const ArquivoNotificacoes = "notificacoes.json"

// InicializarReservas deve ser chamado no main do servidor
func InicializarReservas() error {
	MutexReservas.Lock()
	defer MutexReservas.Unlock()

	if err := persistencia.CarregarJSON(ArquivoReservas, &ReservasRegistradas); err != nil {
		return err
	}

	// Recalcula o proximo ID com base no historico do disco
	for _, r := range ReservasRegistradas {
		if r.ID >= ProximoReservaID {
			ProximoReservaID = r.ID + 1
		}
	}
	return nil
}

/**
 * Procura rotas diretas e combinadas entre diferentes caronas para a data informada.
 */
func ProcessarBuscarItinerarios(conn net.Conn, dadosBrutos string) {
	var req protocolo.BuscarItinerariosRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		responderBusca(conn, false, "JSON malformado", nil)
		return
	}

	grafoAdjacenciaTrechos := make(map[string][]protocolo.TrechoItinerario)

	caronas.MutexCaronas.RLock()
	for _, c := range caronas.CaronasRegistradas {
		if c.Data != req.Data {
			continue
		}
		for _, t := range c.Trechos {
			if t.AssentosLivres > 0 {
				novoTrecho := protocolo.TrechoItinerario{
					CaronaID:  c.ID,
					Motorista: c.Motorista,
					Origem:    t.Origem,
					Destino:   t.Destino,
					Horario:   c.Horario,
					Preco:     t.Preco,
				}
				// Adiciona o trecho diretamente na lista daquela cidade
				grafoAdjacenciaTrechos[t.Origem] = append(grafoAdjacenciaTrechos[t.Origem], novoTrecho)
			}
		}
	}
	caronas.MutexCaronas.RUnlock()

	var itinerariosEncontrados []protocolo.Itinerario
	visitados := make(map[string]bool)

	buscarDFS(
		req.Origem,
		req.Destino,
		req.Data,
		grafoAdjacenciaTrechos,
		visitados,
		[]protocolo.TrechoItinerario{},
		0.0,
		&itinerariosEncontrados,
	)

	if len(itinerariosEncontrados) == 0 {
		responderBusca(conn, true, "Nenhum itinerario disponivel para essa data", []protocolo.Itinerario{})
		return
	}

	msg := fmt.Sprintf("%d itinerario(s) encontrado(s)", len(itinerariosEncontrados))
	responderBusca(conn, true, msg, itinerariosEncontrados)
}

/**
 * Algoritmo recursivo de busca que percorre as conexoes de cidades.
 */
func buscarDFS(
	atual string,
	destino string,
	data string,
	grafo map[string][]protocolo.TrechoItinerario,
	visitados map[string]bool,
	caminho []protocolo.TrechoItinerario,
	custo float64,
	resultados *[]protocolo.Itinerario,
) {
	if atual == destino {
		trechosCopia := make([]protocolo.TrechoItinerario, len(caminho))
		copy(trechosCopia, caminho)

		*resultados = append(*resultados, protocolo.Itinerario{
			Data:       data,
			PrecoTotal: custo,
			Trechos:    trechosCopia,
		})
		return
	}

	visitados[atual] = true

	for _, tr := range grafo[atual] {
		if !visitados[tr.Destino] {
			// Validacao espacial e temporal simples (horario da conexao deve ser >= ao trecho anterior se houver)
			if len(caminho) > 0 {
				ultimoTrecho := caminho[len(caminho)-1]
				// Se for mesma carona ou horario posterior/igual
				if ultimoTrecho.CaronaID != tr.CaronaID && tr.Horario < ultimoTrecho.Horario {
					continue
				}
			}

			caminho = append(caminho, tr)
			buscarDFS(tr.Destino, destino, data, grafo, visitados, caminho, custo+tr.Preco, resultados)
			caminho = caminho[:len(caminho)-1]
		}
	}

	visitados[atual] = false
}

func responderBusca(conn net.Conn, sucesso bool, mensagem string, itinerarios []protocolo.Itinerario) {
	resp := protocolo.BuscarItinerariosResposta{
		Tipo:        protocolo.TipoBuscarItinerariosRes,
		Sucesso:     sucesso,
		Mensagem:    mensagem,
		Itinerarios: itinerarios,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Realiza a reserva atomica, garantindo vaga em todos os trechos antes de decrementar.
 */
func ProcessarReservarItinerario(conn net.Conn, dadosBrutos string) {
	var req protocolo.ReservarRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		responderReserva(conn, false, "JSON malformado", 0)
		return
	}

	if len(req.Trechos) == 0 || req.Passageiro == "" {
		responderReserva(conn, false, "Parametros de reserva invalidos", 0)
		return
	}

	// Bloqueia ambos os recursos mantendo ordem estrita
	caronas.MutexCaronas.Lock()
	defer caronas.MutexCaronas.Unlock()

	MutexReservas.Lock()
	defer MutexReservas.Unlock()

	// Validacao atomica de todos os trechos
	for _, trReq := range req.Trechos {
		encontrado := false
		for _, c := range caronas.CaronasRegistradas {
			if c.ID == trReq.CaronaID {
				for _, t := range c.Trechos {
					if t.Origem == trReq.Origem && t.Destino == trReq.Destino {
						if t.AssentosLivres > 0 {
							encontrado = true
						}
						break
					}
				}
			}
			if encontrado {
				break
			}
		}
		if !encontrado {
			responderReserva(conn, false, "Assento indisponivel em um dos trechos solicitados", 0)
			return
		}
	}

	// Debito efetivo das vagas
	var precoTotal float64
	for _, trReq := range req.Trechos {
		for i := range caronas.CaronasRegistradas {
			if caronas.CaronasRegistradas[i].ID == trReq.CaronaID {
				for j := range caronas.CaronasRegistradas[i].Trechos {
					tr := &caronas.CaronasRegistradas[i].Trechos[j]
					if tr.Origem == trReq.Origem && tr.Destino == trReq.Destino {
						tr.AssentosLivres--
						tr.Passageiros = append(tr.Passageiros, req.Passageiro)
						precoTotal += tr.Preco
					}
				}
			}
		}
	}

	// Persistencia da Reserva
	reservaID := ProximoReservaID
	ProximoReservaID++

	novaReserva := protocolo.ReservaDetalhada{
		ID:         reservaID,
		Passageiro: req.Passageiro,
		Data:       req.Data,
		PrecoTotal: precoTotal,
		Trechos:    req.Trechos,
	}
	ReservasRegistradas = append(ReservasRegistradas, novaReserva)

	// Persiste o estado atualizado de reservas e o decremento das vagas nas caronas
	if err := persistencia.SalvarJSON(ArquivoReservas, ReservasRegistradas); err != nil {
		fmt.Printf("[ERRO] Falha ao persistir reservas: %v\n", err)
	}
	if err := persistencia.SalvarJSON(caronas.ArquivoCaronas, caronas.CaronasRegistradas); err != nil {
		fmt.Printf("[ERRO] Falha ao persistir caronas apos reserva: %v\n", err)
	}

	fmt.Printf("[RESERVA] Reserva #%d confirmada para '%s' (R$ %.2f)\n", reservaID, req.Passageiro, precoTotal)
	responderReserva(conn, true, "Reserva confirmada com sucesso!", reservaID)
}

func responderReserva(conn net.Conn, sucesso bool, mensagem string, id int) {
	resp := protocolo.ReservarResposta{
		Tipo:      protocolo.TipoReservarItinerarioRes,
		Sucesso:   sucesso,
		Mensagem:  mensagem,
		ReservaID: id,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Retorna as reservas pertencentes ao passageiro.
 */
func ProcessarConsultarReservas(conn net.Conn, dadosBrutos string) {
	var req protocolo.ConsultaReservasRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		return
	}

	MutexReservas.Lock()
	var minhas []protocolo.ReservaDetalhada
	for _, res := range ReservasRegistradas {
		if res.Passageiro == req.Passageiro {
			minhas = append(minhas, res)
		}
	}
	MutexReservas.Unlock()

	resp := protocolo.ConsultaReservasResposta{
		Tipo:     protocolo.TipoConsultarReservasRes,
		Sucesso:  true,
		Mensagem: "OK",
		Reservas: minhas,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Cancela a reserva e restaura os assentos mantendo a sincronizacao correta.
 */
func ProcessarCancelarReserva(conn net.Conn, dadosBrutos string) {
	var req protocolo.CancelarReservaRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		return
	}

	caronas.MutexCaronas.Lock()
	defer caronas.MutexCaronas.Unlock()

	MutexReservas.Lock()
	defer MutexReservas.Unlock()

	idxReserva := -1
	var reservaAlvo protocolo.ReservaDetalhada
	for i, res := range ReservasRegistradas {
		if res.ID == req.ReservaID && res.Passageiro == req.Passageiro {
			idxReserva = i
			reservaAlvo = res
			break
		}
	}

	if idxReserva == -1 {
		resp := protocolo.CancelarReservaResposta{
			Tipo:     protocolo.TipoCancelarReservaRes,
			Sucesso:  false,
			Mensagem: "Reserva nao encontrada ou permissao negada",
		}
		_ = json.NewEncoder(conn).Encode(resp)
		return
	}

	// Remove do slice de reservas
	ReservasRegistradas = append(ReservasRegistradas[:idxReserva], ReservasRegistradas[idxReserva+1:]...)

	// Libera os assentos e retira o passageiro do trecho
	for _, trReq := range reservaAlvo.Trechos {
		for i := range caronas.CaronasRegistradas {
			if caronas.CaronasRegistradas[i].ID == trReq.CaronaID {
				for j := range caronas.CaronasRegistradas[i].Trechos {
					tr := &caronas.CaronasRegistradas[i].Trechos[j]
					if tr.Origem == trReq.Origem && tr.Destino == trReq.Destino {
						tr.AssentosLivres++
						for pIdx, pNome := range tr.Passageiros {
							if pNome == req.Passageiro {
								tr.Passageiros = append(tr.Passageiros[:pIdx], tr.Passageiros[pIdx+1:]...)
								break
							}
						}
					}
				}
			}
		}
	}

	// Persiste a remocao da reserva e os assentos devolvidos no disco do servidor
	if err := persistencia.SalvarJSON(ArquivoReservas, ReservasRegistradas); err != nil {
		fmt.Printf("[ERRO] Falha ao persistir reservas apos cancelamento: %v\n", err)
	}
	if err := persistencia.SalvarJSON(caronas.ArquivoCaronas, caronas.CaronasRegistradas); err != nil {
		fmt.Printf("[ERRO] Falha ao persistir caronas apos restauracao de vagas: %v\n", err)
	}

	fmt.Printf("[RESERVA] Reserva #%d cancelada para '%s'\n", req.ReservaID, req.Passageiro)

	resp := protocolo.CancelarReservaResposta{
		Tipo:     protocolo.TipoCancelarReservaRes,
		Sucesso:  true,
		Mensagem: "Reserva cancelada e assentos liberados com sucesso",
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

func BuscarItinerarios(cliente *conexao.ClienteTCP, origem, destino, data string) ([]protocolo.Itinerario, error) {
	req := protocolo.BuscarItinerariosRequisicao{
		Tipo:    protocolo.TipoBuscarItinerariosReq,
		Origem:  origem,
		Destino: destino,
		Data:    data,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return nil, fmt.Errorf("falha ao enviar busca: %w", err)
	}

	var resp protocolo.BuscarItinerariosResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return nil, fmt.Errorf("falha ao ler resposta da busca: %w", err)
	}

	if !resp.Sucesso || len(resp.Itinerarios) == 0 {
		return nil, nil
	}

	return resp.Itinerarios, nil
}

func ReservarItinerario(cliente *conexao.ClienteTCP, passageiro string, itinerario protocolo.Itinerario) (bool, int, error) {
	req := protocolo.ReservarRequisicao{
		Tipo:       protocolo.TipoReservarItinerarioReq,
		Passageiro: passageiro,
		Data:       itinerario.Data,
		Trechos:    itinerario.Trechos,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, 0, fmt.Errorf("falha ao enviar reserva: %w", err)
	}

	var resp protocolo.ReservarResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, 0, fmt.Errorf("falha ao receber confirmacao da reserva: %w", err)
	}

	if !resp.Sucesso {
		return false, 0, nil
	}

	return true, resp.ReservaID, nil
}

func ConsultarMinhasReservas(cliente *conexao.ClienteTCP, passageiro string) ([]protocolo.ReservaDetalhada, error) {
	req := protocolo.ConsultaReservasRequisicao{
		Tipo:       protocolo.TipoConsultarReservasReq,
		Passageiro: passageiro,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return nil, fmt.Errorf("falha ao enviar consulta de reservas: %w", err)
	}

	var resp protocolo.ConsultaReservasResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return nil, fmt.Errorf("falha ao ler reservas: %w", err)
	}

	return resp.Reservas, nil
}

func CancelarMinhaReserva(cliente *conexao.ClienteTCP, reservaID int, passageiro string) (bool, error) {
	req := protocolo.CancelarReservaRequisicao{
		Tipo:       protocolo.TipoCancelarReservaReq,
		ReservaID:  reservaID,
		Passageiro: passageiro,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, fmt.Errorf("falha ao enviar cancelamento: %w", err)
	}

	var resp protocolo.CancelarReservaResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, fmt.Errorf("falha ao receber resposta: %w", err)
	}

	return resp.Sucesso, nil
}

// LimparReservasDaCarona exclui as reservas órfãs e notifica os passageiros
func LimparReservasDaCarona(caronaID int, motorista string) {
	MutexReservas.Lock()
	defer MutexReservas.Unlock()

	// Tenta carregar notificações antigas
	_ = persistencia.CarregarJSON(ArquivoNotificacoes, &NotificacoesRegistradas)

	var reservasRestantes []protocolo.ReservaDetalhada
	for _, res := range ReservasRegistradas {
		afetada := false
		for _, tr := range res.Trechos {
			if tr.CaronaID == caronaID {
				afetada = true
				break
			}
		}

		if afetada {
			// A reserva foi cancelada, cria notificação para o passageiro
			msg := fmt.Sprintf("ATENCAO: Sua reserva #%d foi cancelada pelo sistema porque o motorista '%s' cancelou a carona.", res.ID, motorista)

			novaNot := protocolo.Notificacao{
				Passageiro: res.Passageiro,
				Mensagem:   msg,
				Data:       time.Now().Format("2006-01-02 15:04:05"),
			}
			NotificacoesRegistradas = append(NotificacoesRegistradas, novaNot)
			fmt.Printf("[NOTIFICACAO] Aviso gerado para o passageiro '%s'\n", res.Passageiro)
		} else {
			reservasRestantes = append(reservasRestantes, res)
		}
	}

	// Atualiza banco de dados
	ReservasRegistradas = reservasRestantes
	_ = persistencia.SalvarJSON(ArquivoReservas, ReservasRegistradas)
	_ = persistencia.SalvarJSON(ArquivoNotificacoes, NotificacoesRegistradas)
}

// ProcessarConsultarNotificacoes busca avisos do passageiro e apaga (marca como lido)
func ProcessarConsultarNotificacoes(conn net.Conn, dadosBrutos string) {
	var req protocolo.ConsultarNotificacoesRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		return
	}

	MutexReservas.Lock()
	defer MutexReservas.Unlock()
	_ = persistencia.CarregarJSON(ArquivoNotificacoes, &NotificacoesRegistradas)

	var minhas []protocolo.Notificacao
	var restantes []protocolo.Notificacao

	for _, n := range NotificacoesRegistradas {
		if n.Passageiro == req.Passageiro {
			minhas = append(minhas, n)
		} else {
			restantes = append(restantes, n)
		}
	}

	if len(minhas) > 0 {
		NotificacoesRegistradas = restantes
		_ = persistencia.SalvarJSON(ArquivoNotificacoes, NotificacoesRegistradas)
	}

	resp := protocolo.ConsultarNotificacoesResposta{
		Tipo:         protocolo.TipoConsultarNotificacoesRes,
		Sucesso:      true,
		Notificacoes: minhas,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}
