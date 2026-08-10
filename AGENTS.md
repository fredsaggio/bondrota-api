# BondRota API — contexto para agentes

## O que é este repositório

API REST do BondRota, sistema de transporte universitário para municípios do interior. Ela concentra autenticação e autorização, cadastros, vínculos de clientes, reservas, planejamento de viagens, execução pelos motoristas, rastreamento, rotas via OSRM e arquivos via Supabase Storage.

O consumidor administrativo está no repositório irmão ../bondrota-admin-web. Mudanças em endpoints, payloads, cookies ou permissões podem exigir alteração sincronizada no frontend.

## Stack e execução

- Go: siga a versão declarada em go.mod.
- HTTP: Chi; PostgreSQL: pgx/v5; migrations: Goose.
- Senhas: bcrypt; tokens: JWT HS256.
- Docker Compose fornece o PostgreSQL local; Air é usado para hot reload.
- Base HTTP: /api/v1.

~~~bash
make start
make build
make test
make test/integration
go test ./...
go vet ./...
~~~

Os testes unitários não dependem de banco. Os testes de repository com tag integration iniciam um PostgreSQL temporário via Testcontainers e exigem Docker.

## Arquitetura

A organização é uma arquitetura em camadas por domínio, sem framework pesado:

~~~text
cmd/main.go            configuração, middlewares globais e ciclo do processo
cmd/dependencies.go    composition root e injeção das dependências
internal/server/       rotas HTTP e middlewares transversais
internal/<domínio>/    model -> repository/store -> service -> handler
internal/db/           interface DB, pool e migrations
internal/auth/         JWT, claims, roles e autorização
internal/geo/          OSRM e cálculo geográfico
internal/storage/      URLs assinadas do Supabase
~~~

Fluxo normal:

~~~text
HTTP -> handler -> service -> repository/store -> PostgreSQL
~~~

- Handler decodifica/valida entrada, traduz erros e define status/JSON.
- Service contém regra de negócio e orquestra dependências.
- Repository/store contém SQL e persistência.
- Arquivos *_model.go definem entidades, inputs e interfaces usadas para injeção/mocks.
- O wiring de um novo domínio deve passar por cmd/dependencies.go e internal/server/server.go.

Não coloque SQL em handler, regra HTTP em repository nem acesso direto ao banco no service quando já houver uma interface de store.

## Vocabulário do domínio

- Município: catálogo local importado do IBGE.
- Destino: faculdade/local atendido, pertencente a um município.
- Parada: ponto geográfico.
- Rota interna: sequência ordenada de paradas.
- Cliente: passageiro cadastrado.
- Vínculo: relação estudante/estágio do cliente com destino, turno, rota e horários fixos.
- Reserva: intenção de viajar em uma data, turno e sentido (ida ou volta).
- Ciclo de viagem: agrupamento operacional que associa data, turno, rota, veículo e motorista.
- Viagem: uma perna de ida ou volta do ciclo.
- Viagem-reserva: associação do passageiro à viagem e seu status de presença.
- Rota dinâmica: geometria e ordem de destinos calculadas para uma viagem.
- Execução de planejamento: registro idempotente das tentativas automáticas de criar ciclos/viagens.

Turnos aparecem como MT, VT, NT e, em alguns cadastros, IN. Timestamps operacionais seguem APP_TIMEZONE; não use implicitamente o timezone da máquina.

## Regras de negócio críticas

- Reservas fecham antes da partida; a API sempre revalida prazo e disponibilidade.
- Alterações em reservas podem invalidar/recalcular a rota dinâmica.
- Planejamento agrupa demanda elegível, aloca veículo/motorista e cria ida/volta.
- A volta reutiliza o ciclo da ida e pode precisar existir mesmo sem passageiros elegíveis.
- O processador usa lock/advisory key e execução persistida para evitar duplicidade concorrente.
- Falhas de planejamento têm retry/backoff persistido.
- A localização é atualizada pelo motorista e consultada por admin, motorista ou cliente autenticado.
- Consulte docs/api-reference.md e os testes do domínio antes de simplificar qualquer regra.

## Autenticação e autorização

Roles válidas: admin, cliente e motorista.

