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

func iniciarServidorReal(t *testing.T) (net.Listener, string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Erro ao abrir porta para o servidor de teste: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener encerrado
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
	_ = os.Remove("usuarios.json")
	defer os.Remove("usuarios.json")

	listener, endereco := iniciarServidorReal(t)
	defer listener.Close()

	// Usa o metodo original do seu projeto sem alterar a struct ou pacote conexao
	cliente, err := conexao.ConectarTCP(endereco, 2*time.Second)
	if err != nil {
		t.Fatalf("Erro ao conectar clienteTCP: %v", err)
	}
	defer cliente.Fechar()

	// Cadastro com sucesso
	sucesso, err := usuarios.CadastrarCliente(cliente, "motorista1", "senha123", protocolo.PerfilMotorista)
	if err != nil || !sucesso {
		t.Fatalf("Falha no cadastro: sucesso=%v, err=%v", sucesso, err)
	}

	// Bloqueio de duplicado
	sucessoDuplicado, _ := usuarios.CadastrarCliente(cliente, "motorista1", "outrasenha", protocolo.PerfilMotorista)
	if sucessoDuplicado {
		t.Fatalf("Erro: permitiu cadastrar usuario duplicado!")
	}

	// Login com sucesso
	loginOk, err := usuarios.AutenticarCliente(cliente, "motorista1", "senha123", protocolo.PerfilMotorista)
	if err != nil || !loginOk {
		t.Fatalf("Falha no login com credenciais validas: loginOk=%v, err=%v", loginOk, err)
	}

	// Senha errada
	loginInvalido, _ := usuarios.AutenticarCliente(cliente, "motorista1", "senhaErrada", protocolo.PerfilMotorista)
	if loginInvalido {
		t.Fatalf("Erro: login aceito com senha incorreta!")
	}

	// Perfil divergente
	perfilErrado, _ := usuarios.AutenticarCliente(cliente, "motorista1", "senha123", protocolo.PerfilPassageiro)
	if perfilErrado {
		t.Fatalf("Erro: acesso permitido para perfil divergente!")
	}
}
