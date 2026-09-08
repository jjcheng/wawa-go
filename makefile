target ?= local
ENV_SOURCE := .env.$(target)

env:
	@set -a; source $(ENV_SOURCE); set +a; \
	cp .env.template .env; \
	sed -i '' "s|\$${GIN_MODE}|$${GIN_MODE//&/\\&}|g" .env; \
	sed -i '' "s|\$${DB_USER}|$${DB_USER//&/\\&}|g" .env; \
	sed -i '' "s|\$${DB_PASSWORD}|$${DB_PASSWORD//&/\\&}|g" .env; \
	sed -i '' "s|\$${DB_HOST}|$${DB_HOST//&/\\&}|g" .env; \
	sed -i '' "s|\$${DB_PORT}|$${DB_PORT//&/\\&}|g" .env; \
	sed -i '' "s|\$${DB_NAME}|$${DB_NAME//&/\\&}|g" .env; \
	sed -i '' "s|\$${DB_SSLMODE}|$${DB_SSLMODE//&/\\&}|g" .env; \
	sed -i '' "s|\$${ENVIRONMENT}|$${ENVIRONMENT//&/\\&}|g" .env; \
	sed -i '' "s|\$${GOOGLE_MAP_APIKEY}|$${GOOGLE_MAP_APIKEY//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_OSS_ACCESS_KEY_ID}|$${ALIYUN_OSS_ACCESS_KEY_ID//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_OSS_ACCESS_KEY_SECRET}|$${ALIYUN_OSS_ACCESS_KEY_SECRET//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_SMQ_ENDPOINT}|$${ALIYUN_SMQ_ENDPOINT//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_SMQ_ACCESS_KEY_ID}|$${ALIYUN_SMQ_ACCESS_KEY_ID//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_SMQ_ACCESS_KEY_SECRET}|$${ALIYUN_SMQ_ACCESS_KEY_SECRET//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_SMQ_QUEUE_NAME}|$${ALIYUN_SMQ_QUEUE_NAME//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_SMQ_LISTENING}|$${ALIYUN_SMQ_LISTENING//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_FC_ACCESS_KEY_ID}|$${ALIYUN_FC_ACCESS_KEY_ID//&/\\&}|g" .env; \
	sed -i '' "s|\$${ALIYUN_FC_ACCESS_KEY_SECRET}|$${ALIYUN_FC_ACCESS_KEY_SECRET//&/\\&}|g" .env; \
	sed -i '' "s|\$${WHATSAPP_WEBHOOK_VERIFY_TOKEN}|$${WHATSAPP_WEBHOOK_VERIFY_TOKEN//&/\\&}|g" .env; \
	sed -i '' "s|\$${META_APP_ID}|$${META_APP_ID//&/\\&}|g" .env; \
	sed -i '' "s|\$${META_APP_SECRET}|$${META_APP_SECRET//&/\\&}|g" .env; \
	sed -i '' "s|\$${ABLY_API_KEY}|$${ABLY_API_KEY//&/\\&}|g" .env;
	@echo ".env generated from $(ENV_SOURCE)"
run-api:
	@make env
	go run cmd/api/main.go
run-worker:
	@make env
	go run cmd/worker/main.go
run-cli:
	@make env
	go run cmd/cli/main.go
migration-files:
	@make env
	go run cmd/cli/main.go migration-files
build-api:
	rm -f main
	GOOS=linux GOARCH=amd64 go build cmd/api/main.go
	zip dist/main.zip main
	rm -f main
build-cli:
	rm -f main
	GOOS=linux GOARCH=amd64 go build cmd/cli/main.go
	zip dist/main.zip main 
	rm -f main
test:
	@make env
	@echo "Running vet..."
	@go vet ./...
	@echo "Running tests..."
	go test ./internal/controller/...
restore-db:
	@make env
	go run cmd/cli/main.go restore-db
deploy-staging-fc:
	@make env target=staging
	@echo "Running staging deployment to Aliyun FC..."
	@chmod +x deploy_staging_fc.sh
	@./scripts/deploy_staging_fc.sh
deploy-staging-db:
	@echo "Running staging database migration..."
	@chmod +x deploy_staging_db.sh
	@./scripts/deploy_staging_db.sh
init-debugging:
	@make env
local-ssl:
	ngrok http 9000