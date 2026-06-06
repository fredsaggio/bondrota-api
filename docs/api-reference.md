# BondRota API Reference

Documento para consumo do frontend. As informacoes abaixo foram extraidas de `internal/server/server.go`, handlers, models e migrations do backend.

## Visao Geral

**Base URL de producao**

```text
https://bondrota-api.onrender.com/api/v1
```

Nos exemplos abaixo, `BASE_URL` significa:

```text
https://bondrota-api.onrender.com/api/v1
```

### Formato

A API recebe e responde JSON na maioria dos endpoints.

```http
Content-Type: application/json
Accept: application/json
```

Respostas de erro usam `http.Error`, entao normalmente voltam como texto simples, por exemplo:

```text
invalid request body
```

Deletes bem-sucedidos retornam `204 No Content`, sem corpo.

### Autenticacao

Existem tres logins publicos:

- `POST /admin/login`
- `POST /motoristas/login`
- `POST /clientes/login`

Cada login retorna um JWT:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

O token deve ser enviado nos endpoints autenticados:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

O JWT contem pelo menos:

```json
{
  "user_id": 1,
  "role": "admin",
  "exp": 1791234567,
  "iat": 1791148167
}
```

Roles existentes:

- `admin`
- `cliente`
- `motorista`

O token expira em 24 horas.

### Codigos HTTP

| Status | Uso |
| --- | --- |
| `200 OK` | Consulta, update ou acao executada com sucesso. |
| `201 Created` | Recurso criado com sucesso. |
| `204 No Content` | Recurso removido com sucesso. |
| `400 Bad Request` | JSON invalido, ID invalido, campo obrigatorio faltando em validacao basica. |
| `401 Unauthorized` | Token ausente/invalido ou credenciais de login invalidas. |
| `403 Forbidden` | Usuario autenticado, mas sem role permitida. |
| `404 Not Found` | Recurso nao encontrado. |
| `409 Conflict` | Conflito de unicidade, recurso ja existente ou recurso em uso. |
| `422 Unprocessable Entity` | JSON valido, mas regra de negocio invalida. |
| `500 Internal Server Error` | Erro inesperado no backend. |

### Limite de Body

Todo request body passa por limite global de `250 KB`.

## Autenticacao e Health

### `GET BASE_URL/health`

Verifica se a API esta respondendo.

Autenticacao: nao.

Request body: nenhum.

Response `200 OK`: sem JSON garantido.

Erros comuns: `500`.

### `POST BASE_URL/admin/login`

Autentica administrador.

Autenticacao: nao.

Request:

```json
{
  "email": "admin@bondrota.com",
  "senha": "admin123"
}
```

Response `200 OK`:

```json
{
  "token": "jwt"
}
```

Erros: `400` body invalido, `401` email/senha invalidos, `500`.

### `POST BASE_URL/motoristas/login`

Autentica motorista por CPF e senha.

Autenticacao: nao.

Request:

```json
{
  "cpf": "00000000000",
  "senha": "senha123"
}
```

Response `200 OK`:

```json
{
  "token": "jwt"
}
```

Erros: `400` body invalido ou CPF/senha ausentes, `401`, `500`.

### `POST BASE_URL/clientes/login`

Autentica cliente por CPF e senha.

Autenticacao: nao.

Request:

```json
{
  "cpf": "00000000000",
  "senha": "senha123"
}
```

Response `200 OK`:

```json
{
  "token": "jwt"
}
```

Erros: `400` body invalido ou CPF/senha ausentes, `401`, `500`.

## Endpoints

Observacao: todos os endpoints abaixo, exceto logins e health, exigem `Authorization: Bearer <token>`.

### Administradores

Permissao: `admin`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/admin/` | Cria admin. | `AdminCreateRequest` | `201 AdminCreateResponse` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/admin/` | Lista admins. | nenhum | `200 AdminResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/admin/{adminID}` | Busca admin por ID. | nenhum | `200 AdminResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/admin/{adminID}` | Atualiza email do admin. | `AdminUpdateRequest` | `200 AdminResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/admin/{adminID}` | Remove admin. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Request de criacao:

```json
{
  "email": "admin@bondrota.com",
  "senha": "admin123"
}
```

Response de criacao:

```json
{
  "id": 1
}
```

Response de consulta:

```json
{
  "id": 1,
  "email": "admin@bondrota.com"
}
```

Update:

```json
{
  "email": "novo-admin@bondrota.com"
}
```

### Veiculos

Permissao: `admin`.

Categorias e capacidades validas:

| Categoria | Capacidade |
| --- | --- |
| `executivo` | `46` |
| `escolar` | `24` |
| `carro_7_lugares` | `7` |

