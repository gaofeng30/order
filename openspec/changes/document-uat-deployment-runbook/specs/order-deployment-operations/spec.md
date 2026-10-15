## ADDED Requirements

### Requirement: 可执行的 UAT 部署指南

部署指南 SHALL 只包含正确操作、前置、完成标准与回退，且 SHALL 以实际代码/官方平台能力区分资源功能限制、配置限制和运行容量。

#### Scenario: 按顺序部署 UAT

- **GIVEN** 正式发布版本与本环境资源已冻结
- **WHEN** 操作者执行指南
- **THEN** 先验证客户原生私网 PG、证书校验、完整配置大小、身份和连接预算
- **AND** 单独执行迁移与首次空租户 bootstrap，再发布常驻 API、同源网关/静态和小程序体验版
- **AND** 只有真实 readiness、客户端、完整矩阵、容量及恢复标准满足后才能放量

### Requirement: 生产参考边界

指南 SHALL 明确 UAT CloudRun/PG 与现行生产 CVM/MySQL/SSM 的适用差异，SHALL NOT 将 UAT 配置直接复制为生产方案，或将文档检查表述为真实业务部署成功。

#### Scenario: 为生产准备发布

- **GIVEN** 生产负责人参考 UAT 操作顺序
- **WHEN** 准备生产发布
- **THEN** 使用生产批准的架构、引擎、四份 SSM、身份、域名与 release 客户端配置
- **AND** 生产改变承载或引擎须完成相应受审变更与实际验证
- **AND** 回退不得自动 down、覆盖 bootstrap 状态或重放外部资金操作
