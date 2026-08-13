// Command admin gerencia as contas de administrador fora da aplicacao web.
//
// O painel nao expoe cadastro de administradores de proposito: uma sessao de admin
// comprometida nao deve conseguir criar novos acessos nem apagar os existentes.
// Todas as operacoes de conta passam por aqui, com acesso direto ao banco.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

const (
	targetLocal = "local"
	targetProd  = "prod"

	connectTimeout = 30 * time.Second
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("admin command failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("informe um subcomando")
	}

	name, rest := args[0], args[1:]
	switch name {
	case "seed":
		return runSeed(ctx, rest)
	case "list":
		return runList(ctx, rest)
	case "create":
		return runCreate(ctx, rest)
	case "passwd":
		return runPasswd(ctx, rest)
	case "delete":
		return runDelete(ctx, rest)
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("subcomando desconhecido: %q", name)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Gerenciamento de administradores do BondRota.

Uso:
  go run ./cmd/admin <subcomando> [flags]

Subcomandos:
  seed      Cria o administrador inicial a partir de ADMIN_EMAIL/ADMIN_PASSWORD (idempotente)
  list      Lista os administradores cadastrados
  create    Cria um administrador, pedindo a senha no terminal
  passwd    Troca a senha de um administrador
  delete    Remove um administrador

Flags comuns:
  -target   Banco alvo: local (DATABASE_URL) ou prod (PROD_DATABASE_URL). Padrao: local.

Exemplos:
  go run ./cmd/admin list
  go run ./cmd/admin create -email=novo@prefeitura.gov.br
  go run ./cmd/admin passwd -email=admin@prefeitura.gov.br -target=prod
`)
}

// --- subcomandos ---

// runSeed mantem o comportamento do antigo cmd/seed-admin: cria o administrador
// inicial a partir do ambiente e nao faz nada se o e-mail ja existir, para poder
// ser reexecutado com seguranca durante o provisionamento.
func runSeed(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("seed", flag.ExitOnError)
	target := registerTarget(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}

	if err := loadEnv(*target); err != nil {
		return err
	}

	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" {
		return errors.New("ADMIN_EMAIL is required")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("ADMIN_PASSWORD is required")
	}
	if err := admin.ValidarSenha(password); err != nil {
		return fmt.Errorf("ADMIN_PASSWORD: %w", err)
	}

	pool, store, err := connect(ctx, *target)
	if err != nil {
		return err
	}
	defer pool.Close()

	existing, err := findByEmail(ctx, store, email)
	if err != nil {
		return err
	}
	if existing != nil {
		slog.Info("admin already exists", "email", email, "id", existing.PublicID)
		return nil
	}

	created, err := createAdmin(ctx, store, email, password)
	if err != nil {
		return err
	}

	slog.Info("admin created", "email", created.Email, "id", created.PublicID)
	return nil
}

func runList(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	target := registerTarget(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}

	if err := loadEnv(*target); err != nil {
		return err
	}

	pool, store, err := connect(ctx, *target)
	if err != nil {
		return err
	}
	defer pool.Close()

	admins, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("list admins: %w", err)
	}
	if len(admins) == 0 {
		fmt.Println("Nenhum administrador cadastrado.")
		return nil
	}

	fmt.Printf("%-25s %s\n", "ID", "E-MAIL")
	for _, item := range admins {
		fmt.Printf("%-25s %s\n", item.PublicID, item.Email)
	}
	fmt.Printf("\n%d administrador(es).\n", len(admins))
	return nil
}

func runCreate(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("create", flag.ExitOnError)
	target := registerTarget(flags)
	email := flags.String("email", "", "e-mail do novo administrador")
	if err := flags.Parse(args); err != nil {
		return err
	}

	clean := strings.TrimSpace(*email)
	if clean == "" {
		return errors.New("-email is required")
	}
	if err := loadEnv(*target); err != nil {
		return err
	}

	pool, store, err := connect(ctx, *target)
	if err != nil {
		return err
	}
	defer pool.Close()

	existing, err := findByEmail(ctx, store, clean)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("ja existe um administrador com o e-mail %s (id %s)", clean, existing.PublicID)
	}

	password, err := promptNewPassword(fmt.Sprintf("Senha para %s", clean))
	if err != nil {
		return err
	}

	created, err := createAdmin(ctx, store, clean, password)
	if err != nil {
		return err
	}

	slog.Info("admin created", "email", created.Email, "id", created.PublicID)
	return nil
}

func runPasswd(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("passwd", flag.ExitOnError)
	target := registerTarget(flags)
	email := flags.String("email", "", "e-mail do administrador")
	if err := flags.Parse(args); err != nil {
		return err
	}

	clean := strings.TrimSpace(*email)
	if clean == "" {
		return errors.New("-email is required")
	}
	if err := loadEnv(*target); err != nil {
		return err
	}

	pool, store, err := connect(ctx, *target)
	if err != nil {
		return err
	}
	defer pool.Close()

	existing, err := findByEmail(ctx, store, clean)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("nenhum administrador com o e-mail %s", clean)
	}

	password, err := promptNewPassword(fmt.Sprintf("Nova senha para %s", clean))
	if err != nil {
		return err
	}

	hashed, err := crypto.NewBcryptHasher(crypto.DefaultCost).Hash(password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	// O store escreve email e senha juntos; o service da API so mexe no e-mail, por
	// isso a troca de senha existe apenas aqui.
	if _, err := store.Update(ctx, existing.ID, func(a *admin.Admin) (bool, error) {
		a.Senha = hashed
		return true, nil
	}); err != nil {
		return fmt.Errorf("update admin password: %w", err)
	}

	slog.Info("admin password updated", "email", clean, "id", existing.PublicID)
	warnActiveSessions()
	return nil
}

func runDelete(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("delete", flag.ExitOnError)
	target := registerTarget(flags)
	email := flags.String("email", "", "e-mail do administrador a remover")
	if err := flags.Parse(args); err != nil {
		return err
	}

	clean := strings.TrimSpace(*email)
	if clean == "" {
		return errors.New("-email is required")
	}
	if err := loadEnv(*target); err != nil {
		return err
	}

	pool, store, err := connect(ctx, *target)
	if err != nil {
		return err
	}
	defer pool.Close()

	admins, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("list admins: %w", err)
	}

	var found *admin.Admin
	for i := range admins {
		if strings.EqualFold(admins[i].Email, clean) {
			found = &admins[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("nenhum administrador com o e-mail %s", clean)
	}
	// Sem esta trava o comando consegue apagar o ultimo acesso ao painel, e a unica
	// recuperacao seria rodar `create` de novo com acesso direto ao banco.
	if len(admins) == 1 {
		return errors.New("este e o unico administrador cadastrado: crie outro antes de remover este")
	}

	confirmed, err := confirm(fmt.Sprintf("Digite o e-mail %s para confirmar a remocao: ", found.Email), found.Email)
	if err != nil {
		return err
	}
	if !confirmed {
		return errors.New("confirmacao nao confere: nada foi removido")
	}

	if err := store.Delete(ctx, found.ID); err != nil {
		return fmt.Errorf("delete admin: %w", err)
	}

	slog.Info("admin deleted", "email", found.Email, "id", found.PublicID)
	warnActiveSessions()
	return nil
}

// --- infraestrutura compartilhada ---

func registerTarget(flags *flag.FlagSet) *string {
	return flags.String("target", targetLocal, "banco alvo: local (DATABASE_URL) ou prod (PROD_DATABASE_URL)")
}

// databaseEnvFor devolve a variavel que guarda a URL do banco de cada alvo. O alvo
// padrao e o local: producao exige `-target=prod` explicito, para que rodar o
// comando durante o desenvolvimento nunca toque em producao.
func databaseEnvFor(target string) (string, error) {
	switch target {
	case targetLocal:
		return "DATABASE_URL", nil
	case targetProd:
		return "PROD_DATABASE_URL", nil
	default:
		return "", fmt.Errorf("invalid -target %q: use %s or %s", target, targetLocal, targetProd)
	}
}

func loadEnv(target string) error {
	if _, err := databaseEnvFor(target); err != nil {
		return err
	}

	_ = godotenv.Load(".env")
	if target == targetProd {
		// .env.prod so e carregado no alvo de producao. Carregar sempre faria
		// PROD_DATABASE_URL vazar para execucoes locais.
		_ = godotenv.Overload(".env.prod")
	}
	return nil
}

func connect(ctx context.Context, target string) (*pgxpool.Pool, admin.AdminStore, error) {
	dbEnv, err := databaseEnvFor(target)
	if err != nil {
		return nil, nil, err
	}

	dbURL := strings.TrimSpace(os.Getenv(dbEnv))
	if dbURL == "" {
		return nil, nil, fmt.Errorf("%s is required for -target=%s", dbEnv, target)
	}

	slog.Info("connecting", "target", target, "host", safeHost(dbURL))

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}

	return pool, admin.NewAdminStore(pool), nil
}

// safeHost extrai apenas host:porta da URL de conexao, para registrar o alvo sem
// expor usuario e senha no log.
func safeHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return "desconhecido"
	}
	return parsed.Host
}

// findByEmail normaliza a ausencia de registro em (nil, nil): a maioria dos
// subcomandos precisa distinguir "nao existe" de "falhou a consulta".
func findByEmail(ctx context.Context, store admin.AdminStore, email string) (*admin.Admin, error) {
	found, err := store.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, admin.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin by email: %w", err)
	}
	return found, nil
}

func createAdmin(ctx context.Context, store admin.AdminStore, email, password string) (*admin.Admin, error) {
	hashed, err := crypto.NewBcryptHasher(crypto.DefaultCost).Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash admin password: %w", err)
	}

	created, err := store.Create(ctx, admin.AdminInput{Email: email, Senha: hashed})
	if err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}
	return created, nil
}

func warnActiveSessions() {
	// Os JWTs emitidos nao tem revogacao: continuam valendo ate expirar por conta
	// propria (auth.TokenTTL, hoje 24h). Trocar a senha nao derruba quem ja entrou.
	fmt.Fprintln(os.Stderr, "\nAtencao: sessoes ja abertas continuam validas ate o token expirar (24h). Nao ha revogacao de JWT.")
}

// --- entrada no terminal ---

// promptNewPassword le a senha duas vezes com o eco desligado. A senha nunca vem
// por flag nem por argumento: isso a deixaria no historico do shell e visivel em
// `ps` para qualquer processo da maquina.
func promptNewPassword(label string) (string, error) {
	first, err := promptPassword(label + ": ")
	if err != nil {
		return "", err
	}
	// Mesma regra do painel: admin.ValidarSenha e a fonte unica.
	if err := admin.ValidarSenha(first); err != nil {
		return "", err
	}

	second, err := promptPassword("Repita a senha: ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("as senhas nao conferem")
	}

	return first, nil
}

func promptPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("este subcomando precisa de um terminal interativo para ler a senha")
	}

	fmt.Fprint(os.Stderr, prompt)
	raw, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("ler senha: %w", err)
	}

	return string(raw), nil
}

func confirm(prompt, expected string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("ler confirmacao: %w", err)
		}
		return false, nil
	}

	return strings.TrimSpace(scanner.Text()) == expected, nil
}
