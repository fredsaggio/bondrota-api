# API Endpoints Documentation

Documentação completa de todos os endpoints da API Bondrota. Pronta para testes no Postman ou similar.

**Base URL:** `http://localhost:8080`

---

## 📋 Índice

1. [Health Check](#health-check)
2. [Admin](#admin)
3. [Motoristas](#motoristas)
4. [Veículos](#veículos)
5. [Destinos](#destinos)
6. [Paradas](#paradas)
7. [Rotas Internas](#rotas-internas)
8. [Clientes](#clientes)
9. [Reservas](#reservas)
10. [Viagens](#viagens)

---

## Health Check

### GET /health

Verifica se o servidor está funcionando.

**Request:**

```http
GET /health HTTP/1.1
Host: localhost:8080
```

**Response:** `200 OK`

```json
{}
```

---

## Admin

### POST /admin - Criar Administrador

Cria um novo administrador.

**Request:**

```http
POST /admin HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
  "email": "admin@example.com",
  "senha": "senha123"
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "email": "admin@example.com"
}
```

---

### GET /admin - Listar Todos os Administradores

Lista todos os administradores cadastrados.

**Request:**

```http
GET /admin HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "email": "admin@example.com"
  },
  {
    "id": 2,
    "email": "outro@example.com"
  }
]
```

---

### GET /admin/{adminID} - Obter Administrador por ID

Obtém informações de um administrador específico.

**Request:**

```http
GET /admin/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "email": "admin@example.com"
}
```

---

### PUT /admin/{adminID} - Atualizar Administrador

Atualiza as informações de um administrador.

**Request:**

```http
PUT /admin/1 HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "email": "novoemail@example.com"
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "email": "novoemail@example.com"
}
```

---

### DELETE /admin/{adminID} - Deletar Administrador

Remove um administrador do sistema.

**Request:**

```http
DELETE /admin/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

```
(sem corpo)
```

---

### POST /admin/login - Login de Administrador

Autentica um administrador, define um cookie HttpOnly e mantém o token JWT no JSON por compatibilidade.

**Request:**

```http
POST /admin/login HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
  "email": "admin@example.com",
  "senha": "senha123"
}
```

**Response:** `200 OK`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

A resposta também envia o cookie bondrota_admin_session como HttpOnly. O painel
envia X-Admin-Session-Mode: cookie para receber 204 sem JWT no corpo e
usa GET /admin/session para consultar a sessão e POST /admin/logout para expirar
o cookie.

---

## Motoristas

### POST /motoristas/login - Login de Motorista

Autentica um motorista e retorna um token JWT.

**Request:**

```http
POST /motoristas/login HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
  "cpf": "12345678900",
  "senha": "senha123"
}
```

**Response:** `200 OK`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

---

### POST /motoristas - Criar Motorista

Cria um novo motorista no sistema.

**Request:**

```http
POST /motoristas HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "João Silva",
  "cpf": "12345678900",
  "senha": "senha123",
  "telefone": "11999999999",
  "data_nasc": "1990-05-15",
  "turno": "MT",
  "municipio_trabalho_id": 3550308,
  "foto": "https://example.com/foto.jpg"
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "nome": "João Silva",
  "cpf": "12345678900",
  "telefone": "11999999999",
  "data_nasc": "1990-05-15",
  "turno": "MT",
  "municipio_trabalho_id": 3550308,
  "foto": "https://example.com/foto.jpg"
}
```

**Valores válidos para turno:**

- `MT` - Matutino (06:00 - 14:00)
- `VT` - Vespertino (14:00 - 22:00)
- `NT` - Noturno (22:00 - 06:00)
- `IN` - Integral (Tempo integral)

---

### GET /motoristas - Listar Todos os Motoristas

Lista todos os motoristas cadastrados.

**Request:**

```http
GET /motoristas HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "nome": "João Silva",
    "cpf": "12345678900",
    "telefone": "11999999999",
    "data_nasc": "1990-05-15",
    "turno": "MT",
    "municipio_trabalho_id": 3550308,
    "foto": "https://example.com/foto.jpg"
  },
  {
    "id": 2,
    "nome": "Maria Santos",
    "cpf": "98765432100",
    "telefone": "11988888888",
    "data_nasc": "1988-03-20",
    "turno": "VT",
    "municipio_trabalho_id": 3304557,
    "foto": "https://example.com/foto2.jpg"
  }
]
```

---

### GET /motoristas/{id} - Obter Motorista por ID

Obtém informações de um motorista específico.

**Request:**

```http
GET /motoristas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "João Silva",
  "cpf": "12345678900",
  "telefone": "11999999999",
  "data_nasc": "1990-05-15",
  "turno": "MT",
  "municipio_trabalho_id": 3550308,
  "foto": "https://example.com/foto.jpg"
}
```

---

### PUT /motoristas/{id} - Atualizar Motorista

Atualiza as informações de um motorista.

**Request:**

```http
PUT /motoristas/1 HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "João Silva Santos",
  "telefone": "11987654321",
  "turno": "VT"
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "João Silva Santos",
  "cpf": "12345678900",
  "telefone": "11987654321",
  "data_nasc": "1990-05-15",
  "turno": "VT",
  "municipio_trabalho_id": 3550308,
  "foto": "https://example.com/foto.jpg"
}
```

---

### DELETE /motoristas/{id} - Deletar Motorista

Remove um motorista do sistema.

**Request:**

```http
DELETE /motoristas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

```
(sem corpo)
```

---

## Veículos

### POST /veiculos - Criar Veículo

Cria um novo veículo na frota.

**Request:**

```http
POST /veiculos HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "placa": "ABC-1234",
  "modelo": "Mercedes Benz Sprinter",
  "capacidade": 45,
  "status": "ativo",
  "ar_condicionado": true,
  "banheiro": true,
  "persiana": true,
  "luz_leitura": true,
  "tomada": true
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "placa": "ABC-1234",
  "modelo": "Mercedes Benz Sprinter",
  "capacidade": 45,
  "status": "ativo",
  "ar_condicionado": true,
  "banheiro": true,
  "persiana": true,
  "luz_leitura": true,
  "tomada": true
}
```

---

### GET /veiculos - Listar Todos os Veículos

Lista todos os veículos cadastrados.

**Request:**

```http
GET /veiculos HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "placa": "ABC-1234",
    "modelo": "Mercedes Benz Sprinter",
    "capacidade": 45,
    "status": "ativo",
    "ar_condicionado": true,
    "banheiro": true,
    "persiana": true,
    "luz_leitura": true,
    "tomada": true
  },
  {
    "id": 2,
    "placa": "XYZ-5678",
    "modelo": "Iveco Daily",
    "capacidade": 30,
    "status": "ativo",
    "ar_condicionado": true,
    "banheiro": false,
    "persiana": true,
    "luz_leitura": true,
    "tomada": false
  }
]
```

---

### GET /veiculos/{veiculoID} - Obter Veículo por ID

Obtém informações de um veículo específico.

**Request:**

```http
GET /veiculos/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "placa": "ABC-1234",
  "modelo": "Mercedes Benz Sprinter",
  "capacidade": 45,
  "status": "ativo",
  "ar_condicionado": true,
  "banheiro": true,
  "persiana": true,
  "luz_leitura": true,
  "tomada": true
}
```

---

### PUT /veiculos/{veiculoID} - Atualizar Veículo

Atualiza as informações de um veículo.

**Request:**

```http
PUT /veiculos/1 HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "placa": "ABC-1234",
  "modelo": "Mercedes Benz Sprinter",
  "capacidade": 50,
  "status": "manutencao",
  "ar_condicionado": true,
  "banheiro": true,
  "persiana": true,
  "luz_leitura": true,
  "tomada": true
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "placa": "ABC-1234",
  "modelo": "Mercedes Benz Sprinter",
  "capacidade": 50,
  "status": "manutencao",
  "ar_condicionado": true,
  "banheiro": true,
  "persiana": true,
  "luz_leitura": true,
  "tomada": true
}
```

**Valores válidos para status:**

- `ativo`
- `inativo`
- `manutencao`

---

### DELETE /veiculos/{veiculoID} - Deletar Veículo

Remove um veículo da frota.

**Request:**

```http
DELETE /veiculos/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

```
(sem corpo)
```

---

## Municípios

### GET /municipios/?uf={UF} - Listar Municípios por UF

Lista os municípios ativos importados do IBGE para preencher os dropdowns administrativos.

```http
GET /municipios/?uf=AL HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

```json
[
  {
    "codigo_ibge": 2704302,
    "nome": "Maceió",
    "uf": "AL"
  }
]
```

---

## Destinos

### POST /destinos - Criar Destino

Cria um novo destino, como uma faculdade ou outro local final do cliente.

**Request:**

```http
POST /destinos HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "Universidade Federal de Alagoas",
  "rua": "Av. Lourival Melo Mota",
  "municipio_id": 2704302,
  "latitude": -9.555000,
  "longitude": -35.775000
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "nome": "Universidade Federal de Alagoas",
  "rua": "Av. Lourival Melo Mota",
  "municipio_id": 2704302,
  "latitude": -9.555000,
  "longitude": -35.775000
}
```

---

### GET /destinos - Listar Todos os Destinos

Lista todos os destinos cadastrados.

**Request:**

```http
GET /destinos HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "nome": "Terminal Rodoviário Central",
    "rua": "Avenida Paulista, 1000",
    "municipio_id": 3550308,
    "latitude": -23.5505,
    "longitude": -46.6333
  },
  {
    "id": 2,
    "nome": "Estação da Luz",
    "rua": "Avenida Tiradentes, 100",
    "municipio_id": 3550308,
    "latitude": -23.5407,
    "longitude": -46.6243
  }
]
```

---

### GET /destinos/municipio/{municipioID} - Listar Destinos por Município

Lista todos os destinos vinculados ao código IBGE informado.

**Request:**

```http
GET /destinos/municipio/3550308 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "nome": "Terminal Rodoviário Central",
    "rua": "Avenida Paulista, 1000",
    "municipio_id": 3550308,
    "latitude": -23.5505,
    "longitude": -46.6333
  },
  {
    "id": 2,
    "nome": "Estação da Luz",
    "rua": "Avenida Tiradentes, 100",
    "municipio_id": 3550308,
    "latitude": -23.5407,
    "longitude": -46.6243
  }
]
```

---

### GET /destinos/{id} - Obter Destino por ID

Obtém informações de um destino específico.

**Request:**

```http
GET /destinos/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "Terminal Rodoviário Central",
  "rua": "Avenida Paulista, 1000",
  "municipio_id": 3550308,
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

---

### PUT /destinos/{id} - Atualizar Destino

Atualiza as informações de um destino.

**Request:**

```http
PUT /destinos/1 HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "Terminal Rodoviário Central - Atualizado",
  "rua": "Avenida Paulista, 1500",
  "municipio_id": 3550308,
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "Terminal Rodoviário Central - Atualizado",
  "rua": "Avenida Paulista, 1500",
  "municipio_id": 3550308,
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

---

### DELETE /destinos/{id} - Deletar Destino

Remove um destino do sistema.

**Request:**

```http
DELETE /destinos/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

```
(sem corpo)
```

---

## Paradas

### POST /paradas - Criar Parada

Cria uma nova parada.

**Request:**

```http
POST /paradas HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "Rodoviária de Campinas",
  "latitude": -22.8978,
  "longitude": -47.0739,
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "nome": "Rodoviária de Campinas",
  "latitude": -22.8978,
  "longitude": -47.0739,
}
```

---

### GET /paradas - Listar Todas as Paradas

Lista todas as paradas cadastradas.

**Request:**

```http
GET /paradas HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "nome": "Rodoviária de Campinas",
    "latitude": -22.8978,
    "longitude": -47.0739,
  },
  {
    "id": 2,
    "nome": "Rodoviária de Jundiaí",
    "latitude": -23.1897,
    "longitude": -46.8707,
  }
]
```

---

### GET /paradas/{id} - Obter Parada por ID

Obtém informações de uma parada específica.

**Request:**

```http
GET /paradas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "Rodoviária de Campinas",
  "latitude": -22.8978,
  "longitude": -47.0739,
}
```

---

### PUT /paradas/{id} - Atualizar Parada

Atualiza as informações de uma parada.

**Request:**

```http
PUT /paradas/1 HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "Rodoviária de Campinas - Unidade Principal",
  "latitude": -22.8978,
  "longitude": -47.0739,
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "Rodoviária de Campinas - Unidade Principal",
  "latitude": -22.8978,
  "longitude": -47.0739,
}
```

---

### DELETE /paradas/{id} - Deletar Parada

Remove uma parada do sistema.

**Request:**

```http
DELETE /paradas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

