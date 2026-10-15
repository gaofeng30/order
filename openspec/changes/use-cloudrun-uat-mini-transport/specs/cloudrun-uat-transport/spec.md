## ADDED Requirements

### Requirement: Fixed trial HTTPS entry

The Mini Program SHALL access Order UAT trial through wx.request using the fixed normalized origin https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com, with the existing path, method, DTO and headers. It SHALL preserve wx.login code→server Code2Session→Order bearer and never replace business identity with client OpenID or CloudBase database JWT. It SHALL require neither wx.cloud linkage nor CloudBase client authentication/database SDK.

#### Scenario: Trial login and authenticated read/write

- **WHEN** the configured trial gateway and original backend are available
- **THEN** the existing session exchange uses 201/exact session DTO and subsequent requests carry original Authorization/Idempotency-Key with original HTTP status/data semantics
- **AND** all requests use the fixed origin plus existing path without callContainer, alternative targets or automatic write retry

#### Scenario: Trial invalid context and request failure

- **WHEN** configuration is absent, is a legacy tuple, differs from the fixed HTTPS origin or has an unknown environment version
- **THEN** runtime fails closed with no login or network call
- **WHEN** wx.request fails or throws synchronously
- **THEN** the original consumer failure is preserved without exposing private request errors or switching targets

### Requirement: Existing develop/release and consumer behavior

The Mini Program SHALL preserve develop localhost, keep unconfigured release unavailable, retain asynchronous native random key generation and all awaiting call sites, and preserve reservation query encoding and INVALID_MENU_SELECTION handling. Non-develop environments SHALL preserve exact original server signed HTTPS media URL bytes and reject relative localhost image paths.

#### Scenario: Develop request and release refusal

- **WHEN** develop is selected
- **THEN** the original strict loopback origin, local object URLs and wx.request/session behavior remain
- **WHEN** release lacks explicit real configuration
- **THEN** it does not log in or borrow trial configuration

#### Scenario: SDK randomness and explicit retry

- **WHEN** native random bytes are pending or rejected
- **THEN** a dependent business write waits or sends no request, with original IDEMPOTENCY_UNAVAILABLE mapping
- **WHEN** the user explicitly retries an existing write with its original key
- **THEN** transport preserves that key and performs only the explicitly requested attempt

#### Scenario: Signed media URL preservation

- **WHEN** trial or an explicitly configured release consumes a server signed HTTPS media URL
- **THEN** signature/version query bytes are returned unchanged and a relative /api/v1/objects/ path is rejected
