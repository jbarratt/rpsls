// Pacage store includes all the methods for persisting rock paper scissors games
package store

import (
	"fmt"
	"time"

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
	Players map[string]PlayerItem
	GameID  string
	Expires int
}

type PlayerItem struct {
	ID      string
	Address string
	Play    string
	Round   int
	Score   int
	Expires int
}

// GameStore interface declares the
type GameStore interface {
	Load(string) (*game.Game, error)
	StoreAll(*game.Game) error
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
	for id, p := range gi.Players {
		// Check to see if this game already has that player
		gp, found := g.Players[id]
		if found {
			// need to update any player values
			gp.Address = p.Address
			gp.Play = p.Play
			gp.Round = p.Round
			gp.Score = p.Score
		} else {
			// Need to add a player for this game entry
			g.Players[id] = &game.Player{
				ID:      id,
				Address: p.Address,
				Round:   p.Round,
				Score:   p.Score,
				Play:    p.Play}
		}
	}
}

// UpdateItemFromGame updates a dynamo game item from the game struct
func UpdateItemFromGame(gi *GameItem, g *game.Game) {
	gi.GameID = g.ID
	gi.Round = g.Round
	gi.Plays = g.PlayCount
	for id, gp := range g.Players {
		// Check to see if this GameItem already has that player
		gip, found := gi.Players[id]
		if found {
			// need to update any player values
			gip.Address = gp.Address
			gip.Play = gp.Play
			gip.Round = gp.Round
			gip.Score = gp.Score
		} else {
			// Need to add a player for this game entry
			gi.Players[id] = PlayerItem{
				ID:      id,
				Address: gp.Address,
				Round:   gp.Round,
				Play:    gp.Play,
				Score:   gp.Score,
			}
		}
	}
}

// StoreAll takes a Game and persists the entire thing
// Useful when creating a new game or large operations like round updates
func (s *Store) StoreAll(g *game.Game) error {

	gi := &GameItem{}
	gi.Players = make(map[string]PlayerItem)

	UpdateItemFromGame(gi, g)

	gi.PK = fmt.Sprintf("GAME#%s", g.ID)
	gi.SK = fmt.Sprintf("GAME#%s", g.ID)
	gi.Type = "GameItem"
	// Set the TTL on creation and on every round
	// Does not need to be updated during other gameplay
	gi.Expires = time.Now().Unix() + 2, 592, 000 // TTL: expire in 30 days

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
				S: aws.String(gc.ActingPlayer.Play),
			},
			":count": {
				N: aws.String("1"),
			},
			":round": {
				N: aws.String(fmt.Sprintf("%d", gc.Game.Round)),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#pxid":  aws.String(gc.ActingPlayer.ID),
			"#round": aws.String("Round"),
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
		ConditionExpression: aws.String(fmt.Sprintf("#round = :round and Players.#pxid.Round < :round")),
		UpdateExpression:    aws.String(fmt.Sprintf("SET Plays = Plays + :count, Players.#pxid.Play = :play, Players.#pxid.Round = :round")),
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

	gi := &GameItem{}
	gi.Players = make(map[string]PlayerItem)
	UpdateItemFromGame(gi, gc.Game)

	pv, err := dynamodbattribute.MarshalMap(gi.Players[gc.ActingPlayer.ID])
	if err != nil {
		fmt.Printf("Unable to marshal player object: %s", err)
		return err
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":player": {
				M: pv,
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#pxid":    aws.String(gc.ActingPlayer.ID),
			"#players": aws.String("Players"),
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
		UpdateExpression: aws.String(fmt.Sprintf("SET #players.#pxid = :player")),
	}

	_, err = s.d.UpdateItem(input)
	if err != nil {
		fmt.Printf("got an error storing the player's ID: %+v\n", input)
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// StoreRound takes a Game and stores the next round
// For now takes a tiny risk of a race condition updating non-essential data, by uploading the whole
// item. (e.g. could blow out another player's connection if it changed at the exact wrong time.)
func (s *Store) StoreRound(g *game.Game) error {

	// given that the whole record is being stored, this method works
	err := s.StoreAll(g)
	if err != nil {
		fmt.Printf("Got an error when trying to store a completed round: %s", err)
		return err
	}
	return nil
}