```
(sem corpo)
```

---

## Rotas Internas

### POST /rotas-internas - Criar Rota Interna

Cria uma nova rota interna com suas paradas ordenadas.

**Request:**

```http
POST /rotas-internas HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "paradas": [
    {
      "parada_id": 1,
      "ordem": 1
    },
    {
      "parada_id": 2,
      "ordem": 2
    },
    {
      "parada_id": 3,
      "ordem": 3
    }
  ]
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "paradas": [
    {
      "id": 1,
      "nome": "Rodoviária Central",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "ordem": 1
    },
    {
      "id": 2,
      "nome": "Estação da Luz",
      "latitude": -23.5407,
      "longitude": -46.6243,
      "ordem": 2
    },
    {
      "id": 3,
      "nome": "Shopping Imigrantes",
      "latitude": -23.5611,
      "longitude": -46.5841,
      "ordem": 3
    }
  ]
}
```

**Nota:** Os IDs das paradas e as ordens devem ser maiores que zero, e não pode haver ordens duplicadas.

---

### GET /rotas-internas - Listar Todas as Rotas Internas

Lista todas as rotas internas cadastradas.

**Request:**

```http
GET /rotas-internas HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "paradas": [
      {
        "id": 1,
        "nome": "Rodoviária Central",
        "latitude": -23.5505,
        "longitude": -46.6333,
        "ordem": 1
      },
      {
        "id": 2,
        "nome": "Estação da Luz",
        "latitude": -23.5407,
        "longitude": -46.6243,
        "ordem": 2
      }
    ]
  },
  {
    "id": 2,
    "paradas": [
      {
        "id": 4,
        "nome": "Rodoviária de Campinas",
        "latitude": -22.8978,
        "longitude": -47.0739,
        "ordem": 1
      }
    ]
  }
]
```

