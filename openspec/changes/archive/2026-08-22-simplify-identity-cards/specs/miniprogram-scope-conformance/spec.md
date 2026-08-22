## ADDED Requirements

### Requirement: Identity cards state only the role and never let the auth control carry layout

身份选择页的两张卡片 MUST 只标注身份名称（用户端 / 商户端），MUST NOT 附带描述性副标题。

微信授权控件（`<button open-type="getPhoneNumber">`）MUST NOT 同时承载卡片布局：布局类 MUST 落在其内部的普通容器上。`<button>` 自带 `margin-left/right: auto`、内边距与字号等默认值，其中 auto 外边距在 flex 列容器中会覆盖 `align-self: stretch`，使卡片收缩居中并导致文字逐字折行。在同一元素上与这些默认值竞争 MUST NOT 作为解决方案 —— 压住当前这一版不代表压住下一版基础库。

两张卡片 MUST 使用同一份布局规则，MUST NOT 各自维护样式。

被移除的样式 MUST 一并删除，MUST NOT 留下无人引用的规则。

#### Scenario: The identity screen is rendered

- **WHEN** 用户打开身份选择页
- **THEN** 两张卡片各自只显示身份名称
- **AND** 两张卡片使用同一布局类

#### Scenario: The merchant card markup is audited

- **WHEN** 检查商户端入口的模板
- **THEN** `<button>` 上不存在卡片布局类
- **AND** 布局类位于按钮内部的容器上
