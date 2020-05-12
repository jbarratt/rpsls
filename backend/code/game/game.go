// package Game implements the core logic of rock paper scissors lizard spock
package game

import (
	"crypto/rand"
	"errors"
	"fmt"
)

var (
	winnermap map[string]string
	plays     []string
)

const (
	NUM_PLAYERS   = 2
	GAMEID_LENGTH = 5
)

// Player stores relevant information about a current player's state
type Player struct {
	ID           string
	Number       int
	Play         string
	Score        int
	Game         string
	WonLastRound bool
}

// Game is the key data for the overall game
type Game struct {
	ID           string
	Round        int
	PlayCount    int
	Scores       []int
	PlayerID     []string
	Plays        []string
	RoundSummary string
	Winner       int
}

// GameContext is a container for the overall game and the current player action in it
type GameContext struct {
	Game   *Game
	Player *Player
}

// AssignPlayer sets a player to be P1 or P2
func (gc *GameContext) AssignPlayer() error {
	for idx := 0; idx < NUM_PLAYERS; idx++ {
		if gc.Game.PlayerID[idx] == "" {
			gc.Game.PlayerID[idx] = gc.Player.ID
			gc.Player.Number = idx + 1
			err := gc.UpdatePlayer()
			if err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("unable to assign player, game is already full")
}

// UpdatePlayer updates the player part of the context from the game state
func (gc *GameContext) UpdatePlayer() error {
	// Player needs to be assigned
	if gc.Player.Number == 0 {
		for idx := 0; idx < NUM_PLAYERS; idx++ {
			if gc.Player.ID == gc.Game.PlayerID[idx] {
				gc.Player.Number = idx + 1
			}
		}
		if gc.Player.Number == 0 {
			return errors.New("Player is not a member of the game")
		}
	}
	gc.Player.Game = gc.Game.ID
	gc.Player.Score = gc.Game.Scores[gc.Player.Number-1]
	gc.Player.WonLastRound = (gc.Game.Winner == gc.Player.Number)
	// gc.Player.Play = gc.Game.Plays[pidx]
	return nil
}

func (gc *GameContext) Play(play string) error {
	if !ValidPlay(play) {
		return errors.New("Invalid play " + play)
	}
	if gc.Player.Number < 1 || gc.Player.Number > NUM_PLAYERS {
		return errors.New("Invalid Player Number")
	}
	gc.Player.Play = play
	gc.Game.Plays[gc.Player.Number-1] = play
	gc.Game.PlayCount++
	return nil
}

func NewGame() *Game {
	id, _ := GenerateRandomString(GAMEID_LENGTH)
	g := Game{
		ID:       id,
		Round:    1,
		Scores:   make([]int, NUM_PLAYERS),
		PlayerID: make([]string, NUM_PLAYERS),
		Plays:    make([]string, NUM_PLAYERS),
	}
	return &g
}

func NewGameContext(playerID string, game *Game) *GameContext {
	p := Player{ID: playerID}
	gc := GameContext{
		Game:   game,
		Player: &p,
	}
	gc.UpdatePlayer()
	return &gc
}

// AdvanceGame updates a game to resolve the winner, round, etc
func (g *Game) AdvanceGame() error {

	if g.PlayCount != NUM_PLAYERS {
		return errors.New("Cannot advance game without all plays")
	}

	// Check for a tie
	if g.Plays[0] == g.Plays[1] {
		g.Winner = 0
		g.RoundSummary = fmt.Sprintf("Both played %s, tie", g.Plays[0])
	} else {
		beats, how := Beats(g.Plays[0], g.Plays[1])
		if beats {
			g.Winner = 1
			g.RoundSummary = fmt.Sprintf("%s %s %s", g.Plays[0], how, g.Plays[1])
		} else {
			_, how = Beats(g.Plays[1], g.Plays[0])
			g.Winner = 2
			g.RoundSummary = fmt.Sprintf("%s %s %s", g.Plays[1], how, g.Plays[0])
		}
		g.Scores[g.Winner-1]++
	}

	g.Round = g.Round + 1
	g.PlayCount = 0

	return nil
}

// Beats returns if first would beat second
// also returns the verb needed <first> crushes <second>
// In the case of a tie, returns "ties" as the verb
func Beats(first, second string) (bool, string) {
	if first == second {
		return false, "ties"
	}
	how, ok := winnermap[first+":"+second]
	if ok {
		return true, how
	}
	return false, ""
}

// ValidPlay returns true only if the play given in the argument is valid
func ValidPlay(play string) bool {
	for _, val := range plays {
		if val == play {
			return true
		}
	}
	return false
}

// GenerateRandomString returns a random string of length N
func GenerateRandomString(n int) (string, error) {
	letters := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	data, err := GenerateRandomBytes(n)
	if err != nil {
		return "NONRANDOM", err
	}
	for i, b := range data {
		data[i] = letters[b%byte(len(letters))]
	}
	return string(data), nil
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

func init() {
	plays = []string{"rock", "paper", "scissors", "lizard", "spock"}
	winnermap = map[string]string{
		"scissors:paper":  "cuts",
		"scissors:lizard": "decapitates",
		"paper:rock":      "covers",
		"paper:spock":     "disproves",
		"rock:lizard":     "crushes",
		"rock:scissors":   "smashes",
		"lizard:paper":    "eats",
		"lizard:spock":    "poisons",
		"spock:scissors":  "beams up",
		"spock:rock":      "sits on",
	}
}
