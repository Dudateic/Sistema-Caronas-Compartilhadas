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
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/caronas"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/usuarios"
)

var scanner = bufio.NewScanner(os.Stdin)

// lerEntrada le uma linha do terminal e remove espacos residuais
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
	visual.ExibirCabecalho("VAIJUNTO - MODULO DO MOTORISTA")

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

	// Fluxo de Autenticacao
	motoristaLogado := telaAcesso(cliente)
	if motoristaLogado == "" {
		fmt.Println("Operacao encerrada pelo usuario. Ate logo!")
		return
	}

	// Menu Principal de Operacoes
	menuPrincipalMotorista(cliente, motoristaLogado)
}

/**
 * Controla as opcoes de login e novo cadastro para o perfil de motorista.
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
			sucesso, err := usuarios.AutenticarCliente(cliente, usuario, senha, protocolo.PerfilMotorista)
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
			sucesso, err := usuarios.CadastrarCliente(cliente, usuario, senha, protocolo.PerfilMotorista)
			if err != nil {
				visual.MensagemErro(fmt.Sprintf("Falha ao registrar usuario: %v", err))
				lerEntrada("\nPressione ENTER para continuar...")
				continue
			}
			if sucesso {
				visual.MensagemSucesso("Conta criada com sucesso! Faca login na opcao 1.")
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
 * Loop interativo com as opcoes de gerenciamento de viagens do motorista.
 */
func menuPrincipalMotorista(cliente *conexao.ClienteTCP, motorista string) {
	for {
		LimparTela()
		visual.ExibirCabecalho(fmt.Sprintf("PAINEL DO MOTORISTA: %s", motorista))
		fmt.Println("1. Publicar Nova Carona")
		fmt.Println("2. Consultar Minhas Caronas")
		fmt.Println("3. Cancelar Carona")
		fmt.Println("0. Sair e Desconectar")
		fmt.Println()

		opcao := lerEntrada("Selecione a opcao desejada: ")

		switch opcao {
		case "1":
			acaoPublicarCarona(cliente, motorista)
		case "2":
			acaoConsultarCaronas(cliente, motorista)
		case "3":
			acaoCancelarCarona(cliente, motorista)
		case "0":
			fmt.Println("Desconectando do servidor... Ate logo!")
			return
		default:
			visual.MensagemAviso("Opcao invalida. Digite um numero entre 0 e 3.")
			lerEntrada("\nPressione ENTER para continuar...")
		}
	}
}

/**
 * Coleta os dados de trajeto e publica uma nova oferta no servidor.
 */
