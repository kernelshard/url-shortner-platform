# URL Shortener Platform (System Design Exploration)

This is not just a URL shortener.

This project explores how a simple service evolves into a reliable, distributed system under real-world constraints such as failures, concurrency, and scale.

---

## 🎯 Objectives

- Build a clean and extensible service architecture
- Ensure correctness through idempotency and data constraints
- Introduce asynchronous event-driven design
- Handle failures with retry and backoff strategies
- Move towards guaranteed delivery (Outbox pattern)
- Prepare the system for high-traffic scenarios

---

## 🧠 Key Concepts Implemented

- **Idempotency** via database constraints (unique original URL)
- **Caching + singleflight** to prevent cache stampede
- **Separation of concerns** using interface-driven design
- **Event-driven architecture** using publisher abstraction
- **Retry with exponential backoff** for transient failures
- **Service decoupling** via HTTP-based event publishing

---

## 🏗️ Architecture Evolution

### Phase 1 — Synchronous Core
- Basic URL creation and retrieval
- Database as the source of truth

---

### Phase 2 — In-Memory Events (Simulation)
- Introduced publisher abstraction
- Simulated consumer for validating event flow

---

### Phase 3 — Distributed System (HTTP)
- URL service publishes events over HTTP
- Notification service consumes events
- Established real service boundary

---

### Phase 4 — Resilience
- Retry with exponential backoff
- Improved logging for observability
- Best-effort event delivery

---

### Phase 5 (Next) — Durability
- Outbox pattern for atomicity between DB and events
- Guarantees eventual event delivery

---

### Phase 6 (Future) — High Traffic Readiness
- Redis-based distributed caching
- Horizontal scaling of services
- Read replicas and database optimization
- Message broker (Kafka/Redis Streams) for async communication
- Rate limiting and backpressure handling

---

## 🔁 Current Request Flow

### URL Creation

Client → URL Service → DB  
          ↘ HTTP Event → Notification Service

---

### URL Redirect

Client → URL Service → Cache → DB (fallback)

---

## ⚠️ Known Limitations (Current Stage)

- Event delivery is **best-effort** (may be lost on failure)
- No durable event storage (Outbox not implemented yet)
- Tight coupling via HTTP (no message broker yet)

---

## 🧩 Design Trade-offs

- Prioritized **correctness of core data** over side-effects
- Accepted **eventual inconsistency** in current phase
- Deferred durability in favor of incremental learning
- Used HTTP instead of broker to validate behavior first

---

## 🚀 Why This Project

This project focuses on **how systems evolve**, not just what they do.

It demonstrates:
- Thoughtful trade-offs
- Failure-aware design
- Incremental architecture improvements

Rather than building features, the goal is to build **correct systems under real-world conditions**.

---

## 📌 Next Steps

- Implement Outbox pattern for guaranteed event delivery
- Introduce background worker for event publishing
- Move to message broker (Kafka/Redis Streams)
- Add rate limiting and traffic control

---

## 🧠 Takeaway

> Building scalable systems is not about adding components.
> It is about preserving correctness as complexity increases.
