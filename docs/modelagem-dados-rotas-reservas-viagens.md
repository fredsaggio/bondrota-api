# Modelagem de dados, testes e proximos passos

Documento criado em 2026-06-05 com base nas migrations atuais em `internal/db/migrations`, nas rotas registradas em `internal/server/server.go` e na estrutura dos pacotes em `internal/`.

## Resumo executivo

Hoje o banco ja cobre a base administrativa do sistema: administradores, veiculos, motoristas, destinos, paradas, rotas internas, clientes, vinculos dos clientes, horarios fixos e reservas.

Ja existem migration e endpoints funcionais para `reservas`. Ainda nao existem migrations nem endpoints funcionais para `viagens` e `rotas dinamicas`. A pasta `internal/viagens` existe, mas os arquivos estao praticamente vazios e esses handlers nao sao registrados no servidor.

Para as rotas inteligentes, a melhor decisao e persistir o resultado no banco. Redis pode ser usado depois como cache auxiliar, mas nao deve ser a fonte principal da rota, porque o motorista e o aluno precisam conseguir consultar a viagem planejada mesmo se um cache cair. A rota calculada deve virar dado persistido com prazo de expiracao.

## Modelo logico atual

```mermaid
erDiagram
    ADMINISTRADOR {
        bigint id PK
        text email UK
        text senha
        timestamptz created_at
        timestamptz updated_at
    }

    VEICULOS {
        bigint id PK
        text placa UK
        text modelo
        smallint capacidade
        text cidade_base
        status_veiculo status
        boolean ar_condicionado
        boolean banheiro
        boolean persiana
        boolean luz_leitura
        boolean tomada
        timestamptz created_at
        timestamptz updated_at
    }

    MOTORISTAS {
        bigint id PK
        text nome
        text cpf UK
        text senha
        text telefone
        date data_nasc
        turno_motorista turno
        text cidade_trabalho
        text residencia
        text foto
        timestamptz created_at
        timestamptz updated_at
    }

    DESTINOS {
        bigint id PK
        text nome
        text rua
        text cidade
        numeric latitude
        numeric longitude
        timestamptz created_at
        timestamptz updated_at
    }

    PARADAS {
        bigint id PK
        text nome
        numeric latitude
        numeric longitude
        text cidade
        timestamptz created_at
        timestamptz updated_at
    }

    ROTAS_INTERNAS {
        bigint id PK
        text cidade
        timestamptz created_at
        timestamptz updated_at
    }

    ROTA_INTERNA_PARADAS {
        bigint id PK
        bigint rota_interna_id FK
        bigint parada_id FK
        int ordem
    }

    CLIENTES {
        bigint id PK
        text nome
        text cpf UK
        text senha
        text telefone
        date data_nasc
        text foto
        timestamptz created_at
        timestamptz updated_at
    }

    CLIENTE_VINCULOS {
        bigint id PK
        bigint cliente_id FK
        tipo_conta tipo
        turno_cliente turno
        bigint destino_id FK
        bigint rota_interna_id FK
        text curso
        text comprovante
        date validade
        timestamptz created_at
        timestamptz updated_at
    }

    HORARIOS_FIXOS {
        bigint id PK
        bigint vinculo_id FK
        smallint dia_semana
    }

    ROTAS_INTERNAS ||--o{ ROTA_INTERNA_PARADAS : contem
    PARADAS ||--o{ ROTA_INTERNA_PARADAS : entra_em
    CLIENTES ||--o{ CLIENTE_VINCULOS : possui
    DESTINOS ||--o{ CLIENTE_VINCULOS : destino
    ROTAS_INTERNAS ||--o{ CLIENTE_VINCULOS : rota_base
    CLIENTE_VINCULOS ||--o{ HORARIOS_FIXOS : possui
```

## Como interpretar as tabelas atuais

`clientes` guarda apenas a identidade do usuario: nome, CPF, senha, telefone, data de nascimento e foto.

`cliente_vinculos` guarda a parte operacional do cliente: tipo de conta (`estudante` ou `estagio`), turno (`MT`, `VT`, `NT`, `IN`), destino, rota interna, curso, comprovante e validade. Essa separacao e importante porque um mesmo cliente pode ter mais de um vinculo ao longo do tempo.

`horarios_fixos` guarda os dias da semana do vinculo. A regra atual aceita dias de 1 a 5, ou seja, segunda a sexta.

`destinos` sao as faculdades ou outros locais finais dos clientes. E onde o aluno desembarca na ida e tambem onde ele embarca quando for voltar para casa. Por isso `cliente_vinculos.destino_id` deve ser entendido como o destino do vinculo, nao como a parada de embarque perto da casa do aluno.

