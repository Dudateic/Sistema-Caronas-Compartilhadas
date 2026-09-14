/**
 * Pacote responsavel pelo gerenciamento de estado e regras de negocio das caronas.
 *
 * @author Maria Eduarda
 */
package caronas

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/persistencia"
)

// Estado compartilhado em memoria protegido por Mutex
var (
	CaronasRegistradas []protocolo.CaronaDetalhada
	ProximoCaronaID    = 1
	MutexCaronas       sync.RWMutex
)

const ArquivoCaronas = "caronas.json"

// Callback executado sempre que uma carona é cancelada
var AoCancelarCarona func(caronaID int, motorista string)

// InicializarCaronas deve ser chamado no main do servidor
func InicializarCaronas() error {
	MutexCaronas.Lock()
	defer MutexCaronas.Unlock()

	if err := persistencia.CarregarJSON(ArquivoCaronas, &CaronasRegistradas); err != nil {
		return err
	}

	// Recalcula o proximo ID com base no que veio do disco
	for _, c := range CaronasRegistradas {
		if c.ID >= ProximoCaronaID {
			ProximoCaronaID = c.ID + 1
		}
	}
	return nil
}

/**
 * Valida a rota e os dados da carona, monta os trechos e armazena a viagem em memoria.
 */
func ProcessarPublicarCarona(conn net.Conn, dadosBrutos string) {
	var req protocolo.PublicarCaronaRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		responderPublicacao(conn, false, "JSON malformado", 0)
		return
	}

	if len(req.Rota) < 2 {
		responderPublicacao(conn, false, "A rota precisa de pelo menos 2 cidades", 0)
		return
	}

	if req.AssentosTotais <= 0 {
		responderPublicacao(conn, false, "O total de assentos deve ser maior que zero", 0)
		return
	}

	if req.PrecoPorTrecho <= 0 {
		responderPublicacao(conn, false, "O preco por trecho deve ser positivo", 0)
		return
	}

	MutexCaronas.Lock()
	defer MutexCaronas.Unlock()

	// Divide a sequencia de cidades em trechos individuais consecutivos
	var trechos []protocolo.TrechoInfo
	for i := 0; i < len(req.Rota)-1; i++ {
		trechos = append(trechos, protocolo.TrechoInfo{
			Origem:         req.Rota[i],
			Destino:        req.Rota[i+1],
			AssentosLivres: req.AssentosTotais,
			Preco:          req.PrecoPorTrecho,
			Passageiros:    []string{},
		})
	}

	novaCarona := protocolo.CaronaDetalhada{
		ID:             ProximoCaronaID,
		Motorista:      req.Motorista,
		Data:           req.Data,
		Horario:        req.Horario,
		AssentosTotais: req.AssentosTotais,
		PrecoPorTrecho: req.PrecoPorTrecho,
		Rota:           req.Rota,
		Trechos:        trechos,
	}

	CaronasRegistradas = append(CaronasRegistradas, novaCarona)
	caronaID := ProximoCaronaID
	ProximoCaronaID++

	if err := persistencia.SalvarJSON(ArquivoCaronas, CaronasRegistradas); err != nil {
		fmt.Printf("[ERRO] Falha ao persistir caronas: %v\n", err)
	}

	fmt.Printf("[CARONA] Nova carona #%d cadastrada por '%s' (%s -> %s)\n",
		caronaID, req.Motorista, req.Rota[0], req.Rota[len(req.Rota)-1])

	responderPublicacao(conn, true, "Carona publicada com sucesso", caronaID)
}

