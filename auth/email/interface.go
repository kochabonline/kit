package email

import "time"

type Cache interface {
	Set(key string, value any, expiration time.Duration) error
	Get(key string) (any, error)
	GetTTL(key string) (time.Duration, error)
	Delete(key string) error
}

type EmailAuthenticatorer interface {
	Send(email Email, code string) error
	Validate(email string, code string) (bool, error)
}