---

### GET /rotas-internas/{id} - Obter Rota Interna por ID

Obtém informações de uma rota interna específica.

**Request:**

```http
GET /rotas-internas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "paradas": [
    {
      "id": 1,
      "nome": "Rodoviária Central",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "ordem": 1
    },
    {
      "id": 2,
      "nome": "Estação da Luz",
      "latitude": -23.5407,
      "longitude": -46.6243,
      "ordem": 2
    }
  ]
}
```

---

### PUT /rotas-internas/{id}/paradas - Atualizar Paradas da Rota

Atualiza as paradas e suas ordens em uma rota interna.

**Request:**

```http
PUT /rotas-internas/1/paradas HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "paradas": [
    {
      "parada_id": 1,
      "ordem": 1
    },
    {
      "parada_id": 3,
      "ordem": 2
    },
    {
      "parada_id": 2,
      "ordem": 3
    }
  ]
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "paradas": [
    {
      "id": 1,
      "nome": "Rodoviária Central",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "ordem": 1
    },
    {
      "id": 3,
      "nome": "Shopping Imigrantes",
      "latitude": -23.5611,
      "longitude": -46.5841,
      "ordem": 2
    },
    {
      "id": 2,
      "nome": "Estação da Luz",
      "latitude": -23.5407,
      "longitude": -46.6243,
      "ordem": 3
    }
  ]
}
```

