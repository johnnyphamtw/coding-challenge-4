package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mini-delivery/internal/data"
	"mini-delivery/internal/engine"
	"mini-delivery/internal/models"
)

type ExampleBlock struct {
	Name     string               `json:"name"`
	Orders   []json.RawMessage    `json:"orders"`
	Expected []models.QuoteOutput `json:"expected"`
}

func TestAllWorkedExamples(t *testing.T) {
	ds, err := data.LoadAll("../data")
	if err != nil {
		t.Fatalf("Failed to load reference data: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("..", "examples", "worked-examples.json"))
	if err != nil {
		t.Fatalf("Failed to read worked-examples.json: %v", err)
	}

	var blocks []ExampleBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		t.Fatalf("Failed to parse worked-examples.json: %v", err)
	}

	for _, block := range blocks {
		t.Run(block.Name, func(t *testing.T) {
			// Each block starts from fresh engine with empty hubs
			eng := engine.New(ds)

			if len(block.Orders) != len(block.Expected) {
				t.Fatalf("Block %s: order count (%d) != expected count (%d)", block.Name, len(block.Orders), len(block.Expected))
			}

			for i, rawOrder := range block.Orders {
				exp := block.Expected[i]
				outBytes := eng.ProcessOrder(rawOrder)

				var got models.QuoteOutput
				if err := json.Unmarshal(outBytes, &got); err != nil {
					t.Fatalf("Failed to parse engine output: %v, raw: %s", err, string(outBytes))
				}

				if !reflect.DeepEqual(got, exp) {
					gotJSON, _ := json.Marshal(got)
					expJSON, _ := json.Marshal(exp)
					t.Errorf("Mismatch in block %q, order %d (%s):\nGot:      %s\nExpected: %s",
						block.Name, i, exp.OrderID, string(gotJSON), string(expJSON))
				}
			}
		})
	}
}
