# 🤝 Human-AI Co-Design & Collaboration Log

**Project**: Mini Delivery/works Delivery Pricing Engine  
**Participants**: Johnny (Human Architect / Tech Lead) & Antigravity (AI Assistant / System Designer)  
**Date**: November 20, 2026  

---

## Executive Summary

This log documents the end-to-end collaborative development process between a human engineer and an agentic AI assistant to solve **Coding Challenge #4: Merger Day at Mini Delivery/works**. 

Rather than treating the AI as a pure code generator, the team used an interactive, back-to-back co-design interview process to establish architectural alignment, followed by an autonomous 5-phase agent workflow executed by lightweight models (Gemini Flash), complete with automated self-healing and rigorous verification.

---

## Timeline & Collaboration Dialogue

```mermaid
sequenceDiagram
    autonumber
    actor H as Johnny (Human Architect)
    participant C as Antigravity (Coordinator / Design Lead)
    participant W as Gemini Flash Workflow Agents

    Note over H, C: Phase 1: Collaborative Co-Design & Technical Trade-offs
    H->>C: Provide Challenge Spec & start back-to-back Q&A
    C->>H: Q1: Language selection (Go vs Rust vs Python vs TS)
    H->>C: Narrow to Go/Rust/TS; explain why Go is preferred
    C->>H: Q2: Pipeline Architecture & Streaming I/O
    H->>C: Approve pure-function pipeline + single-pass streaming
    C->>H: Q3: Temporal data modeling & Rule versioning
    H->>C: Approve generic BoardLookup[T] + Entry-point date gating
    C->>H: Q4: Normalization & Bidirectional address mapping
    H->>C: Approve Normalize-then-lookup + O(1) inverted maps
    C->>H: Q5: Dijkstra routing & Slipping Hub state model
    H->>C: Approve Memoized Dijkstra + Candidate-day hub resolution
    C->>H: Q6: Deduplication cache & run.sh binary execution
    H->>C: Approve raw response caching + compiled Go binary

    Note over H, C: Phase 2: Implementation Planning & Approval
    C->>H: Deliver comprehensive implementation-plan.md
    H->>C: Update implementation plan to run subagents on Gemini Flash
    H->>C: [Approved] implementation-plan.md

    Note over C, W: Phase 3: Autonomous Multi-Agent Execution
    C->>W: Launch 5-Phase Workflow with Gemini Flash workers
    W->>W: Phase 1: Foundation (models, normalize, calibrated data)
    W->>W: Phase 2: Data & Routing in parallel (loader, Dijkstra)
    W->>W: Phase 3: Engine & Dispatch (pricing, candidate-day hub logic)
    W->>W: Phase 4: Test Suite (5 mandatory challenge tests)
    W->>W: Phase 5: Verification, Self-Healing & Documentation

    Note over H, C: Phase 4: Permissions & Environment Guidance
    H->>C: How to configure auto-approvals in CLI without Ctrl+K prompts?
    C->>H: Explain settings.json security boundary & provide allow-list snippet

    Note over C, H: Phase 5: Delivery & Verification
    C->>H: Present test passes, worked example validation, and artifacts
```

---

## Detailed Step-by-Step Log

### Round 1: Problem Ingestion & Language Selection

- **Human Prompt**: Johnny initiated the session with `/plan`, sharing the challenge specification describing the Vietnam province merger (63 provinces to 34, removing districts), temporal pricing boards, distance-based road vehicle assignment, hub capacity dispatching, and the 1,000,000 orders/minute performance goal.
- **Antigravity Action**: Antigravity inspected the workspace, discovered it was empty, and initiated the co-design interview.
- **Discussion Point (Language Selection)**:
  - *Antigravity posed*: Evaluated TypeScript (Bun), Python, Go, and Rust against the 6–8 hour timebox and 1M orders in 60s benchmark.
  - *Johnny's feedback*: Noted that Python wouldn't comfortably make the performance cut, while Rust has a high development friction cost. Expressed interest in understanding why Rust has higher overhead.
  - *Antigravity's clarification*: Explained that Rust's ownership model creates friction when managing mutable shared hub state across orders, and string manipulation for Vietnamese Unicode is more verbose.
  - *Consensus*: **Go** was selected as the optimal sweet spot: native compilation speed, predictable memory usage, straightforward concurrency, standard library efficiency, and simple `./run.sh` deployment.