---

### DELETE /rotas-internas/{id} - Deletar Rota Interna

Remove uma rota interna do sistema.

**Request:**

```http
DELETE /rotas-internas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

```
(sem corpo)
```

---

## Clientes

Endpoints para gerenciar clientes e seus vinculos.

### Rotas de cliente

```http
POST /clientes/login
POST /clientes/
GET /clientes/
GET /clientes/{clienteID}
PUT /clientes/{clienteID}
DELETE /clientes/{clienteID}
```

### Rotas de vinculos do cliente

Os vinculos agora ficam aninhados em clientes. Isso deixa explicito a qual cliente cada vinculo pertence.

```http
GET /vinculos/
POST /clientes/{clienteID}/vinculos/
GET /clientes/{clienteID}/vinculos/
GET /clientes/{clienteID}/vinculos/{vinculoID}
PUT /clientes/{clienteID}/vinculos/{vinculoID}
DELETE /clientes/{clienteID}/vinculos/{vinculoID}
POST /clientes/{clienteID}/vinculos/{vinculoID}/reservas/
GET /clientes/{clienteID}/vinculos/{vinculoID}/reservas/
```

Use `GET /vinculos/` quando a tela administrativa precisar de todos os vinculos de uma vez; ela e exclusiva de `admin`, ja devolve `cliente_nome` em cada item e evita uma consulta por cliente. Use `GET /clientes/{clienteID}/vinculos/` quando a tela precisar escolher apenas um vinculo. Use `GET /clientes/{clienteID}` quando precisar do cliente completo com seus vinculos e horarios. Quando a rota recebe `clienteID` e `vinculoID`, a API valida se o vinculo pertence ao cliente informado. Se nao pertencer, retorna `404 Not Found`.

---

## Reservas

Endpoints para gerenciar reservas feitas pelos clientes.

Reservas copiam dados operacionais do vinculo no momento da criacao: `cliente_id`, `destino_id`, `rota_interna_id` e `turno`. O municipio e obtido pelo destino. Para vinculos integrais (`IN`), o campo `turno` deve ser informado como `MT`, `VT` ou `NT`.

### POST /clientes/{clienteID}/vinculos/{vinculoID}/reservas/ - Criar reserva

```http
POST /clientes/1/vinculos/5/reservas/ HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "data_viagem": "2026-06-10",
  "turno": "NT",
  "sentido": "ida"
}
```

O status da reserva nasce como `confirmada` pelo default do banco. O create nao recebe `status`.

O `clienteID` e o `vinculoID` vem da URL. O body nao recebe `vinculo_id`.

**Response:** `201 Created`

```json
{
  "id": 1,
  "cliente_id": 1,
  "vinculo_id": 5,
  "data_viagem": "2026-06-10",
  "turno": "NT",
  "destino_id": 1,
  "rota_interna_id": 1,
  "sentido": "ida",
  "status": "confirmada",
  "created_at": "2026-06-05T19:10:00-03:00",
  "updated_at": "2026-06-05T19:10:00-03:00"
}
```

### GET /reservas/ - Listar reservas

```http
GET /reservas/ HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

