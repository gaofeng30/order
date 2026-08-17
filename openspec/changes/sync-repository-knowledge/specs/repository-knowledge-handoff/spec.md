## ADDED Requirements

### Requirement: Developer documentation states the current data boundary
The repository SHALL state consistently that the Go service persists and serves the anonymous menu catalog, the WeChat mini program home/menu/detail paths consume that catalog, and orders, payment, merchant management and Web Admin remain mock or in-memory behavior.

#### Scenario: Reader identifies the implemented catalog path
- **WHEN** a reader opens the root README, mini-program README, Demo guide or technical document
- **THEN** the document identifies the API-backed catalog without claiming that all front-end behavior is persistent

#### Scenario: Stale all-mock wording is rejected
- **WHEN** the changed documents are checked for the former claims that both frontends do not call the API or all mini-program data is mock
- **THEN** those claims are absent

### Requirement: Repository-local development entry is discoverable
The root README SHALL provide the minimum commands for inspecting and checking the lightweight Harness and SHALL identify its Git common-dir ledger as a local operational index rather than lifecycle, verification or integration proof.

#### Scenario: New developer uses the Harness entry
- **WHEN** a developer follows the root README from a checkout containing `tools/harness`
- **THEN** they can run `./tools/harness status` and `./tools/harness check` and understand the evidence boundary

### Requirement: Documentation separates verified code from unavailable external proof
The changed documents MUST distinguish repository tests and simulated UI evidence from isolated MySQL, WeChat developer tool, real-device, payment, UAT, deployment and production evidence.

#### Scenario: External evidence is unavailable
- **WHEN** the repository has no current external asset or runtime proof
- **THEN** documentation states the boundary and does not label the external behavior as implemented or verified

### Requirement: Handoff links resolve to current repository paths
All relative Markdown links in the changed handoff documents SHALL resolve to existing files or directories in the candidate tree.

#### Scenario: Local link audit
- **WHEN** the candidate scans relative Markdown links in each changed handoff document
- **THEN** every non-anchor, non-HTTP target exists after URL decoding and anchor removal
