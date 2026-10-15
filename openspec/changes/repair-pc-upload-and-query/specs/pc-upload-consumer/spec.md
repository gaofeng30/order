## ADDED Requirements

### Requirement: PC images consume the approved upload protocol

The PC image consumer SHALL use the existing server-generated short-lived intent, PUT and JSON confirmation protocol, preserving one upload identity and its MIME, size and SHA256 across both API phases. It SHALL send raw bytes and only the returned storage headers to the PUT destination, without Admin bearer or cookies. It SHALL expose an image only after a matching successful confirmation and SHALL reject every failed phase without changing product/layer facts. It SHALL NOT introduce multipart compatibility, new backend policy, automatic retries or anonymous storage access.

#### Scenario: An existing PC editor selects a valid image

- **WHEN** the editor uploads an image under the existing authoritative storage contract
- **THEN** intent, PUT and confirmation agree on server object key and the unchanged file specification
- **AND** the successful confirmed image can be used by the existing product/layer flow

#### Scenario: A phase fails

- **WHEN** intent, raw PUT or confirmation fails or returns an inconsistent key
- **THEN** the upload rejects and no unconfirmed image is presented as successful
- **AND** existing product/layer failure shields remain visible

### Requirement: Current PC order query remains visible

The PC order consumer SHALL keep the most recently requested lane/date/search result in its shared read state and rendered order page when earlier responses arrive later. Each read SHALL still return its own server result to its caller. The fix SHALL preserve all six public order states, refund finality, immutable paid amounts, lanes, selected dates and search behavior.

#### Scenario: A previous refunding lane arrives after a refunded-order search

- **WHEN** a refunding lane request starts before an order-number search at the same selected date
- **AND** the refunded-order search completes before the previous empty lane response
- **THEN** the refunded search remains available and visible with its server-backed detail
- **AND** the previous caller receives its empty result without replacing current state or rendering


### Requirement: PC authorization failure and visible logout remove old authorized state
The PC client SHALL call the existing current-token server logout endpoint from the visible logout action. Successful204 or confirmed401 SHALL clear local authorization and display QR login. Network or5xx failures SHALL retain truthful failed state with manual retry and SHALL NOT claim server revocation or retry automatically. A current authenticated401 SHALL clear token, account and authorized presentation;403 SHALL retain the session. An old in-flight response SHALL NOT restore expired authorization.

#### Scenario: Visible logout succeeds for one session
- **WHEN** the server accepts the presented session logout
- **THEN** that session is revoked and local authorized state is cleared
- **AND** another valid session remains usable

#### Scenario: Visible logout service is unavailable
- **WHEN** logout fails with a network or5xx error
- **THEN** the client retains local state, reports failure and permits manual retry without automatic retry

#### Scenario: An active page receives401
- **WHEN** an already-open page or bootstrap receives current401
- **THEN** token and old authorized data disappear and QR login is shown
- **AND** a late response cannot restore the old data while403 alone does not clear the session


### Requirement: Confirmed PC data rendering preserves server facts and selected scope

The PC client SHALL render authenticated account identity, all six server order states and global counts from complete cursor pages; unavailable counts SHALL remain unknown. It SHALL preserve server order notes, flavor and payable line subtotal facts. Finance amounts, rows and tabs SHALL remain bound to the latest selected server-date query, and CSV filenames to the export click's date range. Failed settings reads SHALL expose manual recovery and SHALL NOT display fabricated editable settings. Existing routes and business semantics SHALL remain unchanged.

#### Scenario: A finance query completes after a newer selected query

- **WHEN** the selected dates or tab change while prior finance reads remain in flight
- **THEN** old success or failure cannot replace the current amounts, rows or badges
- **AND** incomplete or failed current facts remain unavailable with visible manual retry

#### Scenario: Settings first read fails and a manual retry succeeds

- **WHEN** settings cannot be read initially
- **THEN** the page displays loading/failure and an explicit retry action without invented fields
- **AND** the successful retry renders only authoritative server fields

#### Scenario: Current401 has an unreadable response body

- **WHEN** the current session receives401 whose JSON or text cannot be parsed
- **THEN** token, account and authorized caches are cleared and the shell receives the login event
- **AND** malformed403/503 and a late old401 do not clear a new authenticated workspace

#### Scenario: A successful old continuation runs after concurrent authorization failure

- **WHEN** a request has passed its response check but a concurrent current401 or replacement login changes authorization before the outer cache write
- **THEN** the old bootstrap, settings, catalog, pending-payment, sold-out or final image confirmation continuation rejects SESSION_EXPIRED
- **AND** it cannot restore old account, store, sale, category, pending or private-image facts
