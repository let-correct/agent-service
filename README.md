# agent-service

This service has nothing to do with AI agents.

This service is the Agent domain — responsible for all agent-related actions.

## Workflows

### 1. Authenticating Agents using SSO (Google Workspace)

Agents authenticate to the service via Google Workspace SSO, brokered through AWS Cognito. On success, Cognito issues a JWT access token used to authorise all subsequent API calls.

```mermaid
sequenceDiagram
    actor Agent
    participant UI as UI
    participant Cognito as AWS Cognito
    participant Google as Google Workspace

    Agent->>UI: Click "Sign in with Google"
    UI->>Cognito: Initiate authorisation
    Cognito->>Google: Redirect to Google login
    Agent->>Google: Enter Workspace credentials
    Google-->>Cognito: Auth code (idpresponse)
    Cognito->>Google: Exchange code for ID token
    Google-->>Cognito: ID token (email, sub, ...)
    Cognito-->>Cognito: Create short-lived code and associate with Google token
    Cognito-->>UI: Redirect to callback_url with auth code
    UI->>Cognito: Exchange auth code for tokens
    Cognito-->>Cognito: Lookup token based on auth code
    Cognito-->>UI: Access token + refresh token
```

---

### 2. Authorising Access to Google Calendar

Once authenticated, an agent authorises the service to access Google Calendar on their behalf. The service stores the resulting OAuth tokens in DynamoDB, keyed by the agent's email (sourced from their Cognito JWT — not caller-supplied).

```mermaid
sequenceDiagram
    actor Agent as Agent via UI
    participant APIGW as API Gateway
    participant Lambda
    participant DynamoDB
    participant Google as Google OAuth

    Agent->>APIGW: Initiate provider authorisation
    APIGW->>APIGW: Validate JWT (401 if invalid)
    APIGW->>Lambda: Forward request
    Lambda->>DynamoDB: Save state token
    Lambda-->>Agent: Return Google Auth URL

    Agent->>Google: Visit auth URL, grant Calendar access
    Google-->>Agent: Redirect to callback with code + state

    Agent->>APIGW: Submit callback
    APIGW->>APIGW: Validate JWT (401 if invalid)
    APIGW->>Lambda: Forward request
    Lambda->>Lambda: Read email from JWT claims
    Lambda->>DynamoDB: Validate & consume state token
    Lambda->>Google: Exchange code for OAuth tokens
    Google-->>Lambda: Access token + refresh token
    Lambda->>DynamoDB: Save tokens (keyed by email)
    Lambda-->>Agent: 200 OK
```

---

## Lambdas

### Auth Lambda (`cmd/auth`)

Handles provider OAuth flows on behalf of authenticated agents. All routes require a valid Cognito JWT (`Authorization: Bearer <token>`).

#### `GET /auth/{provider}`

Initiates an OAuth authorisation flow for the given provider.

| | |
|---|---|
| **Auth** | `Authorization: Bearer <cognito_access_token>` |
| **Path param** | `provider` — the OAuth provider (e.g. `google`) |
| **Response 200** | `{ "url": "<provider_auth_url>" }` |
| **Response 400** | `{ "error": "unsupported provider" }` |
| **Response 401** | JWT missing, invalid, or expired (rejected by API Gateway before Lambda) |

#### `POST /auth/{provider}/callback`

Exchanges the provider auth code for OAuth tokens and persists them. The agent email is read from the 'claims' field in the JWT.

| | |
|---|---|
| **Auth** | `Authorization: Bearer <cognito_access_token>` |
| **Path param** | `provider` — the OAuth provider (e.g. `google`) |
| **Request body** | `{ "code": "<auth_code>", "state": "<state_token>" }` |
| **Response 200** | `{}` |
| **Response 400** | `{ "error": "unsupported provider" }` or `{ "error": "invalid or expired state" }` |
| **Response 401** | JWT missing, invalid, or expired (rejected by API Gateway before Lambda) |

---

## Infrastructure

- **AWS API Gateway v2 (HTTP API)** — routes requests, enforces JWT auth via Cognito authorizer
- **AWS Lambda** — handles `GET /auth/{provider}` and `POST /auth/{provider}/callback`
- **AWS Cognito** — agent identity, Google Workspace SSO federation
- **AWS DynamoDB** — stores OAuth state tokens and provider access/refresh tokens
- **Terraform** — all infrastructure managed as code, deployed via Terraform Cloud