package pricing

import (
	"testing"
	"mini-delivery/internal/models"
)

func TestCalculateChargeableWeight(t *testing.T) {
	// No dimensions: use weight
	if got := CalculateChargeableWeight(1500, nil, nil, nil); got != 1500 {
		t.Errorf("No dims: got %d, want 1500", got)
	}

	// With dimensions: 30x20x20/5 = 2400 > 1500
	l, w, h := 30, 20, 20
	if got := CalculateChargeableWeight(1500, &l, &w, &h); got != 2400 {
		t.Errorf("30x20x20: got %d, want 2400", got)
	}

	// Volume less than weight
	l2, w2, h2 := 10, 10, 5
	if got := CalculateChargeableWeight(1500, &l2, &w2, &h2); got != 1500 {
		t.Errorf("10x10x5: got %d, want 1500", got)
	}

	// Volume with rounding up: 7x7x7 = 343/5 = 68.6 → 69
	l3, w3, h3 := 7, 7, 7
	if got := CalculateChargeableWeight(50, &l3, &w3, &h3); got != 69 {
		t.Errorf("7x7x7: got %d, want 69", got)
	}
}

func TestCalculateBaseFee(t *testing.T) {
	config := models.LaneConfig{
		FirstBlock: 30000, FirstBlockG: 1000,
		PerExtraBlock: 5000, ExtraBlockG: 500,
	}
	// 1500g = 1000 first + 500 extra = 1 extra block
	if got := CalculateBaseFee(config, 1500); got != 35000 {
		t.Errorf("1500g: got %d, want 35000", got)
	}
	// 1000g = exactly first block
	if got := CalculateBaseFee(config, 1000); got != 30000 {
		t.Errorf("1000g: got %d, want 30000", got)
	}
	// 2200g = 1000 + 1200 = 3 started 500g blocks
	if got := CalculateBaseFee(config, 2200); got != 45000 {
		t.Errorf("2200g: got %d, want 45000", got)
	}
}

func TestCalculateSurcharge(t *testing.T) {
	// 1 step above reference: 3% of 35000 = 1050
	got := CalculateSurcharge(35000, 19707, 18000, 3, 15, 1000)
	if got != 1050 {
		t.Errorf("1 step: got %d, want 1050", got)
	}
	// Below reference: 0
	got = CalculateSurcharge(26000, 19906, 20000, 3, 15, 1000)
	if got != 0 {
		t.Errorf("below ref: got %d, want 0", got)
	}
	// 2 steps: 6% of 35000 = 2100
	got = CalculateSurcharge(35000, 20480, 18000, 3, 15, 1000)
	if got != 2100 {
		t.Errorf("2 steps: got %d, want 2100", got)
	}
}

func TestCalculateTotal(t *testing.T) {
	// 36050 → 36000
	if got := CalculateTotal(35000, 1050); got != 36000 {
		t.Errorf("36050: got %d, want 36000", got)
	}
	// 37100 → 37000
	if got := CalculateTotal(35000, 2100); got != 37000 {
		t.Errorf("37100: got %d, want 37000", got)
	}
	// Exactly on boundary
	if got := CalculateTotal(35000, 0); got != 35000 {
		t.Errorf("35000: got %d, want 35000", got)
	}
	// Half up: 500 → rounds up
	if got := CalculateTotal(35000, 500); got != 36000 {
		t.Errorf("35500: got %d, want 36000", got)
	}
}
