package data

import (
	"testing"

	"mini-delivery/internal/models"
)

func TestShortestPath(t *testing.T) {
	rg := &RoadGraph{
		Adj: map[string][]Edge{
			"a": {
				{To: "b", Dist: 10},
				{To: "c", Dist: 30},
			},
			"b": {
				{To: "a", Dist: 10},
				{To: "c", Dist: 15},
				{To: "d", Dist: 50},
			},
			"c": {
				{To: "a", Dist: 30},
				{To: "b", Dist: 15},
				{To: "d", Dist: 10},
			},
			"d": {
				{To: "b", Dist: 50},
				{To: "c", Dist: 10},
			},
			"isolated": {},
		},
		Cache: make(map[string]int),
	}

	// Same depot
	if d := rg.ShortestPath("a", "a"); d != 0 {
		t.Errorf("expected 0 for same depot, got %d", d)
	}

	// Route a -> c via b is 10 + 15 = 25 (better than direct 30)
	if d := rg.ShortestPath("a", "c"); d != 25 {
		t.Errorf("expected 25 for a -> c, got %d", d)
	}

	// Route c -> a should hit cache (symmetric key "a|c")
	if d := rg.ShortestPath("c", "a"); d != 25 {
		t.Errorf("expected 25 for c -> a, got %d", d)
	}
	if rg.Cache["a|c"] != 25 {
		t.Errorf("expected cache key 'a|c' to be 25, got %d", rg.Cache["a|c"])
	}

	// Route a -> d: a -> b (10) -> c (15) -> d (10) = 35
	if d := rg.ShortestPath("a", "d"); d != 35 {
		t.Errorf("expected 35 for a -> d, got %d", d)
	}

	// Disconnected
	if d := rg.ShortestPath("a", "isolated"); d != -1 {
		t.Errorf("expected -1 for isolated node, got %d", d)
	}

	// Unknown node
	if d := rg.ShortestPath("unknown", "a"); d != -1 {
		t.Errorf("expected -1 for unknown start node, got %d", d)
	}
	if d := rg.ShortestPath("a", "unknown"); d != -1 {
		t.Errorf("expected -1 for unknown end node, got %d", d)
	}
}

func TestDetermineVehicle(t *testing.T) {
	if v := DetermineVehicle(50, 50); v != models.VehicleMotorbike {
		t.Errorf("expected motorbike for 50km with max 50km, got %v", v)
	}
	if v := DetermineVehicle(49, 50); v != models.VehicleMotorbike {
		t.Errorf("expected motorbike for 49km with max 50km, got %v", v)
	}
	if v := DetermineVehicle(51, 50); v != models.VehicleTruck {
		t.Errorf("expected truck for 51km with max 50km, got %v", v)
	}
}

func TestExtraDays(t *testing.T) {
	if days := ExtraDays(100, 0); days != 0 {
		t.Errorf("expected 0 extra days for non-positive extraDayKM, got %d", days)
	}
	if days := ExtraDays(100, -10); days != 0 {
		t.Errorf("expected 0 extra days for negative extraDayKM, got %d", days)
	}
	if days := ExtraDays(450, 200); days != 2 {
		t.Errorf("expected 2 extra days for 450km / 200km, got %d", days)
	}
	if days := ExtraDays(150, 200); days != 0 {
		t.Errorf("expected 0 extra days for 150km / 200km, got %d", days)
	}
}
