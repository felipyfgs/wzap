# Chatwoot Docker Setup for Testing

## Prerequisites
- Docker Desktop (Windows/WSL2) or Docker Engine (Linux)
- Docker Compose

## Quick Start

1. Start the services:
```bash
docker compose up -d
```

2. Prepare the database:
```bash
docker compose run --rm rails bundle exec rails db:chatwoot_prepare
```

3. Access Chatwoot at: http://localhost:3000

## Services

| Service | Port | Description |
|---------|------|-------------|
| Rails   | 3000 | Web application |
| Postgres| 5432 | Database (pgvector/pgvector:pg16) |
| Redis   | 6379 | Cache and job queue |

## Credentials (for testing only)

- **Postgres**: `postgres` / `chatwoot_postgres_pass`
- **Redis**: password `chatwoot_redis_pass`

## Create Admin User

After starting, create an admin user:
```bash
docker compose run --rm rails bundle exec rails console
```

Then in the console:
```ruby
# Create admin account
account = Account.create!(name: "Admin Account")
user = User.create!(
  email: "admin@example.com",
  password: "Password123!",
  password_confirmation: "Password123!",
  name: "Admin",
  confirmed_at: Time.current
)
AccountUser.create!(account: account, user: user, role: :administrator)
```

## Stop Services

```bash
docker compose down
```

## Reset Everything

```bash
docker compose down -v
docker compose up -d
docker compose run --rm rails bundle exec rails db:chatwoot_prepare
```
