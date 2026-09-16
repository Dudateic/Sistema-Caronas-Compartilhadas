/**
 * Pacote de comunicacao cliente-servidor via socket TCP.
 * @author Maria Eduarda
 */
package conexao

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	// Host e porta padrao para conexao.
	EnderecoPadrao = ":8081"

	// Tempo limite maximo para aguardar a conexao antes de abortar.
	TimeoutConexaoPadrao = 5 * time.Second
)

/**
 * Encapsula a conexao de rede e os manipuladores de fluxo (stream)
 */
type ClienteTCP struct {
	endereco string
	conn     net.Conn
	leitor   *bufio.Reader
	decoder  *json.Decoder
	encoder  *json.Encoder
}

/**
 * Cria a conexao de rede com controle de tempo limite (timeout)
 */
func ConectarTCP(endereco string, timeout ...time.Duration) (*ClienteTCP, error) {
	if endereco == "" {
		endereco = EnderecoPadrao
	}

	limite := TimeoutConexaoPadrao
	if len(timeout) > 0 && timeout[0] > 0 {
		limite = timeout[0]
	}

	conn, err := net.DialTimeout("tcp", endereco, limite)
	if err != nil {
		return nil, fmt.Errorf("Falha ao conectar a %s: %w", endereco, err)
	}

	return &ClienteTCP{
		endereco: endereco,
		conn:     conn,
		leitor:   bufio.NewReader(conn),
		decoder:  json.NewDecoder(conn),
		encoder:  json.NewEncoder(conn),
	}, nil
}

/**
 * Encerra o socket de rede e libera a porta/recurso do sistema
 */
func (c *ClienteTCP) Fechar() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

/**
 * Tenta restabelecer a conexao com o servidor caso tenha caido
 */
func (c *ClienteTCP) Reconectar() error {
	maxTentativas := 3
	for i := 1; i <= maxTentativas; i++ {
		fmt.Printf("\n[REDE] Tentativa de reconexao automatica %d/%d...\n", i, maxTentativas)

		if c.conn != nil {
			_ = c.conn.Close() // Fecha o socket forçadamente
		}

		// Timeout mais curto (1s) nas tentativas de reconexao para nao travar o usuario
		conn, err := net.DialTimeout("tcp", c.endereco, 1*time.Second)
		if err == nil {
			c.conn = conn
			c.leitor = bufio.NewReader(conn)
			c.decoder = json.NewDecoder(conn)
			c.encoder = json.NewEncoder(conn)
			fmt.Println("[REDE] Conexao restabelecida com sucesso! Retomando operacao...")
			return nil
		}
		time.Sleep(1 * time.Second) // Aguarda 1 segundo antes de tentar de novo
	}
	return fmt.Errorf("falha ao reconectar apos %d tentativas", maxTentativas)
}

/**
 * Converte a struct para JSON e transmite pelo socket, anexando '\n' ao final
 */
func (c *ClienteTCP) EnviarJSON(dados any) error {
	// Garante que o encoder aponte para o socket atual (caso tenha ocorrido reconexao previa)
	c.encoder = json.NewEncoder(c.conn)

	err := c.encoder.Encode(dados)
	if err != nil {
		fmt.Println("\n[AVISO] Conexao instavel detectada no envio. Tentando reconectar...")

		if errRecon := c.Reconectar(); errRecon != nil {
			return fmt.Errorf("servidor indisponivel no momento: %w", errRecon)
		}

		// Atualiza o encoder com o novo socket recem-criado
		c.encoder = json.NewEncoder(c.conn)
		if errEncode := c.encoder.Encode(dados); errEncode != nil {
			return fmt.Errorf("falha ao reenviar dados mesmo apos reconectar: %w", errEncode)
		}
	}
	return nil
}

/**
 * Aguarda a chegada de uma mensagem JSON e preenche a struct informada
 */
func (c *ClienteTCP) LerEDecodificarJSON(destino any) error {
	c.decoder = json.NewDecoder(c.conn)

	err := c.decoder.Decode(destino)
	if err != nil {
		fmt.Println("\n[AVISO] Conexao perdida aguardando resposta. Tentando reconectar...")
		if errRecon := c.Reconectar(); errRecon != nil {
			return fmt.Errorf("resposta interrompida e falha ao reconectar: %w", errRecon)
		}
		return fmt.Errorf("resposta interrompida, conexao reestabelecida (favor repetir a acao): %w", err)
	}
	return nil
}

/**
 * Le texto ate encontrar o delimitador de quebra de linha
 */
func (c *ClienteTCP) LerLinhaTexto() (string, error) {
	linha, err := c.leitor.ReadString('\n')
	if err != nil {
		fmt.Println("\n[AVISO] Conexao perdida durante leitura de texto.")
		_ = c.Reconectar()
		return "", fmt.Errorf("erro ao ler socket: %w", err)
	}
	return strings.TrimSpace(linha), nil
}
