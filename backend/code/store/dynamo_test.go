package store

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/jbarratt/rpsls/backend/code/game"
)

func TestGameStore(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		t.Fatalf("unable to create session: %s", err)
	}

	s := New(dynamodb.New(sess), os.Getenv("TABLE_NAME"))
	g := game.NewGame()
	err = s.StoreNew(g)
	if err != nil {
		t.Fatalf("unable to store game: %s %+v %+v", err, s, g)
	}

	// Simulate first player joining
	p1gc := game.NewGameContext("first", g)
	p1gc.AssignPlayer()

	err = s.StorePlayer(p1gc)
	if err != nil {
		t.Fatalf("unable to store player one: %s", err)
	}

	// Simulate second player joining
	p2gc := game.NewGameContext("second", g)
	p2gc.AssignPlayer()

	err = s.StorePlayer(p2gc)
	if err != nil {
		t.Fatalf("unable to store player two: %s", err)
	}

	// simulate first player reconnecting
	g2, err := s.Load(g.ID)
	if err != nil {
		t.Fatalf("unable to load game from ID: %s", err)
	}
	p2gc2 := game.NewGameContext("second", g2)

	p1gc.Play("rock")
	err = s.StorePlay(p1gc)
	if err != nil {
		t.Errorf("couldn't store player 1's play: %s", err)
	}

	err = p1gc.Game.AdvanceGame()
	if err == nil {
		t.Errorf("game should not be advanceable with one play")
	}

	if p1gc.Game.PlayCount != 1 {
		t.Errorf("PlayCount should be 1: %+v", p1gc.Game)
	}

	p2gc2.Play("scissors")
	err = s.StorePlay(p2gc2)
	if err != nil {
		t.Errorf("couldn't store player 2's play: %s", err)
	}

	err = p2gc2.Game.AdvanceGame()
	if err != nil {
		t.Errorf("game should be advancable: %s\n%+v", err, p2gc2.Game)
	}

	if p2gc2.Game.Scores[0] != 1 {
		t.Errorf("player 1 should have 1 point: %+v", p2gc2.Game)
	}
	if p2gc2.Game.Scores[1] != 0 {
		t.Errorf("player 2 should have 0 points: %+v", p2gc2.Game)
	}

	err = s.StoreRound(p2gc2.Game)
	if err != nil {
		t.Errorf("should be able to advance round: %s\n%+v", err, p2gc2)
	}

	if p2gc2.Game.Round != 2 {
		t.Errorf("round should have advanced: %s\n%+v\n%+v", err, p2gc2.Game, p2gc2.Player)
	}
}
