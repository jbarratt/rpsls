package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jbarratt/rpsls/backend/code/game"
	"github.com/jbarratt/rpsls/backend/code/notify"
	"github.com/jbarratt/rpsls/backend/code/store"
)

type LambdaSvc struct {
	store *store.Store
	ws    *notify.APIGWNotifier
}

// NewLambdaSvc returns a new lambda service
func NewLambdaSvc(store *store.Store, ws *notify.APIGWNotifier) *LambdaSvc {
	return &LambdaSvc{
		store: store,
		ws:    ws,
	}
}

// Connect is currently a no-op
func (s *LambdaSvc) Connect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

// Disconnect is currently a no-op
func (s *LambdaSvc) Disconnect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func (s *LambdaSvc) Default(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$defaut: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)

	// Parse a PlayerMessage
	message := PlayerMessage{}
	if err := json.Unmarshal([]byte(e.Body), &message); err != nil {
		log.Println("Unable to decode player message", err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
		}, nil
	}

	switch strings.ToLower(message.Action) {
	case "play":
		err := s.Play(e.RequestContext.ConnectionID, message)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}
	case "new":
		err := s.NewGame(e.RequestContext.ConnectionID)
		if err != nil {
			log.Println("Unable to create new game", err.Error())
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}
	case "join":
		err := s.JoinGame(e.RequestContext.ConnectionID, message)
		if err != nil {
			log.Println("Unable to join game", err.Error())
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}
	default:
		fmt.Printf("Unknown action %s\n", message.Action)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

// Play handles a single player's play.
func (s *LambdaSvc) Play(connectionID string, message PlayerMessage) error {
	// Validate the plays are correct
	message.Play = strings.ToLower(message.Play)

	g, err := s.store.Load(message.GameID)
	if err != nil {
		fmt.Printf("Unable to load game: %s\n", err)
		return err
	}
	gc := game.NewGameContext(connectionID, g)

	// set this equal to the round from the user
	// so plays will be rejected if too old.
	gc.Game.Round = message.Round

	err = gc.Play(message.Play)
	if err != nil {
		fmt.Printf("invalid play: %s Game: %+v Message: %+v\n", err, gc.Game, message)
		return err
	}
	err = s.store.StorePlay(gc)
	if err != nil {
		fmt.Printf("Unable to store play: %s\n", err)
		return err
	}

	err = gc.Game.AdvanceGame()
	if err != nil {
		fmt.Printf("Round not yet complete")
		return nil
	}

	err = s.store.StoreRound(gc.Game)
	if err != nil {
		fmt.Printf("Unable to store round: %s\n", err)
		return err
	}

	// the round advanced, time to notify all the players
	s.NotifyPlayers(gc.Game)
	return nil
}

// NotifyPlayers sends out a notification about a game round to all connected parties
func (s *LambdaSvc) NotifyPlayers(g *game.Game) error {

	states := make([]GameState, 2)
	states[0] = GameState{
		Round:      g.Round,
		GameID:     g.ID,
		YourScore:  g.Scores[0],
		TheirScore: g.Scores[1],
		YourPlay:   g.Plays[0],
		TheirPlay:  g.Plays[1],
	}
	states[1] = GameState{
		Round:      g.Round,
		GameID:     g.ID,
		YourScore:  g.Scores[1],
		TheirScore: g.Scores[0],
		YourPlay:   g.Plays[1],
		TheirPlay:  g.Plays[0],
	}

	switch g.Winner {
	case 0:
		states[0].Winner = false
		states[1].Winner = false
		states[0].RoundSummary = "Tie Game"
		states[1].RoundSummary = "Tie Game"
	case 1:
		states[0].Winner = true
		states[1].Winner = false
		states[0].RoundSummary = g.RoundSummary
		states[1].RoundSummary = g.RoundSummary
	case 2:
		states[0].Winner = false
		states[1].Winner = true
		states[0].RoundSummary = g.RoundSummary
		states[1].RoundSummary = g.RoundSummary
	default:
		return errors.New("Unusable winner value")
	}

	for i := 0; i < 2; i++ {
		b, err := json.Marshal(states[i])
		if err != nil {
			return err
		}
		err = s.ws.Send(g.PlayerID[i], b)
		if err != nil {
			fmt.Sprintf("Error sending to player %d: %s\n", i+1, err)
		}
	}
	return nil
}

// SendGameState will send a game state to a given connection
func (s *LambdaSvc) SendGameState(gc *game.GameContext) error {
	you := gc.Player.Number - 1
	other := 0
	if you == 0 {
		other = 1
	}
	if you == -1 {
		fmt.Println("player was not assigned to the game")
		return errors.New("player was not a member of the game")
	}

	state := GameState{
		Round:      gc.Game.Round,
		GameID:     gc.Game.ID,
		YourScore:  gc.Game.Scores[you],
		TheirScore: gc.Game.Scores[other],
		YourPlay:   gc.Game.Plays[you],
		TheirPlay:  gc.Game.Plays[other],
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	err = s.ws.Send(gc.Player.ID, b)
	if err != nil {
		return err
	}
	return nil
}

// JoinGame joins a game in progress
func (s *LambdaSvc) JoinGame(connectionID string, message PlayerMessage) error {

	g, err := s.store.Load(message.GameID)
	gc := game.NewGameContext(connectionID, g)
	gc.AssignPlayer()

	err = s.store.StorePlayer(gc)
	if err != nil {
		fmt.Printf("unable to store player: %s\n", err)
	}

	err = s.SendGameState(gc)
	if err != nil {
		fmt.Printf("Got an error sending the game state to the new player: %s\n", err)
		return err
	}
	return nil
}

// NewGame creates a new game record in the database
func (s *LambdaSvc) NewGame(connectionID string) error {
	g := game.NewGame()
	gc := game.NewGameContext(connectionID, g)
	gc.AssignPlayer()

	err := s.store.StoreNew(g)
	if err != nil {
		fmt.Printf("unable to store game: %s %+v", err, g)
		return err
	}

	err = s.SendGameState(gc)
	if err != nil {
		fmt.Println("Error notifying user")
		fmt.Println(err.Error())
		return err
	}
	return nil
}
