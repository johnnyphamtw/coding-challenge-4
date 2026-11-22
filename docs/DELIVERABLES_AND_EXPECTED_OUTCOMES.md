# 📋 Deliverables & Expected Outcomes Guide

**Project**: Mini Delivery/works (Coding Challenge #4)  
**Evaluation Model**: Submission (70%) + Team Presentation (30%)  

---

## 1. Master Requirements & Deliverables Matrix

This matrix maps every explicit requirement from the challenge description to our project implementation:

| Requirement / AC | Deliverable / Artifact | Implemented File & Test | Status |
|---|---|---|---|
| **AC-01**: Lane Classification | LOCAL, SAME_PROVINCE, SAME_REGION, CROSS_REGION | [internal/pricing/pricing.go](file:///Users/johnny/Projects/coding-challenges/internal/pricing/pricing.go#L110) | ✅ Complete |
| **AC-02**: Base Fees & Volumetric Weight | Max(actual, volume/5), tiered block fees | [internal/pricing/pricing.go](file:///Users/johnny/Projects/coding-challenges/internal/pricing/pricing.go#L140), [fees_test.go](file:///Users/johnny/Projects/coding-challenges/internal/pricing/fees_test.go) | ✅ Complete |
| **AC-03**: Fuel Surcharges & 3 Roundings | Petrol (bike) / Diesel (truck), 3 discrete roundings | [internal/pricing/pricing.go](file:///Users/johnny/Projects/coding-challenges/internal/pricing/pricing.go#L170), [fees_test.go](file:///Users/johnny/Projects/coding-challenges/internal/pricing/fees_test.go) | ✅ Complete |
| **AC-04**: Strict Validation Priority | NO_PRICE_BOARD $\rightarrow$ UNKNOWN_PROVINCE $\rightarrow$ MISSING_DISTRICT $\rightarrow$ BAD_WEIGHT $\rightarrow$ BAD_DIMENSIONS | [internal/pricing/pricing.go](file:///Users/johnny/Projects/coding-challenges/internal/pricing/pricing.go#L15), [integration_test.go](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go) | ✅ Complete |
| **AC-05**: Dispatch & Hub Capacity | Earliest open day, TOO_HEAVY check on order date | [internal/dispatch/dispatch.go](file:///Users/johnny/Projects/coding-challenges/internal/dispatch/dispatch.go), [integration_test.go](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go) | ✅ Complete |
| **AC-06**: Merger Day (2025-07-01) | Abolish districts, address translation, slipping hub switch | [internal/engine/engine.go](file:///Users/johnny/Projects/coding-challenges/internal/engine/engine.go#L135), [mapping.csv](file:///Users/johnny/Projects/coding-challenges/data/mapping.csv) | ✅ Complete |
| **AC-07 Rule 1**: Backward Compat (Old Price) | Pre-merger order keeps its historical pricing and lane | [tests/integration_test.go#TestBackwardCompat_OldOrderKeepsPrice](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go#L60) | ✅ Complete |
| **AC-07 Rule 2**: Backward Compat (Old Address) | Post-merger legacy & new addresses give identical quote | [tests/integration_test.go#TestBackwardCompat_OldNewAddressSameQuote](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go#L75) | ✅ Complete |
| **AC-08**: Deduplication & Idempotency | Seen `order_id` returns cached quote with `duplicate: true` | [internal/engine/engine.go](file:///Users/johnny/Projects/coding-challenges/internal/engine/engine.go#L36), [TestDuplicateOrder](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go#L117) | ✅ Complete |
| **AC-09**: 1M Orders Scale ($<60\text{s}$) | Process 1,000,000 orders in under 60 seconds | [test-scale.sh](file:///Users/johnny/Projects/coding-challenges/test-scale.sh) (**5.50s, 181,676 orders/sec**) | ✅ Complete |
| **AC-10**: Distance Rule (2025-09-01) | Shortest road path via Dijkstra, vehicle selection, extra days | [internal/data/routing.go](file:///Users/johnny/Projects/coding-challenges/internal/data/routing.go), [TestIndirectRoute](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go#L136) | ✅ Complete |
| **Executable Entrypoint** | `./run.sh [dataDir]` with stdin $\rightarrow$ stdout streaming | [run.sh](file:///Users/johnny/Projects/coding-challenges/run.sh) & [cmd/engine/main.go](file:///Users/johnny/Projects/coding-challenges/cmd/engine/main.go) | ✅ Complete |
| **Mandatory Tests** | 5 required tests + worked examples runner | [tests/integration_test.go](file:///Users/johnny/Projects/coding-challenges/tests/integration_test.go) & [test-examples.sh](file:///Users/johnny/Projects/coding-challenges/test-examples.sh) | ✅ Complete |
| **System README** | Mermaid diagram, merger date explanation, scale question, AI section | [README.md](file:///Users/johnny/Projects/coding-challenges/README.md) | ✅ Complete |

---

## 2. Evaluation Scorecard Breakdown (70% Submission)

The submission accounts for 70 points allocated across 7 categories:

### 1. Correct Results on Judges' Scenarios (18 points)
- **Goal**: Engine must produce exact, byte-for-byte expected JSON quotes on judges' private scenario files.
- **Verification**: All 19 blocks (22 worked examples) in [examples/worked-examples.json](file:///Users/johnny/Projects/coding-challenges/examples/worked-examples.json) pass with 100% field equality via `./test-examples.sh`.

### 2. Pricing Model Design (8 points)
- **Goal**: Fees, block sizes, fuel settings, and capacity read entirely from configuration files; pure separation of concerns; Vietnamese normalization function; shortest path function; integer arithmetic for money with discrete roundings.
- **Evidence**:
  - `internal/pricing/pricing.go`: Distinct functions for `CalculateChargeableWeight`, `CalculateBaseFee`, `CalculateSurcharge`, `CalculateTotal`.
  - `internal/normalize/normalize.go`: Single `Name()` normalizer covering Unicode NFC, spacing, case, and prefixes.
  - `internal/data/routing.go`: Dedicated Dijkstra implementation.

### 3. Temporal Design (8 points)
- **Goal**: One unified "latest board on or before date" lookup for fuel and capacity; dates read from data files; zero calls to system wall-clock time; pre-merger orders do not convert addresses.
- **Evidence**:
  - `internal/data/board.go`: Generic `BoardLookup[T]` with $O(\log N)$ binary search.
  - Entry-point date gating: `postMerger := !orderDate.Before(e.ds.MergerDate)` in `internal/engine/engine.go`.

### 4. Service Design (8 points)
- **Goal**: Idempotent response cache keyed by `order_id`; input read and written one line at a time; data loaded once into memory; depot distance searched once per pair.
- **Evidence**:
  - `Engine.cache map[string][]byte` in `internal/engine/engine.go`.
  - `RoadGraph.Cache map[string]int` in `internal/data/routing.go` with sorted symmetric keys (`min|max`).
  - Single-pass `bufio.Scanner` and `bufio.Writer` in `cmd/engine/main.go`.

### 5. Working with AI (7 points)
- **Goal**: Documented instructions, description of human vs AI responsibilities, and how AI-generated work was checked, highlighting **at least one mistake it made and how it was discovered**.
- **Evidence**:
  - [docs/COLLABORATION_LOG.md](file:///Users/johnny/Projects/coding-challenges/docs/COLLABORATION_LOG.md) documents the dialogue, human steering, tiered model selection, and the bug detection.
  - **Required Mistake Example**: Prefix normalization regression in `TestName/Xã_An_Xuân` where accented character `"xã"` was mishandled by the code generation subagent and caught by the automated test suite.

### 6. Tests & Reliability (10 points)
- **Goal**: Mandatory tests for backward compatibility rules, hub overflow, duplicate detection, and indirect route.
- **Evidence**:
  - `tests/integration_test.go` has dedicated tests for all 5 required scenarios.
  - `tests/bench_test.go` tests memory allocations and latency.

### 7. Documentation & Communication (11 points)
- **Goal**: Clear architecture diagram, merger date design reasoning, answer to the 20M orders scale question, and crisp instructions.
- **Evidence**:
  - [README.md](file:///Users/johnny/Projects/coding-challenges/README.md) with comprehensive Mermaid flowcharts and scale answers.
  - [docs/TECHNICAL_DECISIONS.md](file:///Users/johnny/Projects/coding-challenges/docs/TECHNICAL_DECISIONS.md) with 10 formal ADRs.
  - [docs/SCALE_BENCHMARK.md](file:///Users/johnny/Projects/coding-challenges/docs/SCALE_BENCHMARK.md) with the 1M performance benchmark.

---

## 3. The Presentation Guide (30% Score)

The presentation is 30% of your total score. The judges look for:
1. Understanding of the problem and trade-offs.
2. Demonstration of the engine in action.
3. How the team used and checked AI.
4. Readiness for questions and the **Live 10-Minute Rule Change**.

### Recommended Presentation Agenda (15–20 minutes)

```
[Slide 1-2] Problem & Domain: Vietnam's Administrative Merger & Temporal Invariance
[Slide 3-4] System Architecture: The Pure Pipeline & Streaming Conveyor Belt
[Slide 5-6] Tricky Technical Challenges & Architectural Decisions (ADRs)
            • Candidate-day Hub Resolution for Slipping Orders
            • O(N) Fast-Forward Frontier for Saturated Hubs
[Slide 7-8] Working with AI: Co-Design, Tiered Workers & Verification
            • Human role (Architect) vs AI role (Implementer/Verifier)
            • Caught defects (prefix normalizer & test types)
[Slide 9-10] Live Demonstration:
            • Running official 22 worked examples (./test-examples.sh)
            • Running 1,000,000 orders scale test (./test-scale.sh: 5.50s)
[Slide 11]  The Scale Question (20,000,000 orders architecture)
[Live Demo] The 10-Minute Rule Change Challenge
```

---

## 4. Live Demo & The 10-Minute Rule Change Strategy

### What Happens at the Demo
At the demonstration, the judge will hand your team **one small rule change**. You have **10 minutes** to adjust the code, and then the judge will execute a scenario file against your updated engine.

### Common Rule Change Scenarios & Where to Modify:

| Likely Rule Change | Location to Modify | What to Change |
|---|---|---|
| **New Vehicle Type** (e.g. Van / Drone) | `internal/models/models.go` & `pricing.go` | Add enum value to `Vehicle`; add vehicle selection threshold in `data/routing.go` or `pricing/pricing.go`. |
| **New Pricing Tier / Formula** | `internal/pricing/pricing.go` | Update `CalculateBaseFee` or `CalculateSurcharge` formula. |
| **Discount / Promotional Surcharge** | `internal/pricing/pricing.go` | Add promo discount step in `CalculateTotal`. |
| **New Administrative Prefix** | `internal/normalize/normalize.go` | Add string to `prefixes` slice (e.g. `"đặc khu"`, `"khu phố"`). |
| **Different Transit Days Formula** | `internal/data/routing.go` | Modify `ExtraDays()` formula. |
| **Hub Capacity Holiday / Blackout** | `internal/dispatch/dispatch.go` | In `Dispatch()`, add condition to skip Sunday/Holiday dates. |

### 10-Minute Workflow:
1. Open the relevant file identified above.
2. Make the small logic tweak.
3. Run unit tests immediately: `go test -v ./...`.
4. Run worked examples: `./test-examples.sh`.
5. Recompile: `go build -o engine ./cmd/engine`.
6. Hand back to judge.

---

## 5. The Scale Question Answer (Key Presentation Talking Point)

**Question**: *"Next year the sale brings 20,000,000 orders in one day, on several machines. Which parts can run in parallel, which must keep the order of arrival, and where does the state live?"*

### Clear Talking Points for the Team:

1. **What Runs in Parallel (Stateless Stages)**:
   - Input JSON parsing & validation (AC-04).
   - Address normalization & merger translation (AC-06).
   - Lane classification (AC-01).
   - Dijkstra shortest path routing (AC-10) — memoized in local read-only cache.
   - Base fee and fuel surcharge calculation (AC-02, AC-03).
   - *These represent 85% of total CPU cycles and scale linearly across any number of worker machines.*

2. **What Must Keep Order of Arrival (Stateful Stages)**:
   - Hub capacity allocation and pickup date assignment (AC-05).
   - Daily hub space is a shared mutable resource allocated chronologically.
   - Idempotency deduplication cache (AC-08).

3. **Where the State Lives**:
   - **Sharded Architecture**: Motorbike hub capacity is independent per province; truck hub capacity is independent per region. Orders are routed to **Kafka partitions keyed by `origin_hub`**.
   - Each hub partition has an independent, sequential single-threaded worker owning its calendar. There is **zero distributed lock contention** across different hubs.
   - Shared distributed state (e.g. Redis Cluster) stores:
     - Deduplication key: `order_id` $\rightarrow$ cached quote.
     - Hub calendar: `hub:{name}:{date}` $\rightarrow$ remaining capacity.
