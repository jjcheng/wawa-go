#!/bin/bash

# migrate-staging-db.sh - Run staging migrations against Cloud DB
# WARNING: This applies 'migration-staging' schemas/data to your configured Cloud DB.
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
    # Configured DSN with safe timeouts
    # use root user to migrate
    DSN="postgres://paix_root:KeWXPs8A6zDt8K@@pgm-t4nr29cnn5b1e56bbo.rwlb.singapore.rds.aliyuncs.com:5432/ai?sslmode=disable&connect_timeout=10&options=-c%20lock_timeout%3D2s"
    
    print_warning "Source: file://migration-staging"
    print_warning "Target: pgm-t4nr29cnn5b1e56bbo.rwlb.singapore.rds.aliyuncs.com / ai"
    
    # Run migration
    migrate -source file://migration-staging -database "$DSN" up
    
    print_success "Cloud Database updated with Staging Migrations!"
}

check_prerequisites
run
