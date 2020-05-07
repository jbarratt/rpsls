package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Package scoped variables aren't great but
// for lambda, it's probably not too terrible
var (
	ddbclient   *dynamodb.DynamoDB
	ddbtable    string
	apigwclient *apigatewaymanagementapi.ApiGatewayManagementApi
	playerNum   map[string]int
	beats       map[string]string
	plays       []string
)

func Connect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$connect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	gameID, err := GetGameFromConnection(e.RequestContext.ConnectionID)
	if err == nil {
		// TODO: send game state to connection
		fmt.Sprintf("need to send the state of game %s to client\n", gameID)
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func Disconnect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$disconnect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	// TODO
	// look up the game if there is one
	// remove the session from it
	//    UpdateGamePlayer("", gameID, playerID)
	// notify the other player
	// remove the connection/game link
	RemoveConnection(e.RequestContext.ConnectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func SendMessage(message []byte, connectionID string) error {
	input := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         message,
	}

	_, err := apigwclient.PostToConnection(input)
	if err != nil {
		log.Println("Error Sending Message", err.Error())
		return err
		// TODO
		//	- on notification failure
		//	- remove the connectionid -> gameid entry
		//	- unset that connection id from the game
	}
	return nil
}

func Default(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
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
		// player making a move
		err := Play(e.RequestContext.ConnectionID, message)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}
	case "new":
		err := NewGame(e.RequestContext.ConnectionID)
		if err != nil {
			log.Println("Unable to create new game", err.Error())
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
			}, nil
		}
	case "join":
		err := JoinGame(e.RequestContext.ConnectionID, message)
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

// GetPlayer returns a memoized player ID from connection and game
func GetOrAssignPlayer(connectionID, gameID string) (int, error) {
	// TODO need to make sure to clear this out on disconnect or connect before it can be trusted
	/*
		number, ok := playerNum[connectionID]
		if ok {
			return number, nil
		}
	*/
	// Otherwise, that means we need to load the game and figure out which player this is
	game, err := LoadGame(gameID)
	if err != nil {
		return 0, err
	}
	player := 0
	if game.P1ConnID == connectionID {
		player = 1
	} else if game.P2ConnID == connectionID {
		player = 2
	}
	if player != 0 {
		playerNum[connectionID] = player
		return player, nil
	}

	// otherwise, player needs to be assigned
	if len(game.P1ConnID) < 2 {
		player = 1
		err := UpdateGamePlayer(connectionID, gameID, player)
		if err != nil {
			fmt.Printf("Error while assigning player 1 to game\n")
			fmt.Println(err.Error())
			return 0, err
		}
	}
	if len(game.P2ConnID) < 2 {
		player = 2
		err := UpdateGamePlayer(connectionID, gameID, player)
		if err != nil {
			fmt.Printf("Error while assigning player 2 to game\n")
			fmt.Println(err.Error())
			return 0, err
		}
	}
	if player != 0 {
		playerNum[connectionID] = player
		return player, nil
	}
	fmt.Printf("unable to assign a player, no slots available\n")
	return 0, errors.New("No player slots available")
}

// UpdateGamePlayer either assigns or unassigns a player
// Set connectionID to the empty string to mark the user as disconnected
func UpdateGamePlayer(connectionID, gameID string, player int) error {
	fmt.Printf("Assigning player %d from connection %s to game %s\n", player, connectionID, gameID)
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":conn": {
				S: aws.String(connectionID),
			},
		},
		TableName: aws.String(ddbtable),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gameID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gameID)),
			},
		},
		UpdateExpression: aws.String(fmt.Sprintf("set P%dConnID = :conn", player)),
	}

	_, err := ddbclient.UpdateItem(input)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	// If a player actually needed to be assigned here,
	// write the game in, just in case. Catches cases where this
	// is a reconnect, or when the client is joining a new game
	StoreConnection(connectionID, gameID)
	return nil
}

