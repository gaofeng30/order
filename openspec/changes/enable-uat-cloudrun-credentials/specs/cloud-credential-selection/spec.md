## ADDED Requirements

### Requirement: UAT exchanges CloudBase environment credentials

The UAT runtime SHALL use native CLOUDBASE_APIKEY and CLOUDBASE_ENV_ID with ORDER_TENCENT_REGION to exchange temporary Tencent credentials through the environment's fixed HTTPS `/capi/credential` endpoint. It SHALL reject invalid configuration, redirects, failed or invalid responses and unavailable credentials without disclosing secrets. It SHALL NOT fall back to CVM metadata, permanent Tencent keys or raw business-secret environment values. Production SHALL retain its existing CVM role provider.

#### Scenario: UAT native SDK needs credentials

- **WHEN** valid UAT configuration receives HTTP200 with numeric code0, a complete temporary credential and expiry covering the maximum fifteen-minute COS signed URL lifetime
- **THEN** the native SDK receives that temporary credential
- **AND** the exchange sends only the approved environment body and Bearer API Key to the fixed HTTPS endpoint

#### Scenario: Exchange is unavailable or unsafe

- **WHEN** configuration, transport, response, credential components or expiry are invalid
- **THEN** the credential load fails with a stable non-sensitive error
- **AND** no alternate identity is attempted or failed provider details exposed

### Requirement: Long-running SDK access refreshes atomically

The UAT credential SHALL refresh at least fifteen minutes before expiry and reject new credentials that cannot cover the maximum COS signed URL lifetime with bounded requests and shared concurrent refresh. A failed refresh SHALL make credentials unavailable rather than returning old values. SSM and COS SHALL consume one complete credential tuple for each signing operation; an incomplete tuple SHALL prevent COS requests and presigned URLs.

#### Scenario: Concurrent operations reach refresh time

- **WHEN** multiple operations request credentials after the cache's refresh time
- **THEN** they share one exchange in progress and receive complete renewed tuples
- **AND** subsequent operations use the valid cache without another exchange

#### Scenario: Refresh fails

- **WHEN** the next exchange fails or returns unusable credentials
- **THEN** no operation receives old or partial credentials
- **AND** a later successful exchange can restore access

### Requirement: Existing cloud secret and upload boundaries remain authoritative

UAT config, WeChat Pay and redemption-token material SHALL use their existing order-uat SSM purpose names through the selected native credential. Production SHALL keep order-production purpose names and CVM role credentials. The change SHALL preserve PostgreSQL TLS validation, the four-purpose secret boundary, native COS, private storage and the existing intent/PUT/confirmation upload protocol. Local adapter checks SHALL NOT be reported as CloudRun deployment, successful SSM access, real object roundtrip or complete UAT acceptance.

#### Scenario: UAT reads business secret material

- **WHEN** the existing runtime loads its database, Mini Program, payment or redemption-token material
- **THEN** the request uses its existing UAT SSM purpose name and selected UAT temporary credential
- **AND** raw secrets, TLS relaxation and fake cloud providers remain prohibited
