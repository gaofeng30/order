#!/usr/bin/env python3
"""
菜品照片生成脚本 —— Vertex AI / Nano Banana 2 (Gemini 3.1 Flash Image Preview)

为小程序里的每个菜品生成「有食欲、商业宣传级」的真实照片，
输出为 1:1 的 PNG，命名与菜品 id 一致（p001.png ...），
存到 miniprogram/assets/dishes/，并自动把 mock/data.js 里对应的
image 路径从 .svg 改成 .png。

────────────────────────────────────────────────────────────
在你自己的 Mac 上运行（沙箱内没有外网，无法直接调用 Vertex）：

    cd "/Users/marcusz/Projects/Test Tool/order/tools/generate-dish-images"
    python3 -m pip install --upgrade google-genai pillow
    python3 generate_dish_images.py            # 生成全部
    python3 generate_dish_images.py p003 p005  # 只生成指定菜品
    python3 generate_dish_images.py --no-update-data   # 不改 data.js

可选环境变量：
    GEMINI_CRED   服务账号 json 路径（默认用 ../../../gemini-reference 里的那个）
    IMAGE_MODEL   图像模型（默认 gemini-3.1-flash-image-preview，即 Nano Banana 2）
                  其他可选：gemini-3-pro-image-preview (Pro) / gemini-2.5-flash-image (初代)
    VERTEX_LOCATION  区域（默认 global）
────────────────────────────────────────────────────────────
"""

import os
import re
import sys
import json
import mimetypes
from pathlib import Path

# ── 路径定位 ──────────────────────────────────────────────────────────
SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[1]                 # .../order
DISHES_DIR = REPO_ROOT / "miniprogram" / "assets" / "dishes"
DATA_JS = REPO_ROOT / "miniprogram" / "mock" / "data.js"

# 凭证：优先环境变量，其次 gemini-reference（与 order 同级）
CRED_CANDIDATES = [
    os.environ.get("GEMINI_CRED"),
    REPO_ROOT.parent / "gemini-reference" / "vivi-gemini-0330-a95a1a91012f.json",
]

VERTEX_LOCATION = os.environ.get("VERTEX_LOCATION", "global")
IMAGE_MODEL = os.environ.get("IMAGE_MODEL", "gemini-3.1-flash-image-preview")  # Nano Banana 2

# ── 统一的摄影风格后缀 ─────────────────────────────────────────────────
STYLE = (
    "Professional commercial food photography, mouth-watering and appetizing, "
    "shot on a full-frame camera with an 85mm lens, shallow depth of field, "
    "soft natural window light from the side, gentle steam where appropriate, "
    "fresh vivid ingredients, glossy and juicy textures, fine condensation/oil sheen, "
    "clean modern restaurant table styling, neutral light background, "
    "high resolution, ultra realistic, photorealistic, no text, no watermark, "
    "no people, square 1:1 centered composition, top-down or 45-degree angle."
)

# ── 每个菜品的针对性 prompt ─────────────────────────────────────────────
PROMPTS = {
    "p001": "A hearty Chinese business set rice bowl: tender black-pepper beef slices and "
            "sliced low-temperature poached chicken breast over fluffy white rice, with "
            "bright stir-fried seasonal greens and a small bowl of clear soup on the side.",
    "p002": "A light Jiangnan-style set meal: plump stir-fried shrimp with assorted mushrooms "
            "and crisp seasonal vegetables, served with steamed rice and a small bowl of "
            "seaweed-and-egg-drop soup, delicate and clean presentation.",
    "p003": "Signature Chinese braised beef brisket (hong shao niu nan): glossy chunks of "
            "slow-braised beef in a rich mahogany sauce, garnished with scallion and star anise, "
            "served over steamed rice, deep savory glaze, irresistible.",
    "p004": "Garlic chicken thigh steak: a juicy boneless chicken thigh fillet, golden seared "
            "and glistening with garlic sauce, sprinkled with chopped garlic and herbs, "
            "served with rice and a few greens, crispy skin texture.",
    "p005": "A healthy quinoa and chicken breast energy bowl: sliced grilled chicken breast, "
            "fluffy tricolor quinoa, ripe avocado fan, cherry tomatoes, edamame and fresh greens, "
            "with vinaigrette in a small dish, fresh low-fat clean-eating styling.",
    "p006": "Chinese yam and pork rib soup (shan yao pai gu tang): a clear nourishing broth "
            "with tender pork ribs and chunks of white Chinese yam and goji berries, "
            "steaming hot in a ceramic bowl, comforting and homestyle.",
    "p007": "Fresh orange sparkling water: a tall glass of chilled fizzy sparkling water with "
            "fresh orange slices and ice cubes, lively rising bubbles, water droplets on the glass, "
            "bright and refreshing, sunlit cold-drink styling.",
}


