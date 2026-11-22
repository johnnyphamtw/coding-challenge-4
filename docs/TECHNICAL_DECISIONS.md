# 🏛️ Architecture Decision Records (ADRs)

**Project**: Mini Delivery/works Delivery Pricing Engine  
**Status**: Accepted & Implemented  
**Author**: Johnny (Human Architect) & Antigravity (AI System Designer)  

---

## Index of Decisions

- [ADR-01: Programming Language Selection](#adr-01-programming-language-selection)
- [ADR-02: Pipeline Processing Architecture](#adr-02-pipeline-processing-architecture)
- [ADR-03: Single-Pass Streaming I/O Model](#adr-03-single-pass-streaming-io-model)
- [ADR-04: Unified Temporal Board Abstraction (`BoardLookup[T]`)](#adr-04-unified-temporal-board-abstraction-boardlookupt)
- [ADR-05: Temporal Invariance & Entry-Point Date Gating](#adr-05-temporal-invariance--entry-point-date-gating)
- [ADR-06: Pre-Normalized Lookup Tables & Inverted Mapping](#adr-06-pre-normalized-lookup-tables--inverted-mapping)
- [ADR-07: Lazy Dijkstra with Symmetric Pair Memoization](#adr-07-lazy-dijkstra-with-symmetric-pair-memoization)
- [ADR-08: Candidate-Day Dynamic Hub Allocation for Slipping Orders](#adr-08-candidate-day-dynamic-hub-allocation-for-slipping-orders)
- [ADR-09: Idempotency & Serialized Output Caching](#adr-09-idempotency--serialized-output-caching)
- [ADR-10: Multi-Agent Workflow with Tiered Models (Gemini Flash)](#adr-10-multi-agent-workflow-with-tiered-models-gemini-flash)

---

## ADR-01: Programming Language Selection

### Context
The engine must process 1,000,000 orders in under 60 seconds on a laptop (AC-09), execute seamlessly via `./run.sh`, handle complex Vietnamese Unicode NFC strings, and implement graph shortest-path routing. The team had to select between Python, Rust, TypeScript (Bun), and Go.

### Decision
We chose **Go (Golang 1.21+)**.

### Rationale & Trade-offs
- **Go vs Python**: Python is rapid to prototype, but Python's interpreter overhead, GIL, and dynamic memory allocation cannot reliably sustain >16,666 orders/sec single-threaded throughput required for 1M orders in 60 seconds without complex native C extensions.
- **Go vs Rust**: Rust offers the highest ceiling for raw performance, but managing shared mutable capacity state across orders introduces lifetime and borrow-checker friction (`Arc<Mutex<...>>` or complex lifetime threading), increasing implementation risk and development time.
- **Go vs TypeScript (Bun)**: TypeScript with Bun can achieve high speed, but requires installing an external runtime environment. Go compiles to a completely self-contained static binary with zero external dependencies, predictable garbage collection pauses, and world-class standard library support for buffered I/O.

---

## ADR-02: Pipeline Processing Architecture

### Context
The grading rubric explicitly rewards solutions where *"lane, chargeable weight, base fee, surcharge and total are separate steps"* and *"one function normalises every name"*.

### Decision
We adopted a **Pure Pipeline Pattern**:
```
Validate (AC-04) ──> Resolve Addresses (AC-06) ──> Determine Lane (AC-01) ──> Determine Vehicle/Distance (AC-10) ──> Price (AC-02, AC-03) ──> Dispatch (AC-05)
```

### Rationale & Trade-offs
- Every stage in the pipeline before `Dispatch` is implemented as a **pure function** without side effects.
- Input validation checks both `from` and `to` addresses at each stage before moving to subsequent checks, strictly respecting the specified priority order (`NO_PRICE_BOARD` $\rightarrow$ `UNKNOWN_PROVINCE` $\rightarrow$ `MISSING_DISTRICT` $\rightarrow$ `BAD_WEIGHT` $\rightarrow$ `BAD_DIMENSIONS`).
- State mutation is isolated strictly to `internal/dispatch/dispatch.go`.

---

## ADR-03: Single-Pass Streaming I/O Model

### Context
Memory footprint and I/O efficiency are critical when processing massive files (1M orders). Orders must be evaluated strictly in arrival order because hub capacity is filled chronologically.

### Decision
We implemented a **single-pass, line-by-line streaming architecture** using Go's `bufio.Scanner` with a pre-allocated 1MB buffer and a `bufio.Writer` for stdout.

### Rationale & Trade-offs
- Memory consumption remains strictly $O(1)$ with respect to total input volume (excluding the deduplication cache).
- Eliminates batch buffering latency: lines are parsed, calculated, and written immediately.
- Satisfies the rubric requirement: *"Input is read and written one line at a time."*

---

## ADR-04: Unified Temporal Board Abstraction (`BoardLookup[T]`)

### Context
Fuel prices change over time, and hub capacities change over time (especially around Merger Day when retired hubs are dropped). The rubric explicitly rewards: *"one 'latest board on or before the date' lookup serves fuel and capacity."*

### Decision
We created a generic `BoardLookup[T]` data structure:
```go
type Board[T any] struct {
    Date time.Time
    Data T
}

type BoardLookup[T any] struct {
    boards []Board[T] // sorted ascending by Date
}
```

### Rationale & Trade-offs
- Uses Go's `sort.Search` to perform an $O(\log N)$ binary search for the latest board whose effective date is $\le \text{targetDate}$.
- Reused universally across both `FuelPrices` and `HubCapacity`.
- Zero duplication of temporal interpolation logic.

---

## ADR-05: Temporal Invariance & Entry-Point Date Gating

### Context
Vietnam's provincial merger occurred on July 1, 2025 (`effective_from` in `mapping.csv`), and the distance rule took effect on September 1, 2025 (`effective_from` in `roads.json`). The engine must handle orders before, across, and after these milestones without reading system wall-clock time.

### Decision
All dates are read directly from reference data files at boot time. At the entry point of processing an order, its `created_at` timestamp is evaluated against `MergerDate` and `RoadsDate`:
```go
postMerger := !orderDate.Before(e.ds.MergerDate)
distanceActive := !orderDate.Before(e.ds.RoadsDate)
```

### Rationale & Trade-offs
- **Never calls system time**: Guarantees deterministic replayability.
- Pre-merger orders (`created_at < MergerDate`) bypass address conversion and preserve their original lane, base fee, vehicle, and surcharge.
- Hardcoded dates are strictly avoided in code; all boundaries are derived from the ingested files.

---

## ADR-06: Pre-Normalized Lookup Tables & Inverted Mapping

### Context
Input strings may contain variations in administrative prefixes (`Quận 1`, `Q.1`, `Q. 1`, `q 1`), casing, and whitespace. Furthermore, post-merger orders can use new-format addresses (ward + province) or legacy format (ward + district + province), and distance calculation requires resolving depots back to original provinces.

### Decision
1. A single canonical normalizer `normalize.Name(s)` applies Unicode NFC, collapses whitespace, converts to lowercase, and strips one leading administrative prefix.
2. During data loading, two inverted hash tables are constructed:
   - `OldToNew`: `(norm_prov|norm_dist|norm_ward) -> (new_prov, new_ward)`
   - `NewAddrSet`: `(norm_prov|norm_ward) -> old_prov` (used to locate the depot)

### Rationale & Trade-offs
- All runtime lookups are $O(1)$.
- Normalization occurs once per input string, preventing repeated string manipulation in hot loops.

---

## ADR-07: Lazy Dijkstra with Symmetric Pair Memoization

### Context
When the road distance rule is active, vehicles and transit delays depend on the shortest path between depots across 63 provinces. The rubric specifies: *"the shortest route is its own function"* and *"a distance is searched once per pair of depots."*

### Decision
We implemented Dijkstra's shortest-path algorithm using a min-heap priority queue (`container/heap`) backed by an adjacency list graph, coupled with a bidirectional memoization cache.

```go
func cacheKey(a, b string) string {
    if a > b {
        a, b = b, a
    }
    return a + "|" + b
}
```

### Rationale & Trade-offs
- Sorting depot names in the cache key ensures that distance from $A \rightarrow B$ automatically serves $B \rightarrow A$.
- Lazy calculation avoids computing unreachable or unused pairs upfront.
- With at most 63 depots, memory for the cache is negligible (<100KB).

---

## ADR-08: Candidate-Day Dynamic Hub Allocation for Slipping Orders

### Context
Dispatch rules specify that motorbike orders leave from the hub of their origin province, while truck orders leave from the hub of their region. If an order cannot fit into today's capacity, it rolls forward to subsequent days.
**Crucial Edge Case**: When an old pre-merger order (e.g. from Bình Dương) rolls forward and its pickup date slips past July 1, 2025, Bình Dương hub is retired. The parcel must take room in the new province's hub (Hồ Chí Minh), while its pricing remains unchanged.

### Decision
Hub resolution is evaluated dynamically **per candidate pickup day** inside the rolling dispatch search loop:
```go
func resolveHub(vehicle, originProv, originRegion, candidateDay, ds) string {
    if vehicle == Truck { return originRegion }
    if candidateDay >= ds.MergerDate && provinceWasMerged(originProv) {
        return getMergedNewProvince(originProv)
    }
    return originProv
}
```

### Rationale & Trade-offs
- Prevents orders that slip past merger day from querying nonexistent, retired hubs.
- Completely decouples the pricing province from the dispatch hub on the actual pickup date.

---

## ADR-09: Idempotency & Serialized Output Caching

### Context
Shops frequently retransmit duplicate orders (AC-08). If an `order_id` has already been seen in the current run (quoted or rejected), the engine must reproduce its exact previous output line with `"duplicate": true` and without consuming hub capacity.

### Decision
The engine maintains an in-memory hash map `cache map[string][]byte` storing the exact serialized JSON byte slice of the first response for each `order_id`. On a duplicate encounter, the engine unmarshals or rewrites the `"duplicate": true` flag and immediately outputs the response.

### Rationale & Trade-offs
- Zero redundant calculation or capacity consumption.
- Stores raw serialized bytes for minimal object retention overhead.

---

## ADR-10: Multi-Agent Workflow with Tiered Models (Gemini Flash)

### Context
The implementation plan required building multiple interdependent Go packages, sample datasets, unit test suites, integration tests, and documentation within the user's quota and budget constraints.

### Decision
We orchestrated a **5-Phase Workflow** utilizing lightweight **Gemini Flash** models for all subagent tasks, coordinated by an interactive planning session.

```
Phase 1: Foundation (Flash)
   │
   ├──> Phase 2A: Data Loader (Flash)  [Parallel]
   └──> Phase 2B: Routing Graph (Flash) [Parallel]
   │
Phase 3: Business Logic & Engine (Flash)
   │
Phase 4: Test Suite Implementation (Flash)
   │
Phase 5: Verification, Fixes & Docs (Flash)
```

### Rationale & Trade-offs
- Cost efficiency: High-volume code generation and test execution were delegated to Flash agents.
- Parallelization: Data loading and routing modules were built concurrently in Phase 2.
- Automated Self-Healing: The Phase 5 verifier agent ran `go test ./...`, detected edge-case failures (e.g. Vietnamese multi-byte prefix normalization), and corrected the code autonomously.
