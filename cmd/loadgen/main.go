package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type OldAddress struct {
	Province string
	District string
	Ward     string
}

type NewAddress struct {
	Province string
	Ward     string
}

func main() {
	count := flag.Int("n", 1000000, "number of orders to generate")
	dataDir := flag.String("data", "./data", "path to data directory")
	outPath := flag.String("out", "load/orders-1m.jsonl", "output file path")
	dupRate := flag.Float64("dup", 0.02, "fraction of duplicate orders (0.0 to 1.0)")
	flag.Parse()

	// Load mapping.csv to get authentic address candidates
	mappingPath := filepath.Join(*dataDir, "mapping.csv")
	f, err := os.Open(mappingPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening mapping file %s: %v\n", mappingPath, err)
		os.Exit(1)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading mapping file: %v\n", err)
		os.Exit(1)
	}

	header := records[0]
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[h] = i
	}

	var oldAddrs []OldAddress
	var newAddrs []NewAddress

	for _, row := range records[1:] {
		oldAddrs = append(oldAddrs, OldAddress{
			Province: row[colIdx["old_province"]],
			District: row[colIdx["old_district"]],
			Ward:     row[colIdx["old_ward"]],
		})
		newAddrs = append(newAddrs, NewAddress{
			Province: row[colIdx["new_province"]],
			Ward:     row[colIdx["new_ward"]],
		})
	}

	if len(oldAddrs) == 0 || len(newAddrs) == 0 {
		fmt.Fprintf(os.Stderr, "No valid addresses found in %s\n", mappingPath)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	outFile, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	defer outFile.Close()

	writer := bufio.NewWriterSize(outFile, 1024*1024) // 1MB write buffer
	defer writer.Flush()

	rng := rand.New(rand.NewSource(42)) // deterministic seed

	weights := []int{500, 800, 1000, 1200, 1500, 2000, 2200, 3000, 5000, 10000, 15000, 25000}
	dimensions := [][3]int{
		{20, 15, 10},
		{30, 20, 20},
		{25, 25, 20},
		{40, 30, 20},
	}

	startTime := time.Now()
	var prevOrderIDs []string
	maxRecent := 1000
	dupCount := 0

	for i := 1; i <= *count; i++ {
		var orderID string

		// Simulate duplicate order
		if len(prevOrderIDs) > 0 && rng.Float64() < *dupRate {
			orderID = prevOrderIDs[rng.Intn(len(prevOrderIDs))]
			dupCount++
		} else {
			orderID = fmt.Sprintf("ord-%07d", i)
			if len(prevOrderIDs) < maxRecent {
				prevOrderIDs = append(prevOrderIDs, orderID)
			} else {
				prevOrderIDs[rng.Intn(maxRecent)] = orderID
			}
		}

		weight := weights[rng.Intn(len(weights))]

		// Address format: 50% new format (post-merger), 50% legacy old format
		var fromStr, toStr string
		if rng.Intn(2) == 0 {
			a := newAddrs[rng.Intn(len(newAddrs))]
			fromStr = fmt.Sprintf(`"from":{"ward":%q,"province":%q}`, a.Ward, a.Province)
		} else {
			a := oldAddrs[rng.Intn(len(oldAddrs))]
			fromStr = fmt.Sprintf(`"from":{"ward":%q,"district":%q,"province":%q}`, a.Ward, a.District, a.Province)
		}

		if rng.Intn(2) == 0 {
			a := newAddrs[rng.Intn(len(newAddrs))]
			toStr = fmt.Sprintf(`"to":{"ward":%q,"province":%q}`, a.Ward, a.Province)
		} else {
			a := oldAddrs[rng.Intn(len(oldAddrs))]
			toStr = fmt.Sprintf(`"to":{"ward":%q,"district":%q,"province":%q}`, a.Ward, a.District, a.Province)
		}

		// Dates: 11.11 sale day (2025-11-11)
		dateStr := "2025-11-11"

		// 20% of orders have dimensions
		var dimStr string
		if rng.Intn(5) == 0 {
			dim := dimensions[rng.Intn(len(dimensions))]
			dimStr = fmt.Sprintf(`,"length_cm":%d,"width_cm":%d,"height_cm":%d`, dim[0], dim[1], dim[2])
		}

		fmt.Fprintf(writer, `{"order_id":%q,"created_at":%q,"weight_g":%d,%s,%s%s}`+"\n",
			orderID, dateStr, weight, fromStr, toStr, dimStr)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Generated %d orders (%d duplicates) to %s in %v (%.0f orders/sec)\n",
		*count, dupCount, *outPath, elapsed, float64(*count)/elapsed.Seconds())
}
