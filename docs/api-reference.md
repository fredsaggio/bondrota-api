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

`GET BASE_URL/config` e publico e retorna a cidade base e o fuso horario da instancia:

```json
{ "cidade_base": "Campo Alegre", "fuso_horario": "America/Maceio" }
```

`fuso_horario` e o mesmo nome IANA configurado em `APP_TIMEZONE` no backend. O
frontend deve usa-lo para calcular datas relativas a "agora" (como "hoje" no
dashboard), em vez do fuso do navegador do admin.

### Autenticacao

Existem tres logins publicos:

- `POST /admin/login`
- `POST /motoristas/login`
- `POST /clientes/login`

Cada login retorna um JWT para compatibilidade com clientes que usam Bearer:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

O token deve ser enviado nos endpoints autenticados:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

O painel web administrativo não persiste esse JWT. O login também define um
cookie HttpOnly, enviado automaticamente nas chamadas com credenciais. O painel
envia X-Admin-Session-Mode: cookie e recebe 204 sem o JWT no corpo. Ele
usa GET /admin/session para restaurar a sessão e POST /admin/logout para
encerrá-la. Requisições mutáveis autenticadas por cookie exigem uma origem
listada em ALLOWED_ORIGINS.

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

### Limite de tentativas de login

Os logins de administrador, motorista e cliente são limitados por IP e por
identidade normalizada. Quando o limite é excedido, a API responde
`429 Too Many Requests` e inclui o header `Retry-After`.

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

A resposta também inclui Set-Cookie para a sessão HttpOnly administrativa.

### GET BASE_URL/admin/session

Retorna user_id, role e expires_at da sessão administrativa autenticada.

### POST BASE_URL/admin/logout

Expira o cookie administrativo. Response 204 No Content.

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

Observacao: todos os endpoints abaixo, exceto logins, logout, health e config, exigem autenticação. O painel administrativo usa o cookie HttpOnly; os demais clientes continuam usando Authorization Bearer.

### Administradores

Permissao: `admin`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/admin/` | Lista admins. | nenhum | `200 AdminResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/admin/{adminID}` | Busca admin por ID. | nenhum | `200 AdminResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/admin/senha` | Troca a senha do proprio admin autenticado. | `AdminChangePasswordRequest` | `204` | `400`, `401`, `403`, `429`, `500` |

Response de consulta:

```json
{
  "id": 1,
  "email": "admin@bondrota.com"
}
```

Troca de senha:

```json
{
  "senha_atual": "senha-de-agora",
  "nova_senha": "senha-nova-com-8+"
}
```

O admin alvo sai do JWT, nunca do corpo nem do path — nao ha como mirar outra conta.
A senha atual e exigida: sem ela, uma sessao roubada trocaria a senha e trancaria o
admin legitimo do lado de fora. A nova senha precisa de pelo menos 8 caracteres
(`admin.MinPasswordLen`, a mesma regra do `cmd/admin`).

Respostas de erro que importam:

| Status | Quando | Por que nao e outro |
| --- | --- | --- |
| `403` | Senha atual incorreta. | `401` faria o painel encerrar a sessao, deslogando quem so errou a digitacao. |
| `400` | Nova senha com menos de 8 caracteres. | — |
| `429` | Tentativas demais. | Limite por admin autenticado, alem do limite por IP. |

Em caso de sucesso o cookie de sessao e reemitido, entao a aba que trocou a senha
continua logada. **As demais sessoes daquele admin seguem validas ate expirarem
sozinhas** — nao ha revogacao de JWT.

**Criar, alterar e remover administrador nao tem endpoint.** Enquanto existir uma
unica role de admin, expor essas operacoes por HTTP significa que qualquer sessao
roubada consegue fabricar acessos proprios e apagar os legitimos. Elas ficam no
comando `cmd/admin`, que exige acesso direto ao banco:

```bash
go run ./cmd/admin create -email=novo@prefeitura.gov.br
go run ./cmd/admin passwd -email=admin@prefeitura.gov.br
go run ./cmd/admin delete -email=antigo@prefeitura.gov.br
```

Reintroduzir essas rotas exige antes uma role acima de `admin`. Ha um teste
(`TestAdminRoutesAreReadOnly`) que falha se elas voltarem ao roteador.

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
| `DELETE` | `BASE_URL/veiculos/{veiculoID}` | Remove veiculo. | nenhum | `204` | `400`, `401`, `403`, `404`, `409`, `500` |

Create:

```json
{
  "placa": "ABC1D23",
  "modelo": "Volare Escolar",
  "categoria": "escolar",
  "capacidade": 24,
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
  "status": "ativo",
  "ar_condicionado": true,
  "banheiro": false,
  "persiana": false,
  "luz_leitura": false,
  "tomada": true
}
```

Update aceita campos parciais. Campos booleanos podem ser enviados como `true` ou `false`.

### Municipios

Permissao: `admin`.

O catalogo local e importado da API oficial de localidades do IBGE. O frontend consulta os municipios por UF para preencher dropdowns; os demais endpoints recebem o codigo IBGE selecionado.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/municipios/?uf=AL` | Lista municipios ativos da UF. | nenhum | `200 MunicipioResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/municipios/{codigoIBGE}` | Busca um municipio pelo codigo IBGE, de qualquer UF. | nenhum | `200 MunicipioResponse` | `400`, `401`, `403`, `404`, `500` |

