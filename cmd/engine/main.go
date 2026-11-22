package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"mini-delivery/internal/data"
	"mini-delivery/internal/engine"
)

func main() {
	dataDir := "./data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	ds, err := data.LoadAll(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading data: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(ds)

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, len(buf))

	writer := bufio.NewWriterSize(os.Stdout, 256*1024)
	defer writer.Flush()

	count := 0
	startTime := time.Now()

	for scanner.Scan() {
		result := eng.ProcessOrder(scanner.Bytes())
		writer.Write(result)
		writer.WriteByte('\n')

		count++
		if count%100000 == 0 {
			elapsed := time.Since(startTime).Seconds()
			rate := float64(count) / elapsed
			fmt.Fprintf(os.Stderr, "[Engine] Processed %d orders in %.1fs (%.0f orders/sec)\n", count, elapsed, rate)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	if count >= 100000 {
		totalElapsed := time.Since(startTime).Seconds()
		fmt.Fprintf(os.Stderr, "[Engine] Completed %d orders in %.2fs (%.0f orders/sec)\n", count, totalElapsed, float64(count)/totalElapsed)
	}
}
