package main

import (
	"errors"
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

// Package scoped variables aren't great but
// for lambda, it's probably not too terrible
var (
	ddbclient   *dynamodb.DynamoDB
	ddbtable    string
	apigwclient *apigatewaymanagementapi.ApiGatewayManagementApi
)

func Connect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$connect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	gameID, err := GetGame(e.RequestContext.ConnectionID)
	if err == nil {
		// TODO: send game state to connection
		fmt.Sprintf("need to send the state of game %s to client\n", gameID)
	}
	//StoreConnection(e.RequestContext.ConnectionID, "temp")
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func Disconnect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$disconnect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	// TODO
	// look up the game if there is one
	// remove the session from it
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
	}
	return nil
}

func Default(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$defaut: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)

	// send a message back to the connectionID
	// could return it inline but this is an example of sending to others
	SendMessage([]byte(e.Body), e.RequestContext.ConnectionID)

	// TODO handle the messages from clients

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

// GetGame will return a GameID given if connectID, if one exists.
func GetGame(connectionID string) (string, error) {

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
	item := ConnectionGame{}
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
	item := ConnectionGame{}
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

func main() {
	lambda.Start(Handler)
}
