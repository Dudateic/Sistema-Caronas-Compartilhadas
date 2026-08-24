/**
 * Pacote com as definicoes de mensagens e perfis de usuarios.
 *
 * @author Maria Eduarda
 */
package protocolo

// Tipos de mensagens trocadas no fluxo de autenticacao e cadastro
const (
	TipoLoginReq    = "login"
	TipoLoginRes    = "login_resposta"
	TipoCadastroReq = "cadastro"
	TipoCadastroRes = "cadastro_resposta"
)

// Perfis de usuario aceitos pelo sistema
const (
	PerfilMotorista  = "motorista"
	PerfilPassageiro = "passageiro"
)

/**
 * Usada para ler apenas o cabecalho da mensagem e decidir qual struct processar.
 *
 * @field Tipo Identificador da acao (ex: "login", "cadastro").
 */
type MensagemBase struct {
	Tipo string `json:"tipo"`
}

/**
 * Dados enviados pelo cliente para autenticar no sistema.
 *
 * @field Tipo    Identificador da requisicao ("login").
 * @field Usuario Identificador ou nome do usuario.
 * @field Senha   Senha informada para validacao.
 */
type LoginRequisicao struct {
	Tipo    string `json:"tipo"`
	Usuario string `json:"usuario"`
	Senha   string `json:"senha"`
}

/**
 * Resposta devolvida pelo servidor apos a tentativa de login.
 *
 * @field Tipo        Identificador da resposta ("login_resposta").
 * @field Sucesso     Indica se o login foi aceito.
 * @field Mensagem    Texto informativo ou descricao de erro.
 * @field TipoUsuario Perfil do usuario ("motorista" ou "passageiro").
 */
type LoginResposta struct {
	Tipo        string `json:"tipo"`
	Sucesso     bool   `json:"sucesso"`
	Mensagem    string `json:"mensagem"`
	TipoUsuario string `json:"tipo_usuario,omitempty"`
}

/**
 * Requisicao enviada pelo cliente para cadastrar um novo usuario.
 *
 * @field Tipo        Identificador da requisicao ("cadastro").
 * @field Usuario     Nome ou identificador do novo usuario.
 * @field Senha       Senha definida para a conta.
 * @field TipoUsuario Perfil a ser atribuido ("motorista" ou "passageiro").
 */
type CadastroRequisicao struct {
	Tipo        string `json:"tipo"`
	Usuario     string `json:"usuario"`
	Senha       string `json:"senha"`
	TipoUsuario string `json:"tipo_usuario"`
}

/**
 * Resposta retornada pelo servidor apos a tentativa de cadastro.
 *
 * @field Tipo     Identificador da resposta ("cadastro_resposta").
 * @field Sucesso  Indica se o cadastro foi realizado.
 * @field Mensagem Descricao do status ou motivo do erro.
 */
type CadastroResposta struct {
	Tipo     string `json:"tipo"`
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem"`
}
