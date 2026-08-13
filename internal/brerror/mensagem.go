package brerror

import (
	"strings"
	"unicode"
)

// generica e usada quando o erro nao carrega uma frase propria para o usuario.
const generica = "Não foi possível concluir a operação. Revise os dados e tente novamente."

// MensagemUsuario devolve o texto que pode ir no corpo da resposta HTTP.
//
// Os erros de dominio sao embrulhados num sentinela para o handler classificar
// o status (`fmt.Errorf("%w: %s", ErrInvalidInput, msg)`). Isso faz o
// `err.Error()` virar "invalid input: <msg>" — e mandar isso para a tela
// entregaria o prefixo em ingles junto com a frase. Aqui o prefixo sai.
//
// Se o que sobrar ainda parecer detalhe interno (texto em ingles, nome de
// coluna, mensagem de driver), a funcao devolve uma frase generica: e melhor
// dizer pouco do que descrever o funcionamento do sistema para quem estiver
// sondando. O detalhe completo continua disponivel no log do servidor, que e
// onde ele serve para alguma coisa.
func MensagemUsuario(err error) string {
	if err == nil {
		return generica
	}

	mensagem := strings.TrimSpace(err.Error())
	for _, sentinela := range []error{
		ErrInvalidInput, ErrNotFound, ErrAlreadyExists,
		ErrUnauthenticated, ErrForbidden, ErrResourceInUse,
	} {
		prefixo := sentinela.Error() + ": "
		if strings.HasPrefix(mensagem, prefixo) {
			mensagem = strings.TrimSpace(strings.TrimPrefix(mensagem, prefixo))
			break
		}
	}

	if mensagem == "" || !ehFraseDeUsuario(mensagem) {
		return generica
	}
	return mensagem
}

// ehFraseDeUsuario reconhece as frases escritas para a tela: elas comecam com
// letra maiuscula e terminam em ponto. Toda mensagem tecnica do projeto segue
// a convencao Go — minuscula, sem pontuacao final — entao a regra separa as
// duas sem precisar de lista.
func ehFraseDeUsuario(mensagem string) bool {
	runas := []rune(mensagem)
	if !unicode.IsUpper(runas[0]) {
		return false
	}
	return runas[len(runas)-1] == '.'
}