// Play handles a single player's play. It includes the majority
// of the game engine logic
// TODO make sure to sanitize the Game ID values
func Play(connectionID string, message PlayerMessage) error {
	// Validate the plays are correct
	message.Play = strings.ToLower(message.Play)
	if ValidPlay(message.Play) == false {
		return errors.New(message.Play + " is not a valid play")
	}

	// Figure out which player we are (1 or 2)
	player, err := GetOrAssignPlayer(connectionID, message.GameID)
	if err != nil {
		fmt.Printf("error identifying player %s\n", connectionID)
		return err
	}
	fmt.Printf("Identified player at %s as %d\n", connectionID, player)

	// Attempt to update. Only do so if this player is the first to get in.
	// Update with plays = 1 and p=play = message.Play unless plays != 0
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":play": {
				S: aws.String(message.Play),
			},
			":count": {
				N: aws.String("1"),
			},
			":zero": {
				N: aws.String("0"),
			},
			":round": {
				N: aws.String(fmt.Sprintf("%d", message.Round)),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#pxplay": aws.String(fmt.Sprintf("P%dPlay", player)),
			"#round":  aws.String("Round"),
		},
		TableName: aws.String(ddbtable),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", message.GameID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", message.GameID)),
			},
		},
		ConditionExpression: aws.String(fmt.Sprintf("Plays = :zero AND #round = :round")),
		UpdateExpression:    aws.String(fmt.Sprintf("SET Plays = :count, #pxplay = :play")),
	}

	result, err := ddbclient.UpdateItem(input)
	if err == nil {
		fmt.Println("first player submitted")
		return nil
	}
	if !(strings.Contains(err.Error(), "ConditionalCheckFailed")) {
		// some other kind of error
		fmt.Printf("got an error making a dynamo play\n")
		fmt.Println(err.Error())
		return err
	}

	// Conditional Check Failed, meaning the other player has already gone.
	// Sadly this means Dynamo will not return the unmodified values to us, so we have to fetch them.
	item, err := LoadGame(message.GameID)
	if err != nil {
		fmt.Println("Error fetching current state of game")
		fmt.Println(err.Error())
		return err
	}
	// update the record with this player's move for judging
	if player == 1 {
		item.P1Play = message.Play
	} else if player == 2 {
		item.P2Play = message.Play
	}

	// TODO how did I get here when only the first player went
	fmt.Printf("Game data to be used for judging: %+v\n", item)

	winningPlayer, how := WinningPlayer(item)

	pxScore := "P1Score"
	scoreInc := "SET "
	if winningPlayer != 0 {
		scoreInc = scoreInc + "#pxscore = #pxscore + :one, "
		pxScore = fmt.Sprintf("P%dScore", winningPlayer)
	} else {
		scoreInc = scoreInc + "#pxscore = #pxscore + :zero, "
	}
	updateExpr := scoreInc + "Round = Round + :one, Plays = :zero, #pxplay = :play"

	input = &dynamodb.UpdateItemInput{
		TableName: aws.String(ddbtable),
		ExpressionAttributeNames: map[string]*string{
			"#pxscore": aws.String(pxScore),
			"#pxplay":  aws.String(fmt.Sprintf("P%dPlay", player)),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":one": {
				N: aws.String("1"),
			},
			":zero": {
				N: aws.String("0"),
			},
			":play": {
				S: aws.String(message.Play),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", message.GameID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", message.GameID)),
			},
		},
		ReturnValues:     aws.String("ALL_NEW"),
		UpdateExpression: aws.String(updateExpr),
	}

	result, err = ddbclient.UpdateItem(input)
	if err != nil {
		fmt.Println("call error: unable to set next round of the game")
		fmt.Printf(err.Error())
		return err
	}

	err = dynamodbattribute.UnmarshalMap(result.Attributes, &item)
	if err != nil {
		fmt.Println("unmarshal error: unable to set next round of the game")
		fmt.Printf(err.Error())
		return err
	}

	err = NotifyPlayers(item, winningPlayer, how)
	if err != nil {
		fmt.Println("unable to notify players")
		return err
	}

	return nil
}

