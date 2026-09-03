package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/caronas"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/reservas"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/usuarios"
)

func main() {
	fmt.Println()
	fmt.Println("         VAIJUNTO   SERVIDOR                ")
	fmt.Println()

	endereco := conexao.EnderecoPadrao
	if len(os.Args) > 1 {
		endereco = os.Args[1] // Ex: go run cmd/servidor/main.go :9000
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

	// Captura interrupcoes do sistema para encerramento gracioso
	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sinais
		fmt.Println()
		fmt.Println("Encerrando o servidor VAIJUNTO...")
		_ = listener.Close()
		os.Exit(0)
	}()

	// Loop principal aceitando conexoes
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Se o listener foi fechado, encerra o loop
			select {
			case <-sinais:
				return
			default:
				fmt.Printf("[ERRO] Falha ao aceitar conexao: %v\n", err)
				continue
			}
		}

		// Processa cada cliente conectado de forma concorrente
		go tratarCliente(conn)
	}
}

/**
 * Atende continuamente as requisicoes de um cliente conectado.
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

		// Roteia para o servico correspondente
		rotearRequisicao(conn, base.Tipo, linhaLimpa, remoto)
	}
}

/**
 * Encaminha o JSON bruto para a funcao de processamento apropriada.
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

	default:
		fmt.Printf("[AVISO] Requisicao desconhecida recebida de %s: '%s'\n", remoto, tipo)
	}
}
