## ADDED Requirements

### Requirement: Isolated structured PostgreSQL connectivity
The backend SHALL construct a lazy PostgreSQL pool from explicit structured connection fields, reject invalid or unverifiable TLS configurations without exposing values, impose nonzero connection/time limits and use UTC. It SHALL NOT inherit ambient connection credentials or connect to a database during pool construction.

#### Scenario: Lazy bounded connection
- **WHEN** valid explicit fields name an unreachable local port
- **THEN** Open succeeds with zero open connections and a maximum of ten, and the first bounded connection attempt fails without exposing credentials

### Requirement: Forward-only atomic migration and readiness
The migration runner SHALL use one connection-scoped PostgreSQL advisory lock and apply each ordered migration and its clean history atomically. The runner SHALL retain exact version/name/checksum semantics, reject dirty/mismatched/too-new history and prevent concurrent migration or reuse of a disconnected or still-locked session. The runner SHALL exclusively control transaction boundaries and reject migration content that can escape those boundaries. Unknown lock-acquisition or lock-release results SHALL discard the physical connection. Readiness SHALL inspect schema without writes and refuse incompatible or incomplete required business tables, used columns and key unique, foreign-key and check constraints through a declared bounded contract; it SHALL NOT scan business rows or claim general data-integrity verification.

#### Scenario: Interrupted version does not partially publish schema
- **WHEN** a migration fails or its connection is terminated before commit
- **THEN** that version's DDL/history are rolled back, prior committed versions remain intact, readiness refuses the missing version and a new runner can safely resume

#### Scenario: Migration content attempts an early commit
- **WHEN** a migration includes explicit transaction control before a subsequent failing statement
- **THEN** it is rejected without publishing any of that version's DDL or history, while legitimate multi-statement migrations remain supported

#### Scenario: History is complete but a business object is missing
- **WHEN** all 44 history rows match but orders or its unique prepayment constraint is missing
- **THEN** read-only readiness refuses the schema with a bounded check instead of accepting history alone

#### Scenario: Lock acquisition outcome is unknown
- **WHEN** cancellation or disconnection prevents confirming the advisory-lock result
- **THEN** the physical connection is discarded, no possibly locked session is reused and a subsequent runner can safely acquire the lock

### Requirement: Existing financial and fulfillment invariants
All existing transaction, identity, idempotency, snapshot, one-payment-one-order, trusted-observation-before-ack, refund intent and fulfillment facts SHALL be preserved by explicit PostgreSQL repositories. Timeout SHALL NOT be treated as a close success; stale observations SHALL NOT override paid facts; an unknown refund SHALL reuse its original external refund identifier; expired worker owners SHALL NOT publish late results.

#### Scenario: Duplicate trusted payment observation
- **WHEN** a valid payment is delivered or queried repeatedly and worker processing is restarted
- **THEN** the recoverable payment observation persists before acknowledgment and creates exactly one order with one effective pickup allocation, without changing immutable amount/item/user/time snapshots
