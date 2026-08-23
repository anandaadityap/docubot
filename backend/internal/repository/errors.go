package repository

import "errors"

var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrEmailTaken is returned when registering with an existing email.
	ErrEmailTaken = errors.New("email already taken")
	// ErrSlugTaken is returned when a bot slug is already used.
	ErrSlugTaken = errors.New("slug already taken")
)
