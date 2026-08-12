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
		{"cpf formatado", "123.456.789-01", "12345678901", nil},
		{"cpf so digitos", "12345678901", "12345678901", nil},
		{"cpf curto", "123.456.789", "", validation.ErrCPFInvalido},
		{"cpf longo demais", "123456789012", "", validation.ErrCPFInvalido},
		{"cpf com todos os digitos iguais", "000.000.000-00", "", validation.ErrCPFInvalido},
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
		{"fixo formatado", "(82) 3333-4444", "8233334444", nil},
		{"vazio e opcional", "", "", nil},
		{"so espacos e opcional", "   ", "", nil},
		{"ddd zero invalido", "(00) 98888-7777", "", validation.ErrTelefoneInvalido},
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
