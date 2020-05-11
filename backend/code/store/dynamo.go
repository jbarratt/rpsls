// Pacage store includes all the methods for persisting rock paper scissors games
package store

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/jbarratt/rpsls/backend/code/game"
)

// GameItem is for game status items
type GameItem struct {
	PK      string
	SK      string
	Type    string
	Round   int
	Plays   int
	P1Play  string
	P2Play  string
	P1Score int
	P2Score int
	P1ID    string
	P2ID    string
	GameID  string
}

// GameStore interface declares the
type GameStore interface {
	Load(string) (*game.Game, error)
	StoreNew(*game.Game) error
	StoreRound(*game.Game) error
	StorePlay(*game.GameContext) error
	StorePlayer(*game.GameContext) error
}

// Store stores the dynamo client and other metadata needed, like the table
type Store struct {
	d         *dynamodb.DynamoDB
	tableName string
}

// New creates a dynamo store
func New(d *dynamodb.DynamoDB, tableName string) *Store {
	return &Store{
		d:         d,
		tableName: tableName,
	}
}

// Load returns a populated game based on a gameID, or error if no game exists
func (s *Store) Load(gameID string) (*game.Game, error) {
	gi := GameItem{}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gameID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gameID)),
			},
		},
	}
	result, err := s.d.GetItem(input)
	if err != nil {
		fmt.Println("Error fetching game by ID")
		fmt.Println(err.Error())
		return nil, err
	}
	err = dynamodbattribute.UnmarshalMap(result.Item, &gi)
	if err != nil {
		fmt.Println("Error reading game record")
		return nil, err
	}
	g := game.NewGame()
	UpdateGameFromItem(g, &gi)
	return g, nil
}

// GameFromItem updates a game struct populated with data from the Dynamo GameItem
func UpdateGameFromItem(g *game.Game, gi *GameItem) {
	g.ID = gi.GameID
	g.Round = gi.Round
	g.PlayCount = gi.Plays
	g.Plays[0] = gi.P1Play
	g.Plays[1] = gi.P2Play
	g.PlayerID[0] = gi.P1ID
	g.PlayerID[1] = gi.P2ID
	g.Scores[0] = gi.P1Score
	g.Scores[1] = gi.P2Score
}

// UpdateItemFromGame updates a dynamo game item from the game struct
func UpdateItemFromGame(gi *GameItem, g *game.Game) {
	gi.GameID = g.ID
	gi.Round = g.Round
	gi.Plays = g.PlayCount
	gi.P1Play = g.Plays[0]
	gi.P2Play = g.Plays[1]
	gi.P1ID = g.PlayerID[0]
	gi.P2ID = g.PlayerID[1]
	gi.P1Score = g.Scores[0]
	gi.P2Score = g.Scores[1]
}

// StoreNew takes a Game and creates a new record for it
func (s *Store) StoreNew(g *game.Game) error {

	gi := &GameItem{}
	UpdateItemFromGame(gi, g)

	gi.PK = fmt.Sprintf("GAME#%s", g.ID)
	gi.SK = fmt.Sprintf("GAME#%s", g.ID)
	gi.Type = "GameItem"

	av, err := dynamodbattribute.MarshalMap(gi)
	if err != nil {
		fmt.Println("Got error marshalling gameitem:")
		fmt.Println(err.Error())
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(s.tableName),
	}

	_, err = s.d.PutItem(input)
	if err != nil {
		fmt.Printf("Error calling PutItem: %+v\n", input)
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// StorePlay takes a GameContext and stores the bits needed if a play has been made
// It updates the Game with the current status as well
func (s *Store) StorePlay(gc *game.GameContext) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":play": {
				S: aws.String(gc.Player.Play),
			},
			":count": {
				N: aws.String("1"),
			},
			":round": {
				N: aws.String(fmt.Sprintf("%d", gc.Game.Round)),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#pxplay": aws.String(fmt.Sprintf("P%dPlay", gc.Player.Number)),
			"#round":  aws.String("Round"),
		},
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gc.Game.ID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gc.Game.ID)),
			},
		},
		ConditionExpression: aws.String(fmt.Sprintf("#round = :round")),
		UpdateExpression:    aws.String(fmt.Sprintf("SET Plays = Plays + :count, #pxplay = :play")),
		ReturnValues:        aws.String("ALL_NEW"),
	}

	result, err := s.d.UpdateItem(input)
	if err != nil {
		fmt.Printf("got an error storing a dynamo play\n")
		fmt.Println(err.Error())
		return err
	}

	item := GameItem{}
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &item)
	if err != nil {
		fmt.Println("unmarshal error: unable to retrieve game values")
		fmt.Printf(err.Error())
		return err
	}
	UpdateGameFromItem(gc.Game, &item)

	return nil
}

// StorePlayer takes a GameContext and stores the bits needed for an added player
func (s *Store) StorePlayer(gc *game.GameContext) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(gc.Player.ID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#pxid": aws.String(fmt.Sprintf("P%dID", gc.Player.Number)),
		},
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gc.Game.ID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", gc.Game.ID)),
			},
		},
		UpdateExpression: aws.String(fmt.Sprintf("SET #pxid = :id")),
	}

	_, err := s.d.UpdateItem(input)
	if err != nil {
		fmt.Printf("got an error storing the player's ID\n")
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// StoreRound takes a Game and stores the next round
func (s *Store) StoreRound(g *game.Game) error {

	updateExpr := "SET P1Score = :p1score, P2Score = :p2score, Round = :round, Plays = :zero"

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":zero": {
				N: aws.String("0"),
			},
			":round": {
				N: aws.String(fmt.Sprintf("%d", g.Round)),
			},
			":p1score": {
				N: aws.String(fmt.Sprintf("%d", g.Scores[0])),
			},
			":p2score": {
				N: aws.String(fmt.Sprintf("%d", g.Scores[1])),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("GAME#%s", g.ID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("GAME#%s", g.ID)),
			},
		},
		ReturnValues:     aws.String("ALL_NEW"),
		UpdateExpression: aws.String(updateExpr),
	}

	result, err := s.d.UpdateItem(input)
	if err != nil {
		fmt.Printf("got an error storing the completed round in dynamo\n")
		fmt.Println(err.Error())
		return err
	}

	item := GameItem{}
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &item)
	if err != nil {
		fmt.Println("unmarshal error: unable to retrieve game values")
		fmt.Printf(err.Error())
		return err
	}
	UpdateGameFromItem(g, &item)

	return nil
}