---

### Round 2: Core Architecture & I/O Model

- **Discussion Point (Pipeline vs State Machine)**:
  - *Antigravity posed*: How to structure order processing to satisfy the rubric requirement that "lane, chargeable weight, base fee, surcharge and total are separate steps".
  - *Johnny's choice*: **Pipeline Architecture**. Each order flows through: `Validate` $\rightarrow$ `Resolve Addresses` $\rightarrow$ `Determine Lane` $\rightarrow$ `Price` $\rightarrow$ `Dispatch`. Stages remain pure functions, keeping state mutation isolated exclusively to the `Dispatch` stage.
- **Discussion Point (Processing Model)**:
  - *Antigravity posed*: Concurrency vs sequential streaming.
  - *Johnny's choice*: **Single-Pass Streaming**. Since hub capacity depends on exact arrival order, processing must be deterministic and sequential. A buffered line-by-line reader/writer (`bufio.Scanner` with a 1MB buffer) was selected for minimal memory overhead.

---

### Round 3: Temporal Design & Rule Versioning

- **Discussion Point (Temporal Lookups)**:
  - *Antigravity posed*: The rubric states "one latest-board-on-or-before-the-date lookup serves fuel and capacity".
  - *Johnny's choice*: Unified `BoardLookup[T]` generic abstraction with sorted slices and binary search (`sort.Search`).
- **Discussion Point (Rule Invariance)**:
  - *Antigravity posed*: Pre-merger vs post-merger rules, and pre-distance vs post-distance rules.
  - *Johnny's choice*: **Entry-point rule selection**. The order's `created_at` timestamp gates the execution path up-front. No scattered conditionals or reading from system clock.

---

### Round 4: Vietnamese Administrative Normalization & Address Resolution

- **Discussion Point (Normalization Strategy)**:
  - *Antigravity posed*: 4-step normalization (Unicode NFC, whitespace collapse, case-folding, prefix stripping for `thành phố`, `quận`, `huyện`, `thị xã`, `phường`, `xã`, etc.).
  - *Johnny's choice*: **Normalize-then-lookup**. Canonicalize strings immediately upon ingestion; pre-normalize all data file lookups at load time.
- **Discussion Point (Merger Mapping Structure)**:
  - *Antigravity posed*: Handling legacy addresses with districts, new format addresses without districts, and looking up original provinces for distance depots.
  - *Johnny's choice*: **Bidirectional O(1) hash maps**:
    1. `OldToNew`: `(old_prov|old_dist|old_ward) -> (new_prov, new_ward)`
    2. `NewToOld`: `(new_prov|new_ward) -> old_prov` (for depot resolution)

---

### Round 5: Routing & Hub Capacity State Tracking

- **Discussion Point (Shortest Path)**:
  - *Antigravity posed*: Dijkstra on demand vs pre-computed all-pairs vs memoized.
  - *Johnny's choice*: **Lazy Dijkstra with symmetric pair memoization**. Cache key is sorted `min(A, B) + "|" + max(A, B)` so queries are computed at most once per depot pair.
- **Discussion Point (Slipping Pre-Merger Orders)**:
  - *Antigravity posed*: A pre-merger order (e.g. from Bình Dương) slipping past July 1, 2025 must take room in the new merged hub (Hồ Chí Minh) while keeping its original price.
  - *Johnny's choice*: **Per-candidate-day hub resolution**. In the dispatch loop, resolve the active hub name for each candidate pickup day.

---

