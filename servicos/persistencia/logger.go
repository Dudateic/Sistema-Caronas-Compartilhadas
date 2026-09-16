package persistencia

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const ArquivoLogRequisicoes = "requisicoes.log"

var mutexLog sync.Mutex

/**
 * Registra uma requisicao recebida pelo servidor com timestamp e IP do cliente
 * Grava em dados/requisicoes.log e exibe no console do servidor
 *
 * @param conn        Conexao TCP de onde veio a requisicao (para capturar IP/porta)
 * @param tipo        Tipo de mensagem (ex: login, buscar_itinerarios, publicar_carona)
 * @param dadosBrutos JSON ou conteudo bruto recebido do socket
 */
func RegistrarLog(conn net.Conn, tipo string, dadosBrutos string) {
	mutexLog.Lock()
	defer mutexLog.Unlock()

	agora := time.Now().Format("2006-01-02 15:04:05")
	origem := "desconhecido"
	if conn != nil && conn.RemoteAddr() != nil {
		origem = conn.RemoteAddr().String()
	}

	linhaLog := fmt.Sprintf("[%s] [ORIGEM: %s] [TIPO: %s] PAYLOAD: %s\n", agora, origem, tipo, dadosBrutos)

	// Exibe no console do servidor
	fmt.Print(linhaLog)

	// Grava no arquivo de persistencia
	caminho := filepath.Join(DiretorioDados, ArquivoLogRequisicoes)
	_ = os.MkdirAll(filepath.Dir(caminho), 0755)

	f, err := os.OpenFile(caminho, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("[ERRO LOG] Falha ao abrir %s: %v\n", caminho, err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(linhaLog); err != nil {
		fmt.Printf("[ERRO LOG] Falha ao escrever log: %v\n", err)
	}
}
