package main

import (
	"context"
	"testing"
)

func TestDatabaseEnvFor(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    string
		wantErr bool
	}{
		{name: "local usa DATABASE_URL", target: "local", want: "DATABASE_URL"},
		{name: "prod exige alvo explicito", target: "prod", want: "PROD_DATABASE_URL"},
		{name: "alvo desconhecido falha", target: "producao", wantErr: true},
		{name: "alvo vazio falha em vez de assumir prod", target: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := databaseEnvFor(tc.target)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error for target %q, got %q", tc.target, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

// O alvo padrao da flag precisa ser local: e ele que impede que rodar o comando
// durante o desenvolvimento acabe mexendo em producao.
func TestDefaultTargetIsLocal(t *testing.T) {
	got, err := databaseEnvFor(targetLocal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "DATABASE_URL" {
		t.Fatalf("want DATABASE_URL for the default target, got %q", got)
	}
}

func TestSafeHostHidesCredentials(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "remove usuario e senha",
			raw:  "postgresql://postgres.abc:sup3rs3cr3t@aws-1-us-east-1.pooler.supabase.com:5432/postgres",
			want: "aws-1-us-east-1.pooler.supabase.com:5432",
		},
		{
			name: "url local",
			raw:  "postgres://postgres:password@localhost:5432/bondrota_db?sslmode=disable",
			want: "localhost:5432",
		},
		{
			name: "url invalida nao vaza a string original",
			raw:  "isso-nao-e-uma-url",
			want: "desconhecido",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := safeHost(tc.raw)
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

// Sem subcomando o comando precisa falhar em vez de assumir um padrao: o antigo
// seed-admin era um binario so, e rodar `go run ./cmd/admin` por engano nao pode
// virar uma escrita silenciosa no banco.
func TestRunRequiresKnownSubcommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "sem argumentos falha", args: nil, wantErr: true},
		{name: "subcomando desconhecido falha", args: []string{"seedadmin"}, wantErr: true},
		{name: "help nao e erro", args: []string{"help"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := run(context.Background(), tc.args)
			if tc.wantErr && err == nil {
				t.Fatalf("want error for args %v, got nil", tc.args)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