```json
[
  {
    "codigo_ibge": 2704302,
    "nome": "Maceio",
    "uf": "AL"
  }
]
```

`GET /municipios/{codigoIBGE}` existe para resolver o nome de um municipio ja
referenciado por outro registro (`destinos.municipio_id`,
`motoristas.municipio_trabalho_id`, `horarios_turno_viagem.municipio_destino_id`)
sem depender de saber a UF de antemao — diferente do `ListByUF`, nao filtra por
`ativo`, pois o registro que aponta para o municipio continua existindo mesmo que
uma reimportacao do IBGE o desative.

Importacao idempotente:

```bash
make municipios/import
make municipios/import uf=AL
make municipios/import/prod
```

### Destinos

Permissao: `admin`.

Destino representa a faculdade/local de desembarque do cliente. Tambem e o local onde o aluno embarca na volta.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/destinos/` | Cria destino. | `DestinoRequest` | `201 { "id": number }` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/destinos/` | Lista destinos. | nenhum | `200 DestinoResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/destinos/municipio/{municipioID}` | Lista destinos por municipio. | nenhum | `200 DestinoResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/destinos/{id}` | Busca destino. | nenhum | `200 DestinoResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/destinos/{id}` | Atualiza destino. | `DestinoRequest` parcial | `200 DestinoResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/destinos/{id}` | Remove destino. | nenhum | `204` | `400`, `401`, `403`, `404`, `409`, `500` |

Request:

```json
{
  "nome": "Universidade Federal de Alagoas",
  "rua": "Av. Lourival Melo Mota",
  "municipio_id": 2704302,
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
  "municipio_id": 2704302,
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
| `GET` | `BASE_URL/paradas/{id}` | Busca parada. | nenhum | `200 ParadaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/paradas/{id}` | Atualiza parada. | `ParadaRequest` parcial | `200 ParadaResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/paradas/{id}` | Remove parada. | nenhum | `204` | `400`, `401`, `403`, `404`, `409` |

Request:

```json
{
  "nome": "Praca Central",
  "latitude": -9.7812,
  "longitude": -36.3501
}
```

Response:

```json
{
  "id": 1,
  "nome": "Praca Central",
  "latitude": -9.7812,
  "longitude": -36.3501
}
```

### Rotas Internas

Permissao: `admin`.

Rota interna e a sequencia de paradas dentro da cidade de origem. Ela e usada para saber por onde o veiculo passa antes de seguir para os destinos.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/rotas-internas/` | Cria rota interna com paradas ordenadas. | `CreateRotaInternaRequest` | `201 RotaInternaResponse` | `400`, `401`, `403`, `422`, `500` |
| `GET` | `BASE_URL/rotas-internas/` | Lista rotas internas. | nenhum | `200 RotaInternaResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/rotas-internas/{id}` | Busca rota interna. | nenhum | `200 RotaInternaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/rotas-internas/{id}/paradas` | Substitui a sequencia de paradas. | `UpdateParadasRequest` | `200 RotaInternaResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/rotas-internas/{id}` | Remove rota interna. | nenhum | `204` | `400`, `401`, `403`, `404`, `409`, `500` |

Create:

