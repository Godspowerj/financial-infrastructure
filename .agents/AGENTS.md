# AGENTS.md — Guide for AI assistants working on fintech-lab

## What this project actually is

A learning lab, not a product. The goal is to understand how fintech/payment
processor backends are engineered — NOT to ship something impressive fast.
The domain modeled is closest to a **payment processor** (Stripe/Paystack/
Flutterwave-style) that sits between customers, merchants, and real banks —
not a bank itself. It coordinates and records money movement; it doesn't
custody funds long-term.

Modules, built roughly in this order, each its own microservice:
1. Payments — lifecycle, idempotency, event-driven architecture
2. Ledger — double-entry bookkeeping, ACID, immutability
3. Transfers — sagas, compensation, distributed transactions
4. Settlement & Reconciliation — batch processing, reporting
5. Notifications — pub/sub, webhooks, retries
6. Fraud & Risk — rule engines, async scoring
7. Scaling — caching, load balancing, CAP theorem, observability

Architecture: real microservices (not a modular monolith), deliberately —
the person building this wants to feel actual network failure modes, not
just simulate boundaries in code. Each service = its own Go module, own
Postgres schema/database, communicates with others only via HTTP APIs or
RabbitMQ events — never by directly touching another service's tables.

## Who you're working with

The person building this is learning Go and SQL *while* learning distributed
systems concepts — three curves at once. Prior experience: solid with SQL
concepts generally shaky, Go is new, comfortable with git. This is explicitly
NOT about speed or impressing anyone. Optimize for understanding over
completion. Concretely:

- Don't dump multiple new concepts/terms in one explanation. Introduce
  vocabulary only when it's actually needed for the step at hand.
- When explaining something technical, prefer plain-language analogies
  first (e.g. database = filing cabinet, table = folder, row = index card,
  constraint = a rule the cabinet itself enforces) before or instead of
  jargon. Check understanding before stacking more concepts on top.
- Don't hand over finished, unexplained code to copy-paste. Build things
  incrementally, explain *why* each piece exists, and what breaks if it's
  left out.
- Never re-send a full project zip after the initial scaffold. From the
  first scaffold onward, give exact file paths + exact content for new
  files, or precise old-vs-new snippets for edits to existing files. The
  person already has the project on disk and uses git to track changes —
  work with that, don't restart it.
- Currency is NGN. Store amounts as BIGINT in kobo (smallest unit, 1 NGN =
  100 kobo) — same reasoning as storing USD in cents. Never use
  FLOAT/DECIMAL for money.
- It is fine, and expected, to slow down, re-explain, or backtrack. Rushing
  to the next module while a concept hasn't landed is the failure mode to
  avoid, not slow progress.

## ADRs (Architecture Decision Records)

Location: `docs/decisions/NNNN-short-title.md`, numbered sequentially
starting at `0001`.

Write one ADR whenever a real tradeoff gets decided — not for every small
choice, only ones where a reasonable alternative existed and was rejected
for a stated reason (e.g. "microservices over modular monolith," "no status
ENUM constraint yet," "BIGINT kobo over DECIMAL naira").

Template:

```markdown
# NNNN — Title

## Status
Proposed | Accepted | Superseded by NNNN

## Context
What problem/decision point triggered this. 2-4 sentences, plain language.

## Decision
What we chose, stated plainly in one or two sentences.

## Alternatives considered
- Option A — why rejected
- Option B — why rejected

## Consequences
What this makes easier, what it makes harder, what we're deferring to later.
```

Keep ADRs short — half a page is fine. The point is capturing *reasoning*,
not producing documentation for its own sake.

## Benchmarks

Location: `docs/benchmarks/`, one markdown file per benchmark run, named
`YYYY-MM-DD-what-was-tested.md`.

Only benchmark things once there's a real question to answer (e.g. Module 7
scaling work, or "does the outbox publisher fall behind under load?") — not
speculatively. Each benchmark file should record:

- What was being measured and why (the question, not just "ran a load test")
- Tool used (k6, custom script, etc.) and exact command/config
- Environment (local Docker, resource limits if any)
- Raw results (latency percentiles, throughput, error rate)
- Interpretation — what it means, what (if anything) changed as a result

## Docs structure

```
docs/
├── decisions/     # ADRs — see above
├── benchmarks/     # perf test results — see above
└── modules/         # one file per module: the business problem, the
                        design, and what was actually learned building it
```

`docs/modules/0N-name.md` is written *after* a module is functionally done,
summarizing: the engineering question it answers, the design chosen, key
code entry points, and what specifically clicked or was hard to understand
building it. This is for the person's own future reference, not a
polished writeup — plain notes are fine.

## Code conventions

- Go: standard project layout (`cmd/api`, `internal/...`). Explain
  non-obvious idioms inline as comments when first introduced in a file
  (e.g. why `*sql.DB` is a pool, why context timeouts matter) — this
  project is also how the person is learning Go, not just distributed
  systems.
- SQL migrations: `services/<service>/migrations/NNNN_description.sql`,
  sequentially numbered, plain SQL files (no migration framework yet —
  intentionally deferred until it's clearly needed).
- No premature constraints/enums on fields still being actively designed
  (e.g. `status` as free TEXT while the state machine is still evolving in
  Go code) — lock down constraints once the design has settled, not before.
- Every service exposes `/health` distinguishing "process alive" from
  "can reach its dependencies" (see payments service `main.go` for the
  pattern).
