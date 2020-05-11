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
	p1 := NewGameContext("first", g)
	p1.AssignPlayer()
	p2 := NewGameContext("second", g)
	p2.AssignPlayer()

	p1.Play("rock")
	err := g.AdvanceGame()
	if err == nil {
		t.Errorf("game should not be advanceable with one play")
	}
	p2.Play("scissors")
	err = g.AdvanceGame()
	if err != nil {
		t.Errorf("valid game play should not cause an error: %s\n%+v", err, g)
	}
	if p1.Game.Scores[0] != 1 {
		t.Errorf("player 1 should have 1 point: %+v", g)
	}
	if p1.Game.Scores[1] != 0 {
		t.Errorf("player 2 should have 0 points: %+v", g)
	}
}
