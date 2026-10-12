## MODIFIED Requirements

### Requirement: UAT exchanges CloudBase environment credentials

The UAT runtime SHALL use native CLOUDBASE_APIKEY and CLOUDBASE_ENV_ID with ORDER_TENCENT_REGION to exchange temporary Tencent credentials through the environment's fixed HTTPS `/capi/credential` endpoint for native COS access. It SHALL reject invalid configuration, redirects, failed or invalid responses and unavailable credentials without disclosing secrets. It SHALL NOT fall back to CVM metadata, permanent Tencent keys or use business-secret environment values as Tencent identity. Production SHALL retain its existing CVM role provider. This delta replaces only the base candidate's UAT business-secret environment prohibition; it does not alter the identity protocol.

#### Scenario: UAT native COS SDK needs credentials

- **WHEN** valid UAT configuration receives HTTP200 with numeric code0, a complete temporary credential and expiry covering the maximum fifteen-minute COS signed URL lifetime
- **THEN** the native COS SDK receives that temporary credential
- **AND** the exchange sends only the approved environment body and Bearer API Key to the fixed HTTPS endpoint

#### Scenario: Exchange is unavailable or unsafe

- **WHEN** configuration, transport, response, credential components or expiry are invalid
- **THEN** the credential load fails with a stable non-sensitive error
- **AND** no alternate identity is attempted or failed provider details exposed

### Requirement: Existing cloud secret and upload boundaries remain authoritative

UAT config, WeChat Pay and redemption-token material SHALL read only four fixed CloudRun service-version environment variables through their existing strict SecretSource parsers: order-uat-db-password maps to ORDER_DB_PASSWORD, order-uat-wechat-miniprogram-app-secret maps to ORDER_WECHAT_MINIPROGRAM_APP_SECRET, order-uat-wechatpay-api-v3 maps to ORDER_WECHATPAY_API_V3, and order-uat-redemption-token-key maps to ORDER_REDEMPTION_TOKEN_KEY. Missing, empty or invalid values SHALL fail closed with no SSM, identity, file or alternate environment fallback. The UAT source SHALL reject unknown and production purpose names. Production SHALL keep order-production SSM purpose names and CVM role credentials. The change SHALL preserve PostgreSQL verified TLS, structured fields, permanent Tencent credential prohibition, native COS, private storage and the existing intent/PUT/confirmation upload protocol. Local adapter checks SHALL NOT be reported as CloudRun business deployment, real object roundtrip or complete UAT acceptance.

#### Scenario: UAT reads four business materials

- **WHEN** the existing public loaders read valid UAT platform variables
- **THEN** they retain the structured database and Mini Program credentials and build the native payment client and versioned AES material
- **AND** no SSM or Tencent credential exchange request is made for business material loading

#### Scenario: A required variable or purpose is unavailable

- **WHEN** a UAT value is absent, empty, invalid, cancelled or requested under an unknown or production purpose name
- **THEN** its load fails with a stable non-sensitive error
- **AND** no alternative provider, environment or raw value is disclosed

#### Scenario: Production configuration contains raw business variables

- **WHEN** production configuration contains database-password or Mini Program AppSecret environment variables
- **THEN** its existing prohibition rejects startup before using those values
- **AND** production SSM and COS identity behavior remains unchanged

## ADDED Requirements

### Requirement: UAT platform material rotation uses a new service revision

UAT material updates SHALL become effective by updating CloudRun service-version variables and starting a new revision/process that loads valid same-environment material. The application SHALL NOT promise dynamic in-process rotation, old-revision secret revocation or equivalence to SSM fine-grained access/audit/automatic rotation. Rollback SHALL require still-valid same-environment material; revoked material SHALL NOT be restored merely by routing to an old revision. Existing exact-SHA receipts SHALL NOT be reused as evidence for this delta.

#### Scenario: A UAT material update is applied

- **WHEN** platform variables are updated and the new revision starts
- **THEN** the new process reads the updated material and its effective runtime values are verified without logging them
- **AND** old revision traffic convergence and continued material validity are checked separately