`paradas` sao os locais presentes dentro de uma rota interna. Elas representam onde o veiculo passa para pegar alunos na ida e onde passa para deixar alunos na volta. Hoje o sistema nao persiste exatamente em qual parada cada aluno embarca perto de casa. O que fica persistido no vinculo do aluno e o `destino_id`, ou seja, a faculdade/destino onde ele desembarca na ida e embarca na volta.

`rotas_internas` nao sao as rotas inteligentes dinamicas. Elas representam uma rota base por cidade com paradas ordenadas. A rota dinamica ainda precisa ser criada e deve usar reservas reais, turno, cidade, capacidade, veiculo, motorista, destinos e as paradas das rotas internas envolvidas.

`veiculos` ja tem a informacao principal para alocacao inteligente: `capacidade`, `cidade_base` e `status`. O admin nao deve escolher manualmente qual unidade do veiculo vai sair em cada viagem. Ele controla o cadastro e o status. O planejador escolhe automaticamente entre os veiculos ativos da cidade, priorizando capacidade adequada. Hoje o modelo do veiculo ja permite diferenciar uma van de um onibus pelo `modelo` e pela `capacidade`; uma coluna futura `categoria` pode ajudar na interface, mas nao e obrigatoria para o algoritmo.

`motoristas` ja tem `turno`, `cidade_trabalho` e `residencia`, que podem ajudar a filtrar quem pode dirigir uma viagem planejada. Para o MVP, recomendo usar `cidade_trabalho` como cidade operacional do motorista, porque `residencia` pode ser apenas onde ele mora. Se a regra do produto for que a residencia define a cidade base do motorista, vale renomear ou documentar isso depois para evitar ambiguidade. Nao recomendo adicionar destino/faculdade no motorista agora, porque isso aumentaria a refatoracao e nao e necessario para calcular as primeiras viagens.

## Status dos endpoints atuais

Base URL local: `http://localhost:8080/api/v1`

Observacao importante: as rotas de colecao foram registradas com `/` dentro de cada grupo. Para evitar erro 404, prefira usar barra final nos endpoints de colecao, por exemplo `/api/v1/veiculos/` em vez de `/api/v1/veiculos`.

| Area | Endpoints registrados |
| --- | --- |
| Health | `GET /health` |
| Admin | `POST /admin/`, `GET /admin/`, `GET /admin/{adminID}`, `PUT /admin/{adminID}`, `DELETE /admin/{adminID}`, `POST /admin/login` |
| Veiculos | `POST /veiculos/`, `GET /veiculos/`, `GET /veiculos/{veiculoID}`, `PUT /veiculos/{veiculoID}`, `DELETE /veiculos/{veiculoID}` |
| Destinos | `POST /destinos/`, `GET /destinos/`, `GET /destinos/cidade/{cidade}`, `GET /destinos/{id}`, `PUT /destinos/{id}`, `DELETE /destinos/{id}` |
| Paradas | `POST /paradas/`, `GET /paradas/`, `GET /paradas/cidade/{cidade}`, `GET /paradas/{id}`, `PUT /paradas/{id}`, `DELETE /paradas/{id}` |
| Rotas internas | `POST /rotas-internas/`, `GET /rotas-internas/`, `GET /rotas-internas/cidade/{cidade}`, `GET /rotas-internas/{id}`, `PUT /rotas-internas/{id}/paradas`, `DELETE /rotas-internas/{id}` |
| Motoristas | `POST /motoristas/login`, `POST /motoristas/`, `GET /motoristas/`, `GET /motoristas/{id}`, `PUT /motoristas/{id}`, `DELETE /motoristas/{id}` |
| Clientes | `POST /clientes/login`, `POST /clientes/`, `GET /clientes/`, `GET /clientes/{clienteID}`, `PUT /clientes/{clienteID}`, `DELETE /clientes/{clienteID}` |
| Cliente vinculos | `POST /clientes/{clienteID}/vinculos/`, `GET /clientes/{clienteID}/vinculos/`, `GET /clientes/{clienteID}/vinculos/{vinculoID}`, `PUT /clientes/{clienteID}/vinculos/{vinculoID}`, `DELETE /clientes/{clienteID}/vinculos/{vinculoID}` |
| Reservas | `POST /clientes/{clienteID}/vinculos/{vinculoID}/reservas/`, `GET /clientes/{clienteID}/vinculos/{vinculoID}/reservas/`, `GET /clientes/{clienteID}/reservas/`, `GET /reservas/`, `GET /reservas/{reservaID}`, `PUT /reservas/{reservaID}`, `POST /reservas/{reservaID}/cancelar`, `DELETE /reservas/{reservaID}` |
| Viagens | Nao implementado |
| Rotas dinamicas | Nao implementado |

O `server.go` aplica middleware de autenticacao nas rotas protegidas. Apenas `GET /health`, `POST /admin/login`, `POST /clientes/login` e `POST /motoristas/login` ficam publicos. Os demais endpoints devem enviar `Authorization: Bearer <token>`.

