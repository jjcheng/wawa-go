#!/bin/bash

# migrate-staging-db.sh - Run staging migrations against Cloud DB
# WARNING: This applies 'migration' schemas/data to your configured Cloud DB.
# Use caution if this overwrites production data.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_step() { echo -e "${BLUE}$1${NC}"; }
print_success() { echo -e "${GREEN}$1${NC}"; }
print_error() { echo -e "${RED}$1${NC}"; }
print_warning() { echo -e "${YELLOW}$1${NC}"; }

check_prerequisites() {
    if ! command -v migrate >/dev/null 2>&1; then print_error "migrate tool not found"; exit 1; fi
    
    if [ -f .env.staging ]; then
        print_step "Using .env.staging..."
        source .env.staging
    elif [ -f .env ]; then
        print_step "Using .env (Local)..."
        source .env
    else
        print_error "No .env.staging or .env file found!"
        exit 1
    fi
}

run() {
    print_step "Connecting to Cloud Database..."

    MIGRATION_DB_HOST="${DB_HOST_EXTERNAL:-}"
    MIGRATION_DB_USER="${DB_USER_EXTERNAL:-}"
    MIGRATION_DB_PASSWORD="${DB_PASSWORD_EXTERNAL:-}"
    MIGRATION_DB_NAME="${DB_NAME:-}"
    MIGRATION_DB_PORT="${DB_PORT:-5432}"
    MIGRATION_DB_SSLMODE="${DB_SSLMODE:-disable}"

    if [ -z "$MIGRATION_DB_USER" ] || [ -z "$MIGRATION_DB_PASSWORD" ] || [ -z "$MIGRATION_DB_HOST" ] || [ -z "$MIGRATION_DB_NAME" ]; then
        print_error "Missing migration database settings. Expected DB_HOST_EXTERNAL, DB_USER_EXTERNAL, DB_PASSWORD_EXTERNAL, DB_NAME, and DB_PORT."
        exit 1
    fi

    encoded_user=$(python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$MIGRATION_DB_USER")
    encoded_password=$(python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$MIGRATION_DB_PASSWORD")

    DSN="postgres://${encoded_user}:${encoded_password}@${MIGRATION_DB_HOST}:${MIGRATION_DB_PORT}/${MIGRATION_DB_NAME}?sslmode=${MIGRATION_DB_SSLMODE}&connect_timeout=10&options=-c%20lock_timeout%3D2s"
    
    print_warning "Source: file://migration"
    print_warning "Target: ${MIGRATION_DB_HOST} / ${MIGRATION_DB_NAME}"
    
    # Run migration
    migrate -source file://migration -database "$DSN" up
    
    print_success "Cloud Database Successfully Migrated!"
}

check_prerequisites
run
