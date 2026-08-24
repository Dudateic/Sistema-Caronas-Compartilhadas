package testes

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/conexao"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/comunicacao/protocolo"
	"VAIJUNTO-Sistema-de-caronas-compartilhadas/servicos/usuarios"
)

// Inicia um servidor TCP mock temporário em uma porta livre
func iniciarServidorMock(t *testing.T) (net.Listener, string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Erro ao iniciar servidor de teste: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()
				leitor := bufio.NewReader(c)

				for {
					linha, err := leitor.ReadString('\n')
					if err != nil {
						return
					}

					var base protocolo.MensagemBase
					if err := json.Unmarshal([]byte(linha), &base); err != nil {
						continue
					}

					switch base.Tipo {
					case protocolo.TipoCadastroReq:
						usuarios.ProcessarCadastro(c, linha)
					case protocolo.TipoLoginReq:
						usuarios.ProcessarLogin(c, linha)
					}
				}
			}(conn)
		}
	}()

	return listener, listener.Addr().String()
}

func TestFluxoCadastroELogin(t *testing.T) {
	// Limpa o arquivo usuarios.json antes e depois do teste
	_ = os.Remove("usuarios.json")
	defer os.Remove("usuarios.json")

	listener, endereco := iniciarServidorMock(t)
	defer listener.Close()

	cliente, err := conexao.ConectarTCP(endereco, 2*time.Second)
	if err != nil {
		t.Fatalf("Erro ao conectar cliente: %v", err)
	}
	defer cliente.Fechar()

	// Cadastro com sucesso
	sucesso, err := usuarios.CadastrarCliente(cliente, "motorista1", "senha123", protocolo.PerfilMotorista)
	if err != nil || !sucesso {
		t.Fatalf("Falha no cadastro: sucesso=%v, err=%v", sucesso, err)
	}

	// Bloqueio de usuário duplicado
	sucessoDuplicado, _ := usuarios.CadastrarCliente(cliente, "motorista1", "outrasenha", protocolo.PerfilMotorista)
	if sucessoDuplicado {
		t.Fatalf("Erro: permitiu cadastrar usuario duplicado!")
	}

	// Login com credenciais válidas
	loginOk, err := usuarios.AutenticarCliente(cliente, "motorista1", "senha123", protocolo.PerfilMotorista)
	if err != nil || !loginOk {
		t.Fatalf("Falha no login com credenciais validas: loginOk=%v, err=%v", loginOk, err)
	}

	// Login com senha errada
	loginInvalido, _ := usuarios.AutenticarCliente(cliente, "motorista1", "senhaErrada", protocolo.PerfilMotorista)
	if loginInvalido {
		t.Fatalf("Erro: login aceito com senha incorreta!")
	}

	// Perfil não autorizado (motorista tentando acessar como passageiro)
	perfilErrado, _ := usuarios.AutenticarCliente(cliente, "motorista1", "senha123", protocolo.PerfilPassageiro)
	if perfilErrado {
		t.Fatalf("Erro: acesso permitido para perfil divergente!")
	}
}