Status validos: `ativo`, `inativo`, `manutencao`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/veiculos/` | Cria veiculo. | `VeiculoCreateRequest` | `201 { "id": number }` | `400`, `401`, `403`, `422`, `500` |
| `GET` | `BASE_URL/veiculos/` | Lista veiculos. | nenhum | `200 VeiculoResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/veiculos/{veiculoID}` | Busca veiculo por ID. | nenhum | `200 VeiculoResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/veiculos/{veiculoID}` | Atualiza veiculo. | `VeiculoUpdateRequest` | `200 VeiculoResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/veiculos/{veiculoID}` | Remove veiculo. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Create:

```json
{
  "placa": "ABC1D23",
  "modelo": "Volare Escolar",
  "categoria": "escolar",
  "capacidade": 24,
  "cidade_base": "campo alegre",
  "status": "ativo",
  "ar_condicionado": true,
  "banheiro": false,
  "persiana": false,
  "luz_leitura": false,
  "tomada": true
}
```

Response:

```json
{
  "id": 1,
  "placa": "ABC1D23",
  "modelo": "Volare Escolar",
  "categoria": "escolar",
  "capacidade": 24,
  "cidade_base": "campo alegre",
  "status": "ativo",
  "ar_condicionado": true,
  "banheiro": false,
  "persiana": false,
  "luz_leitura": false,
  "tomada": true
}
```

Update aceita campos parciais. Campos booleanos podem ser enviados como `true` ou `false`.

### Destinos

Permissao: `admin`.

Destino representa a faculdade/local de desembarque do cliente. Tambem e o local onde o aluno embarca na volta.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/destinos/` | Cria destino. | `DestinoRequest` | `201 { "id": number }` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/destinos/` | Lista destinos. | nenhum | `200 DestinoResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/destinos/cidade/{cidade}` | Lista destinos por cidade. | nenhum | `200 DestinoResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/destinos/{id}` | Busca destino. | nenhum | `200 DestinoResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/destinos/{id}` | Atualiza destino. | `DestinoRequest` parcial | `200 DestinoResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/destinos/{id}` | Remove destino. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Request:

```json
{
  "nome": "Universidade Federal de Alagoas",
  "rua": "Av. Lourival Melo Mota",
  "cidade": "maceio",
  "latitude": -9.5584,
  "longitude": -35.7777
}
```

Response:

```json
{
  "id": 1,
  "nome": "Universidade Federal de Alagoas",
  "rua": "Av. Lourival Melo Mota",
  "cidade": "maceio",
  "latitude": -9.5584,
  "longitude": -35.7777
}
```

### Paradas

Permissao: `admin`.

Parada representa um ponto de embarque dentro da rota interna da cidade de origem. A API nao persiste em qual parada especifica cada aluno embarca na ida; ela persiste o destino/faculdade do aluno via vinculo/reserva.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/paradas/` | Cria parada. | `ParadaRequest` | `201 ParadaResponse` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/paradas/` | Lista paradas. | nenhum | `200 ParadaResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/paradas/cidade/{cidade}` | Lista paradas por cidade. | nenhum | `200 ParadaResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/paradas/{id}` | Busca parada. | nenhum | `200 ParadaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/paradas/{id}` | Atualiza parada. | `ParadaRequest` parcial | `200 ParadaResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/paradas/{id}` | Remove parada. | nenhum | `204` | `400`, `401`, `403`, `404`, `409` |

Request:

```json
{
  "nome": "Praca Central",
  "latitude": -9.7812,
  "longitude": -36.3501,
  "cidade": "campo alegre"
}
```

Response:

```json
{
  "id": 1,
  "nome": "Praca Central",
  "latitude": -9.7812,
  "longitude": -36.3501,
  "cidade": "campo alegre"
}
```

### Rotas Internas

Permissao: `admin`.

Rota interna e a sequencia de paradas dentro da cidade de origem. Ela e usada para saber por onde o veiculo passa antes de seguir para os destinos.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/rotas-internas/` | Cria rota interna com paradas ordenadas. | `CreateRotaInternaRequest` | `201 RotaInternaResponse` | `400`, `401`, `403`, `422`, `500` |
| `GET` | `BASE_URL/rotas-internas/` | Lista rotas internas. | nenhum | `200 RotaInternaResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/rotas-internas/cidade/{cidade}` | Lista rotas internas por cidade. | nenhum | `200 RotaInternaResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/rotas-internas/{id}` | Busca rota interna. | nenhum | `200 RotaInternaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/rotas-internas/{id}/paradas` | Substitui a sequencia de paradas. | `UpdateParadasRequest` | `200 RotaInternaResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/rotas-internas/{id}` | Remove rota interna. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Create:

```json
{
  "cidade": "campo alegre",
  "paradas": [
    {
      "parada_id": 1,
      "ordem": 1
    },
    {
      "parada_id": 2,
      "ordem": 2
    }
  ]
}
```

Response:

```json
{
  "id": 1,
  "cidade": "campo alegre",
  "paradas": [
    {
      "id": 1,
      "nome": "Praca Central",
      "latitude": -9.7812,
      "longitude": -36.3501,
      "cidade": "campo alegre",
      "ordem": 1
    }
  ]
}
```

### Motoristas

Permissao: `admin`, exceto login publico.

