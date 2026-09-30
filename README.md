# WAWAGO

WAWAGO is a Go backend for managing WhatsApp Business communication, customer data, broadcasts, commerce content, and business-agent features. It exposes a Gin HTTP API, persists application state in PostgreSQL, and uses Aliyun Function Compute workers to process asynchronous WhatsApp and broadcast jobs.

The project is designed for a serverless deployment model, but the API, worker, dispatcher, and command-line tools can also be run locally. All application processes share the same configuration, database repositories, DTOs, feature handlers, service clients, and structured response format.

## Key Functionality

### Account and access management

- User login, logout, profile and password management.
- Account sessions with expiry and revocation support.
- Account notifications, assigned phone numbers, and account closure.
- Authenticated access to user-owned WhatsApp, customer, broadcast, and commerce resources.
- Public website lookup and public-facing website routes that do not require a user session.

### WhatsApp Business operations

- Meta embedded signup for WhatsApp Business portfolios, accounts, and phone numbers.
- Phone-number connection, reconnection, disconnection, deletion, business profile, and usage operations.
- Sending, listing, retrieving, retrying, and marking WhatsApp messages as read.
- Typing indicators and real-time notification token creation.
- WhatsApp webhook verification and inbound webhook processing.
- Template storage, sample templates, template creation, deletion, media uploads, and analytics.
- Product catalogs, product sets, products, and catalog usage data.
- WhatsApp Business Agent onboarding, eligibility, budgets, business information, FAQs, files, websites, skills, UI skills, settings, testing, and control transfer.

### Customers and broadcasts

- Customer records and customer phone/WhatsApp identifiers.
- Broadcast creation and recipient management.
- Scheduled and retryable message delivery through the asynchronous queue pipeline.
- Broadcast status, recipient status, delivery history, and error tracking.

### Commerce and public websites

- Commerce catalogs and website content.
- Website pages with create, read, update, delete, status, and synchronization operations.
- Public website resolution by domain and WhatsApp contact links.
- Markdown, HTML, PDF, spreadsheet, and media-related helpers used by content and export workflows.

## Architecture

The repository is organized around four executable processes:

| Process | Entry point | Responsibility |
| --- | --- | --- |
| API | `cmd/api/main.go` | Starts the Gin HTTP server, database connection, middleware, controllers, and API routes. |
| Worker | `cmd/worker/main.go` | Receives Aliyun SMQ messages and processes inbound WhatsApp events, message retries, and broadcasts. |
| Dispatcher | `cmd/dispatcher/main.go` | Runs on a schedule, finds pending broadcasts and messages, and publishes worker jobs. |
| CLI | `cmd/cli/main.go` | Runs migration-file, database-restore, and encryption-backfill tasks. |

The API initializes a `UnitOfWork` backed by GORM and PostgreSQL. Controllers bind and validate request DTOs, feature handlers implement business rules, repositories read and write DAOs, and services encapsulate external APIs and infrastructure clients. The dependency container in `internal/service/dependencies.go` wires the repository, logging, file, queue, real-time, WhatsApp, Facebook, cache, and Cloudflare services.

The API uses Gin middleware for request IDs, CORS, access logging, recovery outside development, request binding, and authentication. The API listens on port `9000` by default, supports graceful shutdown, and uses UTC for application and database operations.

## Data Handling and Persistence

PostgreSQL is the system of record. GORM provides the database driver and query layer, while repository implementations in `internal/repository/gormdb` expose domain-specific interfaces through the Unit of Work. The initial schema is divided into:

- `account`: users, sessions, settings, notifications, and cache data.
- `customer`: customers, broadcasts, and broadcast recipients.
- `wa`: business portfolios, business accounts, phone numbers, messages, message status, templates, and related WhatsApp data.
- `commerce`: catalogs, websites, and pages.
- `site`: feedback and site-level data.

The schema uses PostgreSQL extensions including PostGIS, pgvector, and `citext`, plus JSONB columns for structured payloads and template data. Foreign keys, unique indexes, status fields, timestamps, and transaction boundaries are used to preserve relationships and workflow state.

