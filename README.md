# poc-pg-worker
A Go-based web API with async worker using Postgres LISTEN/NOTIFY for task processing and web push notifications.

## Architecture

- Single Go executable with both API and worker processes
- Uses pgx v5 for Postgres connectivity
- Uses env/v10 for configuration management
- Docker Compose setup with health checks
- Web Push notifications support with VAPID

## Configuration

### API Server
```env
DATABASE_URL=postgres://postgres:postgres@db:5432/postgres?sslmode=disable
SERVER_PORT=8080
VAPID_API_KEY=your_vapid_key
```

### Client
```env
PUBLIC_VAPID_PUBLIC_KEY=your_vapid_public_key
```

## API Endpoints

### Tasks

1. List Tasks
```bash
curl -X GET http://localhost:8080/tasks
```

2. Create Task
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "example",
    "payload": {"message": "Test task"},
    "status": "pending"
  }'
```

### Subscriptions

1. List Subscriptions
```bash
curl -X GET http://localhost:8080/subscriptions
```

2. Create Subscription
```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://updates.push.services.mozilla.com/..."
  }'
```

### Notifications

1. List Notifications
```bash
curl -X GET http://localhost:8080/notifications
```

2. Create Notification
```bash
curl -X POST http://localhost:8080/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "body": "Test notification"
  }'
```

## Database Schema

### Tasks Table
```sql
CREATE TABLE tasks (
    id VARCHAR(255) PRIMARY KEY,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL
);
```

### Subscriptions Table
```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL
);
```

### Notifications Table
```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    body TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL
);
```

## Features

- Async task processing via Postgres LISTEN/NOTIFY
- Web Push notification support
- Database connection resilience with retry logic
- Docker Compose setup with health checks
- TypeScript-based SvelteKit frontend