### GET /reservas/{reservaID} - Buscar reserva por ID

```http
GET /reservas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

### GET /clientes/{clienteID}/reservas/ - Listar reservas do cliente

```http
GET /clientes/1/reservas/ HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

### GET /clientes/{clienteID}/vinculos/{vinculoID}/reservas/ - Listar reservas do vínculo

```http
GET /clientes/1/vinculos/5/reservas/ HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

### PUT /reservas/{reservaID} - Atualizar reserva

```http
PUT /reservas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "data_viagem": "2026-06-11",
  "turno": "NT",
  "sentido": "volta",
  "status": "confirmada"
}
```

### POST /reservas/{reservaID}/cancelar - Cancelar reserva

```http
POST /reservas/1/cancelar HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

### DELETE /reservas/{reservaID} - Remover reserva

```http
DELETE /reservas/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `204 No Content`

---

## Viagens

> **Status:** 🔄 Em desenvolvimento

Endpoints para gerenciar viagens/trajetos.

---

## Headers Padrão

Todos os endpoints, exceto health, config, os três logins e o logout administrativo, requerem autenticação. Apps de cliente e motorista usam:

```http
Authorization: Bearer <seu_token_jwt>
Content-Type: application/json (para requisições com body)
```

O painel administrativo usa o cookie HttpOnly e envia as requisições com credenciais.

## Status HTTP Padrão

- `200 OK` - Requisição bem-sucedida
- `201 Created` - Recurso criado com sucesso
- `204 No Content` - Operação bem-sucedida sem retorno de conteúdo
- `400 Bad Request` - Dados inválidos na requisição
- `401 Unauthorized` - Token ausente ou inválido
- `404 Not Found` - Recurso não encontrado
- `500 Internal Server Error` - Erro do servidor

---

**Última atualização:** 05/06/2026