## Como subir a API para testar

Crie ou confira um `.env` local com valores equivalentes a estes:

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/bondrota_db?sslmode=disable
PORT=8080
ALLOWED_ORIGINS=http://localhost:3000
JWT_SECRET=troque_este_valor_em_producao
```

Suba o banco:

```bash
docker compose up -d db
```

Rode as migrations:

```bash
make migration/up
```

Se o comando `goose` nao existir na maquina, instale antes:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Inicie a API:

```bash
go run ./cmd
```

Teste se esta viva:

```bash
curl -i http://localhost:8080/api/v1/health
```

## Roteiro de teste com curl

Defina a URL base:

```bash
BASE=http://localhost:8080/api/v1
```

Crie um administrador:

```bash
curl -i -X POST "$BASE/admin/" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@bondrota.local","senha":"senha123"}'
```

Faca login do administrador:

```bash
curl -i -X POST "$BASE/admin/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@bondrota.local","senha":"senha123"}'
```

Crie um veiculo:

```bash
curl -i -X POST "$BASE/veiculos/" \
  -H "Content-Type: application/json" \
  -d '{
    "placa":"ABC1D23",
    "modelo":"Van Sprinter",
    "capacidade":15,
    "cidade_base":"maceio",
    "status":"ativo",
    "ar_condicionado":true,
    "banheiro":false,
    "persiana":true,
    "luz_leitura":false,
    "tomada":true
  }'
```

Crie um motorista:

```bash
curl -i -X POST "$BASE/motoristas/" \
  -H "Content-Type: application/json" \
  -d '{
    "nome":"Joao Motorista",
    "cpf":"11122233344",
    "senha":"senha123",
    "telefone":"82999990000",
    "data_nasc":"1985-03-10",
    "turno":"NT",
    "cidade_trabalho":"maceio",
    "residencia":"Maceio",
    "foto":""
  }'
```

Crie um destino/faculdade para o cliente. Neste exemplo, o destino representa onde o aluno desembarca na ida e embarca na volta:

```bash
curl -i -X POST "$BASE/destinos/" \
  -H "Content-Type: application/json" \
  -d '{
    "nome":"Universidade Federal de Alagoas",
    "rua":"Av. Lourival Melo Mota",
    "cidade":"maceio",
    "latitude":-9.665990,
    "longitude":-35.735000
  }'
```

Crie paradas para uma rota interna. As paradas sao os locais por onde o veiculo passa para pegar alunos na ida e deixar alunos na volta:

```bash
curl -i -X POST "$BASE/paradas/" \
  -H "Content-Type: application/json" \
  -d '{"nome":"Parada Centro","latitude":-9.660000,"longitude":-35.730000,"cidade":"maceio"}'

curl -i -X POST "$BASE/paradas/" \
  -H "Content-Type: application/json" \
  -d '{"nome":"Parada Farol","latitude":-9.670000,"longitude":-35.740000,"cidade":"maceio"}'
```

Crie uma rota interna usando os IDs das paradas criadas:

```bash
curl -i -X POST "$BASE/rotas-internas/" \
  -H "Content-Type: application/json" \
  -d '{
    "cidade":"maceio",
    "paradas":[
      {"parada_id":1,"ordem":1},
      {"parada_id":2,"ordem":2}
    ]
  }'
```

Crie um cliente:

```bash
curl -i -X POST "$BASE/clientes/" \
  -H "Content-Type: application/json" \
  -d '{
    "nome":"Maria Cliente",
    "cpf":"55566677788",
    "senha":"senha123",
    "telefone":"82988880000",
    "data_nasc":"2001-08-20",
    "foto":""
  }'
```

Crie um vinculo para o cliente. Troque `destino_id` e `rota_interna_id` pelos IDs reais retornados nos passos anteriores. Aqui, `destino_id` e a faculdade/destino do aluno; nao e a parada onde ele entra perto de casa:

```bash
curl -i -X POST "$BASE/clientes/1/vinculos/" \
  -H "Content-Type: application/json" \
  -d '{
    "tipo":"estudante",
    "turno":"NT",
    "destino_id":1,
    "rota_interna_id":1,
    "curso":"Sistemas de Informacao",
    "comprovante":"https://exemplo.local/comprovante.pdf",
    "validade":"2026-12-31",
    "horarios_fixos":[1,2,3,4,5]
  }'
