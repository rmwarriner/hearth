# ADR-007: PostgreSQL Row-Level Security Strategy

## Status
Accepted

## Context
In server mode, a single PostgreSQL database serves multiple households. Every query must be scoped to the authenticated household — a member of household A must never be able to read or write household B's data, even in the presence of application bugs.

Two approaches were considered:

1. **Application-layer filtering only** — every query includes `WHERE household_id = $1`. Household ID is derived from the validated JWT claim and passed as a query parameter.
2. **Database-layer RLS + application-layer filtering** — PostgreSQL Row-Level Security policies enforce household isolation at the database level; the application also filters by household (defense in depth).

## Decision

**Both layers are used.** Defense in depth: a bug in the application layer (e.g., a missing `WHERE household_id` clause) is caught by the database before data leaks.

**Implementation**: PostgreSQL `SET LOCAL app.household_id = $1` is called at the start of every write transaction (and on read transactions that access household-scoped data). RLS policies on each household-scoped table reference `current_setting('app.household_id', true)`. The `true` parameter makes the setting transaction-local — it resets on commit or rollback, so a pooled connection cannot carry household context from one request to the next.

**Affected tables**: `accounts`, `journal_entries`, `postings`, `fiscal_periods`, `members`, `envelopes`, `envelope_allocations`, `refresh_tokens`. The `households` table does NOT have RLS — it is looked up by ID during authentication before household context is established.

**Tables without a direct `household_id` column** (`postings`, `envelope_allocations`) use subquery policies:
```sql
USING (journal_entry_id IN (
    SELECT id FROM journal_entries WHERE household_id = current_setting('app.household_id', true)
))
```

**Migration note**: `ALTER TABLE ... ENABLE ROW LEVEL SECURITY` requires the table owner. Migration `0011_enable_rls.sql` works as-is with the docker-compose setup where the app role owns the tables. Cloud deployments where the app role is not the table owner may require a superuser to run this migration; this is documented in the migration file.

**SQLite has no equivalent**. The SQLite store (local mode) is single-user; household isolation is enforced only at the application layer via `WHERE household_id` clauses. This asymmetry is acceptable because SQLite local mode has no multi-tenancy requirement.

## Consequences

- Every store method that touches household-scoped data in PostgreSQL must call `SetHouseholdContext` before the first query. This is a code discipline requirement enforced by code review.
- Connection pooling works correctly because `SET LOCAL` resets on transaction end. No per-connection or per-session state leaks between requests.
- The RLS policies add a small per-query overhead (an extra predicate evaluation). At household scale this is unmeasurable.
- If the `app.household_id` setting is not set (e.g., a code path that forgot to call `SetHouseholdContext`), `current_setting('app.household_id', true)` returns an empty string, which matches no `household_id` — the query returns no rows rather than all rows. This is the safe failure mode.
