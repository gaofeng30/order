## ADDED Requirements

### Requirement: WXML tag structure is enforced by a static gate

每个 `.wxml` 文件的标签 MUST 成对闭合。静态门禁 MUST 检出以下三类结构错误，且 MUST 分别报告而非合并为一句「不配平」：

- 栈已空却出现闭合标签（孤立闭合）；
- 闭合标签与栈顶开启标签名不一致（交叉嵌套）；
- 文件结束时仍有未闭合标签。

不匹配的报告 MUST 同时给出**成对另一端的行号** —— 开发者工具只能指到错误暴露的位置，而需要修改的往往是另一端。

检查 MUST 正确处理属性值内的 `>`（如 `wx:if="{{a > 0}}"`），MUST NOT 因此产生假阳性。

标签配对与既有的 `wx:elif` 同级检查 MUST 共用同一份 void 元素清单与同一次遍历，MUST NOT 各自维护，否则同一文件会在两项检查中被解析成不同的树。

#### Scenario: A deleted block leaves an orphan closing tag

- **WHEN** 某个区块被删除但其闭合标签留在原地
- **THEN** 门禁在提交前报出该孤立闭合标签及其行号
- **AND** 不依赖开发者工具在人工打开该页面时才暴露

#### Scenario: A template uses a comparison inside an attribute

- **WHEN** 模板中出现 `wx:if="{{qty > 0}}"` 一类的属性
- **THEN** 该标签被正确识别
- **AND** 不产生任何结构错误报告

#### Scenario: The repository is audited

- **WHEN** 对全部 `.wxml` 运行门禁
- **THEN** 不存在孤立闭合、交叉嵌套或未闭合标签