func responderPublicacao(conn net.Conn, sucesso bool, mensagem string, id int) {
	resp := protocolo.PublicarCaronaResposta{
		Tipo:     protocolo.TipoPublicarCaronaRes,
		Sucesso:  sucesso,
		Mensagem: mensagem,
		CaronaID: id,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Filtra e devolve todas as caronas criadas pelo motorista solicitante.
 */
func ProcessarConsultarCaronas(conn net.Conn, dadosBrutos string) {
	var req protocolo.ConsultarCaronasRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		return
	}

	MutexCaronas.RLock()
	defer MutexCaronas.RUnlock()

	var minhas []protocolo.CaronaDetalhada
	for _, c := range CaronasRegistradas {
		if c.Motorista == req.Motorista {
			minhas = append(minhas, c)
		}
	}

	resp := protocolo.ConsultarCaronasResposta{
		Tipo:     protocolo.TipoConsultarCaronasRes,
		Sucesso:  true,
		Mensagem: "Consulta concluida",
		Caronas:  minhas,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Localiza a viagem pelo ID e remove da memoria caso pertencente ao motorista.
 */
func ProcessarCancelarCarona(conn net.Conn, dadosBrutos string) {
	var req protocolo.CancelarCaronaRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		return
	}

	MutexCaronas.Lock()
	defer MutexCaronas.Unlock()

	idx := -1
	for i, c := range CaronasRegistradas {
		if c.ID == req.CaronaID && c.Motorista == req.Motorista {
			idx = i
			break
		}
	}

	if idx == -1 {
		resp := protocolo.CancelarCaronaResposta{
			Tipo:     protocolo.TipoCancelarCaronaRes,
			Sucesso:  false,
			Mensagem: "Carona nao encontrada ou permissao negada",
		}
		_ = json.NewEncoder(conn).Encode(resp)
		return
	}

	CaronasRegistradas = append(CaronasRegistradas[:idx], CaronasRegistradas[idx+1:]...)

	if AoCancelarCarona != nil {
		go AoCancelarCarona(req.CaronaID, req.Motorista)
	}

	if err := persistencia.SalvarJSON(ArquivoCaronas, CaronasRegistradas); err != nil {
		fmt.Printf("[ERRO] Falha ao persistir cancelamento de carona: %v\n", err)
	}

	fmt.Printf("[CARONA] Carona #%d cancelada pelo motorista '%s'\n", req.CaronaID, req.Motorista)

	resp := protocolo.CancelarCaronaResposta{
		Tipo:     protocolo.TipoCancelarCaronaRes,
		Sucesso:  true,
		Mensagem: "Carona cancelada com sucesso",
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Envia pedido de publicacao de rota pelo ClienteTCP e retorna o ID da viagem criada.
 */
func PublicarCarona(cliente *conexao.ClienteTCP, motorista string, rota []string, data, horario string, assentos int, preco float64) (int, error) {
	req := protocolo.PublicarCaronaRequisicao{
		Tipo:           protocolo.TipoPublicarCaronaReq,
		Motorista:      motorista,
		Rota:           rota,
		Data:           data,
		Horario:        horario,
		AssentosTotais: assentos,
		PrecoPorTrecho: preco,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return 0, fmt.Errorf("falha ao enviar publicacao: %w", err)
	}

	var resp protocolo.PublicarCaronaResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return 0, fmt.Errorf("falha ao receber resposta do servidor: %w", err)
	}

	if !resp.Sucesso {
		return 0, fmt.Errorf("falha ao publicar: %s", resp.Mensagem)
	}

	return resp.CaronaID, nil
}

/**
 * Requisita ao servidor as caronas cadastradas pelo motorista.
 */
func ConsultarCaronas(cliente *conexao.ClienteTCP, motorista string) ([]protocolo.CaronaDetalhada, error) {
	req := protocolo.ConsultarCaronasRequisicao{
		Tipo:      protocolo.TipoConsultarCaronasReq,
		Motorista: motorista,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return nil, fmt.Errorf("falha ao solicitar caronas: %w", err)
	}

	var resp protocolo.ConsultarCaronasResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return nil, fmt.Errorf("falha ao ler caronas: %w", err)
	}

	return resp.Caronas, nil
}

/**
 * Envia pedido ao servidor para cancelar uma carona pelo ID.
 */
func CancelarCarona(cliente *conexao.ClienteTCP, caronaID int, motorista string) (bool, error) {
	req := protocolo.CancelarCaronaRequisicao{
		Tipo:      protocolo.TipoCancelarCaronaReq,
		CaronaID:  caronaID,
		Motorista: motorista,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, fmt.Errorf("falha ao enviar cancelamento: %w", err)
	}

	var resp protocolo.CancelarCaronaResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, fmt.Errorf("falha ao receber resposta do cancelamento: %w", err)
	}

	if !resp.Sucesso {
		return false, fmt.Errorf("erro ao cancelar: %s", resp.Mensagem)
	}

	return true, nil
}
