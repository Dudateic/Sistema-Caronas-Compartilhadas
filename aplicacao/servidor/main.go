package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"Sistema-de-caronas-compartilhadas/servicos/caronas"
	"Sistema-de-caronas-compartilhadas/servicos/persistencia"
	"Sistema-de-caronas-compartilhadas/servicos/reservas"
	"Sistema-de-caronas-compartilhadas/servicos/usuarios"
)

func main() {
	fmt.Println()
	fmt.Println("         VAIJUNTO   SERVIDOR                ")
	fmt.Println()

	// Inicializa os dados persistidos do servidor
	if err := caronas.InicializarCaronas(); err != nil {
		fmt.Printf("[ERRO] Falha ao inicializar caronas: %v\n", err)
		return
	}
	if err := reservas.InicializarReservas(); err != nil {
		fmt.Printf("[ERRO] Falha ao inicializar reservas: %v\n", err)
		return
	}

	caronas.AoCancelarCarona = reservas.LimparReservasDaCarona

	endereco := conexao.EnderecoPadrao
	if len(os.Args) > 1 {
		endereco = os.Args[1]
	}

	listener, err := net.Listen("tcp", endereco)
	if err != nil {
		fmt.Printf("[ERRO] Nao foi possivel iniciar o servidor em %s: %v\n", endereco, err)
		return
	}
	defer listener.Close()

	fmt.Printf("Servidor ativo e escutando conexoes em %s\n", endereco)
	fmt.Println("Pressione Ctrl+C para encerrar o servico")
	fmt.Println()

	// Captura interrupcoes do sistema para encerramento
	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sinais
		fmt.Println()
		fmt.Println("Encerrando o servidor VAIJUNTO...")
		_ = listener.Close()
	}()

	// Loop principal aceitando conexoes
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Se o listener foi fechado intencionalmente, encerra
			if errors.Is(err, net.ErrClosed) {
				break
			}
			fmt.Printf("[ERRO] Falha ao aceitar conexao: %v\n", err)
			continue
		}

		// Processa cada cliente conectado de forma concorrente
		go tratarCliente(conn)
	}

	fmt.Println("Servidor finalizado com sucesso. Ate logo!")
}

/**
 * Atende continuamente as requisicoes de um cliente conectado
 */
func tratarCliente(conn net.Conn) {
	remoto := conn.RemoteAddr().String()
	fmt.Printf("[CONEXAO] Novo cliente conectado: %s\n", remoto)

	defer func() {
		fmt.Printf("[DESCONEXAO] Cliente desconectado: %s\n", remoto)
		_ = conn.Close()
	}()

	leitor := bufio.NewReader(conn)

	for {
		linha, err := leitor.ReadString('\n')
		if err != nil {
			// Conexao encerrada pelo cliente ou timeout
			return
		}

		linhaLimpa := strings.TrimSpace(linha)
		if linhaLimpa == "" {
			continue
		}

		// Decodifica o cabecalho basico para identificar a intencao
		var base protocolo.MensagemBase
		if err := json.Unmarshal([]byte(linhaLimpa), &base); err != nil {
			fmt.Printf("[ERRO] Payload malformado de %s: %v\n", remoto, err)
			continue
		}

		// Registra a requisicao no arquivo de log do servidor
		persistencia.RegistrarLog(conn, base.Tipo, linhaLimpa)

		// Roteia para o servico correspondente
		rotearRequisicao(conn, base.Tipo, linhaLimpa, remoto)
	}
}

/**
 * Encaminha o JSON para a funcao de processamento
 */
func rotearRequisicao(conn net.Conn, tipo string, dadosBrutos string, remoto string) {
	switch tipo {

	// Servico de Usuarios (Autenticacao e Cadastro)
	case protocolo.TipoCadastroReq:
		usuarios.ProcessarCadastro(conn, dadosBrutos)

	case protocolo.TipoLoginReq:
		usuarios.ProcessarLogin(conn, dadosBrutos)

	// Servico de Caronas (Motorista)
	case protocolo.TipoPublicarCaronaReq:
		caronas.ProcessarPublicarCarona(conn, dadosBrutos)

	case protocolo.TipoConsultarCaronasReq:
		caronas.ProcessarConsultarCaronas(conn, dadosBrutos)

	case protocolo.TipoCancelarCaronaReq:
		caronas.ProcessarCancelarCarona(conn, dadosBrutos)

	// Servico de Reservas e Itinerarios (Passageiro)
	case protocolo.TipoBuscarItinerariosReq:
		reservas.ProcessarBuscarItinerarios(conn, dadosBrutos)

	case protocolo.TipoReservarItinerarioReq:
		reservas.ProcessarReservarItinerario(conn, dadosBrutos)

	case protocolo.TipoConsultarReservasReq:
		reservas.ProcessarConsultarReservas(conn, dadosBrutos)

	case protocolo.TipoCancelarReservaReq:
		reservas.ProcessarCancelarReserva(conn, dadosBrutos)

	//  Servico de Notificacoes (Passageiro)
	case protocolo.TipoConsultarNotificacoesReq:
		reservas.ProcessarConsultarNotificacoes(conn, dadosBrutos)

	default:
		fmt.Printf("[AVISO] Requisicao desconhecida recebida de %s: '%s'\n", remoto, tipo)

	}
}
