# YaneMarket Backend API

This is the Go + Gin backend for the YaneMarket platform. It provides a robust, highly concurrent JSON API backed by PostgreSQL.

## Tech Stack
- **Language**: Go 1.25+
- **Framework**: [Gin Web Framework](https://gin-gonic.com/)
- **Database**: PostgreSQL (hosted on Neon)
- **Database Driver/Pool**: `jackc/pgx/v5`
- **Migrations**: `golang-migrate`
- **Configuration**: Strongly typed `.env` parsing via `cleanenv`
- **Logging**: Standard library structured JSON logging (`log/slog`)

## Getting Started

### Prerequisites
- Go 1.25 or higher
- A PostgreSQL database (e.g., Neon serverless Postgres)

### Setup
1. Copy the `.env.example` file to `.env`:
   ```bash
   cp .env.example .env
   ```
2. Update the `DATABASE_URL` in `.env` to point to your PostgreSQL database.
3. Install dependencies:
   ```bash
   go mod download
   ```

### Running the Server
Start the development server:
```bash
go run cmd/server/main.go
```
The server will automatically:
1. Connect to the database.
2. Run any pending database migrations inside the `migrations/` directory.
3. Start the Gin HTTP server on `http://localhost:8080`.

You can test that the server is running by visiting `http://localhost:8080/ping`.

## Project Structure
- `cmd/server/main.go`: The main entry point.
- `internal/config/`: Reads and validates environment variables.
- `internal/db/`: Manages database connections, connection pooling, and programmatic migrations.
- `internal/logger/`: Configures the global structured logger.
- `migrations/`: Contains raw SQL files for version-controlled database schema changes.
