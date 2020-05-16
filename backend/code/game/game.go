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
	// ID is the user-supplied user identifier
	ID string
	// Address is the connection id (or other location) for how to get to the player
	// It may change over time for the same player, e.g. if they reconnect
	Address string
	// Play is the player's last move
	Play string
	// Round is the last round played by the player
	Round int
	// Score is the player's score
	Score int
	// Game is the GameID this player is associated with
	Game string
	// WonLastRound identifies if the player ... won the last round
	WonLastRound bool
}

// Game is the key data for the overall game
type Game struct {
	// ID is the identifier of the overall game
	ID string
	// Round tracks which round of the game is currently active
	Round int
	// PlayCount keeps track of how many plays have been submitted
	PlayCount int
	// Players is indexed by Player.ID and points to player records
	Players map[string]*Player
	// RoundSummary is a summary of the previous round ("Rock beats Scissors")
	RoundSummary string
	// Winner is the Player.ID which won the last round.
	Winner string
}

// GameContext is a container for the overall game and the current player action in it
type GameContext struct {
	Game *Game
	// ID of the current player inside the game
	ActingPlayer *Player
}

// AssignPlayer sets a player to be P1 or P2
func (gc *GameContext) AssignPlayer(p *Player) error {
	_, found := gc.Game.Players[p.ID]
	if found {
		gc.ActingPlayer = gc.Game.Players[p.ID]
		// update address in case it has changed
		gc.ActingPlayer.Address = p.Address
	} else {
		if len(gc.Game.Players) < 2 {
			gc.Game.Players[p.ID] = p
			gc.ActingPlayer = p
		} else {
			return errors.New("unable to assign player, game is already full")
		}
	}
	return nil
}

func (gc *GameContext) Play(play string) error {
	if !ValidPlay(play) {
		return errors.New("Invalid play " + play)
	}
	gc.ActingPlayer.Play = play
	if gc.ActingPlayer.Round < gc.Game.Round {
		gc.Game.PlayCount++
	}
	// Update the round to indicate the player played for this round
	gc.ActingPlayer.Round = gc.Game.Round
	return nil
}

func NewGame() *Game {
	id, _ := GenerateRandomString(GAMEID_LENGTH)
	g := Game{
		ID:      id,
		Round:   1,
		Players: make(map[string]*Player),
	}
	return &g
}

func NewGameContext(playerID, playerAddress string, game *Game) (*GameContext, error) {
	gc := GameContext{
		Game: game,
	}
	p := &Player{ID: playerID, Address: playerAddress, Game: game.ID}
	err := gc.AssignPlayer(p)
	if err != nil {
		return nil, err
	}
	return &gc, nil
}

// AdvanceGame updates a game to resolve the winner, round, etc
func (g *Game) AdvanceGame() error {

	if g.PlayCount != NUM_PLAYERS {
		return errors.New("Cannot advance game without all plays")
	}

	// This seems kind of annoying but this is the only place needing to compare players
	players := make([]*Player, NUM_PLAYERS)
	i := 0
	for k := range g.Players {
		// have to make a read only copy
		players[i] = g.Players[k]
		i++
	}

	// Check for a tie
	if players[0].Play == players[1].Play {
		g.Winner = "Tie"
		g.RoundSummary = fmt.Sprintf("Both played %s, tie", players[0].Play)
	} else {
		beats, how := Beats(players[0].Play, players[1].Play)
		if beats {
			g.Winner = players[0].ID
			players[0].Score++
			players[0].WonLastRound = true
			players[1].WonLastRound = false
			g.RoundSummary = fmt.Sprintf("%s %s %s", players[0].Play, how, players[1].Play)
		} else {
			_, how = Beats(players[1].Play, players[0].Play)
			g.Winner = players[1].ID
			players[1].Score++
			players[1].WonLastRound = true
			players[1].WonLastRound = false
			g.RoundSummary = fmt.Sprintf("%s %s %s", players[1].Play, how, players[0].Play)
		}
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
