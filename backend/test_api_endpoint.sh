#!/bin/bash

# Base URL
BASE_URL="http://localhost:8080"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Function to print test description
print_test() {
    echo -e "\n${GREEN}$1${NC}"
}

print_section "Testing Database Configuration API"

print_section "Testing Database Configuration API"

# Test PostgreSQL configuration
print_test "Creating PostgreSQL configuration..."
curl -X POST $BASE_URL/databases \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-postgres",
    "type": "postgresql",
    "host": "localhost",
    "port": "5432",
    "db_name": "myapp_db",
    "username": "postgres",
    "password": "secret123",
    "options": {
      "ssl_mode": "disable",
      "schema": "public"
    }
  }'

# Test MongoDB configuration
print_test "Creating MongoDB configuration..."
curl -X POST $BASE_URL/databases \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-mongo",
    "type": "mongodb",
    "host": "localhost",
    "port": "27017",
    "db_name": "myapp_db",
    "username": "mongouser",
    "password": "secret123",
    "options": {
      "auth_db": "admin",
      "replica_set": "rs0",
      "write_concern": "majority"
    }
  }'

# Test MySQL configuration
print_test "Creating MySQL configuration..."
curl -X POST $BASE_URL/databases \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-mysql",
    "type": "mysql",
    "host": "localhost",
    "port": "3306",
    "db_name": "myapp_db",
    "username": "mysqluser",
    "password": "secret123",
    "options": {
      "charset": "utf8mb4",
      "collation": "utf8mb4_unicode_ci"
    }
  }'

# Test invalid database type
print_test "Testing invalid database type..."
curl -X POST $BASE_URL/databases \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-db",
    "type": "invalid",
    "host": "localhost",
    "port": "5432",
    "db_name": "myapp_db",
    "username": "user",
    "password": "pass"
  }'

# Test 2: List all database configurations
print_test "Listing all database configurations..."
curl -X GET $BASE_URL/databases

# Test 3: Get specific database configuration
print_test "Getting specific database configuration..."
curl -X GET $BASE_URL/databases/my-postgres

# Test 4: Delete the database configuration
print_test "Deleting database configuration..."
curl -X DELETE $BASE_URL/databases/my-postgres

# Test 5: Verify database deletion by trying to get it (should fail)
print_test "Verifying database deletion (should return 404)..."
curl -X GET $BASE_URL/databases/my-postgres

# Test 6: Try to delete non-existent database configuration
print_test "Testing delete non-existent database configuration (should fail)..."
curl -X DELETE $BASE_URL/databases/non-existent-db

print_section "Testing LLM Configuration API"

# Test 7: Create new OpenAI configuration
# Test OpenAI configuration
print_test "Creating OpenAI configuration..."
curl -X POST $BASE_URL/llm \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gpt4-config",
    "type": "openai",
    "api_key": "sk-test-key-123",
    "model": "gpt-4",
    "options": {
      "organization": "org-123",
      "max_tokens": 4000
    }
  }'

# Test Anthropic configuration
print_test "Creating Anthropic configuration..."
curl -X POST $BASE_URL/llm \
  -H "Content-Type: application/json" \
  -d '{
    "name": "claude-config",
    "type": "anthropic",
    "api_key": "sk-ant-test123",
    "model": "claude-3-opus",
    "options": {
      "max_tokens_to_sample": 2000,
      "temperature": 0.7,
      "top_k": 5
    }
  }'

# Test invalid LLM type
print_test "Testing invalid LLM type..."
curl -X POST $BASE_URL/llm \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-config",
    "type": "invalid",
    "api_key": "test-key",
    "model": "test-model"
  }'

# Test list all configurations
print_test "Listing all LLM configurations..."
curl -X GET $BASE_URL/llm

# Test getting specific configuration
print_test "Getting specific LLM configuration..."
curl -X GET $BASE_URL/llm/openai/gpt4-config

# Test deleting configuration
print_test "Deleting LLM configuration..."
curl -X DELETE $BASE_URL/llm/openai/gpt4-config

print_section "All tests completed!"