```

Liste os dados:

```bash
curl -i "$BASE/veiculos/"
curl -i "$BASE/motoristas/"
curl -i "$BASE/destinos/"
curl -i "$BASE/paradas/"
curl -i "$BASE/rotas-internas/"
curl -i "$BASE/clientes/"
curl -i "$BASE/clientes/1"
curl -i "$BASE/clientes/1/vinculos/"
curl -i "$BASE/clientes/1/vinculos/1"
```

Se uma rota interna falhar com erro de chave estrangeira, provavelmente o ID da parada nao existe. Se o vinculo falhar, confira se `destino_id`, `rota_interna_id`, `tipo`, `turno`, `validade` e `horarios_fixos` estao corretos.

## Como testar com Postman ou Insomnia

Crie um ambiente com:

```text
base_url = http://localhost:8080/api/v1
```

Use sempre `Content-Type: application/json` nas requisicoes com body.

Fluxo recomendado:

1. `GET {{base_url}}/health`
2. `POST {{base_url}}/admin/`
3. `POST {{base_url}}/admin/login`
4. `POST {{base_url}}/veiculos/`
5. `POST {{base_url}}/motoristas/`
6. `POST {{base_url}}/destinos/`
7. `POST {{base_url}}/paradas/` pelo menos duas vezes
8. `POST {{base_url}}/rotas-internas/`
9. `POST {{base_url}}/clientes/`
10. `POST {{base_url}}/clientes/{clienteID}/vinculos/`
11. `GET {{base_url}}/clientes/{clienteID}/vinculos/` para listar somente os vinculos daquele cliente
12. `GET {{base_url}}/clientes/{clienteID}/vinculos/{vinculoID}` quando precisar consultar um vinculo especifico
13. `GET {{base_url}}/clientes/{clienteID}` para conferir o cliente completo com seus vinculos e horarios

Para guardar IDs automaticamente no Postman, adicione este script na aba `Tests` de uma requisicao que retorna `id`:

```javascript
const json = pm.response.json();
if (json.id) {
  pm.environment.set("ultimo_id", json.id);
}
```

## Conferindo direto no banco

Acesse o Postgres:

```bash
docker exec -it bondrota-db psql -U postgres -d bondrota_db
```

Comandos uteis:

```sql
\dt
SELECT id, modelo, capacidade, cidade_base, status FROM veiculos;
SELECT id, nome, turno, cidade_trabalho FROM motoristas;
SELECT id, nome, cidade, latitude, longitude FROM destinos;
SELECT id, cidade FROM rotas_internas;
SELECT rota_interna_id, parada_id, ordem FROM rota_interna_paradas ORDER BY rota_interna_id, ordem;
SELECT id, nome, cpf FROM clientes;
SELECT id, cliente_id, tipo, turno, destino_id, rota_interna_id, curso, validade FROM cliente_vinculos;
SELECT vinculo_id, dia_semana FROM horarios_fixos ORDER BY vinculo_id, dia_semana;
```

## Entidades e ajustes que ainda faltam

### 1. Destinos

Destinos ja existem no banco e representam as faculdades ou outros locais finais dos clientes.

O papel de `destinos` fica assim:

- Na ida, o `destino_id` do vinculo indica onde o aluno vai desembarcar.
- Na volta, o mesmo `destino_id` indica onde o aluno vai embarcar para voltar para casa.
- O sistema nao persiste a parada residencial especifica do aluno.
- A rota interna define as paradas pelas quais o veiculo pode passar para buscar ou deixar alunos.

Como o nome agora e `destinos`, nao precisamos criar outra tabela para faculdades no MVP. Se no futuro houver outros tipos de destino, pode ser adicionada uma coluna `tipo` com valores como `faculdade`, `estagio` ou `outro`.

### 2. Reservas

Reserva e a intencao confirmada do aluno para uma data e turno. Ela deve nascer ligada ao `cliente_vinculo`, porque o vinculo ja contem turno, destino e rota interna.

Campos sugeridos:

```sql
reservas (
  id,
  cliente_id,
  vinculo_id,
  data_viagem,
  turno,
  destino_id, -- faculdade/destino: desembarque na ida e embarque na volta
  rota_interna_id,
  cidade,
  sentido, -- ida, volta
  status, -- confirmada, cancelada
  created_at,
  updated_at
)
```

Essa estrutura foi implementada em `internal/db/migrations/00009_create_reservas.sql`. A coluna `cidade` tambem foi persistida como snapshot operacional da rota interna, para facilitar o agrupamento futuro do planejador por cidade.

Regras implementadas/recomendadas:

- Uma reserva deve copiar `destino_id`, `rota_interna_id`, `turno` e `sentido` no momento da criacao. Isso preserva o historico caso o aluno altere o cadastro depois.
- A criacao da reserva e aninhada no vinculo: `POST /clientes/{clienteID}/vinculos/{vinculoID}/reservas/`. O body nao recebe `vinculo_id`.
- `destino_id` e a faculdade/destino do aluno. Nao e a parada residencial.
- A parada especifica onde o aluno embarca perto de casa nao fica persistida.
- Deve existir uma restricao para evitar duas reservas ativas do mesmo vinculo no mesmo dia, turno e sentido.
- A reserva nasce como `confirmada` pelo default do banco.
- A reserva pode virar `cancelada`.
- Quando o planejador alocar a reserva em uma viagem, isso deve aparecer em `viagem_reservas`, nao no status da reserva.

### 3. Rotas dinamicas

Rota dinamica e o resultado do calculo para uma data, turno, cidade, sentido e conjunto de reservas. Ela deve ser persistida, mas com prazo de vida.

Campos sugeridos:

```sql
rotas_dinamicas (
  id,
  data_viagem,
  turno,
  cidade,
  sentido, -- ida, volta
  status, -- calculada, publicada, em_andamento, concluida, cancelada
  total_reservas,
  distancia_metros,
  duracao_segundos,
  geometria_geojson,
  algoritmo_versao,
  calculada_em,
  expires_at,
  created_at,
  updated_at
)
```

Sequencia operacional da rota dinamica:

```sql
rota_dinamica_paradas (
  id,
  rota_dinamica_id,
  ordem,
  tipo, -- parada_rota, destino_faculdade
  parada_id,
  destino_id,
  nome_snapshot,
  latitude,
  longitude,
  chegada_prevista,
  distancia_acumulada_metros,
  duracao_acumulada_segundos
)
```

Duplicar rotas iguais em turnos ou horarios diferentes e aceitavel e ate recomendado no MVP. Mesmo que a geometria seja igual, o contexto operacional muda: data, turno, reservas, motorista, veiculo e horario. Evitar duplicacao por geometria deixaria o sistema mais complexo sem ganho claro agora.

### 4. Viagens

Viagem nao e apenas historico ou log. Ela e o objeto operacional ativo durante a execucao. Enquanto o veiculo esta rodando, a viagem fica `em_andamento`, o motorista pode atualizar presencas pelo app, e o sistema consegue mostrar para aluno/admin o que esta acontecendo.

Viagem e a execucao planejada ou realizada de uma rota dinamica por um veiculo e um motorista. A atribuicao de `veiculo_id` e `motorista_id` deve ser automatica, feita pelo planejador, nao escolhida manualmente pelo admin.

Campos sugeridos:

```sql
viagens (
  id,
  rota_dinamica_id,
  veiculo_id,
  motorista_id,
  data_viagem,
  turno,
  status, -- programada, em_andamento, concluida, cancelada
  partida_prevista,
  inicio_real,
  fim_real,
  qtd_passageiros_prevista,
  qtd_passageiros_real,
  km_previsto,
  km_real,
  alocada_em,
  expires_at,
  created_at,
  updated_at
)
```

Relacao entre viagem e reservas:

```sql
viagem_reservas (
  id,
  viagem_id,
  reserva_id,
  status_presenca, -- aguardando, embarcou, faltou, cancelado
  horario_confirmacao,
  created_at,
  updated_at
)
```

Quando uma viagem e criada, o sistema deve criar automaticamente uma linha em `viagem_reservas` para cada reserva alocada naquela viagem, iniciando com `status_presenca = 'aguardando'`.

Estados sugeridos de presenca:

- `aguardando`: reserva alocada na viagem, mas o aluno ainda nao embarcou.
- `embarcou`: motorista confirmou que o aluno embarcou.
- `faltou`: aluno nao apareceu ou nao embarcou.
- `cancelado`: reserva saiu da viagem antes ou durante a operacao.

Essa tabela permite saber quantos alunos estavam previstos e quantos realmente foram. `viagens.qtd_passageiros_prevista` pode ser calculado pela quantidade de reservas alocadas; `viagens.qtd_passageiros_real` pode ser calculado pela quantidade de registros com `status_presenca = 'embarcou'`.

Com essa relacao, o aluno consegue saber qual veiculo e motorista ira pegar fazendo o caminho:

```text
reserva -> viagem_reservas -> viagens -> veiculos
reserva -> viagem_reservas -> viagens -> motoristas
reserva -> viagem_reservas -> viagens -> rotas_dinamicas -> rota_dinamica_paradas
```

## Modelo futuro recomendado

```mermaid
erDiagram
    DESTINOS ||--o{ CLIENTE_VINCULOS : destino
    CLIENTES ||--o{ RESERVAS : faz
    CLIENTE_VINCULOS ||--o{ RESERVAS : gera
    DESTINOS ||--o{ RESERVAS : destino_snapshot
    ROTAS_INTERNAS ||--o{ RESERVAS : rota_base_snapshot
    DESTINOS ||--o{ ROTA_DINAMICA_PARADAS : destino
    PARADAS ||--o{ ROTA_DINAMICA_PARADAS : parada_rota
    ROTAS_DINAMICAS ||--o{ ROTA_DINAMICA_PARADAS : possui
    ROTAS_DINAMICAS ||--o| VIAGENS : vira
    VEICULOS ||--o{ VIAGENS : executa
    MOTORISTAS ||--o{ VIAGENS : dirige
    VIAGENS ||--o{ VIAGEM_RESERVAS : transporta
    RESERVAS ||--o{ VIAGEM_RESERVAS : entra_em
```

## Como a rota inteligente deve funcionar

### Entrada do calculo

O calculo deve receber pelo menos:

- `data_viagem`
- `turno`
- `cidade`
- `sentido` (`ida` ou `volta`)

Com isso, o backend busca as reservas ativas daquele dia e turno, juntando:

- reserva
- cliente
- cliente_vinculo
- destino
- rota interna base
- paradas da rota interna

### Turno

O turno deve ser salvo na reserva, mesmo ja existindo em `cliente_vinculos`. Isso cria um snapshot operacional.

Regras sugeridas:

- `MT`: rota da manha
- `VT`: rota da tarde
- `NT`: rota da noite
- `IN`: cliente integral, que pode reservar um turno especifico conforme regra do app

Tambem recomendo criar uma tabela de configuracao de janelas por turno:

```sql
turno_configuracoes (
  id,
  turno,
  sentido,
  horario_saida_padrao,
  horario_chegada_limite,
  tolerancia_minutos
)
```

Assim o algoritmo nao precisa ter horarios fixos escritos no codigo.

### Capacidade inteligente

O algoritmo deve usar `veiculos.capacidade` e `veiculos.status = 'ativo'`.

Fluxo recomendado:

1. Conte quantas reservas existem para `data + turno + cidade + sentido`.
2. Busque veiculos ativos da cidade.
3. Ordene os veiculos pela menor capacidade primeiro.
4. Se todos os alunos cabem em um veiculo, escolha o menor veiculo que comporta todos.
5. Se nao cabem em um veiculo, divida as reservas em grupos.
6. Evite dividir alunos da mesma rota interna ou do mesmo destino quando houver veiculo suficiente.
7. Aloque motorista compativel com cidade e turno.
8. Crie a viagem com o `veiculo_id` e `motorista_id` escolhidos automaticamente.
9. Crie os registros em `viagem_reservas` para todas as reservas daquele grupo.

Exemplos:

- 10 alunos e uma van de 15 disponivel: usar van, nao onibus de 50.
- 40 alunos e um onibus de 50 disponivel: usar um onibus.
- 70 alunos e veiculos de 50 e 30 disponiveis: gerar duas rotas dinamicas, uma para cada veiculo.

Para o MVP, um algoritmo `first-fit decreasing` ja resolve bem:

1. Agrupar reservas por rota interna e destino.
2. Ordenar grupos do maior para o menor.
3. Colocar cada grupo no menor veiculo que ainda tem capacidade.
4. Se nao couber, abrir outro veiculo.

### Alocacao automatica de motorista e veiculo

O admin nao deve escolher manualmente o motorista nem a unidade especifica do veiculo para cada viagem. O papel do admin no MVP deve ser manter os cadastros corretos: veiculos ativos/inativos, capacidade, cidade base, motoristas ativos, cidade de trabalho e turno.

Fluxo recomendado para veiculos:

1. Filtrar por `veiculos.status = 'ativo'`.
2. Filtrar por `veiculos.cidade_base = cidade` da rota/reservas.
3. Remover veiculos ja alocados em outra viagem conflitante no mesmo dia, turno e janela de horario.
4. Escolher o menor veiculo que comporta o grupo de reservas.
5. Se houver empate entre veiculos equivalentes, escolher de forma deterministica, por exemplo menor `id`, ou por uma regra futura de rodizio. Escolha aleatoria tambem funciona no MVP, desde que o resultado seja persistido em `viagens.veiculo_id`.

Fluxo recomendado para motoristas:

1. Filtrar por turno compativel com a viagem.
2. Filtrar por cidade operacional. Para o modelo atual, usar `cidade_trabalho`. Se o produto decidir que `residencia` e a cidade base real do motorista, documentar essa regra ou ajustar o nome da coluna depois.
3. Remover motoristas ja alocados em outra viagem conflitante no mesmo dia, turno e janela de horario.
4. Escolher um motorista disponivel. No MVP pode ser menor `id` ou aleatorio entre disponiveis; depois pode virar rodizio por quantidade de viagens.

Nao recomendo filtrar motorista por destino/faculdade agora. Isso exigiria persistir uma relacao nova entre motorista e destinos atendidos, mas a regra principal da operacao parece ser cidade + turno + disponibilidade. Se no futuro alguns motoristas so puderem atender certas faculdades, ai faria sentido criar algo como `motorista_destinos`.

Depois que o planejador escolhe veiculo e motorista, esses IDs ficam persistidos na viagem. Assim, mesmo que o algoritmo mude depois, aquela viagem continua dizendo exatamente qual unidade foi escalada e qual motorista foi atribuido.

### Ordem das paradas e destinos

O calculo deve respeitar uma regra importante: em rota de ida, as paradas de embarque acontecem antes dos destinos. Em rota de volta, os destinos de embarque acontecem antes das paradas de desembarque perto de casa.

Como o sistema nao persiste a parada especifica de cada aluno, o algoritmo nao deve tentar remover uma parada por achar que "nenhum aluno embarca ali". Sem esse dado individual, a regra mais segura e usar as paradas das rotas internas ligadas as reservas daquele calculo.

Fluxo melhor para ida:

1. Buscar as rotas internas vinculadas as reservas daquele dia, turno, cidade e sentido.
2. Montar a sequencia de paradas de embarque usando a ordem de `rota_interna_paradas`.
3. Deduplicar paradas se mais de uma rota interna compartilhar o mesmo local.
4. Otimizar os destinos envolvidos naquele turno.
5. Unir a sequencia: paradas da rota interna -> destinos.
6. Pedir ao OSRM a geometria final da rota nessa ordem.

Fluxo melhor para volta:

1. Comecar nos destinos, porque e ali que os alunos embarcam para voltar para casa.
2. Otimizar a ordem dos destinos de origem, se houver mais de um.
3. Depois seguir para as paradas das rotas internas.
4. A sequencia de desembarque pode ser a rota interna invertida, se isso fizer sentido para a operacao.

### APIs gratuitas de mapa e rota

Use coordenadas persistidas no banco. Nao geocodifique enderecos toda vez que calcular uma rota.

Recomendacao:

- Nominatim/OpenStreetMap para geocodificar endereco no cadastro.
- OSRM para calcular matriz de tempo/distancia e geometria da rota.
- Banco como fonte da verdade.
- Redis, se entrar no futuro, apenas como cache temporario.

Cuidados importantes:

- Nominatim publico tem limite forte de uso e exige identificacao por `User-Agent` ou `Referer`, atribuicao e cache dos resultados. Ele nao deve ser usado para geocodificacao pesada ou repetitiva.
- OSRM publico tambem e um servidor de demonstracao, com limite de uso e sem garantia de disponibilidade.
- Para producao, a melhor opcao gratuita de verdade e hospedar seu proprio OSRM com dados do OpenStreetMap. Isso evita dependencia de quota publica e melhora estabilidade.
- OSRM usa coordenadas no formato `longitude,latitude`, nao `latitude,longitude`.

Uso sugerido do OSRM:

- `table`: montar matriz de duracao/distancia entre paradas e destinos.
- `trip`: ajudar com ordenacao aproximada quando nao houver restricao de precedencia.
- `route`: gerar a geometria final depois que a ordem das paradas e dos destinos ja foi decidida.

Exemplo conceitual de rota final de ida:

```text
Parada Centro -> Parada Farol -> Parada Tabuleiro -> UFAL -> CESMAC
```

Depois de decidir essa ordem, chamar OSRM `route` com `geometries=geojson`, salvar o GeoJSON em `rotas_dinamicas.geometria_geojson` e salvar cada item da sequencia em `rota_dinamica_paradas`.

## Persistencia, offline e limpeza

Persistir rotas no banco e a decisao correta para este projeto.

Motivos:

- O motorista precisa consultar rota, veiculo, passageiros e paradas.
- O aluno precisa saber qual veiculo/motorista ira pegar.
- O backend precisa auditar o que foi calculado.
- O app pode baixar a viagem enquanto esta online e manter cache local no celular.
- Redis sozinho nao resolve historico nem consulta offline.

Minha recomendacao e salvar:

- rota calculada
- paradas ordenadas
- geometria GeoJSON
- distancia e duracao estimadas
- reservas alocadas
- motorista e veiculo escolhidos automaticamente
- horario previsto

Limpeza:

- `rotas_dinamicas.expires_at = data_viagem + interval '3 months'`
- `viagens.expires_at = data_viagem + interval '3 months'`
- um worker diario remove dados expirados que estejam `concluidos` ou `cancelados`

Exemplo de regra:

```sql
DELETE FROM viagem_reservas
WHERE viagem_id IN (
  SELECT id FROM viagens
  WHERE expires_at < NOW()
    AND status IN ('concluida', 'cancelada')
);

DELETE FROM viagens
WHERE expires_at < NOW()
  AND status IN ('concluida', 'cancelada');

DELETE FROM rotas_dinamicas
WHERE expires_at < NOW()
  AND status IN ('concluida', 'cancelada');
```

Eu manteria `reservas` por mais tempo ou pelo menos avaliaria antes de apagar. Reserva e um registro de solicitacao do aluno; apagar viagem e rota e mais aceitavel do que apagar qualquer prova de que o aluno reservou. Se a decisao for apagar tudo depois de 3 meses, use `ON DELETE CASCADE` com cuidado e deixe isso claro na politica do produto.

## Endpoints futuros sugeridos

Reservas:

```text
POST   /clientes/{clienteID}/vinculos/{vinculoID}/reservas/
GET    /clientes/{clienteID}/vinculos/{vinculoID}/reservas/
GET    /clientes/{clienteID}/reservas/
GET    /reservas/
GET    /reservas/{id}
PUT    /reservas/{id}
POST   /reservas/{id}/cancelar
```

Rotas dinamicas:

```text
POST   /rotas-dinamicas/calcular
GET    /rotas-dinamicas/
GET    /rotas-dinamicas/{id}
POST   /rotas-dinamicas/{id}/publicar
POST   /rotas-dinamicas/{id}/cancelar
```

Viagens:

```text
POST   /viagens/
GET    /viagens/
GET    /viagens/{id}
POST   /viagens/{id}/iniciar
POST   /viagens/{id}/concluir
POST   /viagens/{id}/cancelar
POST   /viagens/{id}/embarques/{reservaID}
POST   /viagens/{id}/faltas/{reservaID}
```

Para o aluno, um endpoint muito util:

```text
GET /minhas-reservas?data=2026-06-05
```

Esse endpoint deve retornar a reserva com dados da viagem quando existir uma relacao em `viagem_reservas`:

```json
{
  "id": 10,
  "data_viagem": "2026-06-05",
  "turno": "NT",
  "status": "confirmada",
  "viagem": {
    "id": 3,
    "status": "programada",
    "partida_prevista": "2026-06-05T18:10:00-03:00",
    "veiculo": {
      "id": 2,
      "modelo": "Van Sprinter",
      "placa": "ABC1D23",
      "capacidade": 15
    },
    "motorista": {
      "id": 4,
      "nome": "Joao Motorista",
      "telefone": "82999990000"
    }
  }
}
```

## Proximos passos mapeados

1. Criar migrations de `rotas_dinamicas`, `rota_dinamica_paradas`, `viagens` e `viagem_reservas`.
2. Criar o modelo de `viagem_reservas` junto com viagens, porque ele e necessario para presenca dos alunos.
3. Implementar repositories, services e handlers de viagens.
4. Criar um `RoutePlannerService` separado para calcular rotas dinamicas e fazer alocacao automatica de veiculo/motorista.
5. Criar cliente HTTP para OSRM com timeout, User-Agent proprio e tratamento de erro.
6. Usar Nominatim somente no cadastro ou atualizacao de endereco, sempre com cache/persistencia da coordenada.
7. Registrar os novos handlers em `cmd/dependencies.go` e `internal/server/server.go`.
8. Implementar endpoint de calculo/publicacao de rotas por `data + turno + cidade + sentido`.
9. Implementar worker de limpeza por `expires_at`.
10. Adicionar testes unitarios para regras de capacidade, turno, disponibilidade de veiculo/motorista e validacao.
11. Adicionar testes de integracao para o fluxo completo: cliente -> vinculo -> reserva -> rota dinamica -> viagem -> viagem_reservas.
12. Aplicar regras de papel e propriedade: admin gerencia cadastros/planejamento, motorista executa viagens atribuidas, cliente consulta as proprias reservas.
13. Depois do MVP, avaliar OSRM self-hosted para nao depender do servidor publico.

## Decisoes recomendadas

Persistir rotas dinamicas no banco: sim.

Permitir rotas duplicadas em turnos/datas diferentes: sim.

Usar Redis como fonte principal: nao.

Usar Redis como cache depois: pode, mas opcional.

Criar uma tabela separada de faculdades antes de rotas dinamicas: nao para o MVP. O conceito ja esta representado por `destinos`.

Calcular rota no momento exato da reserva: possivel, mas pode gerar recalculo demais. Melhor criar reserva e rodar um planejamento por data/turno, com recalculo quando entrar ou sair reserva antes do horario limite.

Aluno saber veiculo/motorista: deve acontecer quando existir `viagem_reservas` ligando a reserva a uma viagem. Antes disso, a API pode retornar `aguardando alocacao`.

Admin escolher motorista/veiculo manualmente: nao para o fluxo principal. O sistema deve alocar automaticamente. No futuro pode existir uma acao administrativa de override, mas ela deve ser excecao operacional, nao o comportamento padrao.

Criar `motorista_destinos` agora: nao. Cidade, turno e disponibilidade resolvem o MVP com menos refatoracao.

## Fontes externas consultadas

- Nominatim Usage Policy: https://operations.osmfoundation.org/policies/nominatim/
- OSRM HTTP API: https://project-osrm.org/docs/v26.6.1/http
- OSRM demo server policy: https://github.com/Project-OSRM/osrm-backend/wiki/Api-usage-policy
- FOSSGIS/OSRM routing server info: https://map.project-osrm.org/about.html
