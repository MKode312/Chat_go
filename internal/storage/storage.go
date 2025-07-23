package storage

import (
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidLoginOrPassword = errors.New("invalid login or password")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrMessageNotFound = errors.New("message not found")
	ErrChatNotFound = errors.New("chat not found")
	ErrChatAlreadyExists = errors.New("chat already exists")
)