Sensitive values are converted between database representations and DTOs in repository code. Searchable sensitive fields use a keyed hash column, while the original value is stored in an encrypted column. WhatsApp payloads and access tokens are encrypted before persistence and decrypted only when a feature needs the plaintext value.

## Encryption and Secret Protection

WAWAGO uses separate mechanisms for passwords, searchable secrets, and recoverable secrets:

- Passwords are hashed with Argon2id using a per-password random salt and constant-time verification.
- Recoverable secrets use AES-256-GCM with a random nonce. The nonce is stored with the ciphertext in a versioned JSON envelope.
- Encryption and HMAC keys are derived once at startup with HKDF-SHA256 from configured master-key material and salt.
- Additional authenticated data binds ciphertext to its purpose and record context, such as a specific user or business portfolio. Decryption fails if the context or ciphertext is changed.
- Searchable values use HMAC-SHA256 with a separate derived HMAC key. The application queries the hash column and decrypts the encrypted column only after locating the record.
- Encryption key versions are tracked in the encrypted payload. The key ring can retain historical versions during rotation while new data uses the current version.

Set encryption material through environment variables such as `ENCRYPTION_CURRENT_VERSION`, `ENCRYPTION_KEY_VERSIONS`, `ENCRYPTION_MASTER_KEY_V1`, and `ENCRYPTION_SALT_V1`. Production key material must be managed through an appropriate secret or key-management process and must never be committed to the repository.

## Authentication and Authorization

Authenticated API requests provide a user access token in the `x-user-access-token` header. The authentication middleware:

1. Hashes the presented token with SHA-256 for lookup.
2. Checks the short-lived in-memory authentication cache.
3. Falls back to the account-session repository when needed.
4. Rejects missing, expired, revoked, inactive, or closed sessions.
5. Updates session last-used time and loads the authenticated user and related WhatsApp resources into the request context.

The cache has a maximum lifetime of 15 seconds and never outlives the session itself. Session and user invalidation removes matching cache entries.

Authorization is enforced by feature handlers and repositories. Handlers verify the authenticated user, resource ownership, account state, and operation-specific permissions before changing data. Unauthorized requests return `401`; authenticated users without permission return `403`. Public routes are explicitly marked in request API settings and are handled without requiring an authenticated user.

WhatsApp webhook requests use Meta's verification token for setup and HMAC-SHA256 signature verification for received request bodies. Login also supports Cloudflare Turnstile verification.

## Error Handling and Observability

All normal API responses use the generic response envelope in `internal/dto/response.go`:

```json
{
	"success": false,
	"status_code": 400,
	"message": "invalid input",
	"request_id": "...",
	"time_taken": 12,
	"input_errors": []
}
```

Errors are represented with safe public messages and HTTP status codes. Validation failures include field-level input errors. Common responses include `400` for invalid input, `401` for authentication failures, `403` for authorization failures, `404` for missing resources, `405` for unsupported methods, `429` for rate limits, and `500` for internal failures. Internal errors are retained for logging rather than exposed in the public message.

Each request receives a correlation ID. Structured access and error logs record the request ID, route, method, status, duration, user ID, and remote address. Request payload details are included only in development logging. The API exposes `/health`; pprof is enabled only in development, and static API assets are served only outside production.

The worker deliberately acknowledges its HTTP invocation with `200` after receiving a queue delivery. Processing failures are logged and the SMQ message is left undeleted so Aliyun SMQ can retry it. Successful jobs delete their receipt handle. The dispatcher is also retry-friendly: it logs queue publication failures and leaves pending database records for the next scheduled cycle.

## OpenAPI Documentation

The API description is generated from the same request objects used to register routes. Each request object provides `APISettings`, including its method, path, summary, description, authentication requirement, tags, parameters, body content type, and declared errors.

`internal/feature/api_generator.go` uses `kin-openapi` to generate an OpenAPI 3.0.3 document. It derives path and query parameters from request tags, generates request and response schemas with reflection, includes standard error responses, and declares the `x-user-access-token` API-key security scheme. In development, controller registration writes the generated document to [`www/api-doc.json`](www/api-doc.json). The generated document is also included in the API deployment package.

