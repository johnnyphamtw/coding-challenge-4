# Handoff Brief: Interactive Architecture & Live Walkthrough Web UI
**Project:** Coding Challenge #4: Merger Day at Mini Delivery/works  
**Repository:** [github.com/johnnyphamtw/coding-challenge-4](https://github.com/johnnyphamtw/coding-challenge-4)  
**Target:** Building a standalone presentation & demonstration Web UI (`cmd/web`)  

---

## 1. Executive Summary & Context

We have successfully engineered, verified, benchmarked, and documented the complete Delivery Pricing & Dispatch Engine in Go (1.23). The engine handles:
1. **Administrative Merger on 2025-07-01**: Vietnam consolidates 63 provinces into 34, removing the district level entirely. Pre-merger addresses with old province/district names must be supported perpetually without breaking historical quotes.
2. **Distance-Based Pricing on 2025-09-01**: Switch from regional lane pricing to Dijkstra shortest road path pricing for orders created on or after 2025-09-01.
3. **Capacity & Dispatch Saturation**: Hub-based dispatch calendar with non-working Sundays and dynamic hub resolution across merger day.
4. **Stepped Fuel Surcharges & Rounding**: 3 discrete roundings per quote.
5. **Extreme Scale**: Processed **1,000,000 orders in 5.50s (181,676 orders/sec)** on a single consumer laptop, meeting AC-09 (<60s) with a 10x safety margin.

The repository is clean, tested against all 19 official worked examples, and pushed to GitHub.

---

## 2. Objective for the Web UI

The challenge includes a **30% presentation component** with an evaluator panel, including a **10-minute live rule change test** where evaluators modify business rules on the fly.

We need an **interactive, highly visual presentation Web UI** that:
1. **Demos the System Architecture**: Shows evaluators that the engine is not a black box, but a clean, deterministic, multi-stage pipeline.
2. **Visualizes the Stage-by-Stage Order Lifecycle**: Like a visual Temporal / DAG workflow execution viewer, showing an order progressing through each stage with live input/output intermediate telemetry.
3. **Simulates "What-If" Live Rule Changes**: Allows tweaking parameters (fuel price, date shift, capacity limits, fee boards) live via a GUI sandbox to prove engine agility without recompilation.
4. **Features Zero Build Dependencies**: Runs directly from Go (`go run ./cmd/web`) using standard library `net/http` and `embed.FS` with modern frontend (Tailwind/DaisyUI via CDN or clean CSS, SVG pipeline graph, Lucide icons).

---

## 3. Existing Project Architecture & Key Files

```
├── cmd/
│   ├── engine/main.go          # Streaming CLI entrypoint (stdin -> stdout)
│   └── loadgen/main.go         # 1M synthetic order generator
├── data/
│   ├── capacity.json           # Daily hub capacity & Sunday closures
│   ├── fees.json               # Temporal base fees & rate boards
│   ├── fuel-prices.json        # Historical diesel price boards
│   ├── mapping.csv             # 63 -> 34 province administrative mapping
│   ├── provinces.json          # Provinces, regions, wards
│   ├── regions.json            # North/Central/South region groupings
│   └── roads.json              # Road distance graph (provinces + hubs)
├── docs/
│   ├── COLLABORATION_LOG.md    # Human-AI co-design log
│   ├── DELIVERABLES_AND_EXPECTED_OUTCOMES.md # 70-point rubric breakdown
│   ├── SCALE_BENCHMARK.md      # 1M scale test & O(N) optimization report
│   └── TECHNICAL_DECISIONS.md  # 10 formal ADRs
├── examples/
│   └── worked-examples.json    # 19 official scenario blocks (22 orders)
├── internal/
│   ├── data/
│   │   ├── board.go            # Generic BoardLookup[T] binary search
│   │   ├── loader.go           # Ingestion & OldToNewProv index
│   │   └── routing.go          # Dijkstra shortest path with min-heap
│   ├── dispatch/
│   │   └── dispatch.go         # Hub calendar & firstAvailable fast-forward
│   ├── engine/
│   │   └── engine.go           # Orchestrator & deduplication cache
│   ├── models/
│   │   └── models.go           # Strongly typed domain models
│   ├── normalize/
│   │   └── normalize.go        # Vietnamese unicode normalization & prefix stripper
│   └── pricing/
│       └── pricing.go          # Lane classification, fee tiers, stepped surcharge
├── tests/
│   ├── integration_test.go     # 5 mandatory edge cases
│   └── worked_examples_test.go # 19 worked example tests
├── run.sh                      # Shell wrapper for engine
├── test-examples.sh            # Runs worked example test suite
└── test-scale.sh               # 1M scale benchmark runner
```

---

## 4. Web UI Specification & Requirements

### A. Architecture Walkthrough Pipeline (The Core Visual)
When an order is submitted or selected, display an interactive pipeline diagram showing 6 animated/highlighted stages with before/after state:

```
[1. Ingest & Deduplicate] 
       │ (RawOrder -> SHA256 Hash -> Cache Check)
       ▼
[2. Temporal Address Resolution]
       │ (Normalize text, 63 -> 34 mapping, drop districts if >= 2025-07-01)
       ▼
[3. Lane & Policy Classifier]
       │ (LOCAL / SAME_PROVINCE / SAME_REGION / CROSS_REGION)
       │ (Check: Does created_at >= 2025-09-01 activate Distance-Based rule?)
       ▼
[4. Routing & Vehicle Selection]
       │ (Weight > 15kg or dim > 60cm ? TRUCK : MOTORBIKE)
       │ (Dijkstra shortest path on road network graph: nodes, hops, km)
       ▼
[5. Pricing & Fuel Surcharge]
       │ (Chargeable weight = max(actual, volumetric), tiered block fees)
       │ (Stepped surcharge: base fuel 18,000, step 1,000 VND -> +3%, 3 roundings)
       ▼
[6. Hub Capacity & Dispatch Calendar]
       │ (Origin hub selection, Sunday skip, capacity reservation, promised date)
       ▼
[Final Quote / Rejection Badge]
```

### B. Feature Set
1. **Preset Picker**: Dropdown with official worked examples:
   - `order-0001`: Pre-merger Same Region quote
   - `order-0002`: Local delivery with prefix normalization (`Q.1`, `TP Hồ Chí Minh`)
   - `order-0004`: Rejected (`NO_PRICE_BOARD`, date before 2025-06-25)
   - `order-0006`: Volumetric weight override ($30 \times 20 \times 20 / 5000 = 2.4\,\text{kg}$)
   - `order-0007`: Post-merger administrative shift
   - `order-0009`: Post-September distance-based pricing (Dijkstra active)
   - `order-0011 to 0013`: Hub capacity saturation & Sunday rollover
   - `order-0015`: Rejected (`TOO_HEAVY`, > 20kg at remote hub)
   - `order-0018`: Duplicate order detection
   - `order-0019`: Indirect route via intermediate hubs
2. **Interactive Order Builder**:
   - Date picker, weight (g), dimensions (L/W/H cm).
   - From / To address fields (Province, District, Ward).
   - Instant "Quote" button.
3. **Inspection Drawer / Modal**:
   - Clicking any stage in the pipeline reveals the exact internal variables, formulas, and decisions made at that stage (e.g. Dijkstra hop list with distances, fuel surcharge step calculation, capacity slots remaining).
4. **Live "What-If" Rule Change Sandbox**:
   - Toggle to override current date to test merger boundary (e.g. 2025-06-30 vs 2025-07-01 vs 2025-09-01).
   - Slider to change Diesel Fuel Price (e.g., test 18,000 vs 22,000 VND).
   - Toggle to inject a custom capacity limit or block a hub.
5. **Dijkstra Road Graph Visualizer**:
   - A clean 2D canvas/SVG showing Vietnam's major hubs and road edges, highlighting the shortest route discovered for distance-based orders.
6. **Scale & Performance Stat Banner**:
   - Real-time quote latency counter (sub-millisecond in Go).
   - Callout badge: *"Tested with 1,000,000 orders in 5.50s (181,676 orders/sec)"*.

---

## 5. Technical Implementation Guidelines

1. **Location**: Place the server code in `cmd/web/main.go` and static assets in `cmd/web/static/` (using `//go:embed static/*`).
2. **Reuse Existing Go Packages**:
   - Import `github.com/johnnyphamtw/coding-challenge-4/internal/engine`, `data`, `pricing`, `dispatch`, `models`, `normalize`.
   - Modify or wrap `internal/engine` to return a `QuoteTelemetry` or `PipelineTrace` struct alongside `QuoteOutput` so intermediate steps are captured for visualization without altering CLI engine performance.
3. **Endpoints**:
   - `GET /`: Serves the Single-Page Application (HTML/JS/CSS).
   - `GET /api/presets`: Returns list of scenarios from `examples/worked-examples.json`.
   - `POST /api/quote`: Accepts `RawOrder` JSON, executes pipeline, returns `{ quote, trace }`.
   - `GET /api/graph`: Returns nodes and edges from `data/roads.json` for the road map.
   - `POST /api/reset`: Resets dispatch hub state and deduplication cache.
4. **No External Runtime Dependencies**:
   - Do NOT require Node.js, npm, or webpack. Keep it runnable with `go run ./cmd/web`.
   - Use CDN for UI utilities (Tailwind CSS, DaisyUI, Lucide icons, or Canvas-confetti for successful quotes).