Turnos validos: `MT`, `VT`, `NT`, `IN`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/motoristas/` | Cria motorista. | `CreateMotoristaRequest` | `201 MotoristaResponse` | `400`, `401`, `403`, `409`, `500` |
| `GET` | `BASE_URL/motoristas/` | Lista motoristas. | nenhum | `200 MotoristaResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/motoristas/{id}` | Busca motorista. | nenhum | `200 MotoristaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/motoristas/{id}` | Atualiza motorista. | `UpdateMotoristaRequest` parcial | `200 MotoristaResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/motoristas/{id}` | Remove motorista. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Create:

```json
{
  "nome": "Joao Motorista",
  "cpf": "00000000000",
  "senha": "senha123",
  "telefone": "82999990000",
  "data_nasc": "1980-05-20",
  "turno": "NT",
  "cidade_trabalho": "campo alegre",
  "residencia": "campo alegre",
  "foto": "https://..."
}
```

Response:

```json
{
  "id": 1,
  "nome": "Joao Motorista",
  "cpf": "00000000000",
  "telefone": "82999990000",
  "data_nasc": "1980-05-20",
  "turno": "NT",
  "cidade_trabalho": "campo alegre",
  "residencia": "campo alegre",
  "foto": "https://..."
}
```

### Clientes

Permissao: `admin` ou `cliente`, exceto login publico.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/clientes/` | Cria cliente. | `CreateClienteRequest` | `201 ClienteResponse` | `400`, `401`, `403`, `409`, `500` |
| `GET` | `BASE_URL/clientes/` | Lista clientes. | nenhum | `200 ClienteResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}` | Busca cliente com vinculos. | nenhum | `200 ClienteComVinculosResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/clientes/{clienteID}` | Atualiza cliente. | `UpdateClienteRequest` parcial | `200 ClienteResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/clientes/{clienteID}` | Remove cliente. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/reservas/` | Lista reservas do cliente. | nenhum | `200 ReservaResponse[]` | `400`, `401`, `403`, `500` |

Create:

```json
{
  "nome": "Maria Cliente",
  "cpf": "11111111111",
  "senha": "senha123",
  "telefone": "82999991111",
  "data_nasc": "2002-08-10",
  "foto": "https://..."
}
```

Response:

```json
{
  "id": 1,
  "nome": "Maria Cliente",
  "cpf": "11111111111",
  "telefone": "82999991111",
  "data_nasc": "2002-08-10",
  "foto": "https://..."
}
```

Response de `GET /clientes/{clienteID}`:

```json
{
  "id": 1,
  "nome": "Maria Cliente",
  "cpf": "11111111111",
  "telefone": "82999991111",
  "data_nasc": "2002-08-10",
  "foto": "https://...",
  "vinculos": [
    {
      "id": 10,
      "cliente_id": 1,
      "tipo": "estudante",
      "turno": "NT",
      "destino_id": 1,
      "rota_interna_id": 1,
      "curso": "Sistemas de Informacao",
      "comprovante": "https://...",
      "validade": "2026-12-31",
      "horarios_fixos": [
        {
          "id": 1,
          "vinculo_id": 10,
          "dia_semana": 1
        }
      ]
    }
  ]
}
```

### Vinculos de Cliente

Permissao: `admin` ou `cliente`.

Vinculo liga um cliente a um destino e a uma rota interna. Ele representa a relacao operacional do cliente com faculdade/estagio, turno, comprovante e dias fixos.

Tipos validos: `estudante`, `estagio`.

Turnos validos: `MT`, `VT`, `NT`, `IN`.

Dias da semana em `horarios_fixos`: `1` a `5`, onde a API apenas valida o intervalo; use a convencao do produto para mapear segunda a sexta.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/clientes/{clienteID}/vinculos/` | Cria vinculo para cliente. | `VinculoRequest` | `201 VinculoResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/vinculos/` | Lista vinculos do cliente. | nenhum | `200 VinculoResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}` | Busca vinculo do cliente. | nenhum | `200 VinculoResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}` | Atualiza vinculo. | `VinculoRequest` | `200 VinculoResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}` | Remove vinculo. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |
| `POST` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}/reservas/` | Cria reserva usando o vinculo. | `CreateReservaRequest` | `201 ReservaResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}/reservas/` | Lista reservas desse vinculo. | nenhum | `200 ReservaResponse[]` | `400`, `401`, `403`, `404`, `500` |

Request de vinculo:

```json
{
  "tipo": "estudante",
  "turno": "NT",
  "destino_id": 1,
  "rota_interna_id": 1,
  "curso": "Sistemas de Informacao",
  "comprovante": "https://...",
  "validade": "2026-12-31",
  "horarios_fixos": [1, 2, 3, 4, 5]
}
```

Response:

```json
{
  "id": 10,
  "cliente_id": 1,
  "tipo": "estudante",
  "turno": "NT",
  "destino_id": 1,
  "rota_interna_id": 1,
  "curso": "Sistemas de Informacao",
  "comprovante": "https://...",
  "validade": "2026-12-31",
  "horarios_fixos": [
    {
      "id": 1,
      "vinculo_id": 10,
      "dia_semana": 1
    }
  ]
}
```

### Reservas

Permissao: `admin` ou `cliente`.

Reserva e criada a partir de um vinculo. Ela guarda snapshot de `cliente_id`, `vinculo_id`, `destino_id`, `rota_interna_id`, `cidade`, `data_viagem`, `turno` e `sentido`. Isso permite manter a reserva historica mesmo se o vinculo mudar depois.

Status validos: `confirmada`, `cancelada`.

Sentidos validos: `ida`, `volta`.

Turnos operacionais validos: `MT`, `VT`, `NT`. Se o vinculo for `IN`, o frontend deve enviar o turno desejado.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/reservas/` | Lista reservas. | nenhum | `200 ReservaResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/reservas/{reservaID}` | Busca reserva. | nenhum | `200 ReservaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/reservas/{reservaID}` | Atualiza dados editaveis da reserva. | `UpdateReservaRequest` parcial | `200 ReservaResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/reservas/{reservaID}/cancelar` | Cancela reserva. | nenhum | `200 ReservaResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/reservas/{reservaID}` | Remove reserva. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Create via vinculo:

