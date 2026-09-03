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
	conn    net.Conn
	leitor  *bufio.Reader
	decoder *json.Decoder
	encoder *json.Encoder
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
		conn:    conn,
		leitor:  bufio.NewReader(conn),
		decoder: json.NewDecoder(conn),
		encoder: json.NewEncoder(conn),
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
 * Converte a struct para JSON e transmite pelo socket, anexando '\n' ao final.
 *
 * @param dados Struct ou map a ser transmitido.
 */
func (c *ClienteTCP) EnviarJSON(dados any) error {
	// Encode serializa direto no socket sem alocar buffers manuais
	if err := c.encoder.Encode(dados); err != nil {
		return fmt.Errorf("Erro ao serializar payload: %w", err)
	}
	return nil
}

/**
 * Aguarda a chegada de uma mensagem JSON e preenche a struct informada.
 * @param destino Ponteiro para a struct que recebera os dados.
 */
func (c *ClienteTCP) LerEDecodificarJSON(destino any) error {
	// Decode le continuamente do socket e desserializa no destino
	if err := c.decoder.Decode(destino); err != nil {
		return fmt.Errorf("Erro ao decodificar payload: %w", err)
	}
	return nil
}

/**
 * Le texto ate encontrar o delimitador de quebra de linha.
 */
func (c *ClienteTCP) LerLinhaTexto() (string, error) {
	linha, err := c.leitor.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("Erro ao ler socket: %w", err)
	}
	// TrimSpace limpa caracteres residuais como \r e \n
	return strings.TrimSpace(linha), nil
}
