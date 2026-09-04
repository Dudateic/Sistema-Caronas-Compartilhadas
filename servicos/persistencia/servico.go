/**
 * Pacote responsavel pela persistencia em disco no servidor.
 */
package persistencia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Mutex para evitar escritas concorrentes no sistema de arquivos
var mutexArquivo sync.Mutex

// Diretorio base onde os arquivos serao gravados
const DiretorioDados = "dados"

/**
 * Garante que a pasta base exista no servidor.
 */
func init() {
	_ = os.MkdirAll(DiretorioDados, 0755)
}

/**
 * Salva qualquer estrutura de dados em formato JSON indentado no caminho informado.
 *
 * @param nomeArquivo Nome do arquivo (ex: "caronas.json", "reservas.json", "usuarios.json")
 * @param dados       Estrutura (slice, map, struct) a ser serializada
 */
func SalvarJSON(nomeArquivo string, dados any) error {
	mutexArquivo.Lock()
	defer mutexArquivo.Unlock()

	caminho := filepath.Join(DiretorioDados, nomeArquivo)

	// Garante que o diretorio exista
	if err := os.MkdirAll(filepath.Dir(caminho), 0755); err != nil {
		return fmt.Errorf("falha ao criar diretorio de persistencia: %w", err)
	}

	conteudo, err := json.MarshalIndent(dados, "", "  ")
	if err != nil {
		return fmt.Errorf("falha ao serializar dados para JSON: %w", err)
	}

	if err := os.WriteFile(caminho, conteudo, 0644); err != nil {
		return fmt.Errorf("falha ao gravar arquivo %s: %w", caminho, err)
	}

	return nil
}

/**
 * Carrega e desserializa o conteudo de um arquivo JSON no destino apontado.
 * Se o arquivo nao existir, simplesmente nao altera o destino e retorna nil.
 *
 * @param nomeArquivo Nome do arquivo dentro da pasta dados (ex: "caronas.json")
 * @param destino     Ponteiro para a estrutura que recebera os dados
 */
func CarregarJSON(nomeArquivo string, destino any) error {
	mutexArquivo.Lock()
	defer mutexArquivo.Unlock()

	caminho := filepath.Join(DiretorioDados, nomeArquivo)

	if _, err := os.Stat(caminho); os.IsNotExist(err) {
		return nil // Primeira inicializacao: arquivo ainda nao existe
	}

	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		return fmt.Errorf("falha ao ler arquivo %s: %w", caminho, err)
	}

	if len(conteudo) == 0 {
		return nil
	}

	if err := json.Unmarshal(conteudo, destino); err != nil {
		return fmt.Errorf("falha ao desserializar conteudo de %s: %w", caminho, err)
	}

	return nil
}
