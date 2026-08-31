# fix-profile-layout — tasks

Base: `8fe591d` (origin/main)
Branch: `worktree-fix-profile-layout`
Worktree: `.claude/worktrees/fix-profile-layout`

Owned paths:
```
apps/wechat-miniprogram/pages/profile/profile.js
apps/wechat-miniprogram/pages/profile/profile.wxml
apps/wechat-miniprogram/pages/profile/profile.wxss
apps/wechat-miniprogram/tests/overnight-user-orders-profile-ui0.test.js
tools/miniprogram-ui/run-ui2.mjs
.scratch/fix-profile-layout/
```

## 1. 冻结范围

- [x] 1.1 定位两处根因，取得代码证据（`.prow` 单行 4 元素；`.extra-*` 三类零定义；`app.wxss:56` 仅 `box-sizing`）
- [x] 1.2 确认附加手机号为单条：PRD §4.1、`000018` 约束、`CONTEXT.md:13`
- [x] 1.3 确认 `confirm.wxml:78` 为无 class 裸 button，排除全局复位路线
- [x] 1.4 确认 UI1 karma 不加载 wxss，几何断言只能放 UI2
- [x] 1.5 确认服务端 `extra_phone.name` 可用于预填（实测返回 `"name":"李四"`）
- [x] 1.6 写 `spec.md`，声明边界与验收对应

## 2. Red

- [x] 2.1 在 `overnight-user-orders-profile-ui0.test.js` 补状态机断言：初始收起、展开、保存后自动收起、姓名预填
- [x] 2.2 同文件补 wxml/wxss 源契约断言：`.extra-*` 三类有定义、三处行不再是 `<button class="prow">`、存在 `.prow-tap`
- [x] 2.3 运行 `npm --prefix apps/wechat-miniprogram test`，记录决定性失败

  新增 4 条全红、既有 11 条全绿。决定性失败：
  ```
  ✖ extra phone form stays collapsed until opened and re-collapses after a save
      undefined !== false            (page.data.extraOpen 不存在)
  ✖ extra phone save stays disabled until both factors are present
      undefined !== false            (page.data.extraSavable 不存在)
  ✖ profile styles define every class the extra phone panel renders
      AssertionError: profile.wxss is missing .prow--extra
  ✖ interactive rows never let a native button carry the row layout
      AssertionError: a native button still carries the .prow layout class
  ```

## 3. Green

- [x] 3.1 `profile.js`：加 `extraOpen` / `extraName` / `extraSavable`、`toggleExtra()`、`refreshExtraSavable()`、保存成功后收起、姓名预填
- [x] 3.2 `profile.wxml`：附加手机号行改折叠面板结构
- [x] 3.3 `profile.wxml`：三处交互行改 view 容器 + `.prow-tap` 透明点击层
- [x] 3.4 `profile.wxss`：补 `.prow--extra` / `.prow-head` / `.extra-*` / `.prow-tap` 规则
- [x] 3.5 重跑 UI0 与 wx lint 到 Green — `tests 105 / pass 105 / fail 0`，含 `wxss and wxml are structurally compilable`

### Green 过程中的两处修正

1. `icon` 组件无 `chevronDown` 资源（图标表仅 21 个名字，只有 `chevron`）。改为保留 `chevron`，用 `.prow--extra.is-open .extra-caret { transform: rotate(90deg) }` 做展开指示，不发明资源。
2. 初版断言用 `class="[^"]*\bprow\b[^"]*"`，`\b` 在 `prow-tap` 的连字符处也成立，导致修好后仍误报。改为提取 class 属性后按 token 精确比对。已回放验证该断言在原始 HEAD 上仍为 Red：

   ```
   原始 HEAD -> 违规 button 数: 3 | 点击层数: 0
   修改后   -> 违规 button 数: 0 | 点击层数: 3
   ```

## 4. Refactor

- [ ] 4.1 只做清晰度重构，重跑同一套门禁

## 5. 几何回归

- [x] 5.1 `run-ui2.mjs` 加 profile 页几何断言（五行等宽、标题不逐字换行、5:3 分栏不重叠、保存全宽且独占行），测量值写入 receipt
- [ ] 5.2 跑 UI2 — **未取得几何证据，阻断于既有缺陷**

### UI2 接入实况

工具链本身可用，三个障碍里前两个已修复：

| # | 现象 | 结论 |
|---|---|---|
| 1 | `Failed connecting to ws://127.0.0.1:19420` | **已修**。`cli auto` 只是请求打开项目窗口，自动化端口要等窗口就绪才监听，原代码立即 connect 必然抢跑。加 `connectWithRetry`（30×1s）。 |
| 2 | fixture 硬绑 `127.0.0.1:8080`，与本地 `order-api` 冲突 | **已识别**。UI2 需独占 8080；跑前必须停掉本地 `order-api`。 |
| 3 | `unbound cold start stopped at pages/launch/launch with entryState="loading"` | **未解决，判定为既有缺陷** |

第 3 项的排除过程：

