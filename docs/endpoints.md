# API Endpoints Documentation

Documentação completa de todos os endpoints da API Bondrota. Pronta para testes no Postman ou similar.

**Base URL:** `http://localhost:8080`

---

## 📋 Índice

1. [Health Check](#health-check)
2. [Admin](#admin)
3. [Motoristas](#motoristas)
4. [Veículos](#veículos)
5. [Pontos](#pontos)
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

Autentica um administrador e retorna um token JWT.

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
  "cidade_trabalho": "São Paulo",
  "residencia": "Rua das Flores, 123",
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
  "cidade_trabalho": "São Paulo",
  "residencia": "Rua das Flores, 123",
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
    "cidade_trabalho": "São Paulo",
    "residencia": "Rua das Flores, 123",
    "foto": "https://example.com/foto.jpg"
  },
  {
    "id": 2,
    "nome": "Maria Santos",
    "cpf": "98765432100",
    "telefone": "11988888888",
    "data_nasc": "1988-03-20",
    "turno": "VT",
    "cidade_trabalho": "Rio de Janeiro",
    "residencia": "Avenida Principal, 456",
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
  "cidade_trabalho": "São Paulo",
  "residencia": "Rua das Flores, 123",
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
  "turno": "VT",
  "residencia": "Rua Nova, 789"
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
  "cidade_trabalho": "São Paulo",
  "residencia": "Rua Nova, 789",
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
  "cidade_base": "São Paulo",
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
  "cidade_base": "São Paulo",
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
    "cidade_base": "São Paulo",
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
    "cidade_base": "Rio de Janeiro",
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
  "cidade_base": "São Paulo",
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
  "cidade_base": "São Paulo",
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
  "cidade_base": "São Paulo",
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

## Pontos

### POST /pontos - Criar Ponto

Cria um novo ponto (ponto de parada ou origem/destino de viagem).

**Request:**

```http
POST /pontos HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "Terminal Rodoviário Central",
  "rua": "Avenida Paulista, 1000",
  "cidade": "São Paulo",
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "nome": "Terminal Rodoviário Central",
  "rua": "Avenida Paulista, 1000",
  "cidade": "São Paulo",
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

---

### GET /pontos - Listar Todos os Pontos

Lista todos os pontos cadastrados.

**Request:**

```http
GET /pontos HTTP/1.1
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
    "cidade": "São Paulo",
    "latitude": -23.5505,
    "longitude": -46.6333
  },
  {
    "id": 2,
    "nome": "Estação da Luz",
    "rua": "Avenida Tiradentes, 100",
    "cidade": "São Paulo",
    "latitude": -23.5407,
    "longitude": -46.6243
  }
]
```

---

### GET /pontos/cidade/{cidade} - Listar Pontos por Cidade

Lista todos os pontos de uma cidade específica.

**Request:**

```http
GET /pontos/cidade/São%20Paulo HTTP/1.1
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
    "cidade": "São Paulo",
    "latitude": -23.5505,
    "longitude": -46.6333
  },
  {
    "id": 2,
    "nome": "Estação da Luz",
    "rua": "Avenida Tiradentes, 100",
    "cidade": "São Paulo",
    "latitude": -23.5407,
    "longitude": -46.6243
  }
]
```

---

### GET /pontos/{id} - Obter Ponto por ID

Obtém informações de um ponto específico.

**Request:**

```http
GET /pontos/1 HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "Terminal Rodoviário Central",
  "rua": "Avenida Paulista, 1000",
  "cidade": "São Paulo",
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

---

### PUT /pontos/{id} - Atualizar Ponto

Atualiza as informações de um ponto.

**Request:**

```http
PUT /pontos/1 HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <token>

{
  "nome": "Terminal Rodoviário Central - Atualizado",
  "rua": "Avenida Paulista, 1500",
  "cidade": "São Paulo",
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
  "cidade": "São Paulo",
  "latitude": -23.5505,
  "longitude": -46.6333
}
```

---

### DELETE /pontos/{id} - Deletar Ponto

Remove um ponto do sistema.

**Request:**

```http
DELETE /pontos/1 HTTP/1.1
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
  "cidade": "Campinas"
}
```

**Response:** `201 Created`

```json
{
  "id": 1,
  "nome": "Rodoviária de Campinas",
  "latitude": -22.8978,
  "longitude": -47.0739,
  "cidade": "Campinas"
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
    "cidade": "Campinas"
  },
  {
    "id": 2,
    "nome": "Rodoviária de Jundiaí",
    "latitude": -23.1897,
    "longitude": -46.8707,
    "cidade": "Jundiaí"
  }
]
```

---

### GET /paradas/cidade/{cidade} - Listar Paradas por Cidade

Lista todas as paradas de uma cidade específica.

**Request:**

```http
GET /paradas/cidade/Campinas HTTP/1.1
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
    "cidade": "Campinas"
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
  "cidade": "Campinas"
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
  "cidade": "Campinas"
}
```

**Response:** `200 OK`

```json
{
  "id": 1,
  "nome": "Rodoviária de Campinas - Unidade Principal",
  "latitude": -22.8978,
  "longitude": -47.0739,
  "cidade": "Campinas"
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
  "cidade": "São Paulo",
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
  "cidade": "São Paulo",
  "paradas": [
    {
      "id": 1,
      "nome": "Rodoviária Central",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "cidade": "São Paulo",
      "ordem": 1
    },
    {
      "id": 2,
      "nome": "Estação da Luz",
      "latitude": -23.5407,
      "longitude": -46.6243,
      "cidade": "São Paulo",
      "ordem": 2
    },
    {
      "id": 3,
      "nome": "Shopping Imigrantes",
      "latitude": -23.5611,
      "longitude": -46.5841,
      "cidade": "São Paulo",
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
    "cidade": "São Paulo",
    "paradas": [
      {
        "id": 1,
        "nome": "Rodoviária Central",
        "latitude": -23.5505,
        "longitude": -46.6333,
        "cidade": "São Paulo",
        "ordem": 1
      },
      {
        "id": 2,
        "nome": "Estação da Luz",
        "latitude": -23.5407,
        "longitude": -46.6243,
        "cidade": "São Paulo",
        "ordem": 2
      }
    ]
  },
  {
    "id": 2,
    "cidade": "Campinas",
    "paradas": [
      {
        "id": 4,
        "nome": "Rodoviária de Campinas",
        "latitude": -22.8978,
        "longitude": -47.0739,
        "cidade": "Campinas",
        "ordem": 1
      }
    ]
  }
]
```

---

### GET /rotas-internas/cidade/{cidade} - Listar Rotas Internas por Cidade

Lista todas as rotas internas de uma cidade específica.

**Request:**

```http
GET /rotas-internas/cidade/São%20Paulo HTTP/1.1
Host: localhost:8080
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "cidade": "São Paulo",
    "paradas": [
      {
        "id": 1,
        "nome": "Rodoviária Central",
        "latitude": -23.5505,
        "longitude": -46.6333,
        "cidade": "São Paulo",
        "ordem": 1
      },
      {
        "id": 2,
        "nome": "Estação da Luz",
        "latitude": -23.5407,
        "longitude": -46.6243,
        "cidade": "São Paulo",
        "ordem": 2
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
  "cidade": "São Paulo",
  "paradas": [
    {
      "id": 1,
      "nome": "Rodoviária Central",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "cidade": "São Paulo",
      "ordem": 1
    },
    {
      "id": 2,
      "nome": "Estação da Luz",
      "latitude": -23.5407,
      "longitude": -46.6243,
      "cidade": "São Paulo",
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
  "cidade": "São Paulo",
  "paradas": [
    {
      "id": 1,
      "nome": "Rodoviária Central",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "cidade": "São Paulo",
      "ordem": 1
    },
    {
      "id": 3,
      "nome": "Shopping Imigrantes",
      "latitude": -23.5611,
      "longitude": -46.5841,
      "cidade": "São Paulo",
      "ordem": 2
    },
    {
      "id": 2,
      "nome": "Estação da Luz",
      "latitude": -23.5407,
      "longitude": -46.6243,
      "cidade": "São Paulo",
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

> **Status:** 🔄 Em desenvolvimento

Endpoints para gerenciar clientes da plataforma.

---

## Reservas

> **Status:** 🔄 Em desenvolvimento

Endpoints para gerenciar reservas de viagens.

---

## Viagens

> **Status:** 🔄 Em desenvolvimento

Endpoints para gerenciar viagens/trajetos.

---

## Headers Padrão

Todos os endpoints (exceto `/health` e `/admin/login`) requerem:

```http
Authorization: Bearer <seu_token_jwt>
Content-Type: application/json (para requisições com body)
```

## Status HTTP Padrão

- `200 OK` - Requisição bem-sucedida
- `201 Created` - Recurso criado com sucesso
- `204 No Content` - Operação bem-sucedida sem retorno de conteúdo
- `400 Bad Request` - Dados inválidos na requisição
- `401 Unauthorized` - Token ausente ou inválido
- `404 Not Found` - Recurso não encontrado
- `500 Internal Server Error` - Erro do servidor

---

**Última atualização:** 04/06/2026
