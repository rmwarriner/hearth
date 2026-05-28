# ADR-005: JWT Authentication Model

## Status
Accepted

## Context
Phase 2 introduces server mode (`hearthd`) with a multi-user REST API. Financial data demands strong session security: stolen credentials must be revocable, and token theft must be detectable. Two approaches were evaluated:

1. **Fully stateless JWT** — both access and refresh tokens are signed JWTs; no server-side state. Simple to implement, horizontally scalable with no shared state.
2. **Short-lived access JWT + DB-backed rotating refresh tokens** — access tokens are stateless JWTs (15 min TTL); refresh tokens are random values stored hashed in the database with rotation on every use.

The stateless approach was rejected because there is no path to revoke a compromised refresh token before its expiry. For a financial application where a stolen refresh token could grant week-long access to account data, this is unacceptable.

## Decision

**Access tokens**: Short-lived signed JWTs (15-minute TTL). Claims include `mid` (member ID), `hid` (household ID), and `role`. Validated on every API request without a database lookup.

**Refresh tokens**: 32-byte cryptographically random values, base64url-encoded. Stored as a bcrypt hash (cost 12) in the `refresh_tokens` table. Expire after 7 days. On every use, the old token is immediately revoked and a new pair is issued (**rotation**).

**Token families**: Each issuance chain (login → refresh → refresh → ...) shares a `family_id` UUID. If a revoked token is presented (replay attack), the entire family is revoked — all active tokens derived from the same login event are invalidated. The member must re-authenticate.

**Logout**: Revokes the presented refresh token. The access token expires naturally (15 min max).

**JWT signing algorithm**: HS256 with a server-side secret of at least 32 bytes. The secret is provided via `HEARTH_JWT_SECRET` environment variable; the server aborts on startup if it is absent or too short.

## Consequences

- Every refresh requires a database write (revoke old, create new). This is one extra write per refresh event, not per API request — acceptable overhead.
- Horizontal scaling requires a shared database (already required by PostgreSQL). No sticky sessions or shared in-memory state needed.
- Token family revocation is aggressive: a replay of any token in a chain revokes all sessions from that login. Users will need to re-authenticate if a token is compromised and replayed. This is the correct trade-off for a financial application.
- Bcrypt at cost 12 is intentionally slow (~250ms). Auth service unit tests use `bcrypt.MinCost` (4) via a config field to keep tests fast.
