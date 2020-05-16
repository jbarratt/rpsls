package game

import (
	"strings"
	"testing"
)

func TestBeats(t *testing.T) {
	beats, how := Beats("rock", "paper")
	if beats {
		t.Errorf("rock should not beat paper")
	}
	beats, how = Beats("paper", "rock")
	if !beats {
		t.Errorf("paper should beat rock")
	}
	if !strings.Contains(how, "cover") {
		t.Errorf("paper should cover rock")
	}
}

func TestGame(t *testing.T) {
	g := NewGame()
	p1, err := NewGameContext("first", "1addr", g)
	if err != nil {
		t.Errorf("error creating game context with player (p1)")
	}
	p2, err := NewGameContext("second", "2addr", g)
	if err != nil {
		t.Errorf("error creating game context with player (p2)")
	}
	_, err = NewGameContext("third", "3addr", g)
	if err == nil {
		t.Errorf("should not have been able to add a third player")
	}

	p1.Play("rock")
	err = g.AdvanceGame()
	if err == nil {
		t.Errorf("game should not be advanceable with one play")
	}
	p2.Play("scissors")
	err = g.AdvanceGame()
	if err != nil {
		t.Errorf("valid game play should not cause an error: %s\n%+v", err, g)
	}

	if p1.ActingPlayer.Score != 1 {
		t.Errorf("player 1 should have 1 point: %+v", g)
	}

	if p2.ActingPlayer.Score != 0 {
		t.Errorf("player 2 should have 0 points: %+v", g)
	}

	p2.Play("scissors")
	p2.Play("scissors")
	err = g.AdvanceGame()
	if err == nil {
		t.Errorf("should not be able to advance game if one player makes multiple plays")
	}
}
