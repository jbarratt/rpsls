package service

// GameState structures for sending to players. Fields are optional especially on setup
type GameState struct {
	Round        int    `json:"round"`
	GameID       string `json:"gameId"`
	YourScore    int    `json:"yourScore"`
	TheirScore   int    `json:"theirScore"`
	Winner       bool   `json:"winner"`
	YourPlay     string `json:"yourPlay,omitempty"`
	TheirPlay    string `json:"theirPlay,omitempty"`
	RoundSummary string `json:"roundSummary,omitempty"`
}

// PlayerMessage are what we get from the players
type PlayerMessage struct {
	Action string `json:"action"`
	UID    string `json:"userId"`
	GameID string `json:"gameId"`
	Play   string `json:"play"`
	Round  int    `json:"round"`
}
