#!/bin/bash

# deploy-staging-api.sh - Deploy the API to Aliyun Function Compute 3.0
# NOTE: Updates function code (binary) and environment variables.
# NOTE: This script does NOT run database migrations. Use deploy-staging-db.sh for that.

set -e
set -o pipefail

# Configuration
FUNC_NAME="wawa-api-go"
FC_REGION="ap-southeast-1"
# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_step() { echo -e "${BLUE}$1${NC}"; }
print_success() { echo -e "${GREEN}$1${NC}"; }
print_error() { echo -e "${RED}$1${NC}"; }

is_deploy_only_env_key() {
    case "$1" in
        ALIYUN_FC_ACCESS_KEY_ID|ALIYUN_FC_ACCESS_KEY_SECRET)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

check_prerequisites() {
    print_step "Checking prerequisites..."
    if ! command -v aliyun >/dev/null 2>&1; then print_error "Aliyun CLI not found"; exit 1; fi
    if ! command -v jq >/dev/null 2>&1; then print_error "jq not found"; exit 1; fi
    if ! command -v zip >/dev/null 2>&1; then print_error "zip not found"; exit 1; fi
    
    if [ -f .env ]; then
        print_step "Using .env for deployment config..."
        ENV_FILE=".env"
    else
        print_error "No .env file found! Please run 'make env target=cloud' first."
        exit 1
    fi
}

deploy() {
    # We no longer source the file because .env values might not be quoted for shell execution
    # instead we will parse it directly.

    # 1. Build Env Var JSON
    print_step "Parsing .env..."
    ENV_JSON="{}"
    SEEN_KEYS=""
    AK=""
    SK=""
    OSS_AK=""
    OSS_SK=""
    OSS_BUCKET=""
    OSS_ENDPOINT=""
    
    # Read file line by line
    while read -r line || [[ -n "$line" ]]; do
        # Skip comments and empty lines
        if [[ $line =~ ^#.* ]] || [[ -z $line ]]; then continue; fi
        
        # Split at the first '='
        key="${line%%=*}"
        value="${line#*=}"
        
        # Strip 'export ' prefix if present
        key=${key#"export "}
        
        # Trim leading/trailing whitespace
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)
        
        # Remove surrounding quotes from value if present
        value="${value%\"}"
        value="${value#\"}"
        value="${value%\'}"
        value="${value#\'}"

        if [ -z "$key" ]; then
            continue
        fi
        if echo "$value" | grep -Eq '\$\{[^}]+\}'; then
            print_error "Unresolved placeholder in .env for $key. Run 'make env target=staging' and check .env.staging."
            exit 1
        fi
        if printf '%s\n' "$SEEN_KEYS" | grep -Fxq "$key"; then
            print_error "Duplicate key in .env: $key"
            exit 1
        fi
        SEEN_KEYS="${SEEN_KEYS}${key}
"

        case "$key" in
            ALIYUN_FC_ACCESS_KEY_ID)
                AK="$value"
                ;;
            ALIYUN_FC_ACCESS_KEY_SECRET)
                SK="$value"
                ;;
            ALIYUN_OSS_ACCESS_KEY_ID)
                OSS_AK="$value"
                ENV_JSON=$(echo "$ENV_JSON" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
                ;;
            ALIYUN_OSS_ACCESS_KEY_SECRET)
                OSS_SK="$value"
                ENV_JSON=$(echo "$ENV_JSON" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
                ;;
            ALIYUN_OSS_BUCKET_NAME)
                OSS_BUCKET="$value"
                ENV_JSON=$(echo "$ENV_JSON" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
                ;;
            ALIYUN_OSS_ENDPOINT)
                OSS_ENDPOINT="$value"
                ;;
        esac

        if ! is_deploy_only_env_key "$key"; then
            ENV_JSON=$(echo "$ENV_JSON" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
        fi
    done < "$ENV_FILE"

    # 1.1 Inject Dynamic Versioning
    CURRENT_VERSION="v$(date +'%Y%m%d-%H%M%S')"
    print_step "Injecting Dynamic Version: $CURRENT_VERSION"
    
    ENV_JSON=$(echo "$ENV_JSON" | jq --arg v "$CURRENT_VERSION" '. + {"VERSION": $v}')

    # Update environment variables only.
    if [ -z "$AK" ] || [ -z "$SK" ]; then
        print_error "Missing Aliyun FC access keys in .env. Set ALIYUN_FC_ACCESS_KEY_ID and ALIYUN_FC_ACCESS_KEY_SECRET."
        exit 1
    fi
    if [ -z "$OSS_AK" ] || [ -z "$OSS_SK" ] || [ -z "$OSS_BUCKET" ] || [ -z "$OSS_ENDPOINT" ]; then
        print_error "Missing OSS settings in .env. Set ALIYUN_OSS_ACCESS_KEY_ID, ALIYUN_OSS_ACCESS_KEY_SECRET, ALIYUN_OSS_BUCKET_NAME, ALIYUN_OSS_ENDPOINT."
        exit 1
    fi

    print_step "Building Go binary..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o main cmd/api/main.go

    print_step "Zipping assets (binary, www)..."
    zip -q -r main.zip main www/

    OSS_OBJECT="fc-deploy/${FUNC_NAME}/${CURRENT_VERSION}/main.zip"
    print_step "Uploading to OSS (${OSS_BUCKET}/${OSS_OBJECT})..."
    aliyun oss cp main.zip "oss://${OSS_BUCKET}/${OSS_OBJECT}" \
        --force \
        --mode AK \
        --access-key-id "$OSS_AK" \
        --access-key-secret "$OSS_SK" \
        --endpoint "$OSS_ENDPOINT" >/dev/null

    rm -f main main.zip

    UPDATE_BODY=$(jq -n \
        --argjson env "$ENV_JSON" \
        --arg bucket "$OSS_BUCKET" \
        --arg object "$OSS_OBJECT" \
        '{environmentVariables: $env, code: {ossBucketName: $bucket, ossObjectName: $object}}')

    print_step "Updating function code and environment variables..."
    ALIBABA_CLOUD_ACCESS_KEY_ID="$AK" \
    ALIBABA_CLOUD_ACCESS_KEY_SECRET="$SK" \
    aliyun fc UpdateFunction --region "$FC_REGION" --functionName "$FUNC_NAME" --body "$UPDATE_BODY" >/dev/null

    print_step "Verifying deployed environment variables..."
    REMOTE_ENV_JSON=$(ALIBABA_CLOUD_ACCESS_KEY_ID="$AK" \
        ALIBABA_CLOUD_ACCESS_KEY_SECRET="$SK" \
        aliyun fc GetFunction --region "$FC_REGION" --functionName "$FUNC_NAME" |
        jq -c '.environmentVariables // .body.environmentVariables // .data.environmentVariables // {}')
    EXPECTED_ENV_SORTED=$(echo "$ENV_JSON" | jq -S -c '.')
    REMOTE_ENV_SORTED=$(echo "$REMOTE_ENV_JSON" | jq -S -c '.')
    if [ "$EXPECTED_ENV_SORTED" != "$REMOTE_ENV_SORTED" ]; then
        print_error "Deployed environment variables do not match .env runtime variables."
        diff <(echo "$EXPECTED_ENV_SORTED" | jq -S '.') <(echo "$REMOTE_ENV_SORTED" | jq -S '.') || true
        exit 1
    fi

    rm -f main main.zip

    print_success "Function code and environment variables updated for ${FUNC_NAME}. Version: ${CURRENT_VERSION}"
}

check_prerequisites
deploy
