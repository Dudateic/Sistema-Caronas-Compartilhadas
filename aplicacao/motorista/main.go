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
	fmt.Println("         VAIJUNTO - MODULO DO MOTORISTA                   ")

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
	motoristaLogado := telaAcesso(cliente)
	if motoristaLogado == "" {
		fmt.Println("Operacao encerrada pelo usuario. Ate logo!")
		return
	}

	// 2. Menu Principal de Operacoes
	menuPrincipalMotorista(cliente, motoristaLogado)
}

/**
 * Controla as opcoes de login e novo cadastro para o perfil de motorista.
 */
func telaAcesso(cliente *conexao.ClienteTCP) string {
	for {
		fmt.Println("\n                  AUTENTICACAO                    ")
		fmt.Println("1. Entrar (Login)")
		fmt.Println("2. Criar Nova Conta (Cadastro)")
		fmt.Println("0. Encerrar")
		fmt.Println("")

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
			sucesso, err := usuarios.AutenticarCliente(cliente, usuario, senha, protocolo.PerfilMotorista)
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
			sucesso, err := usuarios.CadastrarCliente(cliente, usuario, senha, protocolo.PerfilMotorista)
			if err != nil {
				fmt.Printf("[ERRO] Falha ao registrar usuario: %v\n", err)
				continue
			}
			if sucesso {
				fmt.Println("[SUCESSO] Conta criada com sucesso! Agora voce ja pode fazer login (Opcao 1).")
			}

		case "0":
			return ""

		default:
			fmt.Println("[AVISO] Opcao invalida, tente novamente.")
		}
	}
}

/**
 * Loop interativo com as opcoes de gerenciamento de viagens do motorista.
 */
func menuPrincipalMotorista(cliente *conexao.ClienteTCP, motorista string) {
	for {
		LimparTela()
		fmt.Printf("\n             PAINEL DO MOTORISTA: %s                 \n", motorista)
		fmt.Println("1. Publicar Nova Carona")
		fmt.Println("2. Consultar Minhas Caronas")
		fmt.Println("3. Cancelar Carona")
		fmt.Println("0. Sair e Desconectar")
		fmt.Println("")

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
			fmt.Println("[AVISO] Opcao invalida. Digite um numero entre 0 e 3.")
		}
	}
}

/**
 * Coleta os dados de trajeto e publica uma nova oferta no servidor.
 */
func acaoPublicarCarona(cliente *conexao.ClienteTCP, motorista string) {
	fmt.Println("\n--- PUBLICAR NOVA CARONA ---")
	fmt.Println("Informe as cidades da rota em ordem, separadas por virgula.")
	fmt.Println("Exemplo: Salvador, Feira de Santana, Serrinha")
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
		fmt.Println("[ERRO] Uma rota valida precisa de pelo menos 2 cidades (Origem e Destino).")
		return
	}

	data := lerEntrada("Data da viagem (AAAA-MM-DD): ")
	if strings.TrimSpace(data) == "" {
		fmt.Println("[ERRO] A data nao pode ser vazia.")
		return
	}

	horario := lerEntrada("Horario de saida (HH:MM): ")
	if strings.TrimSpace(horario) == "" {
		fmt.Println("[ERRO] O horario nao pode ser vazio.")
		return
	}

	assentosStr := lerEntrada("Quantidade total de vagas no veiculo: ")
	assentos, err := strconv.Atoi(assentosStr)
	if err != nil || assentos <= 0 {
		fmt.Println("[ERRO] Quantidade de assentos deve ser um numero inteiro maior que zero.")
		return
	}

	precoStr := lerEntrada("Preco cobrado por trecho em R$ (ex: 25.50): ")
	precoStr = strings.ReplaceAll(precoStr, ",", ".")
	preco, err := strconv.ParseFloat(precoStr, 64)
	if err != nil || preco <= 0 {
		fmt.Println("[ERRO] Preco invalido. Digite um valor monetario positivo.")
		return
	}

	fmt.Println("\nPublicando carona no servidor...")
	idCarona, err := caronas.PublicarCarona(cliente, motorista, rota, data, horario, assentos, preco)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao enviar carona: %v\n", err)
		return
	}

	if idCarona > 0 {
		fmt.Printf("[SUCESSO] Carona #%d cadastrada e aberta para reservas!\n", idCarona)
	}
}

/**
 * Consulta e formata as caronas publicadas pelo motorista.
 */
func acaoConsultarCaronas(cliente *conexao.ClienteTCP, motorista string) {
	fmt.Println("\nConsultando suas caronas ativas...")
	lista, err := caronas.ConsultarCaronas(cliente, motorista)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao consultar caronas: %v\n", err)
		return
	}

	if len(lista) == 0 {
		return // Mensagem padrao ja e impressa em ConsultarCaronas
	}
}

/**
 * Solicita o cancelamento de uma carona pertencente ao motorista.
 */
func acaoCancelarCarona(cliente *conexao.ClienteTCP, motorista string) {
	fmt.Println("\n                 CANCELAR CARONA                    ")

	// Lista as caronas antes de pedir o ID
	lista, err := caronas.ConsultarCaronas(cliente, motorista)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao consultar caronas: %v\n", err)
		return
	}

	// Se não tiver nenhuma carona, a própria função acima já avisa e retorna vazia
	if len(lista) == 0 {
		return
	}

	idStr := lerEntrada("Informe o ID da carona que deseja cancelar (ou 0 para voltar): ")
	id, err := strconv.Atoi(idStr)

	if err != nil || id < 0 {
		fmt.Println("[ERRO] ID invalido.")
		return
	}
	if id == 0 {
		fmt.Println("Cancelamento abortado.")
		return
	}

	confirmacao := lerEntrada(fmt.Sprintf("Tem certeza que deseja cancelar a carona #%d? (s/N): ", id))
	if strings.ToLower(confirmacao) != "s" {
		fmt.Println("Cancelamento abortado.")
		return
	}

	sucesso, err := caronas.CancelarCarona(cliente, id, motorista)
	if err != nil {
		fmt.Printf("[ERRO] Falha ao comunicar cancelamento: %v\n", err)
		return
	}

	if sucesso {
		fmt.Printf("[OK] Carona #%d cancelada com sucesso.\n", id)
	}
}