### Round 6: Deduplication & Service Packaging

- **Discussion Point (Idempotency)**:
  - *Antigravity posed*: Duplicate order IDs must return original quote with `duplicate: true` without consuming capacity.
  - *Johnny's choice*: Response cache `map[string][]byte` storing the serialized JSON line. On duplicate, modify the `"duplicate"` flag and re-emit.
- **Discussion Point (run.sh Packaging)**:
  - *Johnny's choice*: `go build -o engine ./cmd/engine && ./engine "$@"`.

---

## Autonomous Agent Execution & Human Steering

### Planning Artifact & Delegation
Antigravity generated `implementation-plan.md`. Johnny reviewed it and requested an important adjustment:
> *"L7: change Sonnet to Gemini Flash"*  
> *"ok. as long as the subagents are running on gemini flash (cheaper model), its ok for me."*

Antigravity updated the plan and launched a 5-phase autonomous workflow using Gemini Flash workers:
1. **Foundation Setup Worker**: Set up `go.mod`, models, normalize functions, and reference data files.
2. **Infrastructure Workers (Parallel)**: Worker A built `internal/data/board.go` & `loader.go`; Worker B built `internal/data/routing.go` (Dijkstra algorithm).
3. **Engine & Pricing Worker**: Implemented validation, lane classification, fee calculations, and dispatch rolling-date capacity deductions.
4. **Test Suite Worker**: Built unit tests and integration tests covering all 5 mandatory scenarios.
5. **Verifier & Fixer Worker**: Built the binary, verified all tests, fixed edge-case bugs, and authored comprehensive documentation.

### Human Intervention: Permission Management in Antigravity CLI
During workflow execution, Johnny inquired:
> *"/plan how do i provide the subagents auto-approvals for certain commands? or simply allow them to auto edit."*  
> *"/plan i always have to ctrl+k to approve. how to manage it to make auto-approve?"*

Antigravity analyzed the CLI architecture:
1. Explained that Antigravity prompts `Ctrl+K` whenever a command is executed that is not yet on the allow-list in `~/.gemini/antigravity-cli/settings.json`.
2. Clarified that AI assistants cannot edit `settings.json` directly due to hardcoded safety boundary protection (`Matches hardcoded system protection boundary rule`).
3. Provided Johnny with a one-line terminal snippet to update `settings.json` with wildcard allow rules (`command(go *)`, `command(./run.sh *)`, etc.).

---

## AI Self-Healing & Defect Detection

As required by the challenge submission rubric (*"how you checked its work, with one mistake it made and how you found it"*):

### Defect Identified:
During Phase 4 and Phase 5 verification, the Verifier Agent executed `go test ./...` and identified two issues created by the code generation subagents:

1. **Prefix Normalization Regression**:
   - *Bug*: In `internal/normalize/normalize.go`, the test `TestName/Xã_An_Xuân` failed. The string `"Xã An Xuân"` returned `"xã an xuân"` instead of `"an xuân"`.
   - *Root Cause*: The prefix list contained `"x"` before `"xã"` or failed to properly match the multi-byte UTF-8 accented character `"xã"` when checking word boundaries.
   - *Fix*: The Verifier Agent updated the prefix matching logic in `internal/normalize/normalize.go` to handle multi-byte characters and proper prefix sorting, restoring full test compliance.

2. **Test Assertion Type Mismatch**:
   - *Bug*: In `tests/integration_test.go`, the helper function `assertBool` threw a compilation error: `got := result[key]` did not check existence properly before asserting boolean type.
   - *Fix*: The Test Builder agent updated `assertBool` to use comma-ok idiom (`got, ok := result[key]`) and verified clean compilation.

---

## Final Outcome
- 100% of unit and integration tests passing.
- All 22 worked examples verified.
- Complete documentation delivered in `README.md`, `docs/COLLABORATION_LOG.md`, and `docs/TECHNICAL_DECISIONS.md`.
