# Clockwork Piranha — Salesforce Apex REST Load-Testing Toolkit

[![Go Version](https://img.shields.io/github/go-mod/go-version/aliozdemir13/clockwork-piranha)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go CI](https://github.com/aliozdemir13/clockwork-piranha/actions/workflows/ci.yaml/badge.svg)](https://github.com/aliozdemir13/clockwork-piranha/actions)

A Go-based **toolkit** for load-testing Salesforce Apex REST endpoints. Clockwork Piranha deploys a coordinated pack of concurrent goroutines against a target endpoint at a controlled rate, hunting for the throughput limits and infrastructure weak points that production traffic will eventually find on its own.

The name is deliberate. *Piranha* for the swarm: concurrent goroutines acting as a coordinated pack. *Clockwork* for everything else: rate-limited, deterministic, and — most importantly — wound up by you before it runs. This is not a plug-and-play CLI. It's a small, opinionated codebase you adapt to your own org's endpoints, data schema, and test scenarios. The framework handles concurrency, rate limiting, and data-pool consumption; you describe the requests and bring the data.

Clockwork Piranha is for the case where requests must be parameterized from a managed, shared pool consumed safely by many concurrent workers — the pattern that arises constantly in Salesforce load testing and that the generic tools don't address.

Why not the Bulk API? Clockwork Piranha is designed for Real-Time API performance testing (Apex REST). While Bulk is for data volume, this tool measures how your org’s custom logic, triggers, and synchronous limits hold up under 'Human-speed' but 'Swarm-scale' transactional traffic.

---

## Status: Work In Progress

This is an early-stage personal project. It runs end-to-end and produces useful numbers, but several components are still rough or unimplemented. APIs, package boundaries, and internal architecture **will change** without notice.

**Not accepting feature requests at this stage.** Issues / discussion welcome for design feedback.

### What works today

- Configurable RPS and duration via interactive prompts
- Rate-limited request dispatch using `golang.org/x/time/rate`
- Concurrent request execution via goroutines
- CSV-driven data source with shuffled consumption and refill-on-drain
- Per-endpoint error categorization, top-10 console summary, full error dump as JSON
- Five reference test scenarios for a Loyalty Voucher / Consent API surface (illustrative — you'll replace these for your org)

### What's known-rough

See [Known Limitations](#known-limitations) below. The TL;DR: there's no worker-pool bound, no latency percentiles, no token refresh, no graceful shutdown, and the data strategy doesn't yet handle stateful endpoints honestly. All on the roadmap.

For methodology and how to produce defensible numbers, see [METHODOLOGY.md](METHODOLOGY.md).

---

## Adapt to your org first

Clockwork Piranha ships with reference scenarios that target a specific Loyalty / Consent Apex REST surface. **These are examples, not a product.** Before running against your own org, you'll edit three files:

| File | What to change |
|---|---|
| `internal/data_factory/models.go` | Adjust the `MemberDetails` struct (and its JSON tags) to match the columns and field names you need from your CSV and request bodies. |
| `internal/data_factory/prepare_data.go` | Adjust the CSV parser if your file has a different column count, header, or structure. Update the hardcoded voucher pool or replace it with whatever supporting data your scenarios require. |
| `internal/throughput/test_cases.go` | Replace the factory functions with ones describing your own endpoints: their paths, HTTP methods, and request body shapes. Update `Throughput.TestSuite()` to call your new scenarios. |

The runner itself (`internal/throughput/throughput.go`), the data randomizer, and the rate-limited dispatch loop are designed to be stable across these adaptations. The factory pattern in `test_cases.go` is the single extension point for new endpoints.

---

## Quickstart

**Requirements:** Go 1.22+ (uses `math/rand/v2`).

```bash
git clone https://github.com/aliozdemir13/clockwork-piranha.git
cd clockwork-piranha
# Adapt the three files listed in "Adapt to your org first" above
go run .
```

You will be prompted for:

| Prompt | Value |
|---|---|
| `clientId` | Salesforce Connected App client ID |
| `clientSecret` | Salesforce Connected App client secret |
| `orgBaseUrl` | Your Salesforce instance URL, e.g. `https://your-domain.my.salesforce.com` |
| `RPS` | Requests per second to target |
| `Duration` | Test duration in minutes |

The toolkit will then run each of your configured test scenarios sequentially for the full duration.

### Authentication

Clockwork Piranha uses the OAuth 2.0 **Client Credentials** flow against `/services/oauth2/token`. Your Connected App must have this flow enabled. Tokens are fetched once at startup and reused for the full run — see the limitation on token refresh below if your run exceeds your org's session timeout.

---

## CSV input format

The toolkit expects a file named `membersFromFullSBX.csv` in the working directory by default. **Filename is currently hardcoded** in `main.go` (see roadmap; rename or change in source as needed).

The reference parser expects a header row and three columns:

```csv
Id,MembershipNumber,PersonAccountId
001FAKE000000001AAA,FAKE_MEM_001,001FAKE000000101AAA
001FAKE000000002AAA,FAKE_MEM_002,001FAKE000000102AAA
```

| Column | Description |
|---|---|
| `Id` | Loyalty Program Member record Id |
| `MembershipNumber` | Loyalty Program Member membership number |
| `PersonAccountId` | Person Account Id linked to the member (used for Consent endpoints) |

This schema is illustrative — your org's data model is almost certainly different. The CSV parser lives in `internal/data_factory/prepare_data.go` and is intentionally small; adjust column indices, field names, and the struct it populates to match your data.

---

## How it works (brief)

```
main.go
  └─> loads CSV          (internal/data_factory)
  └─> authenticates      (OAuth Client Credentials)
  └─> Throughput.TestSuite()
        └─> for each scenario:
              shuffle the member pool   (internal/data_factory/randomizer.go)
              runTest(name, factory, pool):
                rate.Limiter dispatches at TestRPS
                each dispatch:
                  - pops one member from the pool
                  - builds a TestCase via the scenario's factory
                  - fires HTTP request in a goroutine
                  - records success / error
                when the pool drains: refill via slice-copy + reshuffled re-fill (ensures zero-overhead data continuity)
                when ctx times out: drain in-flight, write error file
```

The factory pattern (`internal/throughput/test_cases.go`) decouples request construction from dispatch, so each scenario describes its own URL / method / body template independently of the runner.

---

## Extending the toolkit

The most common extension is adding a new endpoint scenario. The shape:

1. **Define the request shape.** If the body has a non-trivial structure, add a Go struct in `internal/data_factory/models.go` with the appropriate JSON tags.
2. **Write a factory.** In `internal/throughput/test_cases.go`, add a function with the signature `func(InstanceURL string) func(*data_factory.MemberDetails) *data_factory.TestCase`. The inner function builds a `TestCase` from a single member of the pool.
3. **Register the scenario.** In `Throughput.TestSuite()` in `internal/throughput/throughput_testing.go`, add a block that shuffles a pool, constructs the factory, and calls `t.runTest(...)` with a meaningful name.

That's it — the runner handles rate limiting, concurrency, error accounting, and output. You don't need to touch any of the dispatch logic to add scenarios.

Other extension points worth knowing about:

- **Different data source:** replace `LoadMembersFromCSV` with anything that returns `[]*MemberDetails`. A SOQL-driven loader or a generator function fits the same shape.
- **Different randomization strategy:** the `data_factory.ShuffleOnInit` function is intentionally narrow; you can write alternatives (no-shuffle, weighted, etc.) and call them from `TestSuite()`.
- **Different reporting:** the post-run reporting at the bottom of `runTest` writes JSON and prints to stdout. Swap it for whatever your CI / dashboard / spreadsheet flow needs.

---

## Known limitations

These are the things you should know before relying on numbers from this toolkit.

**Data strategy**

- Read endpoints use shuffled round-robin with refill. Reasonable.
- Write endpoints (state-mutating) revisit the same rows after wrap-around, which produces state-drift errors that contaminate the success/error ratio. Currently, the recommendation is to size the pool such that `pool_size > RPS × duration`. A state-aware data strategy is planned.

**Concurrency**

- Goroutine fan-out is unbounded — only the rate limiter shapes pacing. At high RPS with slow responses, in-flight goroutines can grow large. No worker pool yet. At extremely high RPS (e.g., >5000) or during network congestion, this can lead to memory exhaustion (OOM) as goroutines queue up waiting for I/O. Use with caution in environments with low RAM.
- A single mutex protects counters and the error map, which becomes contended at high RPS.

**Measurement**

- No latency percentiles. Only success/error counts and computed RPS.
- Error map keys include the full response body, which can grow unbounded if errors carry correlation IDs or timestamps.
- "Actual RPS" includes successes + errors over wall-clock elapsed time. There's no separate "useful throughput" number that excludes business errors.

**Operational**

- No token refresh. Long runs may 401 silently in the second half if the session expires.
- Client secret entered via plain stdin (echoes to terminal). Will move to env vars / secure prompt.
- No SIGINT handling. Ctrl-C kills the run mid-flight without saving partial results.
- Interactive prompts only. No flags, no config file, not CI-friendly.
- Endpoint paths in the reference scenarios are hardcoded — by design, since the toolkit assumes you adapt the scenarios to your org.

**Reproducibility**

- Shuffles are non-deterministic (no seed). Two runs against the same data produce different orderings, which makes run-to-run comparison noisy. Seed support is planned.

---

## Roadmap

In rough priority order:

1. State-aware data strategy for write endpoints (in-memory toggle cache + recirculating channel)
2. Bounded worker pool with configurable concurrency
3. Latency percentiles (p50 / p95 / p99) via a histogram
4. Optional CLI flags / config file for the interactive-prompted values (the toolkit model — adapting source for endpoints and data — remains the primary intent)
5. Optional pool-provisioning mode via Bulk API for sandboxes with storage headroom
6. Token refresh, SIGINT-aware shutdown, structured logging
7. Seeded RNG for reproducible runs

Note: items 4 and 5 are intentionally framed as *optional* moves. Piranha's identity is "code you adapt," not "configuration you fill in." That framing may change as the project matures, but it's the working assumption today.

---

## Disclaimer

Clockwork Piranha is a personal learning project. It is not affiliated with, sponsored by, or endorsed by any employer, client, or third party. The Salesforce-specific endpoint paths and data shapes referenced in the source code are illustrative reference scenarios; replace them with your own Apex REST endpoints as needed.

No real customer or production data is included in this repository. The voucher codes, sample membership numbers, and other identifiers in the source are synthetic.

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
