package brerror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

func TestMensagemUsuario(t *testing.T) {
	const generica = "Não foi possível concluir a operação. Revise os dados e tente novamente."

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			// O caso que motivou a funcao: sem ela o usuario leria
			// "invalid input: Selecione o destino.".
			name: "tira o prefixo do sentinela",
			err:  fmt.Errorf("%w: Selecione o destino.", brerror.ErrInvalidInput),
			want: "Selecione o destino.",
		},
		{
			name: "frase de tela sem sentinela passa direto",
			err:  errors.New("Cliente não encontrado."),
			want: "Cliente não encontrado.",
		},
		{
			// Mensagem tecnica segue a convencao Go (minuscula, sem ponto) e
			// por isso e barrada — e o que impede vazar detalhe interno.
			name: "detalhe tecnico vira generica",
			err:  errors.New("service/rotaInternaService.Create: db error"),
			want: generica,
		},
		{
			name: "erro de driver vira generica",
			err:  errors.New("connection refused to 10.0.0.5"),
			want: generica,
		},
		{
			name: "sentinela embrulhando detalhe tecnico vira generica",
			err:  fmt.Errorf("%w: column \"foo\" does not exist", brerror.ErrInvalidInput),
			want: generica,
		},
		{
			name: "sentinela puro, sem mensagem, vira generica",
			err:  brerror.ErrInvalidInput,
			want: generica,
		},
		{
			name: "nil vira generica",
			err:  nil,
			want: generica,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := brerror.MensagemUsuario(tc.err); got != tc.want {
				t.Fatalf("MensagemUsuario() = %q, want %q", got, tc.want)
			}
		})
	}
}
