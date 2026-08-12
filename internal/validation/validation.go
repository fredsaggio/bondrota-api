// Package validation reune as regras de formato para campos de cadastro
// (nome, cpf, telefone) compartilhadas entre clientes e motoristas. A mascara
// (pontos, tracos, parenteses) e responsabilidade da apresentacao — aqui so
// entra e sai digito puro.
package validation

import (
	"errors"
	"regexp"
	"unicode"
)

var (
	ErrNomeInvalido     = errors.New("nome must contain only letters and spaces")
	ErrCPFInvalido      = errors.New("cpf must have 11 digits")
	ErrTelefoneInvalido = errors.New("telefone must have 10 or 11 digits with a valid ddd")
)

var naoDigito = regexp.MustCompile(`\D`)

// Nome confere que o valor (ja sem espacos nas pontas) tem letras, espacos e
// os sinais comuns em nomes proprios (hifen, apostrofo) — sem digitos ou
// outros simbolos, e ao menos 3 caracteres.
func Nome(nome string) error {
	if len([]rune(nome)) < 3 {
		return ErrNomeInvalido
	}
	for _, r := range nome {
		if unicode.IsLetter(r) || r == ' ' || r == '\'' || r == '-' {
			continue
		}
		return ErrNomeInvalido
	}
	return nil
}

// LimparDigitos remove tudo que nao for digito.
func LimparDigitos(value string) string {
	return naoDigito.ReplaceAllString(value, "")
}

// CPF limpa a pontuacao e confere que sobraram exatamente 11 digitos,
// rejeitando tambem sequencias obviamente invalidas (todos os digitos
// iguais, como "000.000.000-00"). Retorna o CPF ja limpo para gravar.
func CPF(cpf string) (string, error) {
	digits := LimparDigitos(cpf)
	if len(digits) != 11 || todosIguais(digits) {
		return "", ErrCPFInvalido
	}
	return digits, nil
}

// Telefone limpa a pontuacao e confere 10 ou 11 digitos (DDD + numero fixo ou
// celular) com DDD valido. O campo e opcional: string vazia (ou so
// pontuacao/espacos) retorna vazio sem erro. Retorna o telefone ja limpo.
func Telefone(telefone string) (string, error) {
	digits := LimparDigitos(telefone)
	if digits == "" {
		return "", nil
	}
	if (len(digits) != 10 && len(digits) != 11) || digits[0] == '0' {
		return "", ErrTelefoneInvalido
	}
	return digits, nil
}

func todosIguais(digits string) bool {
	for i := 1; i < len(digits); i++ {
		if digits[i] != digits[0] {
			return false
		}
	}
	return true
}