// NotifyPlayers sends out a notification about a game round to all connected parties
func NotifyPlayers(game GameItem, winningPlayer int, how string) error {
	// - notify all players:
	// 		- next round id
	// 		- did they win or lose
	// 		- their and other player's play
	// 		- current scores

	states := make([]GameState, 2)
	states[0] = GameState{
		Round:             game.Round,
		GameID:            game.GameID,
		YourScore:         game.P1Score,
		TheirScore:        game.P2Score,
		YourPlay:          game.P1Play,
		TheirPlay:         game.P2Play,
		OpponentConnected: true,
	}
	states[1] = GameState{
		Round:             game.Round,
		GameID:            game.GameID,
		YourScore:         game.P2Score,
		TheirScore:        game.P1Score,
		YourPlay:          game.P1Play,
		TheirPlay:         game.P2Play,
		OpponentConnected: true,
	}

	switch winningPlayer {
	case 0:
		states[0].Winner = false
		states[1].Winner = false
		states[0].RoundSummary = "Tie Game"
		states[1].RoundSummary = "Tie Game"
	case 1:
		states[0].Winner = true
		states[1].Winner = false
		states[0].RoundSummary = fmt.Sprintf("%s %s %s", game.P1Play, how, game.P2Play)
		states[1].RoundSummary = fmt.Sprintf("%s %s %s", game.P1Play, how, game.P2Play)
	case 2:
		states[0].Winner = false
		states[1].Winner = true
		states[0].RoundSummary = fmt.Sprintf("%s %s %s", game.P2Play, how, game.P1Play)
		states[1].RoundSummary = fmt.Sprintf("%s %s %s", game.P2Play, how, game.P1Play)
	default:
		return errors.New("Unusable winningPlayer value")
	}

	b, err := json.Marshal(states[0])
	if err != nil {
		return err
	}
	SendMessage(b, game.P1ConnID)

	b, err = json.Marshal(states[1])
	if err != nil {
		return err
	}
	SendMessage(b, game.P2ConnID)

	return nil
}

// SendGameState will send a game state to a given connection
func SendGameState(connectionID string, game GameItem) error {
	state := GameState{}
	if game.P1ConnID == connectionID {
		state = GameState{
			Round:             game.Round,
			GameID:            game.GameID,
			YourScore:         game.P1Score,
			TheirScore:        game.P2Score,
			YourPlay:          game.P1Play,
			TheirPlay:         game.P2Play,
			OpponentConnected: false,
		}
		if len(game.P2ConnID) > 2 {
			state.OpponentConnected = true
		}
	} else {
		state = GameState{
			Round:             game.Round,
			GameID:            game.GameID,
			YourScore:         game.P2Score,
			TheirScore:        game.P1Score,
			YourPlay:          game.P1Play,
			TheirPlay:         game.P2Play,
			OpponentConnected: false,
		}
		if len(game.P1ConnID) > 2 {
			state.OpponentConnected = true
		}
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	SendMessage(b, connectionID)
	return nil
}

// JoinGame joins a game in progress
func JoinGame(connectionID string, message PlayerMessage) error {
	GetOrAssignPlayer(connectionID, message.GameID)
	StoreConnection(connectionID, message.GameID)
	// need values after updates from other two queries
	// may need to make a consistent read
	game, err := LoadGame(message.GameID)
	if err != nil {
		return err
	}
	SendGameState(connectionID, game)
	return nil
}

func GetSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalln("unable to create session", err.Error())
	}
	return sess
}

