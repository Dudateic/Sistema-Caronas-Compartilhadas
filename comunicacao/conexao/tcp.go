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
 * Encapsula a conexao de rede e os manipuladores de fluxo (stream).
 *
 * @field conn    Socket de rede bruto (canal de comunicacao).
 * @field leitor  Buffer em memoria para leituras de linha/texto.
 * @field encoder Converte a struct para JSON e ja envia no socket com '\n'.
 */
type ClienteTCP struct {
	endereco string
	conn     net.Conn
	leitor   *bufio.Reader
	decoder  *json.Decoder
	encoder  *json.Encoder
}

/**
 * Cria a conexao de rede com controle de tempo limite (timeout).
 *
 * @param endereco Host e porta no formato "ip:porta".
 * @param timeout  Tempo limite opcional para a tentativa de conexao.
 */
func ConectarTCP(endereco string, timeout ...time.Duration) (*ClienteTCP, error) {
	if endereco == "" {
		endereco = EnderecoPadrao
	}

	limite := TimeoutConexaoPadrao
	if len(timeout) > 0 && timeout[0] > 0 {
		limite = timeout[0]
	}

	// DialTimeout evita travamento infinito se o servidor estiver offline
	conn, err := net.DialTimeout("tcp", endereco, limite)
	if err != nil {
		return nil, fmt.Errorf("Falha ao conectar a %s: %w", endereco, err)
	}

	// Vincula o leitor de buffer e os serializadores JSON a conexao
	return &ClienteTCP{
		endereco: endereco,
		conn:     conn,
		leitor:   bufio.NewReader(conn),
		decoder:  json.NewDecoder(conn),
		encoder:  json.NewEncoder(conn),
	}, nil
}

/**
 * Encerra o socket de rede e libera a porta/recurso do sistema.
 */
func (c *ClienteTCP) Fechar() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

/**
 * Tenta restabelecer a conexao com o servidor caso tenha caido (Auto-Healing).
 */
func (c *ClienteTCP) Reconectar() error {
	maxTentativas := 3
	for i := 1; i <= maxTentativas; i++ {
		fmt.Printf("\n[REDE] Tentativa de reconexao automatica %d/%d...\n", i, maxTentativas)

		if c.conn != nil {
			_ = c.conn.Close() // Garante que a conexao zumbi foi fechada
		}

		conn, err := net.DialTimeout("tcp", c.endereco, TimeoutConexaoPadrao)
		if err == nil {
			// Troca os conectores por baixo dos panos e o app nem percebe
			c.conn = conn
			c.leitor = bufio.NewReader(conn)
			c.decoder = json.NewDecoder(conn)
			c.encoder = json.NewEncoder(conn)
			fmt.Println("[REDE] Conexao restabelecida com sucesso! Retomando operacao...")
			return nil
		}
		time.Sleep(2 * time.Second) // Aguarda 2 segundos antes de tentar de novo
	}
	return fmt.Errorf("falha ao reconectar apos %d tentativas", maxTentativas)
}

/**
 * Converte a struct para JSON e transmite pelo socket, anexando '\n' ao final.
 *
 * @param dados Struct ou map a ser transmitido.
 */
func (c *ClienteTCP) EnviarJSON(dados any) error {
	// Encode serializa direto no socket sem alocar buffers
	err := c.encoder.Encode(dados)
	if err != nil {
		fmt.Println("\n[AVISO] Conexao instavel. O servidor pode ter caido ou a rede oscilou.")
		// Tenta curar a conexao
		if errRecon := c.Reconectar(); errRecon != nil {
			return fmt.Errorf("servidor indisponivel no momento: %w", errRecon)
		}

		// Se reconectou, reenvia a mensagem sem o usuario perceber
		if errEncode := c.encoder.Encode(dados); errEncode != nil {
			return fmt.Errorf("falha ao reenviar dados mesmo apos reconectar: %w", errEncode)
		}
	}
	return nil
}

/**
 * Aguarda a chegada de uma mensagem JSON e preenche a struct informada.
 * @param destino Ponteiro para a struct que recebera os dados.
 */
func (c *ClienteTCP) LerEDecodificarJSON(destino any) error {
	// Decode le continuamente do socket e desserializa no destino
	err := c.decoder.Decode(destino)
	if err != nil {
		// Se caiu aguardando a resposta, a resposta foi perdida.
		// Reconectamos para que a proxima acao do menu funcione.
		fmt.Println("\n[AVISO] Conexao perdida aguardando processamento do servidor.")
		_ = c.Reconectar()
		return fmt.Errorf("resposta interrompida: %w", err)
	}
	return nil
}

/**
 * Le texto ate encontrar o delimitador de quebra de linha.
 */
func (c *ClienteTCP) LerLinhaTexto() (string, error) {
	linha, err := c.leitor.ReadString('\n')
	if err != nil {
		fmt.Println("\n[AVISO] Conexao perdida durante leitura de texto.")
		_ = c.Reconectar()
		return "", fmt.Errorf("erro ao ler socket: %w", err)
	}
	// TrimSpace limpa caracteres residuais como \r e \n
	return strings.TrimSpace(linha), nil
}
