# 味在绥安 · 8 电视价目海报

横屏 1536×1024（≈ 90×60 cm 3:2）国潮风格价目海报，按门店内 8 块电视分别出图。

## 目录

| 路径 | 说明 |
| --- | --- |
| `raw-reference.png` | 风格参考图（哪吒 + 味在绥安 + 价目板） |
| `价目表.docx` | 业务方原始价目表 |
| `价目表.xlsx` | 由 docx 精准转换的电子价目表，三列「电视编号 / 品名 / 价格」，电视编号纵向合并 |
| `scripts/docx-to-xlsx.py` | docx → xlsx 转换脚本（python-docx + openpyxl） |
| `scripts/generate-posters.mjs` | 调 gpt-image-2 批量出海报的脚本 |
| `prompts/0X-…txt` | 每张海报的完整 prompt（可手工调） |
| `posters/0X-…png` | 8 张最终海报 |
| `posters/manifest.json` | 每张图的生成参数、用量、状态 |

## 角色映射

按品类语义匹配《哪吒之魔童闹海》同电影 IP：

1. 电视一 烤肠卤味档 → **太乙真人**
2. 电视二 镇特产 → **殷夫人**
3. 电视三 东北盖饭 → **李靖**
4. 电视四 荤素套餐 → **哪吒**（与参考图一致）
5. 电视五 茶咖 → **敖丙**
6. 电视六 零售/饮料/水 → **敖光**（东海龙王）
7. 电视七 组合套餐 → **鹤童 + 鹿童**
8. 电视八 甜品水果 → **申小豹**

## 运行

```bash
# 1) 价目表 docx → xlsx
python3 scripts/docx-to-xlsx.py

# 2) 一键生成 8 张海报（默认并发 4）
node scripts/generate-posters.mjs

# 仅预览 prompt，不出图
node scripts/generate-posters.mjs --dry-run

# 重抽指定电视（可多选，逗号分隔）
node scripts/generate-posters.mjs --only 6
node scripts/generate-posters.mjs --only 1,3,7

# 调整并发
node scripts/generate-posters.mjs --concurrency 2
```

## 依赖

- Python 3 + `python-docx`、`openpyxl`（`pip3 install --user python-docx openpyxl`）
- Node 20+
- OpenAI key 来自 `/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env`，模型 `gpt-image-2`

## 备注

- 走纯 AI 出图路线（含品名与价格），少量中文笔画可能偶有变形；如某张明显失败，用 `--only N` 单张重抽即可。
- 海报中除「味在绥安」标题与价目板上的品名/价格外，画面禁止出现任何其他中文/英文广告语。
- 重抽时会复用并合并 `posters/manifest.json`，不会丢失其他海报的记录。
