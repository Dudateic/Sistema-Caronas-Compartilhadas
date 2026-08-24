/**
 * Pacote responsavel pelas regras de autenticacao e persistencia de usuarios.
 *
 * @author Maria Eduarda
 */
package usuarios

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
)

var mutexArquivo sync.Mutex

const caminhoBancoUsuarios = "usuarios.json"

/**
 * Estrutura interna para salvar os usuarios no arquivo JSON.
 */
type UsuarioCadastrado struct {
	TipoUsuario string `json:"tipo"`
	Usuario     string `json:"usuario"`
	Senha       string `json:"senha"`
}

/**
 * Abre o arquivo usuarios.json e carrega a lista de contas salvas.
 * Se o arquivo ainda nao existir, retorna uma lista vazia.
 */
func carregarUsuariosDoBanco() ([]UsuarioCadastrado, error) {
	dados, err := os.ReadFile(caminhoBancoUsuarios)
	if err != nil {
		if os.IsNotExist(err) {
			return []UsuarioCadastrado{}, nil
		}
		return nil, fmt.Errorf("falha ao abrir %s: %w", caminhoBancoUsuarios, err)
	}

	var listaUsuarios []UsuarioCadastrado
	if err := json.Unmarshal(dados, &listaUsuarios); err != nil {
		return nil, fmt.Errorf("falha ao decodificar %s: %w", caminhoBancoUsuarios, err)
	}

	return listaUsuarios, nil
}

/**
 * Grava a lista atualizada de usuarios no arquivo usuarios.json.
 */
func salvarUsuariosNoBanco(usuarios []UsuarioCadastrado) error {
	dados, err := json.MarshalIndent(usuarios, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar lista de usuarios: %w", err)
	}
	return os.WriteFile(caminhoBancoUsuarios, dados, 0644)
}

/**
 * Recebe a solicitacao de novo usuario, confere se o nome ja existe e salva no arquivo.
 *
 * @param conn        Conexao do cliente que enviou os dados.
 * @param dadosBrutos Texto JSON com o pedido de cadastro.
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

	mutexArquivo.Lock()
	defer mutexArquivo.Unlock()

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
		TipoUsuario: req.TipoUsuario,
		Usuario:     req.Usuario,
		Senha:       req.Senha,
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
 * Envia de volta para o cliente o resultado da criacao de conta.
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
 * Confere o usuario e a senha informados na lista de contas e autoriza a entrada.
 *
 * @param conn        Conexao do cliente que enviou o login.
 * @param dadosBrutos Texto JSON com as credenciais.
 */
func ProcessarLogin(conn net.Conn, dadosBrutos string) {
	var req protocolo.LoginRequisicao
	if err := json.Unmarshal([]byte(dadosBrutos), &req); err != nil {
		responderLogin(conn, false, "Formato de login invalido", "")
		return
	}

	fmt.Printf("[AUTH] Tentativa de login: Usuario='%s'\n", req.Usuario)

	mutexArquivo.Lock()
	usuarios, err := carregarUsuariosDoBanco()
	mutexArquivo.Unlock()

	if err != nil {
		fmt.Printf("[AUTH] Erro ao carregar base de dados: %v\n", err)
		responderLogin(conn, false, "Erro interno no servidor", "")
		return
	}

	var usuarioLogado *UsuarioCadastrado
	for _, u := range usuarios {
		if u.Usuario == req.Usuario && u.Senha == req.Senha {
			usuarioLogado = &u
			break
		}
	}

	if usuarioLogado != nil {
		msg := fmt.Sprintf("Bem-vindo(a), %s!", usuarioLogado.Usuario)
		responderLogin(conn, true, msg, usuarioLogado.TipoUsuario)
		fmt.Printf("[AUTH] Login aprovado: '%s' (%s)\n", usuarioLogado.Usuario, usuarioLogado.TipoUsuario)
	} else {
		responderLogin(conn, false, "Usuario ou senha incorretos", "")
		fmt.Printf("[AUTH] Login rejeitado para: '%s'\n", req.Usuario)
	}
}

/**
 * Envia de volta para o cliente a resposta de login informando se entrou e qual o perfil.
 */
func responderLogin(conn net.Conn, sucesso bool, mensagem string, tipoUsuario string) {
	resp := protocolo.LoginResposta{
		Tipo:        protocolo.TipoLoginRes,
		Sucesso:     sucesso,
		Mensagem:    mensagem,
		TipoUsuario: tipoUsuario,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

/**
 * Envia os dados de cadastro para o servidor e aguarda a confirmacao.
 *
 * @param cliente Instancia de conexao com o servidor.
 * @param usuario Nome de usuario escolhido.
 * @param senha   Senha definida.
 * @param perfil  Tipo de conta ("motorista" ou "passageiro").
 */
func CadastrarCliente(cliente *conexao.ClienteTCP, usuario string, senha string, perfil string) (bool, error) {
	req := protocolo.CadastroRequisicao{
		Tipo:        protocolo.TipoCadastroReq,
		Usuario:     usuario,
		Senha:       senha,
		TipoUsuario: perfil,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, fmt.Errorf("erro ao transmitir dados de cadastro: %w", err)
	}

	var resp protocolo.CadastroResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, fmt.Errorf("erro ao ler resposta do servidor: %w", err)
	}

	if !resp.Sucesso {
		fmt.Printf("Falha no cadastro: %s\n", resp.Mensagem)
		return false, nil
	}

	fmt.Printf("%s\n", resp.Mensagem)
	return true, nil
}

/**
 * Envia o login para o servidor e confere se a conta bate com o perfil esperado.
 *
 * @param cliente        Instancia de conexao com o servidor.
 * @param usuario        Nome do usuario.
 * @param senha          Senha de acesso.
 * @param perfilEsperado Perfil exigido ("motorista" ou "passageiro").
 */
func AutenticarCliente(cliente *conexao.ClienteTCP, usuario string, senha string, perfilEsperado string) (bool, error) {
	req := protocolo.LoginRequisicao{
		Tipo:    protocolo.TipoLoginReq,
		Usuario: usuario,
		Senha:   senha,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return false, fmt.Errorf("erro ao transmitir credenciais: %w", err)
	}

	var resp protocolo.LoginResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return false, fmt.Errorf("erro ao ler resposta do servidor: %w", err)
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
