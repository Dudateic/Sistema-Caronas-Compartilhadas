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
	LimparTela()
	fmt.Println()
	fmt.Println("         VAIJUNTO   MODULO DO PASSAGEIRO          ")
	fmt.Println()

	endereco := lerEntrada(fmt.Sprintf("Endereco do servidor TCP (Enter para %s): ", conexao.EnderecoPadrao))
	if endereco == "" {
		endereco = conexao.EnderecoPadrao
	}

	fmt.Printf("Conectando ao servidor em %s...\n", endereco)
	cliente, err := conexao.ConectarTCP(endereco, 5*time.Second)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao conectar: %v\n", err)
		return
	}
	defer cliente.Fechar()
	fmt.Println("Conexao estabelecida com sucesso!")

	// 1. Fluxo de Autenticacao
	passageiroLogado := telaAcesso(cliente)
	if passageiroLogado == "" {
		fmt.Println("Operacao encerrada pelo usuario. Ate logo!")
		return
	}

	// 2. Menu Principal de Operacoes do Passageiro
	menuPrincipalPassageiro(cliente, passageiroLogado)
}

/**
 * Controla o login e o cadastro para o perfil de passageiro.
 */
func telaAcesso(cliente *conexao.ClienteTCP) string {
	for {
		fmt.Println()
		fmt.Println("                   AUTENTICACAO                   ")
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
				fmt.Println("[AVISO] Usuario e senha nao podem ser vazios.")
				continue
			}

			fmt.Println("Validando credenciais...")
			sucesso, err := usuarios.AutenticarCliente(cliente, usuario, senha, protocolo.PerfilPassageiro)
			if err != nil {
				fmt.Printf("[ERRO] Falha na comunicacao com servidor: %v\n", err)
				continue
			}
			if sucesso {
				return usuario
			}

		case "2":
			usuario := lerEntrada("Defina seu nome de usuario: ")
			senha := lerEntrada("Defina sua senha: ")

			if usuario == "" || senha == "" {
				fmt.Println("[AVISO] Preencha usuario e senha para se cadastrar.")
				continue
			}

			fmt.Println("Enviando solicitacao de cadastro...")
			sucesso, err := usuarios.CadastrarCliente(cliente, usuario, senha, protocolo.PerfilPassageiro)
			if err != nil {
				fmt.Printf("[ERRO] Falha ao registrar usuario: %v\n", err)
				continue
			}
			if sucesso {
				fmt.Println("[SUCESSO] Conta de passageiro criada! Faca login na opcao 1.")
			}

		case "0":
			return ""

		default:
			fmt.Println("[AVISO] Opcao invalida, tente novamente.")
		}
	}
}

/**
 * Menu principal interativo do passageiro.
 */
func menuPrincipalPassageiro(cliente *conexao.ClienteTCP, passageiro string) {
	for {
		fmt.Println()
		fmt.Printf("        PAINEL DO PASSAGEIRO: %s        \n", passageiro)
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
			fmt.Println("[AVISO] Opcao invalida. Digite um numero entre 0 e 3.")
		}
	}
}

/**
 * Busca caminhos (DFS) e permite reservar uma das opcoes encontradas.
 */
func acaoBuscarEReservar(cliente *conexao.ClienteTCP, passageiro string) {
	fmt.Println()
	fmt.Println("         BUSCAR ITINERARIOS         ")
	origem := lerEntrada("Cidade de partida (Origem): ")
	destino := lerEntrada("Cidade de chegada (Destino): ")
	data := lerEntrada("Data da viagem (AAAA-MM-DD): ")

	if origem == "" || destino == "" || data == "" {
		fmt.Println("[ERRO] Origem, destino e data sao campos obrigatorios.")
		return
	}

	fmt.Println("\nBuscando rotas disponiveis no servidor...")
	itinerarios, err := reservas.BuscarItinerarios(cliente, origem, destino, data)
	if err != nil {
		fmt.Printf("[ERRO] Falha na busca: %v\n", err)
		return
	}

	fmt.Println("\n               ITINERARIOS ENCONTRADOS                ")
	for i, it := range itinerarios {
		fmt.Printf("\nOpcao [%d] - Preco Total: R$ %.2f\n", i+1, it.PrecoTotal)
		for _, t := range it.Trechos {
			fmt.Printf("   Trecho: %s -> %s | Horario: %s | Motorista: %s | R$ %.2f\n",
				t.Origem, t.Destino, t.Horario, t.Motorista, t.Preco)
		}
	}
	fmt.Println()
	escolhaStr := lerEntrada("Deseja reservar alguma dessas opcoes? Digite o numero da opcao (ou 0 para cancelar): ")
	escolha, err := strconv.Atoi(escolhaStr)
	if err != nil || escolha <= 0 || escolha > len(itinerarios) {
		fmt.Println("Nenhuma reserva efetuada.")
		return
	}

	itinerarioEscolhido := itinerarios[escolha-1]
	fmt.Printf("\nSolicitando reserva da opcao %d para %s...\n", escolha, passageiro)

	sucesso, idReserva, err := reservas.ReservarItinerario(cliente, passageiro, itinerarioEscolhido)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao processar reserva: %v\n", err)
		return
	}

	if sucesso {
		fmt.Printf("[SUCESSO] Reserva confirmada! Codigo da Reserva: #%d\n", idReserva)
	} else {
		fmt.Println("[AVISO] Nao foi possivel concluir a reserva (vagas esgotadas em um dos trechos).")
	}
}