```json
{
  "data_viagem": "2026-06-10",
  "turno": "NT",
  "sentido": "ida"
}
```

Response:

```json
{
  "id": 1,
  "cliente_id": 1,
  "vinculo_id": 10,
  "data_viagem": "2026-06-10",
  "turno": "NT",
  "destino_id": 1,
  "rota_interna_id": 1,
  "cidade": "campo alegre",
  "sentido": "ida",
  "status": "confirmada",
  "created_at": "2026-06-06T20:00:00Z",
  "updated_at": "2026-06-06T20:00:00Z"
}
```

Update:

```json
{
  "data_viagem": "2026-06-11",
  "turno": "NT",
  "sentido": "volta",
  "status": "confirmada"
}
```

Regra importante: nao pode existir mais de uma reserva ativa para o mesmo `vinculo_id`, `data_viagem`, `turno` e `sentido`. Se ja existir, a API retorna `409`.

### Horarios por Turno de Viagem

Permissao: `admin`.

Define os horarios padrao que o planejamento usa para criar a partida prevista de ida e volta.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/horarios-turno-viagem/` | Cria horario por cidade/turno. | `HorarioTurnoViagemRequest` | `201 HorarioTurnoViagemResponse` | `400`, `401`, `403`, `409`, `422`, `500` |
| `GET` | `BASE_URL/horarios-turno-viagem/` | Lista horarios. | nenhum | `200 HorarioTurnoViagemResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/horarios-turno-viagem/{horarioTurnoID}` | Busca horario. | nenhum | `200 HorarioTurnoViagemResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/horarios-turno-viagem/{horarioTurnoID}` | Atualiza horario. | `HorarioTurnoViagemRequest` parcial | `200 HorarioTurnoViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `DELETE` | `BASE_URL/horarios-turno-viagem/{horarioTurnoID}` | Remove horario. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Request:

```json
{
  "cidade": "campo alegre",
  "turno": "NT",
  "horario_ida": "17:00",
  "horario_volta": "22:00"
}
```

Response:

```json
{
  "id": 1,
  "cidade": "campo alegre",
  "turno": "NT",
  "horario_ida": "17:00:00",
  "horario_volta": "22:00:00",
  "created_at": "2026-06-06T20:00:00Z",
  "updated_at": "2026-06-06T20:00:00Z"
}
```

`horario_volta` precisa ser maior que `horario_ida`.

### Planejamento de Viagens

Permissao: `admin`.

Este endpoint e o integrador operacional. Ele nao e CRUD de viagem manual: ele planeja ciclos/viagens automaticamente a partir das reservas confirmadas, horario do turno, veiculos disponiveis e motoristas disponiveis.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/planejamentos/viagens` | Planeja ciclos e viagens para data/turno/cidade/rota interna. | `PlanejarViagensRequest` | `201 PlanejamentoViagensResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |

Request:

```json
{
  "data_viagem": "2026-06-10",
  "turno": "NT",
  "cidade": "campo alegre",
  "rota_interna_id": 1,
  "expires_at": "2026-09-10T00:00:00Z"
}
```

Response:

```json
{
  "ciclos": [
    {
      "ciclo": {
        "id": 1,
        "data_viagem": "2026-06-10",
        "turno": "NT",
        "cidade": "campo alegre",
        "rota_interna_id": 1,
        "veiculo_id": 1,
        "motorista_id": 1,
        "status": "planejado",
        "expires_at": "2026-09-10T00:00:00Z",
        "created_at": "2026-06-06T20:00:00Z",
        "updated_at": "2026-06-06T20:00:00Z"
      },
      "viagens": [
        {
          "id": 1,
          "ciclo_viagem_id": 1,
          "sentido": "ida",
          "status": "programada",
          "created_at": "2026-06-06T20:00:00Z",
          "updated_at": "2026-06-06T20:00:00Z"
        },
        {
          "id": 2,
          "ciclo_viagem_id": 1,
          "sentido": "volta",
          "status": "programada",
          "created_at": "2026-06-06T20:00:00Z",
          "updated_at": "2026-06-06T20:00:00Z"
        }
      ]
    }
  ],
  "quantidade_reservas_ida": 25,
  "quantidade_reservas_volta": 20,
  "capacidade_total": 46
}
```

Regras operacionais importantes:

- Usa reservas `confirmada` do mesmo `data_viagem`, `turno`, `cidade`, `rota_interna_id` e `sentido`.
- Usa `horarios_turno_viagem` para definir partida prevista da ida e da volta.
- Aloca veiculos automaticamente por cidade, status e capacidade.
- Aloca motoristas automaticamente por cidade/turno/disponibilidade.
- Cria `ciclos_viagem`, `viagens`, `viagem_horarios` e `viagem_reservas`.
- Um ciclo tem normalmente duas viagens: `ida` e `volta`.

### Viagens

Permissao: `admin` ou `motorista`.

Status de viagem: `programada`, `em_andamento`, `concluida`, `cancelada`.

