## ADDED Requirements

### Requirement: 固定 UAT 组合入口

The system MUST dispatch ORDER_ENV=uat before native configuration or database opening, and compose all existing Order routes and five workers using reviewed ordinary SharedPG ports and original cloud providers. Other environments MUST retain their existing native behavior.

#### Scenario: UAT 无原生连接材料

- **WHEN** fixed UAT identity and all required provider materials are valid without native database inputs
- **THEN** UAT configuration and complete composition succeed without native DB, SSM, administrator credential or fake provider use

#### Scenario: 错误或过宽输入

- **WHEN** a required value is missing, fixed identity mismatches, or native/admin/generic cloud credential is explicitly supplied
- **THEN** startup fails before HTTP or workers with a stable safe reason

### Requirement: 保持完整业务与生命周期合同

The UAT runtime MUST preserve Order sessions, PC and owner authorization, original payment/refund verification, atomic notification behavior, fixed image references, worker cadence, ready checks and shutdown. It MUST use the current complete sharedpg registry.

The registry check MUST bind the ordinary service UID, the payment environment and payment callback from the same runtime configuration, and the original derived refund callback. The approved five-field refund and billing configurations MUST use those same values.

#### Scenario: 原服务链路

- **WHEN** existing authenticated HTTP and background operations execute
- **THEN** original DTOs, errors, idempotency, transaction effects and provider semantics are preserved through the complete Shared ports

#### Scenario: 健康与关闭

- **WHEN** schema/ACL/identity drift occurs or cancellation is requested
- **THEN** ready fails safely within its bound or HTTP and all workers drain under the existing shutdown contract

### Requirement: 公开镜像配置与秘密隔离

The dedicated UAT image MUST contain the complete frozen eleven public subscription settings, with the fourteen runtime inputs injected by the service version. Effective configuration MUST be verified on the exact image and release; secrets MUST remain outside build context, source, layers, history and ordinary evidence.

#### Scenario: 发布配置

- **WHEN** the reviewed image and service configuration are deployed
- **THEN** the effective twenty-five inputs match the original loader contracts, with no service override of the public image tuple and actual API acceptance, ready and worker evidence
