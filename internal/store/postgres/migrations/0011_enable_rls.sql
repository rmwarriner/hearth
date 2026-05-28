-- +goose Up
-- Enable row-level security on all household-scoped tables.
-- This migration must run as the table owner (the role that created the tables).
-- In the docker-compose setup the app user owns all tables, so this works as-is.
-- Cloud deployments where the app role is not the table owner require a superuser
-- to run this migration (e.g. via a one-time DBA step or a privileged migration runner).
--
-- FORCE ROW LEVEL SECURITY is set on each table so that the table owner (the app
-- role in docker-compose) is also subject to the policies. Without FORCE, table
-- owners bypass RLS by default in PostgreSQL, which would nullify the defense-in-depth
-- goal described in ADR-007.
--
-- Every policy has both a USING clause (governs SELECT/UPDATE/DELETE row visibility)
-- and a WITH CHECK clause (governs INSERT/UPDATE target row). This prevents a caller
-- from writing a row into a household they do not own, even if they can read from it.
--
-- The application sets `app.household_id` at the start of every transaction via:
--   SELECT set_config('app.household_id', $1, true)
-- The `true` flag makes the setting transaction-local (resets on commit/rollback),
-- preventing household context from leaking across pooled connections.
--
-- The `households` table does NOT get RLS — it is looked up by ID during
-- authentication before the household context is established.

ALTER TABLE accounts          ENABLE ROW LEVEL SECURITY;
ALTER TABLE journal_entries   ENABLE ROW LEVEL SECURITY;
ALTER TABLE postings          ENABLE ROW LEVEL SECURITY;
ALTER TABLE fiscal_periods    ENABLE ROW LEVEL SECURITY;
ALTER TABLE members           ENABLE ROW LEVEL SECURITY;
ALTER TABLE envelopes         ENABLE ROW LEVEL SECURITY;
ALTER TABLE envelope_allocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens    ENABLE ROW LEVEL SECURITY;

ALTER TABLE accounts          FORCE ROW LEVEL SECURITY;
ALTER TABLE journal_entries   FORCE ROW LEVEL SECURITY;
ALTER TABLE postings          FORCE ROW LEVEL SECURITY;
ALTER TABLE fiscal_periods    FORCE ROW LEVEL SECURITY;
ALTER TABLE members           FORCE ROW LEVEL SECURITY;
ALTER TABLE envelopes         FORCE ROW LEVEL SECURITY;
ALTER TABLE envelope_allocations FORCE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens    FORCE ROW LEVEL SECURITY;

-- Tables with a direct household_id column.
-- USING controls visibility for SELECT/UPDATE/DELETE.
-- WITH CHECK ensures INSERT/UPDATE cannot target a different household.
CREATE POLICY household_isolation ON accounts
    FOR ALL
    USING (household_id = current_setting('app.household_id', true))
    WITH CHECK (household_id = current_setting('app.household_id', true));

CREATE POLICY household_isolation ON journal_entries
    FOR ALL
    USING (household_id = current_setting('app.household_id', true))
    WITH CHECK (household_id = current_setting('app.household_id', true));

CREATE POLICY household_isolation ON fiscal_periods
    FOR ALL
    USING (household_id = current_setting('app.household_id', true))
    WITH CHECK (household_id = current_setting('app.household_id', true));

CREATE POLICY household_isolation ON members
    FOR ALL
    USING (household_id = current_setting('app.household_id', true))
    WITH CHECK (household_id = current_setting('app.household_id', true));

CREATE POLICY household_isolation ON envelopes
    FOR ALL
    USING (household_id = current_setting('app.household_id', true))
    WITH CHECK (household_id = current_setting('app.household_id', true));

CREATE POLICY household_isolation ON refresh_tokens
    FOR ALL
    USING (household_id = current_setting('app.household_id', true))
    WITH CHECK (household_id = current_setting('app.household_id', true));

-- Tables without a direct household_id — policy joins through the parent.
-- WITH CHECK ensures new child rows can only point at a parent owned by the caller.
CREATE POLICY household_isolation ON postings
    FOR ALL
    USING (journal_entry_id IN (
        SELECT id FROM journal_entries
        WHERE household_id = current_setting('app.household_id', true)
    ))
    WITH CHECK (journal_entry_id IN (
        SELECT id FROM journal_entries
        WHERE household_id = current_setting('app.household_id', true)
    ));

CREATE POLICY household_isolation ON envelope_allocations
    FOR ALL
    USING (envelope_id IN (
        SELECT id FROM envelopes
        WHERE household_id = current_setting('app.household_id', true)
    ))
    WITH CHECK (envelope_id IN (
        SELECT id FROM envelopes
        WHERE household_id = current_setting('app.household_id', true)
    ));

-- +goose Down
DROP POLICY IF EXISTS household_isolation ON envelope_allocations;
DROP POLICY IF EXISTS household_isolation ON postings;
DROP POLICY IF EXISTS household_isolation ON refresh_tokens;
DROP POLICY IF EXISTS household_isolation ON envelopes;
DROP POLICY IF EXISTS household_isolation ON members;
DROP POLICY IF EXISTS household_isolation ON fiscal_periods;
DROP POLICY IF EXISTS household_isolation ON journal_entries;
DROP POLICY IF EXISTS household_isolation ON accounts;

ALTER TABLE envelope_allocations NO FORCE ROW LEVEL SECURITY;
ALTER TABLE envelopes             NO FORCE ROW LEVEL SECURITY;
ALTER TABLE members               NO FORCE ROW LEVEL SECURITY;
ALTER TABLE fiscal_periods        NO FORCE ROW LEVEL SECURITY;
ALTER TABLE postings              NO FORCE ROW LEVEL SECURITY;
ALTER TABLE journal_entries       NO FORCE ROW LEVEL SECURITY;
ALTER TABLE accounts              NO FORCE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens        NO FORCE ROW LEVEL SECURITY;

ALTER TABLE envelope_allocations DISABLE ROW LEVEL SECURITY;
ALTER TABLE envelopes             DISABLE ROW LEVEL SECURITY;
ALTER TABLE members               DISABLE ROW LEVEL SECURITY;
ALTER TABLE fiscal_periods        DISABLE ROW LEVEL SECURITY;
ALTER TABLE postings              DISABLE ROW LEVEL SECURITY;
ALTER TABLE journal_entries       DISABLE ROW LEVEL SECURITY;
ALTER TABLE accounts              DISABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens        DISABLE ROW LEVEL SECURITY;
