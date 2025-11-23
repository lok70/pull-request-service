package model

// TeamMember описывает участника команды с его идентификатором, отображаемым именем и признаком активности.
type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// Team описывает команду и список её участников.
type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}
