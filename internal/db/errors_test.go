package db

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func pgError(code, constraint string) error {
	// Os stores embrulham o erro com %w, entao o helper precisa achar o PgError
	// mesmo aninhado.
	return fmt.Errorf("db/store.Delete: %w", &pgconn.PgError{Code: code, ConstraintName: constraint})
}

func TestIsAnyForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "fk de qualquer constraint", err: pgError("23503", "reservas_destino_id_fkey"), want: true},
		{name: "fk de outra constraint tambem conta", err: pgError("23503", "ciclos_viagem_veiculo_id_fkey"), want: true},
		{name: "unique nao e fk", err: pgError("23505", "clientes_cpf_key"), want: false},
		{name: "erro comum nao e fk", err: errors.New("connection refused"), want: false},
		{name: "nil nao e fk", err: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAnyForeignKeyViolation(tc.err); got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestIsForeignKeyViolationStillMatchesByConstraint(t *testing.T) {
	err := pgError("23503", "cliente_vinculos_destino_id_fkey")

	if !IsForeignKeyViolation(err, "cliente_vinculos_destino_id_fkey") {
		t.Fatal("want match for the exact constraint")
	}
	if IsForeignKeyViolation(err, "outra_constraint_fkey") {
		t.Fatal("want no match for a different constraint")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	err := pgError("23505", "clientes_cpf_key")

	if !IsUniqueViolation(err, "clientes_cpf_key") {
		t.Fatal("want match for the exact constraint")
	}
	if IsUniqueViolation(err, "clientes_email_key") {
		t.Fatal("want no match for a different constraint")
	}
	// Uma violacao de FK nunca pode ser lida como unique: os handlers escolhem
	// mensagens diferentes para cada caso.
	if IsUniqueViolation(pgError("23503", "clientes_cpf_key"), "clientes_cpf_key") {
		t.Fatal("foreign key violation must not be reported as unique violation")
	}
}
