# Technical Decisions

## UUIDs for primary keys

Chose UUIDs over auto-increment integers so IDs can't be guessed or enumerated in URLs.
Trade-off: slightly more verbose in Go and SQL.

## Storing hashes of refresh tokens

Only a hash of each refresh token is stored, so a database leak doesn't expose usable tokens.
