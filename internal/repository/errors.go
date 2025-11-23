package repository

import "errors"

var (
	// ErrUserNotFound возвращается, если пользователь не найден в БД.
	ErrUserNotFound = errors.New("user not found")

	// ErrTeamNotFound возвращается, если команда не найдена.
	ErrTeamNotFound = errors.New("team not found")

	// ErrTeamExists возвращается при попытке создать дубликат команды.
	ErrTeamExists = errors.New("team already exists")

	// ErrPRNotFound возвращается, если PR не найден.
	ErrPRNotFound = errors.New("pull request not found")

	// ErrPRExists возвращается при конфликте ID пулл-реквеста.
	ErrPRExists = errors.New("pull request already exists")
)
