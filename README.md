# WAWAGO

WAWAGO is an open source WhatsApp based messaging system. The name comes from _WhatsApp Web App written in GO_. The purpose of this project is to help a few friends who runs very small businesses to better manage their customer related communication. At the same time, I would like to see how AI coding can really create a production grade application.

## Background

Today's SAAS solutions are getting fancier and facier, most of the micro-small companies, sole priprtors who do not have enough manpower or time, find them overly complicated, too expensive and full of irelevant features. As a result they keep their traditional ways of conducting business. However, as AI coding gets 普及的, software development and maintenance cost has come down drastically. I argue that today's SAAS solutions should be cheap to free, offering the 20% essential functionalities, thus enticing an expanded group of potential users who previously relucant to onboard. While running the software may not be profitable, bulding and solving problems to this new group by business is potentially extremely valuable.

WAWAGO is such a simple WhatsApp messaging system which only covers the most essential features such a system should have. I spent slightly over 1 month time building it. The front end of this project WAWAWEB is written in react+next.js, it is 100% AI coded (I have no piror knowledge of react projects). The back end of this project WAWAGO is about 30% assisted by AI.

## WhatsApp based CRM

WhatsApp can be a very good CRM. To sign up as a member, simply link to the desinated WhatsApp contact via QR code, no app to download, no website to sign up, no additional password to remember. Best of all, all future communciation messages can reach the customer directly in WhatsApp which is always better than email or SMS.

## Key Functionalities

- Account management
    1. Onboard with WhatsApp Embedded Signup
    2. Login with password
    3. Manage users, update profile, change password, account clousure
    4. MASTER users with full priviledges, OPERATOR users can only use their assigned phone numbers
- Customer management
    1. Manually creating customers
    2. Import customers from .vcf file (exportable from iPhone/Android)
    3. Automatcially create new customer from incoming messages
- Messaging
    1. Create WhatsApp message templates (covers most of the template types, except flows)
    2. Send broadcast template messages immediately or at a later date
    3. Interact with customers in chat interface
    4. 1 WhatsApp number can be managed by multiple users
- Marketing
    1. Build custom websites using catalogs (created in Meta Commerce Manager)
- Automation
    1. Automate chats using Meta AI (in progress)
    2. AI Worker to manage the system on behalf of the user (in progress)
- Analytics
    1. View usages of phone number and templates

## Artechiture

WAWAGO is primarily running on Alibaba Cloud. 

- API (cmd/api/main.go) runs on Alibaba Function Compute web function
- Dispatcher (cmd/dispatcher/main.go) runs on Alibaba Function Compute task function, it is invoked every 1 minute by a time trigger. The job is to find out any unprocessed scheduled broadcasts, any failed to send messages as well as cleaning up caches. Any occurances are not processed directly, but sent to the Worker function via Alibaba Simple Message Queue (SMQ)
- Worker (cmd/worker/main.go) runs on Alibaba Function Compute task function, it is invoked by SMQ to send broadcasts, retry sening error message and store any incoming WhatsApp messages from the webhook.
- Message queue is using Alibaba Simple Message Queue (SMQ) as mentioned earlier
- Database is running on Aliabab RDS Serverless for Postgres, structure is in migration folder
- Object storage is on Alibaba OSS
- Real time messages are pushed through Ably
- Caching is done using Postgres unlogged table instead of Redis, to save cost
- Logging is through built-in golang slog

The reason to invoke the Worker always by SMQ is for fail-safe auto retry and throtling purposes.

## Encryption and Secret Protection

WAWAGO uses separate mechanisms for passwords, searchable secrets, and recoverable secrets:

- Passwords are hashed with Argon2id using a per-password random salt and constant-time verification.
- Recoverable secrets use AES-256-GCM with a random nonce. The nonce is stored with the ciphertext in a versioned JSON envelope.
- Encryption and HMAC keys are derived once at startup with HKDF-SHA256 from configured master-key material and salt.
- Additional authenticated data binds ciphertext to its purpose and record context, such as a specific user or business portfolio. Decryption fails if the context or ciphertext is changed.
- Searchable values use HMAC-SHA256 with a separate derived HMAC key. The application queries the hash column and decrypts the encrypted column only after locating the record.
- Encryption key versions are tracked in the encrypted payload. The key ring can retain historical versions during rotation while new data uses the current version.

Set encryption material through environment variables such as `ENCRYPTION_CURRENT_VERSION`, `ENCRYPTION_KEY_VERSIONS`, `ENCRYPTION_MASTER_KEY_V1`, and `ENCRYPTION_SALT_V1`. 

No plain text for any personal data is stored in db.

## OpenAPI Documentation

The API description is generated from the same request objects used to register routes. Each request object provides `APISettings`, including its method, path, summary, description, authentication requirement, tags, parameters, body content type, and declared errors. `www/api-doc.json` is updated everytime API is built locally.

## Database Migration

Migration files live in [`migration/`](migration/). The initial schema is in `000001_schema.up.sql` and its rollback is in `000001_schema.down.sql`. Migrations are applied with the `golang-migrate` CLI and use a dedicated migration database configuration when needed.

Useful database commands are:

```sh
make migration-files       # Generate migration files from the CLI workflow
make restore-db             # Restore the local database from migration files
make deploy-staging-db      # Apply migrations to the configured staging database
```

## Local Development

### Prerequisites

- Go `1.25.4` or newer.
- PostgreSQL with the extensions required by the migration.
- A populated environment source file `.env.local`.
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

Staging deployments target Aliyun Function Compute in `ap-southeast-1`. Please make sure you have the necessary services setup already in Alibaba Cloud and Ably.

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

There is a separate wawa-web project for the front end project.