/**
 * Exibe todas as reservas ativas associadas ao passageiro logado.
 */
func acaoConsultarReservas(cliente *conexao.ClienteTCP, passageiro string) {
	fmt.Println()
	fmt.Println("Consultando suas reservas ativas...")

	listaReservas, err := reservas.ConsultarMinhasReservas(cliente, passageiro)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao consultar reservas: %v\n", err)
		return
	}

	if len(listaReservas) == 0 {
		fmt.Println("Voce nao possui nenhuma reserva ativa.")
		return
	}

	// Loop para listar os trechos na tela
	fmt.Println("\n               SUAS RESERVAS                ")
	for _, r := range listaReservas {
		fmt.Printf("\n[Reserva #%d] Data: %s | Preco Total: R$ %.2f\n", r.ID, r.Data, r.PrecoTotal)
		fmt.Println("Trechos reservados:")
		for _, t := range r.Trechos {
			fmt.Printf("  - %s -> %s | Horario: %s | Motorista: %s | R$ %.2f\n",
				t.Origem, t.Destino, t.Horario, t.Motorista, t.Preco)
		}
	}
}

/**
 * Solicita o cancelamento de um bilhete de reserva e a liberacao dos assentos.
 */
func acaoCancelarReserva(cliente *conexao.ClienteTCP, passageiro string) {
	fmt.Println()
	fmt.Println("         CANCELAR RESERVA         ")

	// Busca as reservas do passageiro
	listaReservas, err := reservas.ConsultarMinhasReservas(cliente, passageiro)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao consultar reservas: %v\n", err)
		return
	}

	if len(listaReservas) == 0 {
		fmt.Println("Voce nao possui nenhuma reserva para cancelar.")
		return
	}

	// Lista as reservas na tela mostrando o ID em destaque
	fmt.Println("\nSuas Reservas Ativas:")
	for _, r := range listaReservas {
		origem := r.Trechos[0].Origem
		destino := r.Trechos[len(r.Trechos)-1].Destino
		fmt.Printf(" -> [ID: %d] Data: %s | %s -> %s | R$ %.2f\n",
			r.ID, r.Data, origem, destino, r.PrecoTotal)
	}

	idStr := lerEntrada("Informe o ID da reserva que deseja cancelar (ou 0 para voltar): ")
	id, err := strconv.Atoi(idStr)

	if err != nil || id < 0 {
		fmt.Println("[ERRO] ID invalido.")
		return
	}
	if id == 0 {
		fmt.Println("Cancelamento abortado.")
		return
	}

	confirmacao := lerEntrada(fmt.Sprintf("Tem certeza que deseja cancelar a reserva #%d? (s/N): ", id))
	if strings.ToLower(confirmacao) != "s" {
		fmt.Println("Cancelamento abortado.")
		return
	}

	sucesso, err := reservas.CancelarMinhaReserva(cliente, id, passageiro)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao comunicar cancelamento: %v\n", err)
		return
	}

	if sucesso {
		fmt.Printf("[OK] Reserva #%d cancelada com sucesso e assentos liberados.\n", id)
	}
}

func acaoConsultarNotificacoes(cliente *conexao.ClienteTCP, passageiro string) {
	req := protocolo.ConsultarNotificacoesRequisicao{
		Tipo:       protocolo.TipoConsultarNotificacoesReq,
		Passageiro: passageiro,
	}

	if err := cliente.EnviarJSON(req); err != nil {
		fmt.Printf("[ERRO] Falha ao enviar requisicao: %v\n", err)
		return
	}

	var resp protocolo.ConsultarNotificacoesResposta
	if err := cliente.LerEDecodificarJSON(&resp); err != nil {
		fmt.Printf("[ERRO] Falha ao ler resposta: %v\n", err)
		return
	}

	fmt.Println("\n         CAIXA DE MENSAGENS         ")
	if len(resp.Notificacoes) == 0 {
		fmt.Println("Voce nao possui novas notificacoes.")
		return
	}

	for _, n := range resp.Notificacoes {
		fmt.Printf("-> [%s] %s\n", n.Data, n.Mensagem)
	}
	fmt.Println("\n(Avisos marcados como lidos e apagados da caixa)")
}
