# URL Shortener Platform

This repository applies system design principles to a URL shortener and evolves it toward production-grade reliability.

The platform already demonstrates production-minded building blocks: transactional outbox, background event processing, and idempotent event consumption.

## Current State

Implemented today:
- URL creation and redirect flow with PostgreSQL persistence
- Idempotent create behavior (unique original URL)
- Redis cache with singleflight protection on cache miss
- Outbox pattern on write path (link + event persisted in one DB transaction)
- Background outbox worker claiming and publishing events
- HTTP publisher with retry + exponential backoff + jitter
- Worker-managed durable retry state (retry_count + next_retry_at)
- Separate notification service with idempotent processed-events table
- Docker Compose setup for all services and dependencies

## Services

- url-shortner
	- API for creating and resolving short links
	- Writes to links table and outbox_events table transactionally
	- Runs outbox worker in-process
- notification-service
	- Receives published events on /events
	- Uses insert-first idempotency with unique event_id in processed_events

## Architecture

### URL Create Flow

1. Client calls POST /urls.
2. URL service inserts link into links.
3. In the same transaction, URL service inserts link.created event into outbox_events.
4. Outbox worker claims pending rows (FOR UPDATE SKIP LOCKED + claimed_at).
5. Worker publishes to notification service.
6. On success, event is marked processed.

### Redirect Flow

1. Client calls GET /r/{code}.
2. URL service checks Redis cache first.
3. On miss, service loads from PostgreSQL and fills cache.
4. singleflight collapses concurrent misses for same code.

## Reliability Design Notes

- Atomicity for write + event enqueue is achieved with DB transaction.
- Event consumption is idempotent in notification service (duplicate event_id ignored).
- Outbox claim strategy supports multiple workers safely.
- Event publish retries use exponential backoff + jitter for transient network errors.
- Failed publish attempts are rescheduled durably via retry_count and next_retry_at.

## Known Gaps

- No dead-letter policy yet for events that fail repeatedly over long windows.
- HTTP event transport creates tighter coupling than broker-based asynchronous transport.
- Observability is currently log-based; metrics and tracing are not wired yet.

## Repository Layout

- url-shortner
	- cmd/server/main.go
	- internal/service, internal/repository, internal/worker, internal/cache, internal/event
	- migrations for links and outbox schema
- notification-service
	- cmd/server/main.go
	- internal/handler/http.go
	- migrations for processed_events idempotency table

## Run With Docker Compose

From the repository root:

```bash
docker compose up --build
```

Services:
- URL API: http://localhost:8080
- Notification service: http://localhost:8081
- PostgreSQL (url-shortner): localhost:5432
- PostgreSQL (notification-service): localhost:5434
- Redis: localhost:6399

## API

Create short URL:

```bash
curl -X POST http://localhost:8080/urls \
	-H "Content-Type: application/json" \
	-d '{"url":"https://example.com"}'
```

Redirect:

```bash
curl -i http://localhost:8080/r/<short_code>
```

## Roadmap

Planned next:
- Dead-letter strategy and explicit max retry policy for permanently failing events
- Backoff policy tuning and downstream circuit-breaking
- Broker-based delivery (Kafka or Redis Streams)
- Better observability (metrics, tracing, dashboards)
- Rate limiting and traffic shaping
- Horizontal scaling and operational hardening

## Why This Project

This codebase evolves incrementally through reliability-focused milestones.

The goal is not only feature delivery, but learning how to preserve correctness as distributed-system complexity grows.
