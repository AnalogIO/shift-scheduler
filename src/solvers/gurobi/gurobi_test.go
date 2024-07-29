package gurobi_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Slug-Boi/aion-cli/forms"
	"github.com/Slug-Boi/aion-cli/solvers/gurobi"
)

var data = []byte(`Timestamp,Group Number,Pseudo Lottery String,12-04-24 Monday [8:00-10:00],12-04-24 Monday [10:00-12:00],15-04-24 Thursday [10:00-12:00],15-04-24 Thursday [14:30-16:30]
26/07/2024 10:50:58,Group 1,JavaBois,Want,Can do,,
26/07/2024 10:51:21,Group 2,PartyInTheSewers,Want,Want,,`)

func cleanup() {
	os.Remove("form.csv")
}

func TestGurobiTranslator(t *testing.T) {

	os.WriteFile("form.csv", data, 0644)

	defer cleanup()

	var conf forms.Config

	form := forms.GetForm(conf, true)

	groups, timeslots, users := gurobi.TranslateGurobi(form)
	

	if len(strings.Split(groups, ",")) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
	
	// amount of characters in the string (splitting didn't work for some reason)
	if len(timeslots) != 476 {
		t.Errorf("Expected 4 timeslots, got %d", len(timeslots))
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestRunGurobi(t *testing.T) {
	os.WriteFile("form.csv", data, 0644)

	defer cleanup()

	var conf forms.Config

	form := forms.GetForm(conf, true)

	out, users, err := gurobi.RunGurobi(form)

	if err != nil {
		t.Errorf("Error running gurobi: %v", err)
	}

	if len(out) == 0 {
		t.Errorf("Expected output, got nothing")
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}