Status de ciclo: `planejado`, `em_andamento`, `concluido`, `cancelado`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/viagens/` | Lista viagens com ciclo. | nenhum | `200 ViagemComCicloResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}` | Busca viagem com ciclo. | nenhum | `200 ViagemComCicloResponse` | `400`, `401`, `403`, `404`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/iniciar` | Inicia viagem e registra `inicio_real`. | nenhum | `200 ViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/concluir` | Conclui viagem e registra `fim_real`. | nenhum | `200 ViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/cancelar` | Cancela viagem. | nenhum | `200 ViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}/horarios` | Lista horarios da viagem. | nenhum | `200 ViagemHorarioResponse[]` | `400`, `401`, `403`, `404`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}/reservas/` | Lista reservas alocadas na viagem. | nenhum | `200 ViagemReservaComReservaResponse[]` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/viagens/{viagemID}/reservas/{reservaID}/presenca` | Atualiza presenca do aluno na viagem. | `AtualizarPresencaRequest` | `200 ViagemReservaResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |

Response de viagem com ciclo:

```json
{
  "viagem": {
    "id": 1,
    "ciclo_viagem_id": 1,
    "sentido": "ida",
    "status": "programada",
    "created_at": "2026-06-06T20:00:00Z",
    "updated_at": "2026-06-06T20:00:00Z"
  },
  "ciclo": {
    "id": 1,
    "data_viagem": "2026-06-10",
    "turno": "NT",
    "cidade": "campo alegre",
    "rota_interna_id": 1,
    "veiculo_id": 1,
    "motorista_id": 1,
    "status": "planejado",
    "expires_at": "2026-09-10T00:00:00Z",
    "created_at": "2026-06-06T20:00:00Z",
    "updated_at": "2026-06-06T20:00:00Z"
  }
}
```

Response de status:

```json
{
  "id": 1,
  "ciclo_viagem_id": 1,
  "sentido": "ida",
  "status": "em_andamento",
  "created_at": "2026-06-06T20:00:00Z",
  "updated_at": "2026-06-06T20:05:00Z"
}
```

Horarios:

```json
[
  {
    "id": 1,
    "viagem_id": 1,
    "tipo": "partida_prevista",
    "horario": "2026-06-10T17:00:00Z",
    "created_at": "2026-06-06T20:00:00Z",
    "updated_at": "2026-06-06T20:00:00Z"
  }
]
```

Presenca:

```json
{
  "status_presenca": "embarcou"
}
```

Status de presenca aceitos no update: `embarcou`, `faltou`, `cancelado`. O status inicial criado pelo planejamento e `aguardando`.

Response:

```json
{
  "id": 1,
  "viagem_id": 1,
  "reserva_id": 1,
  "status_presenca": "embarcou",
  "created_at": "2026-06-06T20:00:00Z",
  "updated_at": "2026-06-06T20:10:00Z"
}
```

Lista de reservas da viagem:

```json
[
  {
    "id": 1,
    "viagem_id": 1,
    "reserva_id": 1,
    "status_presenca": "aguardando",
    "created_at": "2026-06-06T20:00:00Z",
    "updated_at": "2026-06-06T20:00:00Z",
    "cliente_id": 1,
    "vinculo_id": 10,
    "data_viagem": "2026-06-10",
    "turno": "NT",
    "destino_id": 1,
    "rota_interna_id": 1,
    "cidade": "campo alegre",
    "sentido": "ida"
  }
]
```

### Localizacao da Viagem

Permissoes:

- `PUT`: `admin` ou `motorista`.
- `GET`: `admin`, `motorista` ou `cliente`.

Regras de acesso:

- Motorista so atualiza/localiza viagem atribuida a ele.
- Cliente so consulta localizacao se tiver reserva vinculada a viagem.
- Para motorista autenticado, o backend usa o `user_id` do JWT como `motorista_id`.
- Para admin, envie `motorista_id` no body do `PUT`.
- Atualizacao de localizacao exige viagem em andamento.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `PUT` | `BASE_URL/viagens/{viagemID}/localizacao` | Atualiza a ultima localizacao da viagem. | `AtualizarLocalizacaoRequest` | `200 ViagemLocalizacaoResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}/localizacao` | Consulta a ultima localizacao. | nenhum | `200 ViagemLocalizacaoResponse` | `400`, `401`, `403`, `404`, `500` |

Request:

```json
{
  "motorista_id": 1,
  "latitude": -9.7812,
  "longitude": -36.3501,
  "velocidade_kmh": 52.4,
  "direcao_graus": 180,
  "precisao_metros": 8
}
```

Response:

```json
{
  "viagem_id": 1,
  "motorista_id": 1,
  "latitude": -9.7812,
  "longitude": -36.3501,
  "velocidade_kmh": 52.4,
  "direcao_graus": 180,
  "precisao_metros": 8,
  "registrada_em": "2026-06-10T17:15:00Z",
  "created_at": "2026-06-10T17:15:00Z",
  "updated_at": "2026-06-10T17:15:00Z"
}
```

Para o frontend mobile, um polling de `GET /viagens/{viagemID}/localizacao` a cada 10 ou 15 segundos funciona com o modelo atual.

### Rotas Dinamicas

Permissao: `admin` ou `motorista`.

A rota dinamica e a rota calculada para uma viagem especifica. Existe no maximo uma rota dinamica por `viagem_id`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/viagens/{viagemID}/rota-dinamica/calcular` | Calcula e persiste rota automaticamente. | nenhum | `201 RotaDinamicaComDestinosResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/rota-dinamica` | Cria rota dinamica manualmente. | `RotaDinamicaRequest` | `201 RotaDinamicaComDestinosResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}/rota-dinamica` | Busca rota dinamica da viagem. | nenhum | `200 RotaDinamicaComDestinosResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/viagens/{viagemID}/rota-dinamica` | Remove rota dinamica da viagem. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Create manual:

