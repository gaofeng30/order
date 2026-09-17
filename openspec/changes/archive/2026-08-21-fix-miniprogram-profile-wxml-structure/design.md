## Context

真实微信开发者工具编译在个人中心模板报出关闭标签不匹配，而仓库现有 `wx-compile-ui1.test.js` 仍为 PASS。直接在正在执行的登录 change 中修改该页面会越过其 owned paths，因此本修复作为无依赖、可独立回滚的前置 change。

## Goals / Non-Goals

**Goals:**

- 让个人中心模板在微信 WXML 编译器中合法解析并保持现有页面内容与交互。
- 让现有零依赖结构检查拒绝多余、错序和未闭合 WXML 标签。
- 用真实微信开发者工具 UI2 验证 exact candidate 的主路径和相邻启动页回归。

**Non-Goals:**

- 不改变个人中心业务行为、组件、样式、身份模型或导航。
- 不引入 XML/WXML 第三方依赖或尝试实现完整微信编译器。
- 不处理登录、正式账号、后端、真机、上传或提审。

## Decisions

### 1. 在现有 linter 中维护打开标签栈

沿用当前 token 正则；每个非自闭合、非 void 标签记录名称与打开行，关闭标签必须精确匹配栈顶，文件结束时栈必须为空。条件分支的同级状态继续按父层维护。相比新增依赖或只为 `profile.wxml` 写正则，这条路径覆盖同类编译阻断且保持 runner 零依赖。

### 2. 用受控负例建立 Red，再修真实模板

先在现有 `wx-compile-ui1.test.js` 创建临时错序标签夹具并要求 linter 非零退出；旧 linter 会错误返回 PASS，形成决定性 Red。随后实现标签栈并删除个人中心唯一多余关闭标签，使同一命令 Green。

### 3. UI2 只使用测试号环境覆盖编译与页面呈现

候选产品 bytes 保持 exact SHA；测试号仅作为开发者工具项目环境注入，不写入候选。主场景实际编译并进入个人中心，相邻回归实际打开启动页。该证据不升级为正式 AppID、真机、登录、支付或提审 PASS。

## Risks / Trade-offs

- [Risk] 正则 token 化不是完整 WXML 语法实现。→ 只声明标签配对和现有条件分支结构能力，最终由微信开发者工具 UI2 作为真实编译 Gate。
- [Risk] 测试号配置与正式 AppID 不同。→ UI2 只证明相同产品源码的编译和模拟器页面；正式平台能力保持未验证。
- [Risk] 修复前置 change 会推进 main，使登录候选失效。→ 集成本 change 后在原登录 writer 上 rebase，重新生成候选并从头验证。

## Migration Plan

1. 独立 writer 完成 Red、Green、Refactor 和 UI2 候选。
2. 两个独立 verifier 对同一 exact SHA PASS 后，按用户既有授权纯快进本地 main 并归档。
3. 登录 writer rebase 新 main，旧候选及旧收据全部失效并重跑。

## Open Questions

无。
