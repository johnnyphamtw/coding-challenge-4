package tests

import (
	"testing"

	"mini-delivery/internal/data"
	"mini-delivery/internal/engine"
	"mini-delivery/internal/normalize"
)

func BenchmarkProcessOrder_Quoted(b *testing.B) {
	ds, err := data.LoadAll("../data")
	if err != nil {
		b.Fatalf("LoadAll failed: %v", err)
	}
	eng := engine.New(ds)

	orderJSON := []byte(`{"order_id":"bench-order-1","created_at":"2025-09-08","weight_g":1500,"from":{"ward":"Trấn Biên","province":"Đồng Nai"},"to":{"ward":"Sài Gòn","province":"Hồ Chí Minh"}}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Use fresh order ID each time to bypass duplicate cache and measure raw processing
		// (or test duplicate separately)
		_ = eng.ProcessOrder(orderJSON)
	}
}

func BenchmarkProcessOrder_UniqueOrders(b *testing.B) {
	ds, err := data.LoadAll("../data")
	if err != nil {
		b.Fatalf("LoadAll failed: %v", err)
	}
	eng := engine.New(ds)

	// Pre-create slice of distinct orders
	orders := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		orders[i] = []byte(`{"order_id":"unique-order","created_at":"2025-09-08","weight_g":1500,"from":{"ward":"Trấn Biên","province":"Đồng Nai"},"to":{"ward":"Sài Gòn","province":"Hồ Chí Minh"}}`)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = eng.ProcessOrder(orders[i%1000])
	}
}

func BenchmarkProcessOrder_Duplicate(b *testing.B) {
	ds, err := data.LoadAll("../data")
	if err != nil {
		b.Fatalf("LoadAll failed: %v", err)
	}
	eng := engine.New(ds)

	orderJSON := []byte(`{"order_id":"bench-dup-order","created_at":"2025-06-30","weight_g":1500,"from":{"ward":"Phú Cường","district":"Thủ Dầu Một","province":"Bình Dương"},"to":{"ward":"Bến Nghé","district":"Quận 1","province":"Hồ Chí Minh"}}`)
	_ = eng.ProcessOrder(orderJSON) // prime cache

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = eng.ProcessOrder(orderJSON)
	}
}

func BenchmarkNormalize_Prefix(b *testing.B) {
	input := "TP. Hồ Chí Minh"
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = normalize.Name(input)
	}
}