```json
{
  "provider": "osrm",
  "origem": {
    "nome": "Ultima parada da rota interna",
    "latitude": -9.7812,
    "longitude": -36.3501
  },
  "destino_final": {
    "nome": "Universidade Federal de Alagoas",
    "latitude": -9.5584,
    "longitude": -35.7777
  },
  "distancia_metros": 100000,
  "duracao_segundos": 5400,
  "geometry": {
    "type": "LineString",
    "coordinates": [
      [-36.3501, -9.7812],
      [-35.7777, -9.5584]
    ]
  },
  "expires_at": "2026-09-10T00:00:00Z",
  "destinos": [
    {
      "destino_id": 1
    }
  ]
}
```

Response:

```json
{
  "rota": {
    "id": 1,
    "viagem_id": 1,
    "provider": "osrm",
    "origem": {
      "nome": "Ultima parada da rota interna",
      "latitude": -9.7812,
      "longitude": -36.3501
    },
    "destino_final": {
      "nome": "Universidade Federal de Alagoas",
      "latitude": -9.5584,
      "longitude": -35.7777
    },
    "distancia_metros": 100000,
    "duracao_segundos": 5400,
    "geometry": {
      "type": "LineString",
      "coordinates": [
        [-36.3501, -9.7812],
        [-35.7777, -9.5584]
      ]
    },
    "expires_at": "2026-09-10T00:00:00Z",
    "created_at": "2026-06-06T20:00:00Z",
    "updated_at": "2026-06-06T20:00:00Z"
  },
  "destinos": [
    {
      "id": 1,
      "rota_dinamica_id": 1,
      "destino_id": 1,
      "ordem": 1,
      "created_at": "2026-06-06T20:00:00Z"
    }
  ]
}
```

Calculo automatico:

- Usa OSRM como roteador externo.
- O frontend nao precisa chamar OSRM diretamente para gerar a rota dinamica.
- Para ida, a origem e a ultima parada da rota interna e os destinos sao ordenados.
- Para volta, a rota sai dos destinos e termina na primeira parada da rota interna.
- Para ate 8 destinos, o backend usa busca por forca bruta para achar a melhor ordem estimada.
- Para mais de 8 destinos, usa heuristica de vizinho mais proximo com melhoria 2-opt.
- Depois da ordem definida, o backend pede a geometria/distancia/duracao ao OSRM.
- O worker tenta calcular rotas dentro da janela padrao de 1 hora antes da partida e respeita uma janela de bloqueio padrao de 30 minutos.

## Modelagem de Dados e Relacionamentos

### Entidades principais

**administrador**

- `id`
- `email`
- `senha`
- `created_at`
- `updated_at`

Usado para login e operacao administrativa.

**clientes**

- `id`
- `nome`
- `cpf`
- `senha`
- `telefone`
- `data_nasc`
- `foto`
- `created_at`
- `updated_at`

Cliente e a pessoa que faz reservas.

**cliente_vinculos**

- `id`
- `cliente_id`
- `tipo`
- `turno`
- `destino_id`
- `rota_interna_id`
- `curso`
- `comprovante`
- `validade`
- `created_at`
- `updated_at`

Relacionamentos:

- `cliente_vinculos.cliente_id -> clientes.id`
- `cliente_vinculos.destino_id -> destinos.id`
- `cliente_vinculos.rota_interna_id -> rotas_internas.id`

Para telas: um cliente pode ter varios vinculos. O usuario escolhe um vinculo para criar reserva.

**horarios_fixos**

- `id`
- `vinculo_id`
- `dia_semana`

Relacionamento:

- `horarios_fixos.vinculo_id -> cliente_vinculos.id`

Para telas: exibir dias recorrentes do vinculo.

**destinos**

- `id`
- `nome`
- `rua`
- `cidade`
- `latitude`
- `longitude`
- `created_at`
- `updated_at`

Destino e faculdade/local de desembarque. Na volta, tambem e o local onde o aluno embarca para retornar.

**paradas**

- `id`
- `nome`
- `latitude`
- `longitude`
- `cidade`
- `created_at`
- `updated_at`

Parada e local da rota interna onde o veiculo passa para pegar alunos na cidade.

**rotas_internas**

- `id`
- `cidade`
- `created_at`
- `updated_at`

**rota_interna_paradas**

- `id`
- `rota_interna_id`
- `parada_id`
- `ordem`

Relacionamentos:

- `rota_interna_paradas.rota_interna_id -> rotas_internas.id`
- `rota_interna_paradas.parada_id -> paradas.id`

Para telas: rota interna e uma lista ordenada de paradas.

**veiculos**

