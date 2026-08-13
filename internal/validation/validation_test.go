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
		want    string
		wantErr error
	}{
		{"nome valido vira maiuscula", "Maria Souza", "MARIA SOUZA", nil},
		{"nome ja minusculo tambem vira maiuscula", "maria souza", "MARIA SOUZA", nil},
		{"nome com acento e hifen", "José Carlos-Neto", "JOSÉ CARLOS-NETO", nil},
		{"nome com apostrofo", "Ana D'Ávila", "ANA D'ÁVILA", nil},
		{"nome com digito", "Maria 2", "", validation.ErrNomeInvalido},
		{"nome so numero", "12345", "", validation.ErrNomeInvalido},
		{"nome curto demais", "Jo", "", validation.ErrNomeInvalido},
		{"nome com simbolo", "Maria@Souza", "", validation.ErrNomeInvalido},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.Nome(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Nome(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Nome(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestModelo(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{"letras numeros e espacos", "Ônibus 1722 Urbano", "Ônibus 1722 Urbano", nil},
		{"remove apenas espacos externos", "  Volare Escolar  ", "Volare Escolar", nil},
		{"modelo com cerquilha", "Ônibus #1722", "", validation.ErrModeloInvalido},
		{"modelo com barra", "Ônibus 1722/Urbano", "", validation.ErrModeloInvalido},
		{"modelo vazio", "   ", "", validation.ErrModeloInvalido},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.Modelo(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Modelo(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Modelo(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestCaminhoDocumento(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{"pdf em pasta temporaria", " _novo/uuid/documento.pdf ", "_novo/uuid/documento.pdf", nil},
		{"imagem em path definitivo", "clientes/10/documento.PNG", "clientes/10/documento.PNG", nil},
		{"extensao executavel", "clientes/10/documento.exe", "", validation.ErrCaminhoDocumentoInvalido},
		{"path com navegacao", "clientes/10/../documento.pdf", "", validation.ErrCaminhoDocumentoInvalido},
		{"path com barra invertida", `clientes\\10\\documento.pdf`, "", validation.ErrCaminhoDocumentoInvalido},
		{"path absoluto", "/clientes/10/documento.pdf", "", validation.ErrCaminhoDocumentoInvalido},
		{"vazio", " ", "", validation.ErrCaminhoDocumentoInvalido},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.CaminhoDocumento(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CaminhoDocumento(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("CaminhoDocumento(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestCurso(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		{"curso vira maiuscula", "Ciência da Computação", "CIÊNCIA DA COMPUTAÇÃO", nil},
		{"hifen e apostrofo", "Língua d'Água-Portuguesa", "LÍNGUA D'ÁGUA-PORTUGUESA", nil},
		{"curso com numero", "Técnico em TI 2", "", validation.ErrCursoInvalido},
		{"curso com simbolo", "Computação@", "", validation.ErrCursoInvalido},
		{"vazio fica para regra do vinculo", "   ", "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validation.Curso(tc.value)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Curso(%q) err = %v, want %v", tc.value, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Curso(%q) = %q, want %q", tc.value, got, tc.want)
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
		{"cpf com letra misturada", "123a45678909", "", validation.ErrCPFInvalido},
		{"cpf com mascara fora de posicao", "12.3456.789-09", "", validation.ErrCPFInvalido},
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
		{"celular formatado sem espaco", "(82)98888-7777", "82988887777", nil},
		{"celular so digitos", "82988887777", "82988887777", nil},
		{"celular com letra misturada", "82abc988887777", "", validation.ErrTelefoneInvalido},
		{"celular com pontuacao fora de posicao", "82-98888-7777", "", validation.ErrTelefoneInvalido},
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
		{"mercosul com hifen", "ABC-1D23", "", validation.ErrPlacaInvalida},
		{"placa com simbolos misturados", "A@B#C1D23", "", validation.ErrPlacaInvalida},
		{"placa antiga com espaco interno", "ABC 1234", "", validation.ErrPlacaInvalida},
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
