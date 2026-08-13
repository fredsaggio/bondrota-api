package publicid

import (
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	id, err := New(Cliente)
	if err != nil {
		t.Fatal(err)
	}
	if !Valid(id, Cliente) {
		t.Fatalf("invalid generated id %q", id)
	}
	if len(id) != len("cli_")+RandomLength {
		t.Fatalf("len(%q) = %d", id, len(id))
	}
	if strings.HasPrefix(id, "mot_") {
		t.Fatalf("wrong prefix: %q", id)
	}
}

func TestValidRejectsWrongFormat(t *testing.T) {
	valid := "cli_123456789012345678901"
	for _, value := range []string{
		"mot_123456789012345678901",
		"cli_123",
		"cli_12345678901234567890/",
	} {
		if Valid(value, Cliente) {
			t.Fatalf("Valid(%q) = true", value)
		}
	}
	if !Valid(valid, Cliente) {
		t.Fatalf("Valid(%q) = false", valid)
	}
}

func TestInsertRetriesOnlyCollisions(t *testing.T) {
	collision := errors.New("collision")
	attempts := 0
	got, err := Insert(Reserva, func(id string) (string, error) {
		attempts++
		if attempts == 1 {
			return "", collision
		}
		return id, nil
	}, func(err error) bool { return errors.Is(err, collision) })
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || !Valid(got, Reserva) {
		t.Fatalf("attempts=%d id=%q", attempts, got)
	}
}
