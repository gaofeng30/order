# Few-shot Examples

## Example 1: TV exhibition food poster intake and strategy

**User input**
> 我要把一个竖版食品品牌宣传图改成45寸1080p电视横屏。左边放品牌吉祥物和烤肠，突出会展价4元，右边放其他商品价目表，不要编号。

**Expected response style**
```markdown
## 先定规格
45寸1080p电视建议做：1920×1080，16:9横屏。不要用原竖图硬拉伸。

## 设计策略
建议采用“左侧2/3主视觉 + 右侧1/3价目板”的结构。左侧第一视觉是“4元/根”，第二视觉是品牌和产品，右侧用菜单灯箱式分组，商品名左对齐、价格右对齐。原图中的手机界面必须删除，只保留品牌黄、吉祥物、奖项感和热卖氛围。

## 需要确认
1. 4元的单位是根、串、个还是份？
2. 右侧商品是否必须全部一页显示？
3. 奖项文案是否真实可用？
```

Then provide a main generation prompt if enough information is available.

## Example 2: Main generation prompt for a food promotion poster

**User constraints**
- brand: 皇家小虎
- size: 1920x1080
- main offer: 皇家小虎烤肠，爆款会展价4元/根
- right-side menu: no numbering
- keep yellow brand tone and mascot
- remove social media UI and “速冻” labels

**Expected prompt**
```text
请基于参考图进行二次创作，生成一张1920×1080、16:9横版会展电视宣传海报。

整体延续参考图中的皇家小虎亮黄色品牌主视觉、可爱小虎吉祥物、商业促销氛围和荣誉感，但不要保留手机截图界面，不要保留竖版构图，不要出现时间、电量、搜索栏、点赞、评论、收藏、头像、定位、账号信息等短视频UI元素。删除“速冻”相关醒目标识。

左侧约三分之二区域为主视觉：包含品牌名“皇家小虎”、主标题“皇家小虎烤肠”、促销文案“爆款会展价”、超大价格“4元/根”。“4元/根”必须是最醒目的视觉焦点。加入小虎吉祥物和诱人的烤肠实物，烤肠要有光泽、热气和食欲感。可加入奖杯、荣誉徽章和热卖贴纸，但不要压过价格。

右侧约三分之一区域为菜单价目板，浅米黄或淡金色底板，深棕文字，商品名左对齐，价格右对齐，不要任何数字编号，不要A/B/C编号。分组为“人气小吃”“酱卤系列”“辣卤系列”。

[粘贴商品清单]

整体要求像专业品牌设计师完成的会展电视广告图，远距离可读，画面干净，高级但有促销冲击力。
```

## Example 3: Review of a first draft

**User sends draft**
A yellow poster with mascot, 4元 price, right-side menu, but logo looks AI-made and menu text has small errors.

**Expected response**
```markdown
## 总体判断
这版是方向稿，评分：82/100。结构已经成立，但还不是最终商用稿。

## 做得好的地方
- 1920×1080横屏适配正确。
- 左侧小虎、烤肠和“4元/根”的主视觉成立。
- 右侧菜单分组清楚，没有编号，符合要求。

## 必须优化的问题
1. 左上角logo像AI临时生成，不够官方。最终最好手动替换官方PNG。
2. 奖项条略抢主标题，建议缩小10%-15%。
3. 右侧文字要放大检查，尤其是“胗、翅、脖、卤、挞”。
4. 烤肠和价格之间略挤，需要更多呼吸空间。

## 下一轮给子Agent的指令
[给出可直接复制的修图指令]
```

## Example 4: Final lock review

**Draft condition**
The poster is strong; only minor risk remains around logo and claims.

**Expected response**
```markdown
这版已经是准最终稿，评分：88/100。不建议再大改，只做最终校对和后期替换。

最终处理方式：
1. 保留当前构图。
2. 后期替换官方logo PNG。
3. 人工放大校对右侧菜单。
4. 确认“4元/根”单位和奖项文案是否真实可用。
5. 导出1920×1080 PNG或JPG。
```

## Example 5: E-commerce hero poster

**User input**
> 做一张咖啡新品电商主图，1:1，用黑金高端风，突出买二送一。

**Expected approach**
- Recommend `1080x1080` or platform-specific size.
- Ask only if product image/logo missing.
- Strategy: center product packshot, large benefit badge, premium texture background, minimal copy.
- Produce main prompt, revision prompt, and text correction prompt.

## Example 6: Store price board

**User input**
> 给奶茶店做价目屏，横屏电视，20个SKU，价格很多。

**Expected approach**
- Warn that 20 SKUs in one AI-generated image may hurt text accuracy.
- Recommend generating visual background/menu frame first, then manually typesetting text.
- If AI must do it, use group headings and a dedicated text correction pass.
- Review should focus on line spacing, price alignment, and legibility.
