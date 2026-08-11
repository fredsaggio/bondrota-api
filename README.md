# BondRota API

API REST para gerenciamento de transporte universitário em cidades do interior do Brasil. Permite o controle de rotas, motoristas, clientes, reservas e viagens, com suporte a planejamento automático de veículos e rota dinâmica via OSRM.

## Funcionalidades

- Autenticação JWT com roles (`admin`, `motorista`, `cliente`)
- Gestão de veículos, motoristas, clientes e destinos
- Reservas de ida e volta por turno
- Planejamento automático de viagens com alocação de veículos e motoristas
- Rota dinâmica calculada via OSRM com worker de atualização automática
- Rastreamento de localização do motorista em tempo real
- Upload e download de arquivos via Supabase Storage (URLs assinadas)

## Stack

- **Go 1.26** com [Chi](https://github.com/go-chi/chi)
- **PostgreSQL** via [pgx/v5](https://github.com/jackc/pgx)
- **Goose** para migrations
- **Docker** para ambiente local
- **Air** para hot reload em desenvolvimento

## Pré-requisitos

- Go 1.26+
- Docker e Docker Compose
- [Air](https://github.com/air-verse/air) (`go install github.com/air-verse/air@latest`)
- [Goose](https://github.com/pressly/goose) (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

## Configuração

Crie um arquivo `.env` na raiz do projeto:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/bondrota_db?sslmode=disable
BASE_CITY=Campo Alegre
APP_TIMEZONE=America/Maceio
PORT=8080
ALLOWED_ORIGINS=http://localhost:3000
JWT_SECRET=seu_jwt_secret
AUTH_COOKIE_SECURE=false
AUTH_COOKIE_SAME_SITE=lax
LOGIN_RATE_LIMIT_PER_IP=20
LOGIN_RATE_LIMIT_PER_IDENTITY=5
LOGIN_RATE_LIMIT_WINDOW=1m
LOGIN_RATE_LIMIT_TRUST_PROXY_HEADERS=false
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_SERVICE_KEY=sua_service_key
OSRM_BASE_URL=https://router.project-osrm.org
```

## Rodando localmente

```bash
# Inicia o banco, roda as migrations e sobe a API com hot reload
make start
```

Ou passo a passo:

```bash
# Sobe apenas o banco
make infra/up

# Roda as migrations
make migration/up

# Importa o catálogo local de municípios uma vez
make municipios/import

# Sobe a API com hot reload
air
```

A API estará disponível em `http://localhost:8080/api/v1`.

A cidade base identifica a única cidade atendida pela instância e fica fora do
banco de dados operacional. O frontend pode consultá-la em `GET /api/v1/config`,
que retorna `{"cidade_base":"Campo Alegre","fuso_horario":"America/Maceio"}`.
O fuso é o mesmo configurado em `APP_TIMEZONE`; use-o para calcular datas como
"hoje" no fuso correto em vez do fuso do navegador.

## Comandos disponíveis

```bash
make help           # Lista todos os comandos
make start          # Inicia banco + migrations + API
make infra/up       # Sobe apenas o banco Docker
make infra/down     # Para o banco Docker
make reset          # Para containers e remove volumes
make build          # Compila o binário
make test           # Roda testes unitários
make test/integration  # Roda testes de repository (requer Docker)
make municipios/import # Importa todos os municipios da API do IBGE
make municipios/import uf=AL # Importa somente uma UF
make municipios/import/prod # Importa no banco definido por PROD_DATABASE_URL
make db             # Abre psql no banco local
make logs           # Tail dos logs da API
```

### Migrations

```bash
make migration/new name=nome_da_migration   # Cria nova migration
make migration/up                           # Aplica migrations localmente
make migration/down                         # Reverte última migration
make migration/status                       # Status das migrations locais
make migration/up/prod                      # Aplica migrations em produção
make migration/status/prod                  # Status das migrations em produção
make migration/fix                          # Converte timestamps para sequencial
```

## Seed de administrador

Para criar o primeiro administrador, configure as variáveis `ADMIN_EMAIL` e
`ADMIN_PASSWORD` no `.env` e execute:

```bash
go run ./cmd/seed-admin
```

O comando usa o banco local de `DATABASE_URL` por padrão. Para semear o banco de
produção é preciso pedir isso explicitamente, e aí ele lê `PROD_DATABASE_URL`:

```bash
go run ./cmd/seed-admin -target=prod
```

Use `-target=prod` apenas de propósito: ele cria um administrador real, com a senha
que estiver em `ADMIN_PASSWORD`. O comando registra o alvo e o host antes de conectar,
sem expor credenciais, para que dê para conferir onde a escrita vai acontecer.

## Estrutura do projeto

```
.
├── cmd/
│   ├── main.go              # Entrypoint
│   ├── dependencies.go      # Wiring de dependências
│   ├── import-municipios/   # Importador do catálogo oficial do IBGE
│   └── seed-admin/          # Comando para criar admin inicial
├── internal/
│   ├── admin/               # Domínio de administradores
│   ├── auth/                # JWT, middleware e roles
│   ├── clientes/            # Domínio de clientes e vínculos
│   ├── destinos/            # Destinos (faculdades/locais)
│   ├── geo/                 # Cliente OSRM e otimizador de rota
│   ├── motoristas/          # Domínio de motoristas
│   ├── municipios/          # Catálogo local de municípios do IBGE
│   ├── paradas/             # Paradas intermediárias
│   ├── reservas/            # Reservas de clientes
│   ├── rotasdinamicas/      # Cálculo e persistência de rotas dinâmicas
│   ├── rotasinternas/       # Rotas fixas com paradas ordenadas
│   ├── server/              # Registro de rotas HTTP
│   ├── storage/             # Integração com Supabase Storage
│   ├── veiculos/            # Domínio de veículos e alocação
│   └── viagens/             # Viagens, planejamento, presença e localização
│       └── db/migrations/   # Migrations SQL (Goose)
└── docs/                    # Documentação
```

## Documentação da API

Consulte [`docs/api-reference.md`](docs/api-reference.md) para a referência completa de endpoints, exemplos de request/response, regras de negócio e fluxo da aplicação.

## Deploy

A API é containerizada via Docker e pode ser deployada em qualquer plataforma que suporte containers. O `Dockerfile` usa build multi-stage com imagem final `distroless` para manter o binário mínimo e seguro.

Variáveis de ambiente necessárias em produção: `DATABASE_URL`, `BASE_CITY`, `APP_TIMEZONE`, `PORT`, `ALLOWED_ORIGINS`, `JWT_SECRET`, `PLANNING_CRON_SECRET`, `SUPABASE_URL` e `SUPABASE_SERVICE_KEY`. O painel administrativo autentica por cookie HttpOnly: em produção, use `AUTH_COOKIE_SECURE=true`; use `AUTH_COOKIE_SAME_SITE=none` apenas quando painel e API estiverem em sites diferentes (essa opção exige HTTPS). `AUTH_COOKIE_NAME` e `AUTH_COOKIE_DOMAIN` são opcionais. `ALLOWED_ORIGINS` deve listar origens exatas e não aceita `*`. `APP_TIMEZONE` deve usar um nome IANA, como `America/Maceio`, e determina o fuso dos horários operacionais e dos limites de reserva. `PLANNING_CRON_SECRET` deve ter pelo menos 32 caracteres e autentica exclusivamente o processador interno de planejamentos. Configure também `OSRM_BASE_URL` para usar uma instância dedicada do OSRM; quando omitida, a API usa o servidor público de demonstração.

Os três logins possuem rate limit em memória por IP e por e-mail/CPF. Os padrões
são 20 tentativas por IP e 5 por identidade a cada minuto; ajuste-os com
`LOGIN_RATE_LIMIT_PER_IP`, `LOGIN_RATE_LIMIT_PER_IDENTITY` e
`LOGIN_RATE_LIMIT_WINDOW`. Em múltiplas réplicas, complemente essa proteção no
proxy ou WAF, pois cada processo mantém seu próprio contador.

Mantenha `LOGIN_RATE_LIMIT_TRUST_PROXY_HEADERS=false` quando a API puder receber
tráfego direto. Ative-o somente quando a API estiver exclusivamente atrás de um
proxy confiável que sobrescreva `X-Forwarded-For`, como no serviço web do Render.

### CI/CD com GitHub Actions e Render

O workflow `.github/workflows/ci.yml` é executado em pull requests e pushes para
`main`. Ele verifica a formatação, executa `go vet`, testes unitários, testes de
integração, valida o `Dockerfile` e, em pushes para `main`, aplica as migrations
de produção.

No GitHub, crie um environment chamado `production` em
`Settings > Environments` e adicione o secret:

```text
PROD_DATABASE_URL=postgresql://usuario:senha@host:porta/database?sslmode=require
```

No serviço do Render, mantenha o repositório e a branch `main` conectados e
configure `Settings > Auto-Deploy` como `After CI Checks Pass`. O Render fará o
deploy somente depois que os testes, o build da imagem e as migrations passarem.
As variáveis usadas durante a execução da API continuam configuradas diretamente
no Render, incluindo a variável `DATABASE_URL`.

Para preparar um Supabase vazio, acesse `Actions > Bootstrap production database
> Run workflow`. Esse workflow aplica as migrations e, opcionalmente, importa o
catálogo de municípios do IBGE. A UF pode ser deixada vazia para importar todo o
Brasil. O bootstrap não é executado em deploys comuns.

### Cron de planejamento no Supabase

1. Gere um segredo com `openssl rand -hex 32` e configure o resultado como `PLANNING_CRON_SECRET` no Render.
2. No Supabase, habilite os módulos Cron (`pg_cron`) e `pg_net` pelo Dashboard.
3. Abra [`deploy/supabase/planning_cron.sql`](deploy/supabase/planning_cron.sql), substitua `<PLANNING_CRON_SECRET>` pelo mesmo segredo do Render e `<RENDER_PLANNING_ENDPOINT>` pela URL completa `https://SEU-SERVICO.onrender.com/api/v1/internal/planejamentos/processar`.
4. Execute o SQL uma única vez no SQL Editor do Supabase. Os valores ficam criptografados no Vault e o job chama a API a cada minuto.

### Cron de retenção de dados no Supabase

As tabelas operacionais (`ciclos_viagem`, `viagens` e dependentes, `reservas` e
`execucoes_planejamento`) crescem a cada dia de operação e nunca são limpas pelo
fluxo normal. A limpeza roda **diariamente** e apaga o que passou da janela de
retenção, mantendo os últimos 3 meses para auditoria externa.

1. Execute o `planning_cron.sql` primeiro: o script de retenção reutiliza o schema
   `bondrota_internal` e o segredo `bondrota_planning_cron_secret`.
2. Abra [`deploy/supabase/retention_cron.sql`](deploy/supabase/retention_cron.sql),
   substitua `<RENDER_RETENTION_ENDPOINT>` pela URL completa
   `https://SEU-SERVICO.onrender.com/api/v1/internal/retencao/limpar` e execute o SQL
   uma única vez no SQL Editor do Supabase.

A janela e o tamanho do lote são configuráveis no Render:

| Variável | Padrão | Para que serve |
| --- | --- | --- |
| `RETENTION_MONTHS` | `3` | Meses de dados mantidos, contados no fuso de `APP_TIMEZONE`. |
| `RETENTION_BATCH_LIMIT` | `5000` | Máximo de linhas removidas por tabela em cada execução. |

O lote existe porque o `pg_net` encerra a chamada HTTP em 55s; apagar um trimestre
inteiro de uma vez estouraria o tempo e seguraria lock nas tabelas. Quando o limite
é atingido, a resposta traz `"lote_saturado": true` — sinal de que sobrou trabalho
para a execução seguinte e que o limite pode precisar ser aumentado.

A ordem de remoção importa: os ciclos saem primeiro e o `ON DELETE CASCADE` leva
junto viagens, `viagem_reservas`, `viagem_reserva_confirmacoes`, `viagem_horarios`,
`viagem_localizacoes`, `rotas_dinamicas` e `rota_dinamica_destinos`. Só então as
reservas ficam livres da FK `RESTRICT` de `viagem_reservas`. Cadastros (clientes,
vínculos, motoristas, veículos, destinos, rotas) nunca são tocados.

Para consultar as execuções do cron:

```sql
select * from cron.job_run_details order by start_time desc limit 20;
select * from net._http_response order by created desc limit 20;
```

## Testes

```bash
# Testes unitários (sem banco)
make test

# Testes de integração dos repositories
# Inicia um PostgreSQL 16 temporário e aplica as migrations automaticamente
make test/integration
```

Os testes de integração ficam em `internal/integration/repositories`. Cada teste
executa em uma transação própria, revertida ao final, e não utiliza o banco local
nem as variáveis `DATABASE_URL` ou `E2E_DATABASE_URL`.

## Licença

Projeto acadêmico. Todos os direitos reservados.
