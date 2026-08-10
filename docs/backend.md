# BondRota API Documentation

**Base URL:** `/api/v1`

## Authentication

Endpoints que requerem autenticação devem enviar o token JWT no cabeçalho da requisição:

```http
Authorization: Bearer <seu_token_aqui>
```

## Endpoints disponíveis

Nota: a aplicação atualmente registra as rotas via `server.RegisterRoutes` em `internal/server/server.go`. As rotas expostas são listadas abaixo.

- **GET** `/health`
  - Descrição: verifica saúde da API. Retorna `200 OK` quando o servidor está operacional.

---

**Admin** (prefixo `/admin`)

- **POST** `/admin`
  - Descrição: cria um novo administrador.
  - Request JSON:
    ```json
    {
      "email": "string",
      "senha": "string"
    }
    ```
  - Response (201 Created):
    ```json
    { "id": 123 }
    ```

- **GET** `/admin`
  - Descrição: lista administradores.
  - Response (200 OK): array de administradores:
    ```json
    [{ "id": 1, "email": "a@b.com" }]
    ```

- **GET** `/admin/{adminID}`
  - Descrição: obtém administrador por ID.
  - Response (200 OK):
    ```json
    [{ "id": 1, "email": "a@b.com" }]
    ```

- **PUT** `/admin/{adminID}`
  - Descrição: atualiza campos do administrador (parciais permitidas).
  - Request JSON:
    ```json
    {
      "email": "novo@email.com"
    }
    ```
  - Response (200 OK): objeto atualizado do administrador.

- **DELETE** `/admin/{adminID}`
  - Descrição: remove o administrador indicado.
  - Response: `204 No Content` em caso de sucesso.

- **POST** `/admin/login`
  - Descrição: autentica um administrador e retorna um JWT.
  - Request JSON:
    ```json
    { "email": "a@b.com", "senha": "senha" }
    ```
  - Response (200 OK):
    ```json
    { "token": "<jwt_token>" }
    ```

## Observações

- As rotas acima são montadas em `/api/v1` pelo `cmd/main.go`, portanto o caminho final é, por exemplo, `/api/v1/admin`.
- Outras pastas do projeto (`clientes`, `veiculos`, `viagens`, etc.) possuem handlers e serviços, mas atualmente não estão registrados no `server.RegisterRoutes` — por isso não estão expostas pela API.
- Variáveis de ambiente relevantes: `DATABASE_URL`, `JWT_SECRET`, `PLANNING_CRON_SECRET`, `BASE_CITY`, `APP_TIMEZONE`, `PORT` e `ALLOWED_ORIGINS`.

Se quiser, posso também registrar as rotas restantes (clientes, motoristas, veículos, viagens, destinos, reservas, rotasinternas) no servidor e documentá-las automaticamente.
