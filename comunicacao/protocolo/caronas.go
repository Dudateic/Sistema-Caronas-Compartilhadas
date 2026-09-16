package protocolo

// Tipos de mensagens trocadas nas operacoes de caronas
const (
	TipoPublicarCaronaReq   = "publicar_carona"
	TipoPublicarCaronaRes   = "publicar_carona_resposta"
	TipoConsultarCaronasReq = "consultar_caronas_motorista"
	TipoConsultarCaronasRes = "consultar_caronas_resposta"
	TipoCancelarCaronaReq   = "cancelar_carona"
	TipoCancelarCaronaRes   = "cancelar_carona_resposta"
)

/**
 * Representa um trecho especifico dentro de uma rota de carona
 *
 * @field Origem         Cidade de partida do trecho
 * @field Destino        Cidade de chegada do trecho
 * @field AssentosLivres Quantidade de assentos disponiveis no trecho
 * @field Preco          Valor cobrado por passageiro neste trecho.
 * @field Passageiros    Lista de passageiros com assento confirmado no trecho
 */
type TrechoInfo struct {
	Origem         string   `json:"origem"`
	Destino        string   `json:"destino"`
	AssentosLivres int      `json:"assentos_livres"`
	Preco          float64  `json:"preco"`
	Passageiros    []string `json:"passageiros"`
}

/**
 * Representacao estruturada completa de uma carona
 *
 * @field ID             Identificador unico da carona
 * @field Motorista      Identificador do motorista que cadastrou
 * @field Data           Data da viagem (AAAA-MM-DD)
 * @field Horario        Horario de saida (HH:MM)
 * @field AssentosTotais Capacidade total do veiculo
 * @field PrecoPorTrecho Valor cobrado por trecho percorrido
 * @field Rota           Sequencia ordenada de cidades da viagem
 * @field Trechos        Lista de trechos derivados da rota com controle de vagas
 */
type CaronaDetalhada struct {
	ID             int          `json:"id"`
	Motorista      string       `json:"motorista"`
	Data           string       `json:"data"`
	Horario        string       `json:"horario"`
	AssentosTotais int          `json:"assentos_totais"`
	PrecoPorTrecho float64      `json:"preco_por_trecho"`
	Rota           []string     `json:"rota"`
	Trechos        []TrechoInfo `json:"trechos"`
}

/**
 * Requisicao enviada pelo motorista para registrar nova carona
 */
type PublicarCaronaRequisicao struct {
	Tipo           string   `json:"tipo"`
	Motorista      string   `json:"motorista"`
	Rota           []string `json:"rota"`
	Data           string   `json:"data"`
	Horario        string   `json:"horario"`
	AssentosTotais int      `json:"assentos_totais"`
	PrecoPorTrecho float64  `json:"preco_por_trecho"`
}

/**
 * Resposta retornada apos a criacao de uma carona
 */
type PublicarCaronaResposta struct {
	Tipo     string `json:"tipo"`
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem"`
	CaronaID int    `json:"carona_id,omitempty"`
}

/**
 * Requisicao para consultar caronas publicadas por um motorista
 */
type ConsultarCaronasRequisicao struct {
	Tipo      string `json:"tipo"`
	Motorista string `json:"motorista"`
}

/**
 * Resposta com a lista de caronas do motorista
 */
type ConsultarCaronasResposta struct {
	Tipo     string            `json:"tipo"`
	Sucesso  bool              `json:"sucesso"`
	Mensagem string            `json:"mensagem"`
	Caronas  []CaronaDetalhada `json:"caronas,omitempty"`
}

/**
 * Requisicao para cancelamento de carona pelo motorista
 */
type CancelarCaronaRequisicao struct {
	Tipo      string `json:"tipo"`
	CaronaID  int    `json:"carona_id"`
	Motorista string `json:"motorista"`
}

/**
 * Resposta retornada apos a tentativa de cancelamento
 */
type CancelarCaronaResposta struct {
	Tipo     string `json:"tipo"`
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem"`
}