- Admin web: cookie HttpOnly bondrota_admin_session; login em /admin/login, sessão em /admin/session e logout em /admin/logout.
- O login do painel envia X-Admin-Session-Mode: cookie e recebe 204 sem JWT no corpo.
- Clientes e motoristas continuam usando Authorization: Bearer <JWT>.
- O grupo autenticado aceita Bearer ou o cookie administrativo.
- Admin pode operar recursos administrativos.
- Cliente só pode acessar o próprio cliente, vínculos e reservas.
- Motorista só pode operar viagens para as quais está alocado; admin pode fazer bypass explícito.
- O processador interno usa PLANNING_CRON_SECRET, separado dos JWTs de usuário.
- Os três logins possuem rate limit por IP e por identidade.

Autenticação não substitui autorização. Ao criar endpoint com ID na URL, determine explicitamente dono, role permitida e eventual bypass de admin. Teste acessos do dono, de outro usuário e de role incorreta.

## Segurança que não deve regredir

- Nunca exponha senha, service key ou cron secret. JWT só aparece nas respostas de login Bearer/legado; o login cookie-mode do painel não deve devolvê-lo.
- Preserve cookie HttpOnly, Secure em produção, SameSite e validação de Origin/CSRF.
- ALLOWED_ORIGINS usa origens exatas; não habilite wildcard com credentials.
- Não confie em X-Forwarded-For sem configuração explícita de proxy confiável.
- LOGIN_RATE_LIMIT_TRUST_PROXY_HEADERS deve ficar false quando houver acesso direto à API.
- Queries de recurso pertencente a usuário devem filtrar/validar ownership no backend.
- Use mensagens genéricas em login para não revelar se a conta existe.
- O limite global de body é 250 KiB; uploads passam por URLs assinadas, não pelo corpo da API.

## Persistência e migrations

- A interface central de banco está em internal/db/db.go.
- Use context.Context recebido; não crie context.Background dentro do fluxo de request.
- Preserve wrapping de erros com %w e sentinels existentes.
- Detecte constraints com helpers do pacote db quando aplicável.
- Mudança de schema exige uma nova migration sequencial em internal/db/migrations.
- Não reescreva migration já aplicada, salvo pedido explícito para corrigir histórico ainda não publicado.
- Operações multi-etapa que precisam ser atômicas devem usar transação.

## Mocks gerados

internal/mocks é gerado pelo Mockery.

- Não leia nem analise arquivos gerados inteiros.
- Descubra contratos nos arquivos internal/*/*_model.go.
- Em testes, use apenas construtores e métodos públicos, como mocks.NewReservaStore(t) e mockStore.EXPECT().Method(...).
- Se um mock precisar mudar, edite somente .mockery.yml e execute:

~~~bash
mockery
~~~

Não edite arquivos dentro de internal/mocks manualmente.

## Testes e convenções

- Testes de handler cobrem parsing, status HTTP e tradução de erros.
- Testes de service cobrem regras de negócio com interfaces/mocks.
- Testes de repository ficam em internal/integration/repositories e usam transação revertida ao final.
- Testes end-to-end ficam em internal/tests.
- Prefira testes table-driven quando houver matriz de roles, erros ou estados.
- Depois de editar Go, rode gofmt nos arquivos tocados.
- Para mudanças concorrentes, rode o teste relevante também com -race.
- Não use banco/serviço live em teste comum; testes live devem ser explícitos e opt-in.

## Serviços externos e jobs

- OSRM: cálculo de rotas; OSRM_BASE_URL pode apontar para instância dedicada.
- Supabase Storage: somente URLs assinadas; SUPABASE_SERVICE_KEY fica no backend.
- Supabase Cron chama POST /internal/planejamentos/processar a cada minuto.
- O worker de rota dinâmica também roda no processo principal e respeita cancelamento do contexto.

## Checklist de conclusão

1. Confirme a camada e o domínio corretos para a alteração.
2. Verifique autenticação, role e ownership de cada rota afetada.
3. Adicione/atualize testes proporcionais ao risco.
4. Rode gofmt, go test ./... e go vet ./....
5. Para repository/schema, rode os testes de integração e revise a migration.
6. Atualize docs/api-reference.md e docs/swagger.yaml se o contrato HTTP mudar.
7. Confira o frontend irmão quando payload, sessão ou endpoint administrativo mudar.
8. Não inclua secrets, artefatos gerados ou alterações alheias à tarefa.
