package validation_test

import (
	"errors"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/validation"
)

func TestNome(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{"nome valido", "Maria Souza", nil},
		{"nome com acento e hifen", "José Carlos-Neto", nil},
		{"nome com apostrofo", "Ana D'Ávila", nil},
		{"nome com digito", "Maria 2", validation.ErrNomeInvalido},
		{"nome so numero", "12345", validation.ErrNomeInvalido},
		{"nome curto demais", "Jo", validation.ErrNomeInvalido},
		{"nome com simbolo", "Maria@Souza", validation.ErrNomeInvalido},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validation.Nome(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Nome(%q) = %v, want %v", tc.value, err, tc.wantErr)
			}
		})
	}
}

func TestCPF(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{"cpf formatado", "123.456.789-09", "12345678909", nil},
		{"cpf so digitos", "12345678909", "12345678909", nil},
		// Exemplo canonico usado em tutoriais e ferramentas de teste — nao
		// pertence a ninguem, mas passa no calculo do digito verificador.
		{"outro cpf valido", "111.444.777-35", "11144477735", nil},
		{"cpf curto", "123.456.789", "", validation.ErrCPFInvalido},
		{"cpf longo demais", "123456789012", "", validation.ErrCPFInvalido},
		{"cpf com todos os digitos iguais", "000.000.000-00", "", validation.ErrCPFInvalido},
		// 11 digitos, sem repeticao, mas o digito verificador nao bate — o bug
		// que esta validacao existe para pegar.
		{"digito verificador errado", "123.456.789-01", "", validation.ErrCPFInvalido},
		{"cpf vazio", "", "", validation.ErrCPFInvalido},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.CPF(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CPF(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("CPF(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestTelefone(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{"celular formatado", "(82) 98888-7777", "82988887777", nil},
		{"fixo nao e aceito", "(82) 3333-4444", "", validation.ErrTelefoneInvalido},
		{"vazio e opcional", "", "", nil},
		{"so espacos e opcional", "   ", "", nil},
		{"ddd zero invalido", "(00) 98888-7777", "", validation.ErrTelefoneInvalido},
		{"nono digito ausente", "82888887777", "", validation.ErrTelefoneInvalido},
		{"curto demais", "988887777", "", validation.ErrTelefoneInvalido},
		{"longo demais", "829888877771", "", validation.ErrTelefoneInvalido},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.Telefone(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Telefone(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Telefone(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestPlaca(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{"antiga com hifen", "ABC-1234", "ABC1234", nil},
		{"antiga limpa", "ABC1234", "ABC1234", nil},
		{"antiga minuscula", "abc-1234", "ABC1234", nil},
		{"mercosul", "ABC1D23", "ABC1D23", nil},
		{"mercosul minuscula com espacos", " abc1d23 ", "ABC1D23", nil},
		// O hifen nao faz parte do padrao Mercosul, mas se alguem digitar ele
		// sai na limpeza como qualquer outra pontuacao.
		{"mercosul com hifen sobra", "ABC-1D23", "ABC1D23", nil},
		{"obrigatoria", "", "", validation.ErrPlacaInvalida},
		{"curta demais", "ABC123", "", validation.ErrPlacaInvalida},
		{"longa demais", "ABC12345", "", validation.ErrPlacaInvalida},
		{"letra na posicao de digito", "ABCD234", "", validation.ErrPlacaInvalida},
		// A quinta posicao aceita letra ou digito; a sexta e a setima, so digito.
		{"letra na sexta posicao", "ABC1DA3", "", validation.ErrPlacaInvalida},
		{"digito no lugar de letra", "AB11234", "", validation.ErrPlacaInvalida},
		{"acento nao e letra de placa", "ABÇ1234", "", validation.ErrPlacaInvalida},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.Placa(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Placa(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Placa(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