## External Services

The service adapters in `internal/service` integrate with:

- **Meta/Facebook Graph APIs** for WhatsApp Business accounts, phone numbers, messages, templates, catalogs, analytics, webhooks, and Business Agent operations.
- **Aliyun Function Compute** for the deployed API, worker, and scheduled dispatcher runtimes.
- **Aliyun Simple Message Queue (SMQ)** for durable asynchronous job delivery and retry behavior.
- **Aliyun Object Storage Service (OSS)** for permanent files and short-lived signed download URLs.
- **Ably** for real-time notification and message-channel token creation.
- **Cloudflare Turnstile** for login bot protection.
- **Google Maps** configuration for map-related client features through API keys and map IDs.

External credentials and endpoints are supplied through environment variables. The application does not hardcode access tokens or service secrets in source code.

## Database Migration

Migration files live in [`migration/`](migration/). The initial schema is in `000001_schema.up.sql` and its rollback is in `000001_schema.down.sql`. Migrations are applied with the `golang-migrate` CLI and use a dedicated migration database configuration when needed.

The staging migration script:

- Reads migration connection settings from `.env.staging` or `.env`.
- Requires the `migrate` and `psql` command-line tools.
- Applies `file://migration` migrations with a connection timeout and lock timeout.
- Grants the configured application role access to application schemas, tables, sequences, and future objects.

Useful database commands are:

```sh
make migration-files       # Generate migration files from the CLI workflow
make restore-db             # Restore the local database from migration files
make deploy-staging-db      # Apply migrations to the configured staging database
```

## Local Development

### Prerequisites

- Go `1.25.4` or a compatible newer Go toolchain.
- PostgreSQL with the extensions required by the migration.
- A populated environment source file such as `.env.local`.
- Credentials for any external services needed by the feature being exercised.

Generate the working `.env` file from an environment source:

```sh
make env
make env target=staging
```

Run the API, worker, or CLI locally:

```sh
make run-api
make run-worker
make run-cli
```

The API listens on `http://localhost:9000` by default. Check `GET /health` for the running version and environment. In development, the generated API document is available at `www/api-doc.json` and the Gin pprof routes are registered.

Run the repository checks with:

```sh
make test
```

This runs `go vet ./...` and the controller package tests. Focused package tests can be run directly with `go test ./...`.

## Deployment

Staging deployments target Aliyun Function Compute in `ap-southeast-1`. Each deployment script builds a static Linux AMD64 binary, packages it as a ZIP archive, uploads the archive to Aliyun OSS, updates the Function Compute function, injects environment variables, and verifies that the deployed environment matches the generated configuration.

Deploy the individual functions with:

```sh
make deploy-staging-api
make deploy-staging-worker
make deploy-staging-dispatcher
```

Database migration is intentionally separate from code deployment:

```sh
make deploy-staging-db
```

The scripts require the Aliyun CLI, `jq`, and `zip`; the database deployment additionally requires `migrate` and `psql`. Deployment credentials are used by the scripts but are excluded from the runtime environment where appropriate. The scripts also inject a timestamped `VERSION` value into the deployed function configuration.

## Repository Layout

```text
cmd/api/          HTTP API entry point
cmd/worker/       SMQ-triggered asynchronous worker
cmd/dispatcher/   Scheduled job dispatcher
cmd/cli/          Operational and migration CLI
internal/cfg/     Environment-backed configuration
internal/controller/ HTTP route registration
internal/feature/ Business use cases and request objects
internal/middleware/ HTTP middleware and authentication
internal/repository/ Repository interfaces and GORM implementations
internal/service/ External services, logging, caching, and dependencies
internal/dto/     API request and response data-transfer objects
internal/dao/     Database access objects
internal/helper/  Crypto, validation, formatting, and shared helpers
migration/        PostgreSQL schema migrations
scripts/          Staging deployment and migration scripts
docs/             Feature-specific API payload documentation
www/              Generated API document and development web assets
```

