package usuarios

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"Sistema-de-caronas-compartilhadas/servicos/persistencia"
)

var mutexUsuarios sync.Mutex

const ArquivoUsuarios = "usuarios.json"

/**
 * Estrutura interna para salvar os usuarios no arquivo JSON
 */
type UsuarioCadastrado struct {
	TipoUsuario string `json:"tipo"`
	IDUsuario   string `json:"id_usuario"`
	Usuario     string `json:"usuario"`
	Senha       string `json:"senha"`
}

// hashSimples aplica a formula (x² + 1) acumulando cada caractere
func hashSimples(senha string) string {
	var total uint64 = 0

	for i := 0; i < len(senha); i++ {
		x := uint64(senha[i]) // Valor numerico do caractere
		total += (x*x + 1)    // Aplica x² + 1 e soma
	}

	return fmt.Sprintf("%d", total)
}

// gerarID gera um identificador hexadecimal unico de 8 bytes
func gerarID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

/**
 * Carrega a lista de contas salvas utilizando o servico de persistencia
 */
func carregarUsuariosDoBanco() ([]UsuarioCadastrado, error) {
	var listaUsuarios []UsuarioCadastrado
	if err := persistencia.CarregarJSON(ArquivoUsuarios, &listaUsuarios); err != nil {
		return nil, fmt.Errorf("falha ao carregar usuarios via persistencia: %w", err)
	}
	return listaUsuarios, nil
}

/**
 * Grava a lista atualizada de usuarios no arquivo via servico de persistencia
 */
func salvarUsuariosNoBanco(usuarios []UsuarioCadastrado) error {
	if err := persistencia.SalvarJSON(ArquivoUsuarios, usuarios); err != nil {
		return fmt.Errorf("falha ao persistir lista de usuarios: %w", err)
	}
	return nil
}

/**
 * Recebe a solicitacao de novo usuario, confere se o nome ja existe e salva no arquivo
 *
 * @param conn        Conexao do cliente que enviou os dados
 * @param dadosBrutos Texto JSON com o pedido de cadastro
 */
func ProcessarCadastro(conn net.Conn, dadosBrutos string) {
	var req protocolo.CadastroRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		responderCadastro(conn, false, "Formato de cadastro invalido")
		return
	}

	if req.Usuario == "" || req.Senha == "" {
		responderCadastro(conn, false, "Usuario e senha nao podem ser vazios")
		return
	}

	if req.TipoUsuario != protocolo.PerfilMotorista && req.TipoUsuario != protocolo.PerfilPassageiro {
		responderCadastro(conn, false, "Perfil de usuario invalido")
		return
	}

	mutexUsuarios.Lock()
	defer mutexUsuarios.Unlock()

	usuarios, err := carregarUsuariosDoBanco()
	if err != nil {
		fmt.Printf("[CADASTRO] Erro ao abrir banco: %v\n", err)
		responderCadastro(conn, false, "Erro interno ao consultar banco de dados")
		return
	}

	for _, u := range usuarios {
		if u.Usuario == req.Usuario {
			responderCadastro(conn, false, "Nome de usuario ja em uso")
			return
		}
	}

	novoUsuario := UsuarioCadastrado{
		IDUsuario:   gerarID(),
		TipoUsuario: req.TipoUsuario,
		Usuario:     req.Usuario,
		Senha:       hashSimples(req.Senha),
	}

	usuarios = append(usuarios, novoUsuario)
	if err := salvarUsuariosNoBanco(usuarios); err != nil {
		fmt.Printf("[CADASTRO] Erro ao gravar banco: %v\n", err)
		responderCadastro(conn, false, "Erro interno ao salvar novo usuario")
		return
	}

	fmt.Printf("[CADASTRO] Novo usuario registrado: '%s' (%s)\n", req.Usuario, req.TipoUsuario)
	responderCadastro(conn, true, "Usuario cadastrado com sucesso!")
}

/**
 * Envia de volta para o cliente o resultado da criacao de conta
 */
