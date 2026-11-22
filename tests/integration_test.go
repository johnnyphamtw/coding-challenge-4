package tests

import (
	"encoding/json"
	"testing"

	"mini-delivery/internal/data"
	"mini-delivery/internal/engine"
)

func loadEngine(t *testing.T) *engine.Engine {
	t.Helper()
	ds, err := data.LoadAll("../data")
	if err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}
	return engine.New(ds)
}

// parseOutput parses a JSON output line into a map for flexible checking
func parseOutput(t *testing.T, output []byte) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}
	return result
}

func TestWorkedExample_Order0001(t *testing.T) {
	eng := loadEngine(t)
	input := []byte(`{"order_id":"order-0001","created_at":"2025-06-30","weight_g":1500,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)
	result := parseOutput(t, eng.ProcessOrder(input))

	assertEqual(t, result, "status", "quoted")
	assertEqual(t, result, "lane", "SAME_REGION")
	assertEqualNum(t, result, "chargeable_g", 1500)
	assertEqualNum(t, result, "base", 35000)
	assertEqualNum(t, result, "surcharge", 1050)
	assertEqualNum(t, result, "total", 36000)
	assertEqual(t, result, "pickup_date", "2025-06-30")
	assertEqual(t, result, "promised_date", "2025-07-02")
	assertNull(t, result, "distance_km")
}

func TestWorkedExample_Order0007_PostMerger(t *testing.T) {
	eng := loadEngine(t)
	input := []byte(`{"order_id":"order-0007","created_at":"2025-07-07","weight_g":1500,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)
	result := parseOutput(t, eng.ProcessOrder(input))

	assertEqual(t, result, "status", "quoted")
	assertEqual(t, result, "lane", "SAME_PROVINCE")
	assertEqualNum(t, result, "chargeable_g", 1500)
	assertEqualNum(t, result, "base", 26000)
	assertEqualNum(t, result, "surcharge", 0)
	assertEqualNum(t, result, "total", 26000)
	assertEqual(t, result, "pickup_date", "2025-07-07")
	assertEqual(t, result, "promised_date", "2025-07-08")
}

// === MANDATORY TEST 1: Backward compat - old orders keep their price ===
func TestBackwardCompat_OldOrderKeepsPrice(t *testing.T) {
	eng := loadEngine(t)
	// Pre-merger order: should use SAME_REGION (Bình Dương and HCM are different provinces, same region)
	input := []byte(`{"order_id":"bc-old-price","created_at":"2025-06-30","weight_g":1500,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)
	result := parseOutput(t, eng.ProcessOrder(input))

	// Even though merger makes them same province, pre-merger order keeps SAME_REGION
	assertEqual(t, result, "lane", "SAME_REGION")
	assertEqualNum(t, result, "base", 35000)
	assertEqualNum(t, result, "surcharge", 1050)
	assertEqualNum(t, result, "total", 36000)
}

// === MANDATORY TEST 2: Backward compat - old and new address give same quote ===
func TestBackwardCompat_OldNewAddressSameQuote(t *testing.T) {
	// Two separate engines (separate runs) to avoid capacity interference
	eng1 := loadEngine(t)
	eng2 := loadEngine(t)

	// Old-format address (with district)
	oldInput := []byte(`{"order_id":"bc-old-fmt","created_at":"2025-07-07","weight_g":1500,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)
	// New-format address (without district) - same physical location
	newInput := []byte(`{"order_id":"bc-new-fmt","created_at":"2025-07-07","weight_g":1500,"from":{"ward":"Thủ Dầu Một","province":"Hồ Chí Minh"},"to":{"ward":"Sài Gòn","province":"Hồ Chí Minh"}}`)

	oldResult := parseOutput(t, eng1.ProcessOrder(oldInput))
	newResult := parseOutput(t, eng2.ProcessOrder(newInput))

	// Should get same quote (except order_id)
	assertEqual(t, oldResult, "lane", newResult["lane"].(string))
	assertEqualNum(t, oldResult, "base", int(newResult["base"].(float64)))
	assertEqualNum(t, oldResult, "surcharge", int(newResult["surcharge"].(float64)))
	assertEqualNum(t, oldResult, "total", int(newResult["total"].(float64)))
}

// === MANDATORY TEST 3: Hub fills up ===
func TestHubOverflow(t *testing.T) {
	eng := loadEngine(t)
	// Hồ Chí Minh hub has 60kg = 60000g capacity pre-merger on 2025-06-28
	// Send three 25kg orders on the same day
	order1 := []byte(`{"order_id":"hub-1","created_at":"2025-06-28","weight_g":25000,"from":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"},"to":{"ward":"Tân Phong","district":"Quận 7","province":"Hồ Chí Minh"}}`)
	order2 := []byte(`{"order_id":"hub-2","created_at":"2025-06-28","weight_g":25000,"from":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"},"to":{"ward":"Tân Phong","district":"Quận 7","province":"Hồ Chí Minh"}}`)
	order3 := []byte(`{"order_id":"hub-3","created_at":"2025-06-28","weight_g":25000,"from":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"},"to":{"ward":"Tân Phong","district":"Quận 7","province":"Hồ Chí Minh"}}`)

	r1 := parseOutput(t, eng.ProcessOrder(order1))
	r2 := parseOutput(t, eng.ProcessOrder(order2))
	r3 := parseOutput(t, eng.ProcessOrder(order3))

	// First two fit on 06-28
	assertEqual(t, r1, "pickup_date", "2025-06-28")
	assertEqual(t, r2, "pickup_date", "2025-06-28")
	// Third spills to 06-29
	assertEqual(t, r3, "pickup_date", "2025-06-29")
}

// === MANDATORY TEST 4: Duplicate order ===
func TestDuplicateOrder(t *testing.T) {
	eng := loadEngine(t)
	input1 := []byte(`{"order_id":"dup-test","created_at":"2025-06-30","weight_g":1500,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)
	// Second time: different data but same order_id
	input2 := []byte(`{"order_id":"dup-test","created_at":"2025-07-07","weight_g":3000,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)

	r1 := parseOutput(t, eng.ProcessOrder(input1))
	r2 := parseOutput(t, eng.ProcessOrder(input2))

	// First is not duplicate
	assertBool(t, r1, "duplicate", false)
	// Second IS duplicate
	assertBool(t, r2, "duplicate", true)
	// Second returns first result's values (except duplicate flag)
	assertEqual(t, r2, "lane", "SAME_REGION")
	assertEqualNum(t, r2, "total", 36000)
}

// === MANDATORY TEST 5: Indirect route ===
func TestIndirectRoute(t *testing.T) {
	eng := loadEngine(t)
	// Bình Phước to Đồng Nai: direct 120km, through Bình Dương 85+25=110km
	// Distance rule active from 2025-09-01
	input := []byte(`{"order_id":"route-test","created_at":"2025-09-08","weight_g":1000,"from":{"ward":"Bình Phước","province":"Đồng Nai"},"to":{"ward":"Trấn Biên","province":"Đồng Nai"}}`)
	result := parseOutput(t, eng.ProcessOrder(input))

	assertEqual(t, result, "status", "quoted")
	// Should use the indirect route: 110km, not 120km
	assertEqualNum(t, result, "distance_km", 110)
}

// Helper assertions
func assertEqual(t *testing.T, result map[string]interface{}, key, expected string) {
	t.Helper()
	got, ok := result[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	if got != expected {
		t.Errorf("%s = %v, want %s", key, got, expected)
	}
}

func assertEqualNum(t *testing.T, result map[string]interface{}, key string, expected int) {
	t.Helper()
	got, ok := result[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	f, ok := got.(float64)
	if !ok {
		t.Errorf("%s is not a number: %v (%T)", key, got, got)
		return
	}
	if int(f) != expected {
		t.Errorf("%s = %v, want %d", key, got, expected)
	}
}

func assertNull(t *testing.T, result map[string]interface{}, key string) {
	t.Helper()
	got := result[key]
	if got != nil {
		t.Errorf("%s = %v, want null", key, got)
	}
}

func assertBool(t *testing.T, result map[string]interface{}, key string, expected bool) {
	t.Helper()
	got, ok := result[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	b, ok := got.(bool)
	if !ok {
		t.Errorf("%s is not a bool: %v", key, got)
		return
	}
	if b != expected {
		t.Errorf("%s = %v, want %v", key, got, expected)
	}
}