- `id`
- `placa`
- `modelo`
- `categoria`
- `capacidade`
- `cidade_base`
- `status`
- opcionais booleanos de conforto
- `created_at`
- `updated_at`

Para telas: veiculo fica disponivel para planejamento se estiver ativo e na cidade correta. A categoria determina capacidade fixa.

**motoristas**

- `id`
- `nome`
- `cpf`
- `senha`
- `telefone`
- `data_nasc`
- `turno`
- `cidade_trabalho`
- `residencia`
- `foto`
- `created_at`
- `updated_at`

Para telas: motorista e atribuido automaticamente no planejamento.

**reservas**

- `id`
- `cliente_id`
- `vinculo_id`
- `data_viagem`
- `turno`
- `destino_id`
- `rota_interna_id`
- `cidade`
- `sentido`
- `status`
- `created_at`
- `updated_at`

Relacionamentos:

- `reservas.cliente_id -> clientes.id`
- `reservas.vinculo_id -> cliente_vinculos.id`
- `reservas.destino_id -> destinos.id`
- `reservas.rota_interna_id -> rotas_internas.id`

Para telas: reservas sao o ponto central do app do cliente. A reserva sabe o dia, turno, sentido, destino e rota interna. `destino_id`, `rota_interna_id` e `cidade` sao snapshot do vinculo.

**horarios_turno_viagem**

- `id`
- `cidade`
- `turno`
- `horario_ida`
- `horario_volta`
- `created_at`
- `updated_at`

Para telas admin: configurar horario padrao antes de planejar viagens.

**ciclos_viagem**

- `id`
- `data_viagem`
- `turno`
- `cidade`
- `rota_interna_id`
- `veiculo_id`
- `motorista_id`
- `status`
- `expires_at`
- `created_at`
- `updated_at`

Relacionamentos:

- `ciclos_viagem.rota_interna_id -> rotas_internas.id`
- `ciclos_viagem.veiculo_id -> veiculos.id`
- `ciclos_viagem.motorista_id -> motoristas.id`

Para telas: ciclo representa o bloco operacional com o mesmo veiculo e motorista. Normalmente agrupa ida e volta.

**viagens**

- `id`
- `ciclo_viagem_id`
- `sentido`
- `status`
- `created_at`
- `updated_at`

Relacionamento:

- `viagens.ciclo_viagem_id -> ciclos_viagem.id`

Para telas: viagem e o trecho operacional de ida ou volta.

**viagem_horarios**

- `id`
- `viagem_id`
- `tipo`
- `horario`
- `created_at`
- `updated_at`

Tipos:

- `partida_prevista`
- `inicio_real`
- `fim_real`

Para telas: mostrar previsao e historico real de execucao.

**viagem_reservas**

- `id`
- `viagem_id`
- `reserva_id`
- `status_presenca`
- `created_at`
- `updated_at`

Relacionamentos:

- `viagem_reservas.viagem_id -> viagens.id`
- `viagem_reservas.reserva_id -> reservas.id`

Para telas do motorista: lista de alunos/reservas a marcar presenca.

**viagem_reserva_confirmacoes**

- `viagem_reserva_id`
- `registro_presenca`
- `created_at`
- `updated_at`

Relacionamento:

- `viagem_reserva_confirmacoes.viagem_reserva_id -> viagem_reservas.id`

Para telas: a confirmacao registra quando a presenca saiu de `aguardando`.

**rotas_dinamicas**

- `id`
- `viagem_id`
- `provider`
- `origem_nome`
- `origem_latitude`
- `origem_longitude`
- `destino_final_nome`
- `destino_final_latitude`
- `destino_final_longitude`
- `distancia_metros`
- `duracao_segundos`
- `geometry`
- `expires_at`
- `created_at`
- `updated_at`

Relacionamento:

- `rotas_dinamicas.viagem_id -> viagens.id`

Para telas: usar `geometry` para desenhar a rota no mapa e `destinos` para ordem de desembarque/embarque.

**rota_dinamica_destinos**

- `id`
- `rota_dinamica_id`
- `destino_id`
- `ordem`
- `created_at`

Relacionamentos:

- `rota_dinamica_destinos.rota_dinamica_id -> rotas_dinamicas.id`
- `rota_dinamica_destinos.destino_id -> destinos.id`

**viagem_localizacoes**

- `viagem_id`
- `motorista_id`
- `latitude`
- `longitude`
- `velocidade_kmh`
- `direcao_graus`
- `precisao_metros`
- `registrada_em`
- `created_at`
- `updated_at`

Relacionamentos:

- `viagem_localizacoes.viagem_id -> viagens.id`
- `viagem_localizacoes.motorista_id -> motoristas.id`

Para telas: mostra a ultima posicao enviada pelo app do motorista.

### Diagrama Logico Simplificado

```text
clientes
  1 ── N cliente_vinculos
          N ── 1 destinos
          N ── 1 rotas_internas
          1 ── N horarios_fixos
          1 ── N reservas

rotas_internas
  1 ── N rota_interna_paradas
          N ── 1 paradas

reservas
  N ── 1 clientes
  N ── 1 cliente_vinculos
  N ── 1 destinos
  N ── 1 rotas_internas
  1 ── N viagem_reservas

ciclos_viagem
  N ── 1 rotas_internas
  N ── 1 veiculos
  N ── 1 motoristas
  1 ── N viagens

viagens
  1 ── N viagem_horarios
  1 ── N viagem_reservas
  1 ── 0..1 rotas_dinamicas
  1 ── 0..1 viagem_localizacoes

rotas_dinamicas
  1 ── N rota_dinamica_destinos
          N ── 1 destinos
```