func Handler(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("Entered handler\n")

	// Set up the AWS connections. TODO refactor
	sess := GetSession()
	baseURL := fmt.Sprintf("https://%s/%s/", e.RequestContext.DomainName, e.RequestContext.Stage)
	apigwclient = apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(baseURL))

	// TODO refactor around a dynamo implementation of a store interface
	ddbtable = os.Getenv("TABLE_NAME")
	ddbclient = dynamodb.New(GetSession())

	switch e.RequestContext.RouteKey {
	case "$connect":
		return Connect(e)
	case "$disconnect":
		return Disconnect(e)
	default:
		return Default(e)
	}
}

// GameState structures for sending to players. Fields are optional especially on setup
type GameState struct {
	Round             int    `json:"round"`
	GameID            string `json:"gameId"`
	YourScore         int    `json:"yourScore"`
	TheirScore        int    `json:"theirScore"`
	Winner            bool   `json:"winner,omitempty"`
	YourPlay          string `json:"yourPlay,omitempty"`
	TheirPlay         string `json:"theirPlay,omitempty"`
	RoundSummary      string `json:"roundSummary,omitempty"`
	OpponentConnected bool   `json:"opponentConnected"`
}

// TODO use PlayerSession in the places where normally game is passed around
// Alternatively GameContext which has Player info (ID, etc) as well as the game info we know about
type PlayerSession struct {
	ConnectionID string
	GameID       string
	Number       int
}

// PlayerMessage are what we get from the players
type PlayerMessage struct {
	Action string `json:"action"`
	GameID string `json:"gameId"`
	Play   string `json:"play"`
	Round  int    `json:"round"`
}

// ConnectionItem tracks the link between a game and connection
type ConnectionItem struct {
	PK     string
	SK     string
	Type   string
	GameID string
}

// GameItem is for game status items
type GameItem struct {
	PK       string
	SK       string
	Type     string
	Round    int
	Plays    int
	P1Play   string
	P2Play   string
	P1Score  int
	P2Score  int
	P1ConnID string
	P2ConnID string
	GameID   string
}

// StoreConnection will store the game ID with the connection ID
func StoreConnection(connectionID, gameID string) error {

	item := ConnectionItem{
		PK:     fmt.Sprintf("CONN#%s", connectionID),
		SK:     fmt.Sprintf("CONN#%s", connectionID),
		Type:   "ConnectionItem",
		GameID: gameID,
		// TODO add TTL
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		fmt.Println("Got error marshalling connectiongame:")
		fmt.Println(err.Error())
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(ddbtable),
	}

	_, err = ddbclient.PutItem(input)
	if err != nil {
		fmt.Println("Error calling PutItem")
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func NewGameID() (string, error) {
	for i := 0; i < 3; i++ {
		id, err := GenerateRandomString(5)
		if err != nil {
			return "", err
		}
		_, err = LoadGame(id)
		// in this case an error means a collision and need to try a new ID
		// no error means this id is good to return
		if err == nil {
			return id, nil
		}
	}
	// if we can't find something in 3 tries something's wrong.
	return "", errors.New("Unable to find unused gameID")
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

//GenerateRandomString returns a random string of length N
func GenerateRandomString(n int) (string, error) {
	letters := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	data, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range data {
		data[i] = letters[b%byte(len(letters))]
	}
	return string(data), nil
}

// NewGame creates a new game record in the database
func NewGame(connectionID string) error {

	gameID, err := NewGameID()
	if err != nil {
		return err
	}

	item := GameItem{
		PK:       fmt.Sprintf("GAME#%s", gameID),
		SK:       fmt.Sprintf("GAME#%s", gameID),
		Type:     "GameItem",
		GameID:   gameID,
		P1ConnID: connectionID,
		Round:    1,
		P1Score:  0,
		P2Score:  0,
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		fmt.Println("Got error marshalling gameitem:")
		fmt.Println(err.Error())
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(ddbtable),
	}

	_, err = ddbclient.PutItem(input)
	if err != nil {
		fmt.Println("Error calling PutItem")
		fmt.Println(err.Error())
		return err
	}
	err = SendGameState(connectionID, item)
	if err != nil {
		fmt.Println("Error notifying user")
		fmt.Println(err.Error())
		return err
	}
	StoreConnection(connectionID, gameID)
	return nil
}

// LoadGame returns a GameItem given a GameID
func LoadGame(gameID string) (GameItem, error) {

	game := GameItem{}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(ddbtable),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gameID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gameID)),
			},
		},
	}
	result, err := ddbclient.GetItem(input)
	if err != nil {
		fmt.Println("Error fetching game by ID")
		fmt.Println(err.Error())
		return game, err
	}
	err = dynamodbattribute.UnmarshalMap(result.Item, &game)
	if err != nil {
		fmt.Println("Error reading game record")
		return game, err
	}
	return game, nil
}

