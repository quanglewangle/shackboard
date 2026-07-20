package main

import (
	"testing"
	"time"

	"github.com/quanglewangle/shackboard/internal/adif"
	"github.com/quanglewangle/shackboard/internal/cluster"
	"github.com/quanglewangle/shackboard/internal/parkspots"
)

func TestDecorateSpotsWorkedBeforeFlip(t *testing.T) {
	idx := adif.NewIndex()
	spots := []cluster.Spot{
		{DXCall: "W1AW", Band: "20m"},
	}

	before := decorateSpots(spots, idx)
	if before[0].WorkedAny || before[0].WorkedBand {
		t.Fatalf("before upload: expected both false, got %+v", before[0])
	}

	idx.Replace([]adif.QSO{{Call: "W1AW", Band: "20m"}}, time.Now())

	after := decorateSpots(spots, idx)
	if !after[0].WorkedAny || !after[0].WorkedBand {
		t.Fatalf("after upload: expected both true, got %+v", after[0])
	}

	// Worked on a different band: WorkedAny true, WorkedBand false.
	spots[0].Band = "40m"
	mixed := decorateSpots(spots, idx)
	if !mixed[0].WorkedAny || mixed[0].WorkedBand {
		t.Fatalf("different band: expected WorkedAny=true WorkedBand=false, got %+v", mixed[0])
	}
}

func TestDecorateParkSpotsWorkedBeforeFlip(t *testing.T) {
	idx := adif.NewIndex()
	idx.Replace([]adif.QSO{{Call: "W7DLZ", Band: "20m"}}, time.Now())

	spots := []parkspots.Spot{{Activator: "W7DLZ", Band: "20m"}}
	decorated := decorateParkSpots(spots, idx)
	if !decorated[0].WorkedAny || !decorated[0].WorkedBand {
		t.Fatalf("expected both true for a worked activator, got %+v", decorated[0])
	}

	spots = []parkspots.Spot{{Activator: "UNHEARD", Band: "20m"}}
	decorated = decorateParkSpots(spots, idx)
	if decorated[0].WorkedAny || decorated[0].WorkedBand {
		t.Fatalf("expected both false for a never-worked activator, got %+v", decorated[0])
	}
}