def build_prompt(pid: str, name: str = "", intro: str = "") -> str:
    base = PROMPTS.get(pid)
    if not base:
        # 未来新增菜品的通用回退：用名称 + 简介拼接
        base = f"A delicious dish: {name}. {intro}".strip()
    return f"{base} {STYLE}"


# ── 从 data.js 里取出菜品（id / name / intro）──────────────────────────
def load_products():
    text = DATA_JS.read_text(encoding="utf-8")
    items = []
    for m in re.finditer(r'id:\s*"(p\d+)"', text):
        pid = m.group(1)
        seg = text[m.start(): m.start() + 600]
        nm = re.search(r'name:\s*"([^"]*)"', seg)
        intro = re.search(r'intro:\s*"([^"]*)"', seg)
        items.append({
            "id": pid,
            "name": nm.group(1) if nm else "",
            "intro": intro.group(1) if intro else "",
        })
    return items


# ── Vertex 客户端 ─────────────────────────────────────────────────────
def get_client():
    from google import genai
    cred = next((Path(c) for c in CRED_CANDIDATES if c and Path(c).exists()), None)
    if not cred:
        sys.exit("找不到服务账号 json，请用 GEMINI_CRED 指定路径。")
    os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = str(cred.resolve())
    project_id = json.loads(cred.read_text()).get("project_id", "vivi-gemini-0330")
    print(f"• 凭证: {cred.name}  项目: {project_id}  区域: {VERTEX_LOCATION}  模型: {IMAGE_MODEL}")
    return genai.Client(vertexai=True, project=project_id, location=VERTEX_LOCATION)


def generate_one(client, prompt: str) -> bytes | None:
    from google.genai import types
    # 尽量请求 1:1；老版本 SDK 不支持 image_config 时自动降级
    try:
        config = types.GenerateContentConfig(
            response_modalities=["TEXT", "IMAGE"],
            image_config=types.ImageConfig(aspect_ratio="1:1"),
        )
    except Exception:
        config = types.GenerateContentConfig(response_modalities=["TEXT", "IMAGE"])

    resp = client.models.generate_content(model=IMAGE_MODEL, contents=[prompt], config=config)
    for cand in (resp.candidates or []):
        for part in (cand.content.parts or []):
            data = getattr(part, "inline_data", None)
            if data and data.data:
                raw = data.data
                if isinstance(raw, str):  # 兼容 base64 字符串
                    import base64
                    raw = base64.b64decode(raw)
                return raw
    return None


# ── data.js：把对应 id 的 image 后缀改成 .png ───────────────────────────
def update_data_js(done_ids):
    text = DATA_JS.read_text(encoding="utf-8")
    changed = 0
    for pid in done_ids:
        new, n = re.subn(
            rf'(image:\s*"/assets/dishes/{pid})\.svg"',
            r'\1.png"',
            text,
        )
        if n:
            text = new
            changed += n
    if changed:
        DATA_JS.write_text(text, encoding="utf-8")
    print(f"• 已更新 data.js 图片路径 {changed} 条 (.svg → .png)")


# ── 主流程 ────────────────────────────────────────────────────────────
def main():
    args = [a for a in sys.argv[1:] if not a.startswith("-")]
    update_data = "--no-update-data" not in sys.argv

    DISHES_DIR.mkdir(parents=True, exist_ok=True)
    products = load_products()
    if args:
        products = [p for p in products if p["id"] in args]
    if not products:
        sys.exit("没有匹配的菜品。")

    client = get_client()
    done = []
    for p in products:
        pid = p["id"]
        prompt = build_prompt(pid, p["name"], p["intro"])
        print(f"\n→ {pid} {p['name']}")
        try:
            img = generate_one(client, prompt)
        except Exception as e:
            print(f"  ✗ 调用失败: {e}")
            continue
        if not img:
            print("  ✗ 未返回图片")
            continue
        out = DISHES_DIR / f"{pid}.png"
        out.write_bytes(img)
        print(f"  ✓ 已保存 {out.relative_to(REPO_ROOT)}  ({len(img)//1024} KB)")
        done.append(pid)

    print(f"\n完成 {len(done)}/{len(products)} 张。")
    if done and update_data:
        update_data_js(done)
    if done:
        print("提示：旧的 .svg 仍保留，可在确认效果后手动删除。")


if __name__ == "__main__":
    main()
