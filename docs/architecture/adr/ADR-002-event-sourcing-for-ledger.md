# ADR-002: Event Sourcing for the Ledger

## Status
Accepted

## Context
A double-entry ledger is naturally an append-only event log: transactions are facts, not state mutations. Traditional CRUD approaches add UPDATE/DELETE operations that introduce the risk of corrupting the audit trail — a core GAAP requirement. The alternative (standard mutable rows with soft-delete flags) requires out-of-band audit log infrastructure and relies on application-level discipline to maintain immutability.

## Decision
Journal entries and postings are append-only (event-sourced). No UPDATE or DELETE is ever issued against `journal_entries` or `postings`. Corrections are made exclusively via reversing entries — a new journal entry that negates the original — exactly as GAAP requires. Account balances, envelope states, and all reports are projections computed from the immutable event stream.

## Consequences
- Immutability is structural, not policy-based: the GAAP guard enforces it before persistence, and the store layer has no update/delete methods for these tables.
- The full audit trail is implicit; there is no separate audit log table to maintain.
- Balance queries scan postings for the account up to a given date; performance is managed via indexed `account_id` + `posted_at` and precomputed balance snapshots (future optimization, not Phase 1).
- The `sync_event` table extends this model to the sync layer: device-to-server sync is also append-only event replay.
- Conflict resolution is simpler because the conflicting fact (the journal entry) never changes; only metadata (account names, envelope config) can conflict.
