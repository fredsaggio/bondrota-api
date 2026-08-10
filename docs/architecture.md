# Arquitetura da BondRota API

## Visão geral

Projeto escrito em Go, organizado como uma API HTTP REST leve. A aplicação segue uma arquitetura em camadas com responsabilidades bem definidas: apresentação (handlers), regras de negócio (services), persistência (stores/repositories) e infraestrutura (db, crypto, server).

## Estrutura do repositório (resumo)

- `cmd/` - ponto de entrada da aplicação (`main.go`) e construção das dependências.
- `internal/server` - inicialização do servidor HTTP e registro de rotas (utiliza `chi`).
- `internal/*` - pacotes por domínio (admin, auth, clientes, motoristas, destinos, reservas, rotasinternas, veiculos, viagens).
- `internal/db` - abstração de acesso ao banco e pool (`pgxpool`).
- `internal/crypto` - implementações de hashing de senha (`bcrypt`).
- `migrations/` - scripts SQL de migração do schema.

## Fluxo de requisição

1. `cmd/main.go` configura o router `chi`, middlewares (logger, recoverer, CORS) e monta um `apiRouter` registrado em `/api/v1`.
2. `server.NewServer` recebe um conjunto de handlers (`server.Handlers`) e os registra via `RegisterRoutes`.
3. Cada handler (p.ex. `internal/admin/handler.go`) parseia a requisição, valida dados e delega para o service correspondente.
4. Os services (`internal/*/service.go`) contêm a lógica de negócio e orquestram chamadas ao store (repositório) e ao `auth` quando necessário.
5. Stores/Repositories (`internal/*/repository.go`) executam queries usando a interface `db.DB` baseada em `pgx`/`pgxpool`.
6. Responses são serializadas em JSON e retornadas ao cliente.

## Autenticação e autorização

- O projeto usa JWTs via `github.com/golang-jwt/jwt/v5` implementado em `internal/auth`.
- `AuthService` fornece `GenerateToken`, `ValidateToken`, `HashPassword` e `CheckPassword`.
- Há middlewares auxiliares: `Authenticate` (verifica header `Authorization: Bearer ...`) e `RequireRole` para autorização baseada em papel.
- O `AdminService` usa `AuthService` para hash de senha e geração/validação do token.

## Banco de dados

- Conexão gerenciada por `internal/db/pool.go` usando `pgxpool`.
- A interface `db.DB` é definida para facilitar testes e abstrair `pgx`.
- Queries usam named args e funções utilitárias do `pgx` para coleta de linhas.
- Transações são feitas com `pgx.BeginFunc` onde necessário.
- Migrações SQL estão em `internal/db/migrations`.

## Segurança

- Senhas armazenadas com `bcrypt` (`internal/crypto`).
- JWTs assinados com `HS256` usando `JWT_SECRET`.
- Middlewares `Recoverer` e `Logger` do `chi` ajudam a estabilidade e observabilidade.
- CORS configurado via `github.com/go-chi/cors` com `ALLOWED_ORIGINS`.

## Padrões e boas práticas adotadas

- Separação clara entre handlers, services e stores (Repository pattern).
- Injeção de dependências via `buildHandlers` em `cmd/dependencies.go`.
- Interface `db.DB` para facilitar mocks em testes.
- Tratamento de erros consistente com valores sentinel (`ErrNotFound`, `ErrInvalidCredentials`).
- Uso de context (`context.Context`) para propagação de prazo/cancelamento e dados (claims JWT no contexto).

## Escalabilidade e manutenção

- Adicionar novos domínios: criar pacote em `internal/`, implementar `service`, `repository` e `handler`, e registrar o handler em `server.RegisterRoutes` e `buildHandlers`.
- Para alta disponibilidade, a configuração atual já permite rodar múltiplas instâncias por trás de um load balancer; o banco deve ser escalado separadamente.

## Processamento do planejamento de viagens

- `AgendadorPlanejamentoStore` descobre candidatos de ida a partir de reservas confirmadas e candidatos de volta a partir dos ciclos de ida existentes.
- `ProcessadorPlanejamento.Processar` executa uma varredura unica. Ele considera o dia atual e o seguinte no fuso de `APP_TIMEZONE`, necessario para partidas logo apos a meia-noite.
- Um candidato so e processado entre `partida - 30 minutos` e o instante da partida.
- `ExecucaoPlanejamentoStore` adquire cada combinacao de data, turno, municipio de destino, rota e sentido. Isso impede processamento duplicado por chamadas concorrentes.
- Uma execucao termina como `concluido`, `sem_demanda` ou `falhou`. Falhas podem ser adquiridas novamente; execucoes interrompidas liberam o bloqueio depois de cinco minutos.
- A volta reutiliza os ciclos da ida. Por isso, ela e processada mesmo sem passageiros elegiveis, garantindo a criacao da viagem de retorno dos veiculos.
- O processador nao possui ticker proprio nem endpoint neste estágio. O disparo periodico externo deve apenas chamar `Processar`.

## Deploy e execução

- `Dockerfile` e `docker-compose.yml` estão presentes para facilitar execução local e containerização.
- Variáveis importantes: `DATABASE_URL`, `JWT_SECRET`, `PORT`, `ALLOWED_ORIGINS`.

## Próximos passos recomendados

- Registrar rotas dos demais módulos (clientes, veiculos, viagens, etc.) e adicionar testes de integração.
- Adicionar documentação OpenAPI/Swagger para facilitar consumo externo.
- Implementar rate limiting e logs estruturados mais completos (ex.: request id tracing).