func acaoPublicarCarona(cliente *conexao.ClienteTCP, motorista string) {
	LimparTela()
	visual.ExibirCabecalho("PUBLICAR NOVA CARONA")
	fmt.Println("Informe as cidades da rota em ordem, separadas por virgula.")
	fmt.Println("Exemplo: Salvador, Feira de Santana, Serrinha")
	visual.LinhaDivisoria()

	rotaTexto := lerEntrada("Rota: ")

	partes := strings.Split(rotaTexto, ",")
	var rota []string
	for _, p := range partes {
		cidade := strings.TrimSpace(p)
		if cidade != "" {
			rota = append(rota, cidade)
		}
	}

	if len(rota) < 2 {
		visual.MensagemErro("Uma rota valida precisa de pelo menos 2 cidades (Origem e Destino).")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	data := lerEntrada("Data da viagem (AAAA-MM-DD): ")
	if strings.TrimSpace(data) == "" {
		visual.MensagemErro("A data nao pode ser vazia.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	horario := lerEntrada("Horario de saida (HH:MM): ")
	if strings.TrimSpace(horario) == "" {
		visual.MensagemErro("O horario nao pode ser vazio.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	assentosStr := lerEntrada("Quantidade total de vagas no veiculo: ")
	assentos, err := strconv.Atoi(assentosStr)
	if err != nil || assentos <= 0 {
		visual.MensagemErro("Quantidade de assentos deve ser um numero inteiro maior que zero.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	precoStr := lerEntrada("Preco cobrado por trecho em R$ (ex: 25.50): ")
	precoStr = strings.ReplaceAll(precoStr, ",", ".")
	preco, err := strconv.ParseFloat(precoStr, 64)
	if err != nil || preco <= 0 {
		visual.MensagemErro("Preco invalido. Digite um valor monetario positivo.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	fmt.Println("\nPublicando carona no servidor...")
	idCarona, err := caronas.PublicarCarona(cliente, motorista, rota, data, horario, assentos, preco)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao enviar carona: %v", err))
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	if idCarona > 0 {
		visual.MensagemSucesso(fmt.Sprintf("Carona #%d cadastrada e aberta para reservas!", idCarona))
		lerEntrada("\nPressione ENTER para voltar ao menu...")
	}
}

/**
 * Consulta e formata as caronas publicadas pelo motorista usando tabelas padronizadas.
 */
func acaoConsultarCaronas(cliente *conexao.ClienteTCP, motorista string) {
	LimparTela()
	visual.ExibirCabecalho("SUAS CARONAS CADASTRADAS")

	listaCaronas, err := caronas.ConsultarCaronas(cliente, motorista)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao consultar caronas: %v", err))
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	if len(listaCaronas) == 0 {
		visual.MensagemAviso("Voce nao possui nenhuma carona cadastrada.")
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	// Exibição em Tabela Padronizada
	colunasCabecalho := []string{"ID", "Data / Horario", "Rota Completa"}
	largurasColunas := []int{4, 20, 35}

	visual.TabelaCabecalho(colunasCabecalho, largurasColunas)

	for _, c := range listaCaronas {
		dataHora := fmt.Sprintf("%s as %s", c.Data, c.Horario)
		rotaCompleta := strings.Join(c.Rota, " -> ")

		visual.TabelaLinha([]string{
			strconv.Itoa(c.ID),
			dataHora,
			rotaCompleta,
		}, largurasColunas)
	}
	visual.TabelaRodape(largurasColunas)

	// Detalhes dos trechos e preços de forma limpa abaixo da tabela principal
	fmt.Println()
	visual.LinhaDivisoria()
	fmt.Println("DETALHES DOS TRECHOS POR CARONA:")
	for _, c := range listaCaronas {
		fmt.Printf("\n[Carona #%d]\n", c.ID)
		if len(c.Trechos) > 0 {
			trechoCols := []string{"Trecho (Origem -> Destino)", "Preco por Trecho"}
			trechoLarguras := []int{35, 15}
			visual.TabelaCabecalho(trechoCols, trechoLarguras)
			for _, t := range c.Trechos {
				trechoStr := fmt.Sprintf("%s -> %s", t.Origem, t.Destino)
				precoStr := fmt.Sprintf("R$ %.2f", t.Preco)
				visual.TabelaLinha([]string{trechoStr, precoStr}, trechoLarguras)
			}
			visual.TabelaRodape(trechoLarguras)
		}
	}

	lerEntrada("\nPressione ENTER para voltar ao menu...")
}

/**
 * Solicita o cancelamento de uma carona pertencente ao motorista.
 */
func acaoCancelarCarona(cliente *conexao.ClienteTCP, motorista string) {
	LimparTela()
	visual.ExibirCabecalho("CANCELAR CARONA")

	lista, err := caronas.ConsultarCaronas(cliente, motorista)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao consultar caronas: %v", err))
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	if len(lista) == 0 {
		visual.MensagemAviso("Voce nao possui nenhuma carona cadastrada para cancelar.")
		lerEntrada("\nPressione ENTER para continuar...")
		return
	}

	// Tabela rápida para seleção de cancelamento
	colunasCabecalho := []string{"ID", "Data / Horario", "Rota"}
	largurasColunas := []int{4, 20, 30}
	visual.TabelaCabecalho(colunasCabecalho, largurasColunas)
	for _, c := range lista {
		dataHora := fmt.Sprintf("%s as %s", c.Data, c.Horario)
		rotaStr := strings.Join(c.Rota, " -> ")
		visual.TabelaLinha([]string{strconv.Itoa(c.ID), dataHora, rotaStr}, largurasColunas)
	}
	visual.TabelaRodape(largurasColunas)
	fmt.Println()

	idStr := lerEntrada("Informe o ID da carona que deseja cancelar (ou 0 para voltar): ")
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

	confirmacao := lerEntrada(fmt.Sprintf("Tem certeza que deseja cancelar a carona #%d? (s/N): ", id))
	if strings.ToLower(confirmacao) != "s" {
		fmt.Println("Cancelamento abortado.")
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	sucesso, err := caronas.CancelarCarona(cliente, id, motorista)
	if err != nil {
		visual.MensagemErro(fmt.Sprintf("Falha ao comunicar cancelamento: %v", err))
		lerEntrada("\nPressione ENTER para voltar...")
		return
	}

	if sucesso {
		visual.MensagemSucesso(fmt.Sprintf("Carona #%d cancelada com sucesso.", id))
		lerEntrada("\nPressione ENTER para voltar ao menu...")
	}
}
