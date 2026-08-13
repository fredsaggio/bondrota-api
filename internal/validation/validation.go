// Package validation reune as regras de formato para campos de cadastro
// compartilhadas entre dominios. Mascaras de campos como CPF, telefone e placa
// sao responsabilidade da apresentacao; a API normaliza e valida novamente os
// valores recebidos antes de persistir.
package validation

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// O texto destes erros vai direto para a tela de quem usa o painel, entao e
// escrito em portugues e sem nome de coluna. Detalhe tecnico de falha
// inesperada nao entra aqui: isso vai para o log do servidor, em ingles.
var (
	ErrNomeInvalido     = errors.New("O nome deve conter apenas letras e espaços.")
	ErrModeloInvalido   = errors.New("O modelo deve conter apenas letras, números e espaços.")
	ErrCursoInvalido    = errors.New("O curso deve conter apenas letras, espaços, hífen e apóstrofo.")
	ErrCPFInvalido      = errors.New("CPF inválido. Confira os dígitos digitados.")
	ErrTelefoneInvalido = errors.New("O telefone deve ser um celular válido: DDD + 9 dígitos.")
	ErrPlacaInvalida    = errors.New("A placa deve seguir um dos padrões: ABC-1234 ou ABC1D23.")
)

var (
	naoDigito               = regexp.MustCompile(`\D`)
	formatoCPFLimpo         = regexp.MustCompile(`^[0-9]{11}$`)
	formatoCPFMascarado     = regexp.MustCompile(`^[0-9]{3}\.[0-9]{3}\.[0-9]{3}-[0-9]{2}$`)
	formatoTelefoneLimpo    = regexp.MustCompile(`^[0-9]{11}$`)
	formatoTelefoneMascara  = regexp.MustCompile(`^\([0-9]{2}\) ?[0-9]{5}-[0-9]{4}$`)
	formatoPlaca            = regexp.MustCompile(`^[A-Z]{3}[0-9][0-9A-Z][0-9]{2}$`)
	formatoPlacaAntigaHifen = regexp.MustCompile(`^[A-Z]{3}-[0-9]{4}$`)
)

// Os dois padroes em circulacao no Brasil cabem num so formato: as tres letras
// e o quarto digito sao comuns, e o que os separa e a quinta posicao — letra no
// Mercosul (ABC1D23), digito no modelo antigo (ABC1234).
// Nome confere que o valor (ja sem espacos nas pontas) tem letras, espacos e
// os sinais comuns em nomes proprios (hifen, apostrofo) — sem digitos ou
// outros simbolos, e ao menos 3 caracteres. Retorna o nome em maiusculas: e
// consistente independente de como foi digitado, e evita o problema do title
// case em portugues, que capitalizaria preposicoes como "Sistemas De
// Informacao" sem uma lista de excecoes para manter minusculas.
func Nome(nome string) (string, error) {
	if len([]rune(nome)) < 3 {
		return "", ErrNomeInvalido
	}
	for _, r := range nome {
		if unicode.IsLetter(r) || r == ' ' || r == '\'' || r == '-' {
			continue
		}
		return "", ErrNomeInvalido
	}
	return strings.ToUpper(nome), nil
}

// Modelo aceita os nomes comerciais usuais de veiculos, inclusive acentos e
// numeros de serie, mas rejeita pontuacao e simbolos. A validacao fica na API
// para que clientes diretos nao consigam contornar a restricao da interface.
func Modelo(modelo string) (string, error) {
	limpo := strings.TrimSpace(modelo)
	if limpo == "" {
		return "", ErrModeloInvalido
	}
	for _, r := range limpo {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == ' ' {
			continue
		}
		return "", ErrModeloInvalido
	}
	return limpo, nil
}

// Curso segue o mesmo conjunto de caracteres exibido pelo painel e retorna o
// valor normalizado em maiusculas. Vazio continua permitido aqui porque a
// obrigatoriedade depende do tipo de vinculo e e validada pelo seu dominio.
func Curso(curso string) (string, error) {
	limpo := strings.TrimSpace(curso)
	for _, r := range limpo {
		if unicode.IsLetter(r) || r == ' ' || r == '\'' || r == '-' {
			continue
		}
		return "", ErrCursoInvalido
	}
	return strings.ToUpper(limpo), nil
}

// LimparDigitos remove tudo que nao for digito.
func LimparDigitos(value string) string {
	return naoDigito.ReplaceAllString(value, "")
}

// CPF aceita apenas os 11 digitos ou a mascara canonica, confere o formato e
// valida os dois digitos
// verificadores pelo calculo oficial (modulo 11) — sem isso qualquer
// sequencia de 11 numeros passava, e um CPF digitado errado só seria
// percebido quando alguem tentasse usá-lo de verdade. Sequencias com todos os
// digitos iguais (000.000.000-00, 111.111.111-11, ...) satisfazem esse
// calculo por coincidencia matematica mas nunca foram emitidas, entao seguem
// rejeitadas à parte. Retorna o CPF ja limpo para gravar.
func CPF(cpf string) (string, error) {
	value := strings.TrimSpace(cpf)
	if !formatoCPFLimpo.MatchString(value) && !formatoCPFMascarado.MatchString(value) {
		return "", ErrCPFInvalido
	}
	digits := LimparDigitos(value)
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

// Telefone aceita apenas os 11 digitos ou a mascara brasileira canonica e
// confere os 11 digitos de celular: DDD valido
// seguido do 9 que todo celular brasileiro tem desde a expansao do nono
// digito. Fixo nao e aceito aqui — so cadastramos celular. O campo e
// opcional: string vazia ou so espacos retorna vazio sem erro.
// Retorna o telefone ja limpo.
func Telefone(telefone string) (string, error) {
	value := strings.TrimSpace(telefone)
	if value == "" {
		return "", nil
	}
	if !formatoTelefoneLimpo.MatchString(value) && !formatoTelefoneMascara.MatchString(value) {
		return "", ErrTelefoneInvalido
	}
	digits := LimparDigitos(value)
	if len(digits) != 11 || digits[0] == '0' || digits[2] != '9' {
		return "", ErrTelefoneInvalido
	}
	return digits, nil
}

// Placa aceita a forma limpa dos dois padroes brasileiros e, para o padrao
// antigo, a mascara canonica com hifen. Normaliza para maiuscula e retorna a
// placa limpa, que e a unica forma que vai para
// o banco: a coluna tem UNIQUE, e guardar "ABC-1234" e "ABC1234" lado a lado
// criaria dois veiculos para a mesma placa sem o indice reclamar.
func Placa(placa string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(placa))
	if !formatoPlaca.MatchString(value) && !formatoPlacaAntigaHifen.MatchString(value) {
		return "", ErrPlacaInvalida
	}
	limpa := strings.ReplaceAll(value, "-", "")
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