- 把 profile 三个文件 `git stash` 回基线后重跑，**同样失败**，故与本 change 无关
- 冷启动轮询窗口从 20×100ms 放宽到 60×250ms（2s → 15s），**仍停在 `loading`**，故非竞态
- 环境读数正常：开发者工具 `2.01.2510290`、基础库 `3.6.6`、CLI 已登录、项目权限探测通过

即入口分流的 `entryRouting` 始终未落定。这属于 launch 冷启动链路，不在本 change 的 owned paths，按 AGENTS.md「不得顺手处理相邻问题」不在此修。

保留几何断言代码：它本身正确，待第 3 项修复后即可产出 receipt。

**P0-6.5 未关闭**，阻断原因是既有 UI2 冷启动缺陷，不是外部资产缺失，因此**不标 `BLOCKED_EXTERNAL`**——那个标签是给平台/账号类外部依赖的，用在这里会掩盖一个自有缺陷。

## 5b. 第二批：首页品牌与图标补齐

- [x] 5b.1 Red：补 3 条源契约断言（五行图标齐备且图标名在 icons.js 内、首页无「服务功能」且 hero 已收紧、hero 含徽标品牌行）
- [x] 5b.2 记录决定性失败 — `tests 18 / pass 15 / fail 3`
- [x] 5b.3 `home.wxml` 删 `.sec-h` 标题；`home.wxss` hero 底部 `128rpx → 48rpx`、`.body` margin-top `-88rpx → 24rpx`
- [x] 5b.4 `home.wxml` 加 `.hero-brand` 品牌锁定行；`home.wxss` 加 `.hero-brand` / `.hero-logo` / `.hero-emblem`
- [x] 5b.5 `profile.wxml` 三行补 `.prow-ico`：`phone` 绿、`user` 绿、`store` 蓝
- [x] 5b.6 转 Green — `tests 108 / pass 108 / fail 0`，含 wxml/wxss 编译门禁

前置确认：全仓库测试无人引用「服务功能」或 `sec-h`；`.grid-item` 被三个 UI1 spec 断言（数量 3、文案、dataset），本次不动宫格项本身。

## 6. 本地验收

- [ ] 6.1 `npm --prefix apps/wechat-miniprogram test`（14 套件）
- [ ] 6.2 `python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram`
- [ ] 6.3 UI1 composed 门禁（需 order-api 在跑）
- [ ] 6.4 开发者工具 iPhone SE 与 iPhone 14 人工确认
- [ ] 6.5 产出候选 SHA，工作区干净

## 7. 独立验证

- [x] 7.1 另一个干净 detached worktree、精确候选 SHA、只读重跑全部门禁
- [x] 7.2 记录 PASS/FAIL、SHA、命令结果与剩余限制

### 验证结果：PASS（有限定）

精确 SHA `ab3e77eafc241ecc2be0ebdac2e95472e31d8c2f`，全新 detached worktree，未复用实现 worktree 的任何产物。

```
npm --prefix apps/wechat-miniprogram test
  → tests 108 / pass 108 / fail 0   （含 wxss/wxml 编译门禁）

python3 apps/wechat-miniprogram/tests/lint_wx.py apps/wechat-miniprogram
  → WX_LINT=PASS

git status -s
  → 空（验证过程未修改任何跟踪文件，满足只读要求）
```

### 剩余限制（未被本次验证覆盖）

1. **几何证据缺失**。P0-6.5 要求断言真实 bounding box、行宽与无逐字换行的 rendered 回归，UI2 代码已就位但跑不出结果，阻断于既有冷启动缺陷（`entryRouting` 恒停 `loading`，基线代码同样复现）。本次交付**不含**几何证据，P0-6 第 5 条**未关闭**。
2. **UI1 composed 门禁未跑**。它需要 `order-api` + MySQL 在跑，且 fixture 硬绑 `127.0.0.1:8080` 与之冲突，本次未纳入验证。
3. **真机与开发者工具人工核对未做**。P0-6 第 4 条要求小屏与标准屏分别验证，未执行。
4. 验证层级为 `UI0_LOCAL` + 静态源契约，不构成 UI2/UI3 证据。

即：**代码层面的结构与状态机已独立验证；视觉与几何层面未取得证据。**

## 决定记录

| 决定 | 结论 | 依据 |
|---|---|---|
| 拆分粒度 | 单 change | 同页同批元素，P0-6 作为一组验收 |
| 条数 | 单条，不做加号 | PRD §4.1 是已确认产品决策，属硬约束 |
| 收起态 | 不显示结果 | 用户指定；附带避免脱敏号常驻曝光 |
| 复位作用域 | `button.prow` 局部 | 保护 `confirm.wxml:78` |
| button 处理 | 点击层，退出布局流 | `isolate-auth-hit-area/design.md` 记录纯复位已失败两次 |
| 几何层 | UI2 | UI1 无 wxss，无样式 DOM 上测几何无意义 |
| UI2 不可用 | 标 BLOCKED_EXTERNAL，代码照常集成 | 不让布局修复被从未跑通的工具链阻塞 |
