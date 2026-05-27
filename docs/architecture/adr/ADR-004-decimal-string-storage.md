# ADR-004: Decimal String Storage for Monetary Amounts

## Status
Accepted

## Context
SQLite's `REAL` type and PostgreSQL's `FLOAT` type use IEEE 754 binary floating-point, which cannot represent most decimal fractions exactly. Storing `$1.10` as a float may round-trip as `1.0999999999999999`. For a financial application this is unacceptable. The alternatives are: (a) store as an integer number of minor units (cents), (b) use the database's `DECIMAL`/`NUMERIC` type, or (c) store as a TEXT decimal string.

Integer storage (cents) requires knowing the currency's decimal places at storage time and breaks for currencies like KWD (3 decimal places) or JPY (0). PostgreSQL `NUMERIC` is correct but SQLite has no true `NUMERIC` type — it silently promotes to float. TEXT storage works identically on both backends and is lossless for any decimal representation.

## Decision
All monetary amounts are stored as TEXT columns containing the exact decimal string representation (e.g., `"1.10"`, `"-500.00"`, `"0.001"`). The currency code is stored in an adjacent column. The application layer uses `shopspring/decimal` for all arithmetic and converts to/from string at the store boundary.

## Consequences
- Arithmetic is never done in the database; all computations happen in Go using `shopspring/decimal`.
- Balance queries (sum of postings for an account) retrieve decimal strings and sum them in Go. This is acceptable for household-scale data volumes.
- Indexes on amount columns are not useful for range queries; date and account indexes are the primary performance levers.
- The `currency` column is always stored alongside `amount`; there is no bare decimal field in the schema.
- Display rounding uses banker's rounding (round-half-to-even) applied at presentation time only, never stored.