func responderCadastro(conn net.Conn, sucesso bool, mensagem string) {
	resp := protocolo.CadastroResposta{
		Tipo:     protocolo.TipoCadastroRes,
		Sucesso:  sucesso,
		Mensagem: mensagem,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Confere o usuario e a senha informados na lista de contas e autoriza a entrada
 *
 * @param conn        Conexao do cliente que enviou o login
 * @param dadosBrutos Texto JSON com as credenciais
 */
func ProcessarLogin(conn net.Conn, dadosBrutos string) {
	var req protocolo.LoginRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		responderLogin(conn, false, "Formato de login invalido", "", "")
		return
	}

	fmt.Printf("[AUTH] Tentativa de login: Usuario='%s'\n", req.Usuario)

	mutexUsuarios.Lock()
	usuarios, err := carregarUsuariosDoBanco()
	mutexUsuarios.Unlock()

	if err != nil {
		fmt.Printf("[AUTH] Erro ao carregar base de dados: %v\n", err)
		responderLogin(conn, false, "Erro interno no servidor", "", "")
		return
	}

	senhaCalculada := hashSimples(req.Senha)
	var usuarioLogado *UsuarioCadastrado

	for i := range usuarios {
		if usuarios[i].Usuario == req.Usuario && usuarios[i].Senha == senhaCalculada {
			usuarioLogado = &usuarios[i]
			break
		}
	}

	if usuarioLogado != nil {
		msg := fmt.Sprintf("Bem-vindo(a), %s!", usuarioLogado.Usuario)
		responderLogin(conn, true, msg, usuarioLogado.TipoUsuario, usuarioLogado.IDUsuario)
		fmt.Printf("[AUTH] Login aprovado: '%s' (%s)\n", usuarioLogado.Usuario, usuarioLogado.TipoUsuario)
	} else {
		responderLogin(conn, false, "Usuario ou senha incorretos", "", "")
		fmt.Printf("[AUTH] Login rejeitado para: '%s'\n", req.Usuario)
	}
}

/**
 * Envia de volta para o cliente a resposta de login informando se entrou, o perfil e o ID
 */
func responderLogin(conn net.Conn, sucesso bool, mensagem string, tipoUsuario string, idUsuario string) {
	resp := protocolo.LoginResposta{
		Tipo:        protocolo.TipoLoginRes,
		IDUsuario:   idUsuario,
		Sucesso:     sucesso,
		Mensagem:    mensagem,
		TipoUsuario: tipoUsuario,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Envia os dados de cadastro para o servidor e aguarda a confirmacao
 *
 * @param cliente Instancia de conexao com o servidor
 * @param usuario Nome de usuario escolhido
 * @param senha   Senha definida
 * @param perfil  Tipo de conta ("motorista" ou "passageiro")
 */
func CadastrarCliente(cliente *conexao.ClienteTCP, usuario string, senha string, perfil string) (bool, error) {
	req := protocolo.CadastroRequisicao{
		Tipo:        protocolo.TipoCadastroReq,
		Usuario:     usuario,
		Senha:       senha,
		TipoUsuario: perfil,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, fmt.Errorf("Erro ao transmitir dados de cadastro: %w", err)
	}

	var resp protocolo.CadastroResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, fmt.Errorf("Erro ao ler resposta do servidor: %w", err)
	}

	if !resp.Sucesso {
		fmt.Printf("Falha no cadastro: %s\n", resp.Mensagem)
		return false, nil
	}

	fmt.Printf("%s\n", resp.Mensagem)
	return true, nil
}

/**
 * Envia o login para o servidor e confere se a conta bate com o perfil esperado
 *
 * @param cliente        Instancia de conexao com o servidor
 * @param usuario        Nome do usuario
 * @param senha          Senha de acesso
 * @param perfilEsperado Perfil exigido ("motorista" ou "passageiro")
 */
func AutenticarCliente(cliente *conexao.ClienteTCP, usuario string, senha string, perfilEsperado string) (bool, error) {
	req := protocolo.LoginRequisicao{
		Tipo:    protocolo.TipoLoginReq,
		Usuario: usuario,
		Senha:   senha,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, fmt.Errorf("Erro ao transmitir credenciais: %w", err)
	}

	var resp protocolo.LoginResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, fmt.Errorf("Erro ao ler resposta do servidor: %w", err)
	}

	if !resp.Sucesso {
		fmt.Printf("Falha no login: %s\n", resp.Mensagem)
		return false, nil
	}

	if resp.TipoUsuario != perfilEsperado {
		fmt.Printf("Acesso negado: perfil '%s' nao autorizado (esperado '%s').\n", resp.TipoUsuario, perfilEsperado)
		return false, nil
	}

	fmt.Printf("%s\n", resp.Mensagem)
	return true, nil
}