// GetGame will return a GameID given if connectID, if one exists.
func GetGameFromConnection(connectionID string) (string, error) {

	input := &dynamodb.GetItemInput{
		TableName: aws.String(ddbtable),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("CONN#%s", connectionID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("CONN#%s", connectionID)),
			},
		},
	}
	result, err := ddbclient.GetItem(input)
	if err != nil {
		fmt.Println("Error fetching game from connection")
		fmt.Println(err.Error())
		return "", err
	}
	item := ConnectionItem{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		fmt.Println("Error reading game record")
		return "", nil
	}
	if item.GameID != "" {
		return item.GameID, nil
	}
	return "", errors.New("connection had no game set")
}

// RemoveConnection removes a connection (i.e. on disconnect)
func RemoveConnection(connectionID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName:    aws.String(ddbtable),
		ReturnValues: aws.String("ALL_OLD"),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("CONN#%s", connectionID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("CONN#%s", connectionID)),
			},
		},
	}

	result, err := ddbclient.DeleteItem(input)
	if err != nil {
		fmt.Println("Error deleting connection")
		fmt.Println(err.Error())
		return err
	}
	item := ConnectionItem{}
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &item)
	if err != nil {
		fmt.Println("Error reading game record, not unlinking from game")
		return nil
	}
	if item.GameID != "" {
		fmt.Println("TODO call RemoveConnectionFromGame")
		//RemoveConnectionFromGame(connectionID, item.GameID)
	}
	return nil
}

// WinningPlayer takes a game item and returns
// 0 for a tie
// 1 or 2 for winning player
// the name of the winner
// the verb of the action
func WinningPlayer(game GameItem) (int, string) {
	beats, how := Beats(game.P1Play, game.P2Play)
	if how == "ties" {
		return 0, "ties"
	}
	if beats {
		return 1, how
	}
	beats, how = Beats(game.P2Play, game.P1Play)
	if beats {
		return 2, how
	}
	return 0, "a bug"
}

// Beats returns if first would beat second
// also returns the verb needed <first> crushes <second>
// In the case of a tie, returns "ties" as the verb
func Beats(first, second string) (bool, string) {
	fmt.Printf("Checking to see if %s beats %s\n", first, second)
	if first == second {
		fmt.Println("they are equal so they tie")
		return false, "ties"
	}
	how, ok := beats[first+":"+second]
	if ok {
		fmt.Println("It does! returning how")
		return true, how
	}
	fmt.Println("It does not beat it.")
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

func main() {
	lambda.Start(Handler)
}

func init() {
	playerNum = make(map[string]int)
	plays = []string{"rock", "paper", "scissors", "lizard", "spock"}
	beats = map[string]string{
		"scissors:paper":  "cuts",
		"scissors:lizard": "decapitates",
		"paper:rock":      "covers",
		"paper:spock":     "disproves",
		"rock:lizard":     "crushes",
		"rock:scissors":   "smashes",
		"lizard:paper":    "eats",
		"lizard:spock":    "poisons",
		"spock:scissors":  "beams up",
		"spock:rock":      "vaporizes",
	}
	rand.Seed(time.Now().Unix())
}
