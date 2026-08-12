// Package validation reune as regras de formato para campos de cadastro
// (nome, cpf, telefone) compartilhadas entre clientes e motoristas. A mascara
// (pontos, tracos, parenteses) e responsabilidade da apresentacao — aqui so
// entra e sai digito puro.
package validation

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrNomeInvalido     = errors.New("nome must contain only letters and spaces")
	ErrCPFInvalido      = errors.New("cpf is not a valid CPF number")
	ErrTelefoneInvalido = errors.New("telefone must be a valid cellphone number: ddd + 9 digits starting with 9")
	ErrPlacaInvalida    = errors.New("placa must have 7 characters: LLLNNNN (old) or LLLNLNN (mercosul)")
)

var naoDigito = regexp.MustCompile(`\D`)

var naoAlfanumerico = regexp.MustCompile(`[^A-Z0-9]`)

// Os dois padroes em circulacao no Brasil cabem num so formato: as tres letras
// e o quarto digito sao comuns, e o que os separa e a quinta posicao — letra no
// Mercosul (ABC1D23), digito no modelo antigo (ABC1234).
var formatoPlaca = regexp.MustCompile(`^[A-Z]{3}[0-9][0-9A-Z][0-9]{2}$`)

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

// CPF limpa a pontuacao, confere 11 digitos e valida os dois digitos
// verificadores pelo calculo oficial (modulo 11) — sem isso qualquer
// sequencia de 11 numeros passava, e um CPF digitado errado só seria
// percebido quando alguem tentasse usá-lo de verdade. Sequencias com todos os
// digitos iguais (000.000.000-00, 111.111.111-11, ...) satisfazem esse
// calculo por coincidencia matematica mas nunca foram emitidas, entao seguem
// rejeitadas à parte. Retorna o CPF ja limpo para gravar.
func CPF(cpf string) (string, error) {
	digits := LimparDigitos(cpf)
	if len(digits) != 11 || todosIguais(digits) || !cpfDigitosVerificadoresValidos(digits) {
		return "", ErrCPFInvalido
	}
	return digits, nil
}

func cpfDigitosVerificadoresValidos(digits string) bool {
	return digits[9] == cpfDigitoVerificador(digits[:9]) && digits[10] == cpfDigitoVerificador(digits[:10])
}

// cpfDigitoVerificador aplica o modulo 11: o primeiro digito da base pesa
// len(base)+1, decrescendo ate 2 no ultimo. Resto da divisao por 11 menor que
// 2 vira '0'; caso contrario o digito verificador e 11 menos o resto. A mesma
// funcao serve para os dois digitos verificadores — o segundo so entra com
// uma base um digito maior (que já inclui o primeiro verificador).
func cpfDigitoVerificador(base string) byte {
	peso := len(base) + 1
	soma := 0
	for _, r := range base {
		soma += int(r-'0') * peso
		peso--
	}
	resto := soma % 11
	if resto < 2 {
		return '0'
	}
	return byte('0' + (11 - resto))
}

// Telefone limpa a pontuacao e confere 11 digitos de celular: DDD valido
// seguido do 9 que todo celular brasileiro tem desde a expansao do nono
// digito. Fixo nao e aceito aqui — so cadastramos celular. O campo e
// opcional: string vazia (ou so pontuacao/espacos) retorna vazio sem erro.
// Retorna o telefone ja limpo.
func Telefone(telefone string) (string, error) {
	digits := LimparDigitos(telefone)
	if digits == "" {
		return "", nil
	}
	if len(digits) != 11 || digits[0] == '0' || digits[2] != '9' {
		return "", ErrTelefoneInvalido
	}
	return digits, nil
}

// Placa remove hifen e espacos, normaliza para maiuscula e confere os dois
// padroes brasileiros. Retorna a placa limpa, que e a unica forma que vai para
// o banco: a coluna tem UNIQUE, e guardar "ABC-1234" e "ABC1234" lado a lado
// criaria dois veiculos para a mesma placa sem o indice reclamar.
func Placa(placa string) (string, error) {
	limpa := naoAlfanumerico.ReplaceAllString(strings.ToUpper(strings.TrimSpace(placa)), "")
	if !formatoPlaca.MatchString(limpa) {
		return "", ErrPlacaInvalida
	}
	return limpa, nil
}

func todosIguais(digits string) bool {
	for i := 1; i < len(digits); i++ {
		if digits[i] != digits[0] {
			return false
		}
	}
	return true
}
