//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/stretchr/testify/require"
)

func TestTelefoneUnicoEntreClientesEMotoristas(t *testing.T) {
	ctx, tx := beginTestTx(t)

	_, err := tx.Exec(ctx, `
		INSERT INTO motoristas (
			nome, cpf, senha, telefone, data_nasc, turno, municipio_trabalho_id
		) VALUES ('Motorista Telefone', '30000000001', 'hash', '82999995555', '1985-05-20', 'MT', $1)`,
		testMunicipioID,
	)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, `
		INSERT INTO clientes (
			nome, cpf, senha, telefone, data_nasc,
			documento_identificacao, comprovante_residencia
		) VALUES (
			'Cliente Telefone', '40000000001', 'hash', '82999995555', '2002-08-10',
			'identidade.pdf', 'residencia.pdf'
		)`)
	require.Error(t, err)
	require.True(t, db.IsUniqueViolation(err, "telefones_cadastrados_pkey"), "%v", err)
}

func TestTelefoneUnicoNaMesmaEntidade(t *testing.T) {
	ctx, tx := beginTestTx(t)

	_, err := tx.Exec(ctx, `
		INSERT INTO clientes (
			nome, cpf, senha, telefone, data_nasc,
			documento_identificacao, comprovante_residencia
		) VALUES
			('Cliente Um', '40000000002', 'hash', '82999996666', '2002-08-10', 'id-1.pdf', 'res-1.pdf')`)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, `
		INSERT INTO clientes (
			nome, cpf, senha, telefone, data_nasc,
			documento_identificacao, comprovante_residencia
		) VALUES
			('Cliente Dois', '40000000003', 'hash', '82999996666', '2002-08-10', 'id-2.pdf', 'res-2.pdf')`)
	require.Error(t, err)
	require.True(t, db.IsUniqueViolation(err, "telefones_cadastrados_pkey"), "%v", err)
}

func TestTelefoneVazioPodeSeRepetir(t *testing.T) {
	ctx, tx := beginTestTx(t)

	_, err := tx.Exec(ctx, `
		INSERT INTO clientes (
			nome, cpf, senha, telefone, data_nasc,
			documento_identificacao, comprovante_residencia
		) VALUES
			('Sem Telefone Um', '40000000004', 'hash', '', '2002-08-10', 'id-1.pdf', 'res-1.pdf'),
			('Sem Telefone Dois', '40000000005', 'hash', '', '2002-08-10', 'id-2.pdf', 'res-2.pdf')`)
	require.NoError(t, err)
}

func TestTelefoneDuplicadoEmUpdate(t *testing.T) {
	ctx, tx := beginTestTx(t)

	_, err := tx.Exec(ctx, `
		INSERT INTO clientes (
			nome, cpf, senha, telefone, data_nasc,
			documento_identificacao, comprovante_residencia
		) VALUES
			('Cliente Update Um', '40000000006', 'hash', '82999997771', '2002-08-10', 'id-1.pdf', 'res-1.pdf'),
			('Cliente Update Dois', '40000000007', 'hash', '82999997772', '2002-08-10', 'id-2.pdf', 'res-2.pdf')`)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, `
		UPDATE clientes SET telefone = '82999997771' WHERE cpf = '40000000007'`)
	require.Error(t, err)
	require.True(t, db.IsUniqueViolation(err, "telefones_cadastrados_pkey"), "%v", err)
}
