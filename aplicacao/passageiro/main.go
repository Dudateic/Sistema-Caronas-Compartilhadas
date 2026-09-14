package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/visual"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/reservas"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/usuarios"
)

var scanner = bufio.NewScanner(os.Stdin)

func lerEntrada(rotulo string) string {
	fmt.Print(rotulo)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

/**
 * Identifica o sistema operacional e limpa o terminal.
 */
func LimparTela() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

func main() {
	visual.ExibirCabecalho("VAIJUNTO - MODULO DO PASSAGEIRO")

	endereco := lerEntrada(fmt.Sprintf("Endereco do servidor TCP (Enter para %s): ", conexao.EnderecoPadrao))
	if endereco == "" {
		endereco = conexao.EnderecoPadrao
	}

	fmt.Printf("Conectando ao servidor em %s...\n", endereco)
	cliente, err := conexao.ConectarTCP(endereco, 5*time.Second)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao conectar: %v", err))
		return
	}
	defer cliente.Fechar()
	visual.MensagemSucesso("Conexao estabelecida com sucesso!")
	time.Sleep(1 * time.Second)

	passageiroLogado := telaAcesso(cliente)
	if passageiroLogado == "" {
		fmt.Println("Operacao encerrada pelo usuario. Ate logo!")
		return
	}

	menuPrincipalPassageiro(cliente, passageiroLogado)
}

/**
 * Controla o login e o cadastro para o perfil de passageiro.
 */
func telaAcesso(cliente *conexao.ClienteTCP) string {
	for {
		LimparTela()
		visual.ExibirCabecalho("AUTENTICACAO")
		fmt.Println("1. Entrar (Login)")
		fmt.Println("2. Criar Nova Conta (Cadastro)")
		fmt.Println("0. Encerrar")
		fmt.Println()

		opcao := lerEntrada("Escolha uma opcao: ")

		switch opcao {
		case "1":
			usuario := lerEntrada("Usuario: ")
			senha := lerEntrada("Senha: ")

			if usuario == "" || senha == "" {
				visual.MensagemAviso("Usuario e senha nao podem ser vazios.")
				lerEntrada("\nPressione ENTER para continuar...")
				continue
			}

			fmt.Println("Validando credenciais...")
			sucesso, err := usuarios.AutenticarCliente(cliente, usuario, senha, protocolo.PerfilPassageiro)
			if err != nil {
				visual.MensagemErro(fmt.Sprintf("Falha na comunicacao com servidor: %v", err))
				lerEntrada("\nPressione ENTER para continuar...")
				continue
			}
			if sucesso {
				return usuario
			} else {
				visual.MensagemAviso("Credenciais invalidas.")
				lerEntrada("\nPressione ENTER para continuar...")
			}

		case "2":
			usuario := lerEntrada("Defina seu nome de usuario: ")
			senha := lerEntrada("Defina sua senha: ")

			if usuario == "" || senha == "" {
				visual.MensagemAviso("Preencha usuario e senha para se cadastrar.")
				lerEntrada("\nPressione ENTER para continuar...")
				continue
			}

			fmt.Println("Enviando solicitacao de cadastro...")
			sucesso, err := usuarios.CadastrarCliente(cliente, usuario, senha, protocolo.PerfilPassageiro)
			if err != nil {
				visual.MensagemErro(fmt.Sprintf("Falha ao registrar usuario: %v", err))
				lerEntrada("\nPressione ENTER para continuar...")
				continue
			}
			if sucesso {
				visual.MensagemSucesso("Conta de passageiro criada! Faca login na opcao 1.")
				lerEntrada("\nPressione ENTER para continuar...")
			}

		case "0":
			return ""

		default:
			visual.MensagemAviso("Opcao invalida, tente novamente.")
			lerEntrada("\nPressione ENTER para continuar...")
		}
	}
}

/**
 * Menu principal interativo do passageiro.
 */
func menuPrincipalPassageiro(cliente *conexao.ClienteTCP, passageiro string) {
	// Checa as notificações assim que loga
	LimparTela()
	fmt.Println("\nVerificando caixa de mensagens...")
	acaoConsultarNotificacoesSilenciosa(cliente, passageiro)

	for {
		LimparTela()
		visual.ExibirCabecalho(fmt.Sprintf("PAINEL DO PASSAGEIRO: %s", passageiro))
		fmt.Println("1. Buscar Itinerarios e Reservar")
		fmt.Println("2. Consultar Minhas Reservas")
		fmt.Println("3. Cancelar Reserva")
		fmt.Println("4. Ver Notificacoes de Cancelamento")
		fmt.Println("0. Sair e Desconectar")
		fmt.Println()

		opcao := lerEntrada("Selecione a opcao desejada: ")

		switch opcao {
		case "1":
			acaoBuscarEReservar(cliente, passageiro)
		case "2":
			acaoConsultarReservas(cliente, passageiro)
		case "3":
			acaoCancelarReserva(cliente, passageiro)
		case "4":
			acaoConsultarNotificacoes(cliente, passageiro)
		case "0":
			fmt.Println("Desconectando do servidor... Ate logo!")
			return
		default:
			visual.MensagemAviso("Opcao invalida. Digite um numero entre 0 e 4.")
			lerEntrada("\nPressione ENTER para continuar...")
		}
	}
}

