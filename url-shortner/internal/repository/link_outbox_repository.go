package repository

type LinkOutboxRepository interface {
	LinkRepository
	OutboxRepository
}
