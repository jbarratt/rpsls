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
		err := s.NewGame(e.RequestContext.ConnectionID, message)
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
	gc, err := game.NewGameContext(message.UID, connectionID, g)

	// set this equal to the round from the user
	// so plays will be rejected if too old.
	gc.ActingPlayer.Round = message.Round

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
	s.NotifyPlayers(gc)
	return nil
}

func otherPlayer(gc *game.GameContext) (*game.Player, error) {
	var them *game.Player
	for id, player := range gc.Game.Players {
		if gc.ActingPlayer.ID != id {
			them = player
		}
	}

	if them == nil {
		return nil, errors.New("Unable to find second player when messaging round completion")
	}
	return them, nil
}

// NotifyPlayers sends out a notification about a game round to all connected parties
func (s *LambdaSvc) NotifyPlayers(gc *game.GameContext) error {

	you := gc.ActingPlayer
	them, err := otherPlayer(gc)
	if err != nil {
		fmt.Printf("Unable to find other player, critical.")
		return err
	}

	youState := GameState{
		Round:      gc.Game.Round,
		GameID:     gc.Game.ID,
		YourScore:  you.Score,
		TheirScore: them.Score,
		YourPlay:   you.Play,
		TheirPlay:  them.Play,
	}
	themState := GameState{
		Round:      gc.Game.Round,
		GameID:     gc.Game.ID,
		YourScore:  them.Score,
		TheirScore: you.Score,
		YourPlay:   them.Play,
		TheirPlay:  you.Play,
	}

	switch gc.Game.Winner {
	case "Tie":
		youState.Winner = false
		themState.Winner = false
		youState.RoundSummary = "Tie Game"
		themState.RoundSummary = "Tie Game"
	case you.ID:
		youState.Winner = true
		themState.Winner = false
		youState.RoundSummary = gc.Game.RoundSummary
		themState.RoundSummary = gc.Game.RoundSummary
	case them.ID:
		youState.Winner = false
		themState.Winner = true
		youState.RoundSummary = gc.Game.RoundSummary
		themState.RoundSummary = gc.Game.RoundSummary
	default:
		return errors.New("Unusable winner value")
	}

	s.SendStateMessage(&youState, you.Address)
	s.SendStateMessage(&themState, them.Address)
	return nil
}

func (s *LambdaSvc) SendStateMessage(gs *GameState, address string) error {
	b, err := json.Marshal(gs)
	if err != nil {
		return err
	}
	err = s.ws.Send(address, b)
	if err != nil {
		// TODO maybe handle a disconnection event here
		fmt.Printf("Error sending to player %s\n", err)
	}
	return nil
}

// SendGameState will send a game state to a given connection
func (s *LambdaSvc) SendGameState(gc *game.GameContext) error {

	you := gc.ActingPlayer
	them, err := otherPlayer(gc)
	if err != nil {
		fmt.Printf("unable to find other player, but this may be new game")
		them = &game.Player{}
	}

	state := GameState{
		Round:      gc.Game.Round,
		GameID:     gc.Game.ID,
		YourScore:  you.Score,
		TheirScore: them.Score,
		YourPlay:   you.Play,
		TheirPlay:  them.Play,
	}

	s.SendStateMessage(&state, you.Address)
	return nil
}

// JoinGame joins a game in progress
func (s *LambdaSvc) JoinGame(connectionID string, message PlayerMessage) error {

	g, err := s.store.Load(message.GameID)
	gc, err := game.NewGameContext(message.UID, connectionID, g)

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
func (s *LambdaSvc) NewGame(connectionID string, message PlayerMessage) error {
	g := game.NewGame()
	gc, err := game.NewGameContext(message.UID, connectionID, g)

	err = s.store.StoreAll(g)
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
