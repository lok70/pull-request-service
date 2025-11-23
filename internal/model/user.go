package model

// User описывает пользователя, его юзернейм, команду, статус активности.
type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}
