package service

import (
	"context"
	"log"
)

type EmailSender interface {
	Send(ctx context.Context, email, subject, body string) error
}

type emailService struct{}

func NewEmailService() *emailService {
	return &emailService{}
}

func (s *emailService) Send(ctx context.Context, email, subject, body string) error {
	log.Printf("sending email to %s: %s - %s", email, subject, body)
	return nil
}