/**
 * Busca caminhos (DFS) e permite reservar uma das opcoes encontradas
 */
func acaoBuscarEReservar(cliente *conexao.ClienteTCP, passageiro string) {
	LimparTela()
	visual.ExibirCabecalho("BUSCAR ITINERARIOS")

	origem := lerEntrada("Cidade de partida (Origem): ")
	destino := lerEntrada("Cidade de chegada (Destino): ")
	data := lerEntrada("Data da viagem (AAAA-MM-DD): ")

	if origem == "" || destino == "" || data == "" {
		visual.MensagemErro("Origem, destino e data sao campos obrigatorios.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	fmt.Println("\nBuscando rotas disponiveis no servidor...")
	itinerarios, err := reservas.BuscarItinerarios(cliente, origem, destino, data)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha na busca: %v", err))
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	if len(itinerarios) == 0 {
		visual.MensagemAviso("Nenhum itinerario disponivel para essa rota e data.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	LimparTela()
	visual.ExibirCabecalho("ITINERARIOS ENCONTRADOS")

	for i, it := range itinerarios {
		fmt.Printf("\n[Opcao %d] - Preco Total: R$ %.2f\n", i+1, it.PrecoTotal)
		trechoCols := []string{"Origem -> Destino", "Horario", "Motorista", "Preco"}
		trechoLarguras := []int{30, 8, 15, 10}
		visual.TabelaCabecalho(trechoCols, trechoLarguras)
		for _, t := range it.Trechos {
			rotaStr := fmt.Sprintf("%s -> %s", t.Origem, t.Destino)
			precoStr := fmt.Sprintf("R$ %.2f", t.Preco)
			visual.TabelaLinha([]string{rotaStr, t.Horario, t.Motorista, precoStr}, trechoLarguras)
		}
		visual.TabelaRodape(trechoLarguras)
	}

	fmt.Println()
	escolhaStr := lerEntrada("Deseja reservar alguma dessas opcoes? Digite o numero da opcao (ou 0 para cancelar): ")
	escolha, err := strconv.Atoi(escolhaStr)
	if err != nil || escolha <= 0 || escolha > len(itinerarios) {
		fmt.Println("Nenhuma reserva efetuada.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	itinerarioEscolhido := itinerarios[escolha-1]
	fmt.Printf("\nSolicitando reserva da opcao %d para %s...\n", escolha, passageiro)

	sucesso, idReserva, err := reservas.ReservarItinerario(cliente, passageiro, itinerarioEscolhido)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao processar reserva: %v", err))
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	if sucesso {
		visual.MensagemSucesso(fmt.Sprintf("Reserva confirmada! Codigo da Reserva: #%d", idReserva))
		lerEntrada("\nPressione ENTER para voltar ao menu...")
	} else {
		visual.MensagemAviso("Nao foi possivel concluir a reserva (vagas esgotadas em um dos trechos).")
		lerEntrada("\nPressione ENTER para voltar ao menu...")
	}
}

/**
 * Exibe todas as reservas ativas associadas ao passageiro logado em formato tabular.
 */
func acaoConsultarReservas(cliente *conexao.ClienteTCP, passageiro string) {
	LimparTela()
	visual.ExibirCabecalho("SUAS RESERVAS ATIVAS")

	listaReservas, err := reservas.ConsultarMinhasReservas(cliente, passageiro)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao consultar reservas: %v", err))
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	if len(listaReservas) == 0 {
		visual.MensagemAviso("Voce nao possui nenhuma reserva ativa.")
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	// Tabela Geral de Reservas
	colunasCabecalho := []string{"ID", "Data da Viagem", "Preco Total"}
	largurasColunas := []int{6, 20, 15}

	visual.TabelaCabecalho(colunasCabecalho, largurasColunas)
	for _, r := range listaReservas {
		visual.TabelaLinha([]string{
			strconv.Itoa(r.ID),
			r.Data,
			fmt.Sprintf("R$ %.2f", r.PrecoTotal),
		}, largurasColunas)
	}
	visual.TabelaRodape(largurasColunas)

	// Detalhes dos trechos de cada reserva
	fmt.Println()
	visual.LinhaDivisoria()
	fmt.Println("DETALHES DOS TRECHOS RESERVADOS:")
	for _, r := range listaReservas {
		fmt.Printf("\n[Reserva #%d]\n", r.ID)
		if len(r.Trechos) > 0 {
			trechoCols := []string{"Trecho (Origem -> Destino)", "Horario", "Motorista", "Preco"}
			trechoLarguras := []int{30, 8, 15, 10}
			visual.TabelaCabecalho(trechoCols, trechoLarguras)
			for _, t := range r.Trechos {
				trechoStr := fmt.Sprintf("%s -> %s", t.Origem, t.Destino)
				precoStr := fmt.Sprintf("R$ %.2f", t.Preco)
				visual.TabelaLinha([]string{trechoStr, t.Horario, t.Motorista, precoStr}, trechoLarguras)
			}
			visual.TabelaRodape(trechoLarguras)
		}
	}

	lerEntrada("\nPressione ENTER para voltar ao menu...")
}

/**
 * Solicita o cancelamento de um bilhete de reserva e a liberacao dos assentos.
 */
func acaoCancelarReserva(cliente *conexao.ClienteTCP, passageiro string) {
	LimparTela()
	visual.ExibirCabecalho("CANCELAR RESERVA")

	listaReservas, err := reservas.ConsultarMinhasReservas(cliente, passageiro)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao consultar reservas: %v", err))
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	if len(listaReservas) == 0 {
		visual.MensagemAviso("Voce nao possui nenhuma reserva para cancelar.")
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	// Tabela para seleção do ID de cancelamento
	colunasCabecalho := []string{"ID", "Data", "Rota", "Preco Total"}
	largurasColunas := []int{4, 12, 25, 12}
	visual.TabelaCabecalho(colunasCabecalho, largurasColunas)
	for _, r := range listaReservas {
		origem := r.Trechos[0].Origem
		destino := r.Trechos[len(r.Trechos)-1].Destino
		rotaStr := fmt.Sprintf("%s -> %s", origem, destino)
		visual.TabelaLinha([]string{
			strconv.Itoa(r.ID),
			r.Data,
			rotaStr,
			fmt.Sprintf("R$ %.2f", r.PrecoTotal),
		}, largurasColunas)
	}
	visual.TabelaRodape(largurasColunas)
	fmt.Println()

	idStr := lerEntrada("Informe o ID da reserva que deseja cancelar (ou 0 para voltar): ")
	id, err := strconv.Atoi(idStr)

	if err != nil || id < 0 {
		visual.MensagemErro("ID invalido.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}
	if id == 0 {
		fmt.Println("Cancelamento abortado.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	confirmacao := lerEntrada(fmt.Sprintf("Tem certeza que deseja cancelar a reserva #%d? (s/N): ", id))
	if strings.ToLower(confirmacao) != "s" {
		fmt.Println("Cancelamento abortado.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	sucesso, err := reservas.CancelarMinhaReserva(cliente, id, passageiro)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao comunicar cancelamento: %v", err))
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	if sucesso {
		visual.MensagemSucesso(fmt.Sprintf("Reserva #%d cancelada com sucesso e assentos liberados.", id))
		lerEntrada("\nPressione ENTER para voltar ao menu...")
	}
}

/**
 * Consulta notificações e pausa para leitura.
 */
func acaoConsultarNotificacoes(cliente *conexao.ClienteTCP, passageiro string) {
	LimparTela()
	visual.ExibirCabecalho("CAIXA DE MENSAGENS")

	req := protocolo.ConsultarNotificacoesRequisicao{
		Tipo:       protocolo.TipoConsultarNotificacoesReq,
		Passageiro: passageiro,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao enviar requisicao: %v", err))
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	var resp protocolo.ConsultarNotificacoesResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao ler resposta: %v", err))
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	if len(resp.Notificacoes) == 0 {
		visual.MensagemAviso("Voce nao possui novas notificacoes.")
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	for _, n := range resp.Notificacoes {
		fmt.Printf("\n[Recebido em: %s]\n-> %s\n", n.Data, n.Mensagem)
		visual.LinhaDivisoria()
	}

	fmt.Println("\n(Avisos marcados como lidos e apagados da caixa)")
	lerEntrada("\nPressione ENTER para voltar ao menu...")
}

/**
 * Consulta notificações no login
 */
func acaoConsultarNotificacoesSilenciosa(cliente *conexao.ClienteTCP, passageiro string) {
	req := protocolo.ConsultarNotificacoesRequisicao{
		Tipo:       protocolo.TipoConsultarNotificacoesReq,
		Passageiro: passageiro,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		return
	}

	var resp protocolo.ConsultarNotificacoesResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		return
	}

	if len(resp.Notificacoes) > 0 {
		visual.ExibirCabecalho("NOVAS NOTIFICACOES")
		for _, n := range resp.Notificacoes {
			fmt.Printf("\n[Recebido em: %s]\n-> %s\n", n.Data, n.Mensagem)
		}
		visual.LinhaDivisoria()
		lerEntrada("\nPressione ENTER para ir para o Painel Principal...")
	}
}
