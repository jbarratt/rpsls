package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func Connect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$connect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	StoreConnection(e.RequestContext.ConnectionID, "temp")
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func Disconnect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$disconnect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func Default(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$defaut: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)

	baseURL := fmt.Sprintf("https://%s/%s/", e.RequestContext.DomainName, e.RequestContext.Stage)

	// send a message back to the connectionID
	// could return it inline but this is an example of sending to others
	sess := GetSession()

	input := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: &e.RequestContext.ConnectionID,
		Data:         []byte(e.Body),
	}

	apigateway := apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(baseURL))

	_, err := apigateway.PostToConnection(input)
	if err != nil {
		log.Println("Error Posting", err.Error())
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
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
	Round         int    `json:"round"`
	GameID        string `json:"gameId"`
	Winner        bool   `json:"winner"`
	YourPlay      string `json:"yourPlay"`
	TheirPlay     string `json:"theirPlay"`
	BattleSummary string `json:"battleSummary"`
	YourScore     int    `json:"yourScore"`
	TheirScore    int    `json:"theirScore"`
}

// PlayerMessage are what we get from the players
type PlayerMessage struct {
	Action string `json:"action"`
	GameID string `json:"gameId"`
	Play   string `json:"play"`
	Round  int    `json:"round"`
}

// ConnectionGame tracks the link between a game and connection
type ConnectionGame struct {
	PK     string
	SK     string
	Type   string
	GameID string
}

// StoreConnection will store the game ID with the connection ID
func StoreConnection(connectionID, gameID string) error {

	sess := GetSession()
	svc := dynamodb.New(sess)

	item := ConnectionGame{
		PK:     fmt.Sprintf("CONN#%s", connectionID),
		SK:     fmt.Sprintf("CONN#%s", connectionID),
		Type:   "ConnectionGame",
		GameID: gameID,
		// TODO add TTL
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		fmt.Println("Got error marshalling connectiongame:")
		fmt.Println(err.Error())
		return err
	}

	table := os.Getenv("TABLE_NAME")

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(table),
	}

	_, err = svc.PutItem(input)
	if err != nil {
		fmt.Println("Error calling PutItem")
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// GetGame will return a GameID given if connectID, if one exists.
// func GetGame(connectionID string) (string, error) {
// }

/*

Global cache of
- connid -> {gameid, player number}

on play:
- do I know which player I am? cool
- else, fetch game state and figure it out based on conn id. Cache this data.
- if I'm not one of the connection ids
- if one is blank, then that's me, add my connection id there
- add a CONN#connectionid -> gameid entry
- update GAME#$gameid, plays:1, p1play: move UNLESS plays>0
- else, fetch game state and
- determine the winner
- inc the winner's score
- unset the plays
- bump the round id
- set plays to zero
- store that row back in DB
- notify all players:
- next round id
- did they win or lose
- their and other player's play
- current scores
- on notification failure
- unset that connection id from the game
- remove the connectionid -> gameid entry
*/

func main() {
	lambda.Start(Handler)
}
