package publicid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
)

const (
	RandomLength = 21
	alphabet     = "_-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxAttempts  = 3
)

type Prefix string

const (
	Admin     Prefix = "adm"
	Cliente   Prefix = "cli"
	Motorista Prefix = "mot"
	Vinculo   Prefix = "vin"
	Reserva   Prefix = "res"
	Viagem    Prefix = "via"
)

var ErrCollision = errors.New("não foi possível gerar um identificador público único")

// New cria um identificador público URL-safe. O alfabeto possui 64 símbolos,
// portanto cada byte aleatório pode ser mapeado uniformemente usando seus seis
// bits menos significativos, sem viés de módulo.
func New(prefix Prefix) (string, error) {
	random := make([]byte, RandomLength)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate public id: %w", err)
	}
	for i, value := range random {
		random[i] = alphabet[value&63]
	}
	return string(prefix) + "_" + string(random), nil
}

func Valid(value string, prefix Prefix) bool {
	if len(value) != len(prefix)+1+RandomLength || !strings.HasPrefix(value, string(prefix)+"_") {
		return false
	}
	for _, char := range value[len(prefix)+1:] {
		if !strings.ContainsRune(alphabet, char) {
			return false
		}
	}
	return true
}

// Insert gera o identificador e repete apenas quando o callback confirma que
// houve colisão na constraint de public_id. Outros erros não são mascarados.
func Insert[T any](prefix Prefix, insert func(string) (T, error), isCollision func(error) bool) (T, error) {
	var zero T
	for range maxAttempts {
		id, err := New(prefix)
		if err != nil {
			return zero, err
		}
		created, err := insert(id)
		if err == nil {
			return created, nil
		}
		if !isCollision(err) {
			return zero, err
		}
	}
	return zero, ErrCollision
}