```json
{
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
  "paradas": [
    {
      "id": 1,
      "nome": "Praca Central",
      "latitude": -9.7812,
      "longitude": -36.3501,
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
| `DELETE` | `BASE_URL/motoristas/{id}` | Remove motorista. | nenhum | `204` | `400`, `401`, `403`, `404`, `409`, `500` |

`telefone`, `residencia` e `foto` sao opcionais no update: omita a chave (ou envie
`null`) para manter o valor atual, e envie `""` explicitamente para limpar o campo.
`nome`, `data_nasc`, `turno` e `municipio_trabalho_id` continuam sendo ignorados
quando enviados em branco, porque nao existe valor valido em branco para eles.

Create:

```json
{
  "nome": "Joao Motorista",
  "cpf": "00000000000",
  "senha": "senha123",
  "telefone": "82999990000",
  "data_nasc": "1980-05-20",
  "turno": "NT",
  "municipio_trabalho_id": 2704302,
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
  "municipio_trabalho_id": 2704302,
  "residencia": "campo alegre",
  "foto": "https://..."
}
```

### Clientes

Permissoes:

- `POST /clientes/` e `GET /clientes/`: somente `admin`.
- Rotas com `{clienteID}`: `admin` ou o proprio cliente, quando `clienteID` for igual ao `user_id` do JWT.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/clientes/` | Cria cliente. | `CreateClienteRequest` | `201 ClienteResponse` | `400`, `401`, `403`, `409`, `500` |
| `GET` | `BASE_URL/clientes/?cursor=&limit=50&q=` | Lista clientes, paginada por cursor. | nenhum | `200 ClienteListResponse` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/clientes/resumo` | Total de clientes cadastrados. | nenhum | `200 ClienteResumoResponse` | `401`, `403`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}` | Busca cliente com vinculos. | nenhum | `200 ClienteComVinculosResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/clientes/{clienteID}` | Atualiza cliente. | `UpdateClienteRequest` parcial | `200 ClienteResponse` | `400`, `401`, `403`, `404`, `500` |
| `DELETE` | `BASE_URL/clientes/{clienteID}` | Remove cliente. | nenhum | `204` | `400`, `401`, `403`, `404`, `409`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/reservas/` | Lista reservas do cliente. | nenhum | `200 ReservaResponse[]` | `400`, `401`, `403`, `500` |

Listagem paginada (`GET /clientes/`):

| Query param | Obrigatorio | Descricao |
| --- | --- | --- |
| `limit` | nao (padrao 50, teto 200) | tamanho da pagina |
| `cursor` | nao (ausente = primeira pagina) | opaco, devolvido em `next_cursor` |
| `q` | nao | busca por nome, telefone ou CPF |

A busca por CPF ignora pontuacao (`300.000.000-01` acha `30000000001`), mas so
quando o termo **nao tem letra**. Sem essa regra, procurar "Ana 13" viraria
tambem uma busca por documento contendo "13" e traria gente sem relacao com o
nome digitado.

```json
{
  "items": [{ "id": 1, "nome": "Maria Cliente", "cpf": "11111111111" }],
  "next_cursor": "MTIw",
  "has_more": true
}
```

`GET /clientes/resumo` responde `{"total": 137}`. Ele existe porque o painel
precisa do numero de cadastros, e contar isso baixando a tabela nao escala.

`telefone` e `foto` sao opcionais no update: omita a chave (ou envie `null`) para
manter o valor atual, e envie `""` explicitamente para limpar o campo. `nome` e
`data_nasc` continuam sendo ignorados quando enviados em branco.

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

Permissao: `admin` ou o proprio cliente identificado por `{clienteID}`. A excecao e
`GET BASE_URL/vinculos/`, que e exclusiva de `admin` por listar vinculos de todos os clientes.

Vinculo liga um cliente a um destino e a uma rota interna. Ele representa a relacao operacional do cliente com faculdade/estagio, turno, comprovante e dias fixos.

Tipos validos: `estudante`, `estagio`.

Turnos validos: `MT`, `VT`, `NT`, `IN`.

Dias da semana em `horarios_fixos`: `1` a `5`, onde a API apenas valida o intervalo; use a convencao do produto para mapear segunda a sexta.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/vinculos/?cursor=&limit=50&q=` | Lista vinculos de todos os clientes, paginada por cursor. Somente `admin`. | nenhum | `200 VinculoListResponse` | `400`, `401`, `403`, `500` |
| `POST` | `BASE_URL/clientes/{clienteID}/vinculos/` | Cria vinculo para cliente. | `VinculoRequest` | `201 VinculoResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/vinculos/` | Lista vinculos do cliente. | nenhum | `200 VinculoResponse[]` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}` | Busca vinculo do cliente. | nenhum | `200 VinculoResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}` | Atualiza vinculo. | `VinculoRequest` | `200 VinculoResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}` | Remove vinculo. | nenhum | `204` | `400`, `401`, `403`, `404`, `409`, `500` |
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

`GET BASE_URL/vinculos/` devolve os mesmos campos acrescidos de `cliente_nome` e
`destino_nome`, no mesmo nivel do objeto, ordenados por nome do cliente. Ela existe
para o painel montar a lista sem consultar os vinculos cliente a cliente.

| Query param | Obrigatorio | Descricao |
| --- | --- | --- |
| `limit` | nao (padrao 50, teto 200) | tamanho da pagina |
| `cursor` | nao (ausente = primeira pagina) | opaco, devolvido em `next_cursor` |
| `q` | nao | busca por nome do cliente, nome do destino, curso, tipo ou turno |

O cursor carrega o par que ordena a listagem (nome do cliente, id) — o nome sozinho
nao serve por nao ser unico.

Um detalhe da consulta: `horarios_fixos` vem de um LEFT JOIN que multiplica as
linhas (uma por dia da semana). O recorte da pagina acontece **antes** desse join,
numa CTE; aplicado depois, o LIMIT cortaria no meio de um vinculo e ele viria com
parte dos dias.

```json
{
  "items": [
    {
      "id": 10,
      "cliente_id": 1,
      "cliente_nome": "Maria Souza",
      "tipo": "estudante",
      "turno": "NT",
      "destino_id": 1,
      "destino_nome": "Campus Central",
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
  ],
  "next_cursor": "TWFyaWEgU291emF8MTA",
  "has_more": true
}
```

### Reservas

Permissoes:

- `GET /reservas/`: somente `admin`.
- Rotas com `{reservaID}`: `admin` ou o cliente proprietario da reserva.
- Rotas aninhadas em `/clientes/{clienteID}`: `admin` ou o proprio cliente.

Reserva e criada a partir de um vinculo. Ela guarda snapshot de `cliente_id`, `vinculo_id`, `destino_id`, `rota_interna_id`, `data_viagem`, `turno` e `sentido`. Isso permite manter a reserva historica mesmo se o vinculo mudar depois.

Status validos: `confirmada`, `cancelada`.

Sentidos validos: `ida`, `volta`.

Turnos operacionais validos: `MT`, `VT`, `NT`. Se o vinculo for `IN`, o frontend deve enviar o turno desejado.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/clientes/{clienteID}/vinculos/{vinculoID}/reservas/disponibilidade?data_viagem=2026-06-10&turno=NT&sentido=ida` | Consulta partida, fechamento e disponibilidade para o vinculo. | nenhum | `200 DisponibilidadeReservaResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `GET` | `BASE_URL/reservas/?cursor=&limit=50&q=&data_inicio=&data_fim=` | Lista reservas, paginada por cursor. | nenhum | `200 ReservaListResponse` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/reservas/resumo` | Contagens agregadas de reservas confirmadas. | nenhum | `200 ReservaResumoResponse` | `401`, `403`, `500` |
| `GET` | `BASE_URL/reservas/{reservaID}` | Busca reserva. | nenhum | `200 ReservaResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/reservas/{reservaID}` | Atualiza dados editaveis da reserva. | `UpdateReservaRequest` parcial | `200 ReservaResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/reservas/{reservaID}/cancelar` | Cancela reserva. | nenhum | `200 ReservaResponse` | `400`, `401`, `403`, `404`, `422`, `500` |
| `DELETE` | `BASE_URL/reservas/{reservaID}` | Remove reserva. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Listagem paginada (`GET /reservas/`):

| Query param | Obrigatorio | Descricao |
| --- | --- | --- |
| `limit` | nao (padrao 50, teto 200) | tamanho da pagina |
| `cursor` | nao (ausente = primeira pagina) | opaco, devolvido em `next_cursor`; nunca monte um na mao |
| `q` | nao | busca livre por nome do cliente, nome do destino, status, turno ou sentido. **Nao busca por data** — use `data_inicio`/`data_fim` |
| `data_inicio`, `data_fim` | nao | filtro de intervalo por `data_viagem`, formato `YYYY-MM-DD`, inclusivo nas duas pontas |

```json
{
  "items": [
    {
      "id": 1,
      "cliente_id": 1,
      "cliente_nome": "Maria Souza",
      "vinculo_id": 10,
      "data_viagem": "2026-06-10",
      "turno": "NT",
      "destino_id": 1,
      "destino_nome": "Campus A",
      "rota_interna_id": 1,
      "sentido": "ida",
      "status": "confirmada",
      "created_at": "2026-06-06T20:00:00Z",
      "updated_at": "2026-06-06T20:00:00Z"
    }
  ],
  "next_cursor": "MjAyNi0wNi0xMHwx",
  "has_more": true
}
```

`next_cursor` so aparece quando `has_more` e `true`. Repassar esse valor no `cursor`
da proxima chamada continua exatamente de onde a pagina anterior parou — sem
`OFFSET`, entao o custo da consulta nao cresce conforme a tabela cresce.

Resumo (`GET /reservas/resumo`) existe para o painel exibir totais sem baixar a
tabela: ele agrega no banco (`GROUP BY turno`), entao o custo nao cresce junto com
o numero de reservas. Conta apenas reservas `confirmada`.

```json
{
  "confirmadas_total": 7,
  "confirmadas_por_turno": { "MT": 4, "NT": 3 }
}
```

Regras de cancelamento:

- `admin`: pode cancelar qualquer reserva.
- `cliente`: pode cancelar apenas a propria reserva. Tentar cancelar reserva de outro cliente retorna `403`.
- Reservas canceladas nao entram no planejamento de viagens.
- Se a reserva estiver vinculada a uma viagem com rota dinamica e for cancelada antes da janela de bloqueio, a rota dinamica pode ser invalidada para recalculo automatico.

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

Cada sentido fecha separadamente 30 minutos antes da partida configurada para o municipio de destino e turno. A ida usa `horario_ida`; a volta usa `horario_volta`. No instante exato do fechamento, criar ou reativar uma reserva retorna `409`. O horario precisa estar configurado antes da reserva; caso contrario, a API retorna `422`.

Consulta de disponibilidade:

```json
{
  "data_viagem": "2026-06-10",
  "turno": "NT",
  "sentido": "ida",
  "partida_em": "2026-06-10T17:00:00-03:00",
  "fechamento_em": "2026-06-10T16:30:00-03:00",
  "consultado_em": "2026-06-10T15:40:00-03:00",
  "disponivel": true
}
```

O frontend pode usar `fechamento_em` para desabilitar o botao antecipadamente, mas o backend sempre repete a validacao ao criar ou alterar a reserva. Os timestamps usam o fuso configurado em `APP_TIMEZONE`.

### Horarios por Turno de Viagem

Permissao: `admin`.

Define os horarios padrao por municipio de destino que o planejamento usa para criar a partida prevista de ida e volta.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/horarios-turno-viagem/` | Cria horario por municipio de destino/turno. | `HorarioTurnoViagemRequest` | `201 HorarioTurnoViagemResponse` | `400`, `401`, `403`, `409`, `422`, `500` |
| `GET` | `BASE_URL/horarios-turno-viagem/` | Lista horarios. | nenhum | `200 HorarioTurnoViagemResponse[]` | `401`, `403`, `500` |
| `GET` | `BASE_URL/horarios-turno-viagem/{horarioTurnoID}` | Busca horario. | nenhum | `200 HorarioTurnoViagemResponse` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/horarios-turno-viagem/{horarioTurnoID}` | Atualiza horario. | `HorarioTurnoViagemRequest` parcial | `200 HorarioTurnoViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `DELETE` | `BASE_URL/horarios-turno-viagem/{horarioTurnoID}` | Remove horario. | nenhum | `204` | `400`, `401`, `403`, `404`, `500` |

Request:

```json
{
  "municipio_destino_id": 2704302,
  "turno": "NT",
  "horario_ida": "17:00",
  "horario_volta": "22:00"
}
```

Response:

```json
{
  "id": 1,
  "municipio_destino_id": 2704302,
  "turno": "NT",
  "horario_ida": "17:00:00",
  "horario_volta": "22:00:00",
  "created_at": "2026-06-06T20:00:00Z",
  "updated_at": "2026-06-06T20:00:00Z"
}
```

`horario_volta` precisa ser maior que `horario_ida`.

### Planejamento de Viagens

O planejamento e iniciado exclusivamente pelo processador automatico. Nao existe endpoint publico para admin criar viagens manualmente.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/planejamentos/execucoes/falhas?limit=50` | Lista execucoes que aguardam retry. | - | `200 ExecucaoPlanejamentoFalhaResponse[]` | `400`, `401`, `403`, `500` |

Regras operacionais importantes:

- Cada execucao processa somente um sentido: `ida` ou `volta`.
- A ida usa reservas `confirmada` do mesmo `data_viagem`, `turno`, `rota_interna_id` e municipio de destino.
- A volta usa apenas reservas confirmadas de clientes com presenca `embarcou` em uma ida do mesmo planejamento.
- Usa `horarios_turno_viagem` para definir a partida prevista do sentido solicitado.
- Calcula `expires_at` automaticamente como `data_viagem + 3 meses`. O frontend nao envia esse campo no request.
- Na ida, aloca veiculos por capacidade/disponibilidade e motoristas por cidade de destino, turno e disponibilidade; depois cria os ciclos.
- Na volta, reutiliza os ciclos, veiculos e motoristas criados pela ida. Todos os ciclos da ida recebem uma viagem de volta, mesmo quando nao ha passageiro elegivel.
- A ida deve ser planejada antes da volta.

O processador interno de planejamento executa todos os candidatos devidos encontrados em uma unica chamada, inclusive quando varias cidades ou rotas possuem o mesmo horario. Ele usa `execucoes_planejamento` para impedir duplicidade e permitir recuperacao de falhas. Os retries usam intervalos progressivos de 1, 2, 4 e no maximo 5 minutos; `proxima_tentativa_em` informa quando uma falha volta a ser elegivel.

Exemplo de falha consultada pelo admin:

```json
{
  "id": 7,
  "data_viagem": "2026-08-12",
  "turno": "NT",
  "municipio_destino_id": 2704302,
  "rota_interna_id": 3,
  "sentido": "ida",
  "partida_em": "2026-08-12T17:00:00-03:00",
  "fechamento_em": "2026-08-12T16:30:00-03:00",
  "status": "falhou",
  "tentativas": 2,
  "ultimo_erro": "vehicles unavailable",
  "proxima_tentativa_em": "2026-08-12T16:32:00-03:00",
  "finalizado_em": "2026-08-12T16:30:00-03:00"
}
```

#### Disparo interno do planejamento

| Metodo | Path completo | Autenticacao | Sucesso | Erros |
| --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/internal/planejamentos/processar` | `Authorization: Bearer <PLANNING_CRON_SECRET>` | `200 ResumoProcessamentoPlanejamentoResponse` | `401`, `500` |

Resposta:

```json
{
  "candidatos": 4,
  "devidos": 2,
  "adquiridos": 2,
  "concluidos": 2,
  "sem_demanda": 0,
  "falhos": 0
}
```

Esse endpoint nao aceita JWT de admin, cliente ou motorista. O Supabase Cron o chama a cada minuto com um segredo exclusivo armazenado no Vault. A configuracao executavel esta em `deploy/supabase/planning_cron.sql`.

#### Limpeza interna de retencao

| Metodo | Path completo | Autenticacao | Sucesso | Erros |
| --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/internal/retencao/limpar` | `Authorization: Bearer <PLANNING_CRON_SECRET>` | `200 ResumoLimpezaResponse` | `401`, `500` |

Remove os dados operacionais fora da janela de retencao (padrao: 3 meses, contados
no fuso de `APP_TIMEZONE`). Como o planejamento, nao aceita JWT de nenhuma role. O
Supabase Cron o chama uma vez por dia; a configuracao esta em
`deploy/supabase/retention_cron.sql`.

Resposta:

```json
{
  "corte": "2026-05-11T00:00:00-03:00",
  "ciclos_removidos": 12,
  "reservas_removidas": 340,
  "execucoes_removidas": 24,
  "lote_saturado": false
}
```

`corte` e a data limite: tudo com `data_viagem` anterior a ela foi removido. Os
ciclos saem primeiro e o `ON DELETE CASCADE` leva junto viagens, `viagem_reservas`,
`viagem_reserva_confirmacoes`, `viagem_horarios`, `viagem_localizacoes`,
`rotas_dinamicas` e `rota_dinamica_destinos`; so entao as reservas ficam livres da
FK `RESTRICT` de `viagem_reservas`. Cadastros nunca sao removidos.

`lote_saturado: true` indica que alguma tabela atingiu `RETENTION_BATCH_LIMIT` e
ainda ha registros vencidos para a proxima execucao.

### Viagens

Permissao: `admin` ou `motorista`. O motorista lista apenas as viagens atribuidas a ele e recebe `403` ao tentar consultar ou operar uma viagem de outro motorista.

Status de viagem: `programada`, `em_andamento`, `concluida`, `cancelada`.

Status de ciclo: `planejado`, `em_andamento`, `concluido`, `cancelado`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `GET` | `BASE_URL/viagens/?cursor=&limit=50&q=&data_inicio=&data_fim=&status=&ordem=` | Lista viagens com ciclo, paginada por cursor. | nenhum | `200 ViagemListResponse` | `400`, `401`, `403`, `500` |
| `GET` | `BASE_URL/viagens/resumo` | Agregados de viagens para o painel. Somente `admin`. | nenhum | `200 ViagemResumoResponse` | `401`, `403`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}` | Busca viagem com ciclo. | nenhum | `200 ViagemComCicloResponse` | `400`, `401`, `403`, `404`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/iniciar` | Inicia viagem e registra `inicio_real`. | nenhum | `200 ViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/concluir` | Conclui viagem e registra `fim_real`. | nenhum | `200 ViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `POST` | `BASE_URL/viagens/{viagemID}/cancelar` | Cancela viagem. | nenhum | `200 ViagemResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}/horarios` | Lista horarios da viagem. | nenhum | `200 ViagemHorarioResponse[]` | `400`, `401`, `403`, `404`, `500` |
| `GET` | `BASE_URL/viagens/{viagemID}/reservas/` | Lista reservas alocadas na viagem. | nenhum | `200 ViagemReservaComReservaResponse[]` | `400`, `401`, `403`, `404`, `500` |
| `PUT` | `BASE_URL/viagens/{viagemID}/reservas/{reservaID}/presenca` | Atualiza presenca do aluno na viagem. | `AtualizarPresencaRequest` | `200 ViagemReservaResponse` | `400`, `401`, `403`, `404`, `409`, `422`, `500` |

Listagem paginada (`GET /viagens/`):

| Query param | Obrigatorio | Descricao |
| --- | --- | --- |
| `limit` | nao (padrao 50, teto 200) | tamanho da pagina |
| `cursor` | nao (ausente = primeira pagina) | opaco, devolvido em `next_cursor` |
| `q` | nao | busca por nome do municipio, placa do veiculo, status, turno ou sentido. **Nao busca por data** |
| `data_inicio`, `data_fim` | nao | intervalo por `data_viagem`, formato `YYYY-MM-DD`, inclusivo |
| `status` | nao (repetivel) | filtra por status de viagem; sem ele, traz todos |
| `ordem` | nao (padrao `desc`) | `asc` traz a viagem mais proxima primeiro; usado pelo monitoramento |

Cada item traz `municipio_nome` e `veiculo_placa` resolvidos, alem dos ids. O
motorista autenticado recebe apenas as proprias viagens — o recorte acontece na
consulta, entao a paginacao dele nao vem com paginas incompletas.

```json
{
  "items": [
    {
      "viagem": { "id": 1, "sentido": "ida", "status": "programada" },
      "ciclo": { "id": 1, "data_viagem": "2026-06-10", "turno": "NT", "municipio_destino_id": 2704302 },
      "municipio_nome": "Maceio",
      "veiculo_placa": "ABC1D23"
    }
  ],
  "next_cursor": "MjAyNi0wNi0xMHwx",
  "has_more": true
}
```

Resumo (`GET /viagens/resumo`) agrega no banco o que o painel precisa, para nao
baixar a tabela inteira so para contar. `hoje` e resolvido no fuso da operacao,
nao no do servidor. `proximas` vem em ordem crescente, limitada a 6.

```json
{
  "por_status": { "programada": 12, "concluida": 40 },
  "por_turno": { "MT": 20, "NT": 32 },
  "hoje_total": 4,
  "hoje_em_andamento": 1,
  "proximas": []
}
```

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

Permissao: `admin` ou o motorista atribuido a viagem.

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
- O backend herda `expires_at` da viagem/ciclo. O frontend nao envia esse campo no request.
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

**municipios**

- `codigo_ibge`
- `nome`
- `uf`
- `ativo`
- `created_at`
- `updated_at`

Catalogo oficial importado do IBGE e fonte unica para nomes e UFs dos municipios.

**destinos**

- `id`
- `nome`
- `rua`
- `municipio_id`
- `latitude`
- `longitude`
- `created_at`
- `updated_at`

Destino e faculdade/local de desembarque. Na volta, tambem e o local onde o aluno embarca para retornar.

Relacionamento: `destinos.municipio_id -> municipios.codigo_ibge`.

**paradas**

- `id`
- `nome`
- `latitude`
- `longitude`
- `created_at`
- `updated_at`

Parada e local da rota interna onde o veiculo passa para pegar alunos na cidade.

**rotas_internas**

- `id`
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
- `status`
- opcionais booleanos de conforto
- `created_at`
- `updated_at`

Para telas: veiculo fica disponivel para planejamento se estiver ativo. A categoria determina capacidade fixa.

**motoristas**

- `id`
- `nome`
- `cpf`
- `senha`
- `telefone`
- `data_nasc`
- `turno`
- `municipio_trabalho_id`
- `residencia`
- `foto`
- `created_at`
- `updated_at`

Para telas: motorista e atribuido automaticamente no planejamento.

Relacionamento: `motoristas.municipio_trabalho_id -> municipios.codigo_ibge`.

**reservas**

- `id`
- `cliente_id`
- `vinculo_id`
- `data_viagem`
- `turno`
- `destino_id`
- `rota_interna_id`
- `sentido`
- `status`
- `created_at`
- `updated_at`

Relacionamentos:

- `reservas.cliente_id -> clientes.id`
- `reservas.vinculo_id -> cliente_vinculos.id`
- `reservas.destino_id -> destinos.id`
- `reservas.rota_interna_id -> rotas_internas.id`

Para telas: reservas sao o ponto central do app do cliente. A reserva sabe o dia, turno, sentido, destino e rota interna. `destino_id` e `rota_interna_id` sao snapshots do vinculo.

**horarios_turno_viagem**

- `id`
- `municipio_destino_id`
- `turno`
- `horario_ida`
- `horario_volta`
- `created_at`
- `updated_at`

Para telas admin: configurar horario padrao antes de planejar viagens.

Relacionamento: `horarios_turno_viagem.municipio_destino_id -> municipios.codigo_ibge`.

**ciclos_viagem**

- `id`
- `data_viagem`
- `turno`
- `municipio_destino_id`
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
- `ciclos_viagem.municipio_destino_id -> municipios.codigo_ibge`

Para telas: ciclo representa o bloco operacional com o mesmo veiculo, motorista e municipio de destino. Normalmente agrupa ida e volta. `municipio_destino_id` preserva a identidade oficial do municipio no historico.

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

municipios
  1 ── N destinos
  1 ── N motoristas
  1 ── N horarios_turno_viagem
  1 ── N ciclos_viagem

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

Antes de usar o painel, precisa existir um administrador. Com as variaveis `ADMIN_EMAIL`, `ADMIN_PASSWORD` e `DATABASE_URL` configuradas, rode:

```bash
go run ./cmd/admin seed
```

O painel nao expoe cadastro de administradores: criar, listar, trocar senha e remover
contas de admin sao operacoes do `cmd/admin`, feitas com acesso direto ao banco. Veja
"Gerenciamento de administradores" no README.

Depois, o frontend faz:

```http
POST /admin/login
```

e guarda o JWT para as operacoes administrativas.

### 2. Configuracao base pelo admin

Ordem recomendada:

1. Importar municipios com `make municipios/import`.
2. Criar destinos.
3. Criar paradas.
4. Criar rotas internas com paradas ordenadas.
5. Criar veiculos.
6. Criar motoristas.
7. Criar horarios por municipio de destino/turno em `horarios-turno-viagem`.
8. Criar clientes.
9. Criar vinculos dos clientes.

Dependencias:

- Vinculo precisa de `cliente_id`, `destino_id` e `rota_interna_id`.
- Rota interna precisa de paradas existentes.
- Planejamento precisa de horario configurado para `municipio_destino_id + turno`.
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

### 4. Sistema planeja viagens automaticamente

O Supabase Cron chama `POST /internal/planejamentos/processar` a cada minuto. Ao atingir 30 minutos antes da partida, o processador fecha as reservas daquela data, turno, cidade de destino, rota interna e sentido e inicia o planejamento.

Na ida, o backend:

1. Busca reservas confirmadas de ida. Reservas canceladas sao ignoradas.
2. Busca o horario de ida em `horarios_turno_viagem`.
3. Calcula veiculos por capacidade.
4. Aloca veiculos disponiveis.
5. Aloca motoristas disponiveis.
6. Calcula `expires_at` automaticamente para retencao de dados.
7. Cria ciclos de viagem.
8. Cria as viagens de ida.
9. Cria os horarios previstos.
10. Cria `viagem_reservas` para as reservas alocadas.

No horario de fechamento da volta, o processador reutiliza os ciclos da ida. Os veiculos e motoristas nao sao recalculados, e somente clientes com presenca `embarcou` na ida entram como passageiros da volta.

**Regras de alocacao de veiculos:**

O backend tenta alocar o veiculo ideal para a quantidade de alunos. Se nao existir o ideal disponivel, usa fallback para veiculo maior:

- `carro_7_lugares` → `escolar` → `executivo`
- `escolar` → `executivo`

Veiculos com status `inativo` ou `manutencao` nao sao alocados. Veiculos ja alocados em outro ciclo no mesmo dia e turno tambem nao entram.

**Regras de alocacao de motoristas:**

- Motorista de outro turno ou cidade de destino nao e alocado.
- Motorista ja alocado em outro ciclo no mesmo dia e turno nao e reutilizado.

### 5. Rota dinamica

Depois que a viagem existe, a rota dinamica pode ser gerada:

```http
POST /viagens/{viagemID}/rota-dinamica/calcular
```

Tambem existe worker automatico que processa viagens dentro da janela de calculo. Mesmo assim, para apresentacao ou painel admin, o endpoint manual e util para forcar o calculo.

**Comportamento em caso de falha:** se o servico de roteamento (OSRM) falhar, a API nao persiste rota parcial. O frontend deve tratar ausencia de rota dinamica como estado valido e permitir nova tentativa.

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

### Supabase Storage

Permissao: `admin`, `cliente` ou `motorista`.

Estes endpoints nao recebem o arquivo em si. Eles retornam URLs assinadas para que o frontend envie ou visualize arquivos diretamente no Supabase Storage. Isso evita enviar arquivos grandes pela API e respeita os limites configurados nos buckets privados do Supabase.

Buckets aceitos:

- `fotos`
- `documentos`

Regras de acesso por path:

- `admin`: pode assinar paths permitidos em qualquer bucket.
- `cliente`: pode assinar apenas paths iniciando com `clientes/{user_id}/`.
- `motorista`: pode assinar apenas paths iniciando com `motoristas/{user_id}/` e somente no bucket `fotos`.

Content types aceitos:

- `fotos`: `image/jpeg`, `image/png`, `image/webp`.
- `documentos`: `application/pdf`, `image/jpeg`, `image/png`, `image/webp`.

| Metodo | Path completo | Descricao | Body | Sucesso | Erros |
| --- | --- | --- | --- | --- | --- |
| `POST` | `BASE_URL/storage/signed-upload-url` | Gera URL assinada para upload direto no Supabase. | `SignedUploadURLRequest` | `201 SignedUploadURLResponse` | `400`, `401`, `403`, `422`, `500` |
| `POST` | `BASE_URL/storage/signed-download-url` | Gera URL temporaria para visualizar/download de arquivo privado. | `SignedDownloadURLRequest` | `200 SignedDownloadURLResponse` | `400`, `401`, `403`, `422`, `500` |

Request de upload:

```json
{
  "bucket": "fotos",
  "path": "clientes/1/foto.png",
  "content_type": "image/png",
  "upsert": true
}
```

Response:

```json
{
  "bucket": "fotos",
  "path": "clientes/1/foto.png",
  "signed_url": "https://project.supabase.co/storage/v1/object/upload/sign/fotos/clientes/1/foto.png?token=...",
  "token": "token-opcional"
}
```

Depois disso, o frontend envia o arquivo diretamente para `signed_url`, por exemplo:

```ts
await fetch(response.signed_url, {
  method: "PUT",
  headers: {
    "Content-Type": file.type
  },
  body: file
});
```

Depois do upload, salve o `path` no recurso correspondente da API, por exemplo:

- `clientes.foto`
- `motoristas.foto`
- `cliente_vinculos.comprovante`

Request de download:

```json
{
  "bucket": "documentos",
  "path": "clientes/1/vinculos/10/comprovante.pdf",
  "expires_in_seconds": 900
}
```

Response:

```json
{
  "bucket": "documentos",
  "path": "clientes/1/vinculos/10/comprovante.pdf",
  "signed_url": "https://project.supabase.co/storage/v1/object/sign/documentos/clientes/1/vinculos/10/comprovante.pdf?token=...",
  "expires_in_seconds": 900
}
```

Se `expires_in_seconds` nao for enviado, a API usa `900` segundos. O valor aceito fica entre `60` e `3600`.
