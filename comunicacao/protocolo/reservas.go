/**
 * Pacote com as definicoes de mensagens para busca, reserva e cancelamento por passageiros.
 * @author Maria Eduarda
 */
package protocolo

// Tipos de mensagens trocadas nas operacoes de passageiro
const (
	TipoBuscarItinerariosReq  = "buscar_itinerarios"
	TipoBuscarItinerariosRes  = "buscar_itinerarios_resposta"
	TipoReservarItinerarioReq = "reservar_itinerario"
	TipoReservarItinerarioRes = "reservar_resposta"
	TipoConsultarReservasReq  = "consultar_reservas_passageiro"
	TipoConsultarReservasRes  = "consultar_reservas_resposta"
	TipoCancelarReservaReq    = "cancelar_reserva_passageiro"
	TipoCancelarReservaRes    = "cancelar_reserva_resposta"
)

// --- NOVAS MENSAGENS PARA NOTIFICAÇÕES ---

const (
	TipoConsultarNotificacoesReq = "consultar_notificacoes"
	TipoConsultarNotificacoesRes = "consultar_notificacoes_resposta"
)

type Notificacao struct {
	Passageiro string `json:"passageiro"`
	Mensagem   string `json:"mensagem"`
	Data       string `json:"data"`
}

type ConsultarNotificacoesRequisicao struct {
	Tipo       string `json:"tipo"`
	Passageiro string `json:"passageiro"`
}

type ConsultarNotificacoesResposta struct {
	Tipo         string        `json:"tipo"`
	Sucesso      bool          `json:"sucesso"`
	Notificacoes []Notificacao `json:"notificacoes"`
}

/**
 * Representa um segmento de viagem associado a uma carona especifica.
 */
type TrechoItinerario struct {
	CaronaID  int     `json:"carona_id"`
	Motorista string  `json:"motorista"`
	Origem    string  `json:"origem"`
	Destino   string  `json:"destino"`
	Horario   string  `json:"horario"`
	Preco     float64 `json:"preco"`
}

/**
 * Opcao de viagem completa (direta ou combinando múltiplos motoristas).
 */
type Itinerario struct {
	Data       string             `json:"data"`
	PrecoTotal float64            `json:"preco_total"`
	Trechos    []TrechoItinerario `json:"trechos"`
}

/**
 * Registro de confirmacao de reserva de um passageiro.
 */
type ReservaDetalhada struct {
	ID         int                `json:"id"`
	Passageiro string             `json:"passageiro"`
	Data       string             `json:"data"`
	PrecoTotal float64            `json:"preco_total"`
	Trechos    []TrechoItinerario `json:"trechos"`
}

/**
 * Requisicao para consultar rotas disponiveis entre duas cidades.
 */
type BuscarItinerariosRequisicao struct {
	Tipo    string `json:"tipo"`
	Origem  string `json:"origem"`
	Destino string `json:"destino"`
	Data    string `json:"data"`
}

/**
 * Resposta com as opcoes de itinerarios encontrados via algoritmo DFS.
 */
type BuscarItinerariosResposta struct {
	Tipo        string       `json:"tipo"`
	Sucesso     bool         `json:"sucesso"`
	Mensagem    string       `json:"mensagem"`
	Itinerarios []Itinerario `json:"itinerarios,omitempty"`
}

/**
 * Requisicao para efetuar reserva atomica dos trechos escolhidos.
 */
type ReservarRequisicao struct {
	Tipo       string             `json:"tipo"`
	Passageiro string             `json:"passageiro"`
	Data       string             `json:"data"`
	Trechos    []TrechoItinerario `json:"trechos"`
}

/**
 * Resposta da tentativa de reserva atomica.
 */
type ReservarResposta struct {
	Tipo      string `json:"tipo"`
	Sucesso   bool   `json:"sucesso"`
	Mensagem  string `json:"mensagem"`
	ReservaID int    `json:"reserva_id,omitempty"`
}

/**
 * Requisicao para listar as reservas ativas do passageiro.
 */
type ConsultaReservasRequisicao struct {
	Tipo       string `json:"tipo"`
	Passageiro string `json:"passageiro"`
}

/**
 * Resposta com a lista de reservas efetuadas pelo passageiro.
 */
type ConsultaReservasResposta struct {
	Tipo     string             `json:"tipo"`
	Sucesso  bool               `json:"sucesso"`
	Mensagem string             `json:"mensagem"`
	Reservas []ReservaDetalhada `json:"reservas,omitempty"`
}

/**
 * Requisicao de cancelamento de reserva pelo passageiro.
 */
type CancelarReservaRequisicao struct {
	Tipo       string `json:"tipo"`
	ReservaID  int    `json:"reserva_id"`
	Passageiro string `json:"passageiro"`
}

/**
 * Resposta confirmando o cancelamento e liberacao de vagas.
 */
type CancelarReservaResposta struct {
	Tipo     string `json:"tipo"`
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem"`
}
