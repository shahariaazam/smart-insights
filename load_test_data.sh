curl -X POST http://localhost:8080/databases/postgresql \
  -H "Content-Type: application/json" \
  -d '{
    "name": "sales_db",
    "host": "127.0.0.1",
    "port": "5432",
    "db_name": "app",
    "username": "postgres",
    "password": "pass"
  }'


curl -X POST http://localhost:8080/llm/openai \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI - GPT 4",
    "api_key": "xxxxxxxx",
    "model": "gpt-4"
  }'