## Flow da Aplicacao

### 1. Bootstrap/Admin inicial

Antes de usar o painel, precisa existir um administrador. O projeto possui comando de seed admin. Com as variaveis `ADMIN_EMAIL`, `ADMIN_PASSWORD` e `DATABASE_URL` configuradas, rode:

```bash
go run ./cmd/seed-admin
```

Depois, o frontend faz:

```http
POST /admin/login
```

e guarda o JWT para as operacoes administrativas.

### 2. Configuracao base pelo admin

Ordem recomendada:

1. Criar destinos.
2. Criar paradas.
3. Criar rotas internas com paradas ordenadas.
4. Criar veiculos.
5. Criar motoristas.
6. Criar horarios por cidade/turno em `horarios-turno-viagem`.
7. Criar clientes.
8. Criar vinculos dos clientes.

Dependencias:

- Vinculo precisa de `cliente_id`, `destino_id` e `rota_interna_id`.
- Rota interna precisa de paradas existentes.
- Planejamento precisa de horario configurado para `cidade + turno`.
- Planejamento precisa de reservas confirmadas.
- Planejamento precisa de veiculos ativos/disponiveis e motoristas disponiveis.

### 3. Cliente cria reserva

Fluxo no app do cliente:

1. Cliente faz `POST /clientes/login`.
2. Frontend lista ou busca o cliente com vinculos:
   - `GET /clientes/{clienteID}`
   - ou `GET /clientes/{clienteID}/vinculos/`
3. Usuario escolhe o vinculo.
4. Frontend cria reserva:
   - `POST /clientes/{clienteID}/vinculos/{vinculoID}/reservas/`
5. Se quiser ida e volta, crie duas reservas:
   - uma com `sentido: "ida"`
   - outra com `sentido: "volta"`

### 4. Admin planeja viagens

Quando houver reservas para a data/turno/cidade/rota interna:

```http
POST /planejamentos/viagens
```

O backend:

1. Busca reservas confirmadas de ida e volta.
2. Busca horarios de ida/volta em `horarios_turno_viagem`.
3. Calcula veiculos por capacidade.
4. Aloca veiculos disponiveis.
5. Aloca motoristas disponiveis.
6. Cria ciclos de viagem.
7. Cria viagens de ida e volta.
8. Cria horarios previstos.
9. Cria `viagem_reservas` para as reservas alocadas.

Para o frontend, o retorno ja informa quais `viagem_id`, `veiculo_id` e `motorista_id` foram definidos.

### 5. Rota dinamica

Depois que a viagem existe, a rota dinamica pode ser gerada:

```http
POST /viagens/{viagemID}/rota-dinamica/calcular
```

Tambem existe worker automatico que processa viagens dentro da janela de calculo. Mesmo assim, para apresentacao ou painel admin, o endpoint manual e util para forcar o calculo.

Para renderizar no frontend:

1. Buscar `GET /viagens/{viagemID}/rota-dinamica`.
2. Usar `rota.geometry` no mapa.
3. Usar `destinos[].ordem` para exibir a ordem dos destinos.

### 6. Motorista executa viagem

Fluxo do app do motorista:

1. Motorista faz `POST /motoristas/login`.
2. Lista viagens: `GET /viagens/`.
3. Abre uma viagem: `GET /viagens/{viagemID}`.
4. Consulta reservas/alunos: `GET /viagens/{viagemID}/reservas/`.
5. Inicia viagem: `POST /viagens/{viagemID}/iniciar`.
6. App envia localizacao periodicamente:
   - `PUT /viagens/{viagemID}/localizacao`
7. Motorista marca presencas:
   - `PUT /viagens/{viagemID}/reservas/{reservaID}/presenca`
8. Conclui viagem:
   - `POST /viagens/{viagemID}/concluir`

### 7. Cliente acompanha localizacao

Fluxo do app do cliente:

1. Cliente autenticado abre a viagem associada a sua reserva.
2. Frontend chama:
   - `GET /viagens/{viagemID}/localizacao`
3. Atualiza mapa a cada 10 ou 15 segundos.

O backend valida se o cliente possui reserva vinculada aquela viagem.

## Observacoes para o Frontend

- Guarde o token por perfil. O mesmo app pode ter telas por role lendo `role` do JWT.
- Paths com barra final existem em algumas rotas aninhadas, por exemplo `/reservas/`. Mantenha a barra para evitar diferenca de roteamento.
- Use `data_nasc`, `data_viagem` e `validade` em `YYYY-MM-DD`.
- Use timestamps em RFC3339, por exemplo `2026-09-10T00:00:00Z`.
- Coordenadas de mapas usam latitude/longitude numericas.
- Distancia da rota dinamica vem em metros.
- Duracao da rota dinamica vem em segundos.
- O frontend nao precisa chamar OSRM para calcular rota. A API calcula e persiste.
- Para mapa visual, o frontend pode usar biblioteca gratuita baseada em OpenStreetMap, como Leaflet, consumindo a `geometry` retornada pela API.
