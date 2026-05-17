# Review Rubric for Product Poster Drafts

Use this rubric when the user sends a generated poster draft.

## Score bands
- **0-59**: not usable; core direction failed. Needs redesign.
- **60-74**: direction draft; some core ideas visible but major layout/text/brand issues remain.
- **75-84**: promising draft; structure works but needs commercial polish.
- **85-91**: near-final; only local fixes, proofreading, or brand asset replacement needed.
- **92-100**: final candidate; ready after human verification of factual claims and text.

## Review dimensions

### 1. Format and channel fit
Check:
- correct aspect ratio and pixel target.
- landscape vs portrait.
- TV readability if applicable.
- no unintended safe-area cropping.

### 2. Brand recognition
Check:
- brand color accuracy.
- logo accuracy and placement.
- mascot/product packaging consistency.
- whether AI invented an unreliable logo.

### 3. Visual hierarchy
Check:
- first-glance focal point.
- price and promotion strength.
- title vs subtitle vs support text.
- whether awards or decorations overpower the main offer.

### 4. Product appetite or desirability
For food/retail:
- product looks appealing and recognizable.
- product does not look distorted, plastic, raw, burnt, or unrelated.
- packaging and real product are balanced.

### 5. Information architecture
Check:
- grouped product list.
- no unnecessary numbering if user requested none.
- price alignment.
- line spacing.
- menu density.
- whether all required items are included.

### 6. Text accuracy
Check every critical character, especially Chinese text. Flag fragile characters and require manual zoom-proofing.

### 7. Commercial-use risk
Flag:
- official logos generated from scratch.
- exact award/certificate replication without authorization.
- unsupported claims such as “first”, “best”, “official”, “Forbes award”.
- misleading pricing units.
- platform UI remnants from screenshots.

### 8. Finish quality
Check:
- AI artifacts.
- clutter.
- background consistency.
- object edges.
- contrast.
- final export readiness.

## Review output template
```markdown
## 总体判断
这版是 [阶段]，评分：[分数]/100。建议：[重做/继续精修/准最终/可定稿]。

## 做得好的地方
[3-5条具体优点]

## 必须优化的问题
[按优先级列出具体问题]

## 商用落地检查
[logo、奖项、价格单位、文字校对、授权风险]

## 下一轮给子Agent的指令
[可直接复制的修图指令]
```
