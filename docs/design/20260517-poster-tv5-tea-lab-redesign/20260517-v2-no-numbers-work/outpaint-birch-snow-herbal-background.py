import base64
import json
import os
import time
from pathlib import Path

from openai import OpenAI
from PIL import Image


ROOT = Path(__file__).resolve().parent
ENV_FILE = Path("/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env")
WORK_DIR = ROOT / "outpaint-work"
RAW_DIR = ROOT / "raw-3-2"

MODEL = "gpt-image-2"
QUALITY = "medium"
SIZE = "1536x1024"
FORMAT = "png"

SOURCE_11_6 = ROOT / "02-birch-snow-herbal.png"
SOURCE_RAW = RAW_DIR / "02-birch-snow-herbal.png"
CANVAS_FILE = WORK_DIR / "02-birch-snow-herbal-centered-canvas.png"
MASK_FILE = WORK_DIR / "02-birch-snow-herbal-outpaint-mask.png"
OUTPAINT_RAW = RAW_DIR / "02-birch-snow-herbal-bg-extended.png"
OUTPAINT_11_6 = ROOT / "02-birch-snow-herbal-bg-extended.png"
SAFE_RAW = RAW_DIR / "02-birch-snow-herbal-bg-expanded-safe-crop.png"
SAFE_11_6 = ROOT / "02-birch-snow-herbal-bg-expanded-safe-crop.png"
MODEL_BACKGROUND = WORK_DIR / "02-birch-snow-herbal-model-background.png"
PROMPT_FILE = ROOT / "prompts" / "02-birch-snow-herbal-bg-extended.txt"
MANIFEST_FILE = ROOT / "manifest-bg-extended.json"


def load_env() -> None:
    for raw in ENV_FILE.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key, value.strip().strip("\"'"))


def build_canvas_and_mask() -> None:
    WORK_DIR.mkdir(parents=True, exist_ok=True)
    source = Image.open(SOURCE_11_6).convert("RGBA")
    canvas = Image.new("RGBA", (1536, 1024), (0, 0, 0, 0))
    top = (1024 - source.height) // 2
    canvas.alpha_composite(source, (0, top))
    canvas.save(CANVAS_FILE)

    # Transparent mask areas are the only regions gpt-image-2 should fill.
    mask = Image.new("RGBA", (1536, 1024), (0, 0, 0, 0))
    opaque = Image.new("RGBA", source.size, (255, 255, 255, 255))
    mask.alpha_composite(opaque, (0, top))
    mask.save(MASK_FILE)


def prompt() -> str:
    return """对输入图片做上下背景扩充（outpainting），只补全透明的上边和下边区域，保持中间已有海报内容完全不变。

目标：
- 输出 1536x1024 横版 3:2 原图，用于后续居中裁切成 11:6。
- 中间已有 11:6 海报区域必须保持原样：不要移动、不要缩放、不要重绘标题、商品名、价格、葫芦、人物、饮品卡片、白桦树。
- 只在上方和下方透明留白区域自然延展背景，让它看起来像原本就是完整画面，而不是相框、白边、灰边、模糊边框或贴图边框。

上方扩充：
- 延续浅蓝天空、冷雾、远山、白桦林枝叶的氛围。
- 可以增加柔和蓝白渐变、薄雾、远处树梢和天空留白。
- 不要新增文字，不要新增标题，不要把原标题改掉。

下方扩充：
- 延续冰雪水面、浅蓝反射、雪地、轻微雾气和柔和光影。
- 可以自然扩展水面波纹、雪地反光和冷色阴影。
- 不要新增商品、不要新增饮品卡片、不要新增文字。

风格：
高级商务蓝、雾白、银蓝、寒地白桦林、药食同源实验室氛围，整体干净、清爽、和谐。

严格负面要求：
do not change the central poster, no frame, no border, no white border, no gray border, no shrink effect, no duplicate text, no new text, no cropped title, no altered Chinese characters, no extra products, no logo, no QR code, no watermark, no red festive style, no lanterns.
"""


def call_image_api(prompt_text: str) -> bytes:
    client = OpenAI(api_key=os.environ["OPENAI_API_KEY"])
    with CANVAS_FILE.open("rb") as image_file, MASK_FILE.open("rb") as mask_file:
        response = client.images.edit(
            model=MODEL,
            image=image_file,
            mask=mask_file,
            prompt=prompt_text,
            n=1,
            size=SIZE,
            quality=QUALITY,
            output_format=FORMAT,
            timeout=360,
        )
    return base64.b64decode(response.data[0].b64_json)


def crop_to_11_6(raw_path: Path, out_path: Path) -> None:
    image = Image.open(raw_path).convert("RGB")
    width, height = image.size
    crop_height = round(width * 6 / 11)
    top = (height - crop_height) // 2
    image.crop((0, top, width, top + crop_height)).save(out_path)


def composite_original_center(background_path: Path, out_path: Path) -> None:
    background = Image.open(background_path).convert("RGBA")
    source = Image.open(SOURCE_11_6).convert("RGBA")
    top = (background.height - source.height) // 2

    # Keep every original pixel intact. The model output is used only as the
    # extra top/bottom background around the preserved 11:6 poster.
    background.alpha_composite(source, (0, top))
    background.convert("RGB").save(out_path)


def composite_safe_crop(background_path: Path, out_path: Path) -> None:
    background = Image.open(background_path).convert("RGBA")
    source = Image.open(SOURCE_RAW).convert("RGBA")
    scale = 0.84
    target_size = (round(source.width * scale), round(source.height * scale))
    content = source.resize(target_size, Image.Resampling.LANCZOS)
    x = (background.width - content.width) // 2
    y = (background.height - content.height) // 2

    mask = Image.new("L", content.size, 255)
    pixels = mask.load()
    feather = 24
    for row in range(content.height):
        vertical = min(row, content.height - 1 - row)
        alpha_y = min(255, round(255 * vertical / feather)) if vertical < feather else 255
        for col in range(content.width):
            horizontal = min(col, content.width - 1 - col)
            alpha_x = min(255, round(255 * horizontal / feather)) if horizontal < feather else 255
            pixels[col, row] = min(alpha_x, alpha_y)

    background.paste(content, (x, y), mask)
    background.convert("RGB").save(out_path)


def main() -> None:
    load_env()
    build_canvas_and_mask()
    prompt_text = prompt()
    PROMPT_FILE.write_text(prompt_text + "\n")

    print(f"Outpainting {SOURCE_11_6.name} with {MODEL} ...", flush=True)
    start = time.time()
    if MODEL_BACKGROUND.exists():
        print(f"Reusing {MODEL_BACKGROUND.relative_to(ROOT)} ...", flush=True)
    elif OUTPAINT_RAW.exists():
        OUTPAINT_RAW.replace(MODEL_BACKGROUND)
    else:
        MODEL_BACKGROUND.write_bytes(call_image_api(prompt_text))
    composite_original_center(MODEL_BACKGROUND, OUTPAINT_RAW)
    crop_to_11_6(OUTPAINT_RAW, OUTPAINT_11_6)
    composite_safe_crop(MODEL_BACKGROUND, SAFE_RAW)
    crop_to_11_6(SAFE_RAW, SAFE_11_6)
    elapsed = round(time.time() - start, 1)
    print(f"OK {OUTPAINT_RAW.relative_to(ROOT)} ({elapsed}s)", flush=True)

    MANIFEST_FILE.write_text(
        json.dumps(
            {
                "model": MODEL,
                "task": "Extend top and bottom background for birch snow herbal poster",
                "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "source_file": SOURCE_11_6.name,
                "canvas_file": str(CANVAS_FILE.relative_to(ROOT)),
                "mask_file": str(MASK_FILE.relative_to(ROOT)),
                "model_background_file": str(MODEL_BACKGROUND.relative_to(ROOT)),
                "raw_file": str(OUTPAINT_RAW.relative_to(ROOT)),
                "cropped_11_6_file": OUTPAINT_11_6.name,
                "safe_crop_raw_file": str(SAFE_RAW.relative_to(ROOT)),
                "safe_crop_11_6_file": SAFE_11_6.name,
                "prompt_file": str(PROMPT_FILE.relative_to(ROOT)),
                "generated_size": SIZE,
                "quality": QUALITY,
                "elapsed_seconds": elapsed,
            },
            ensure_ascii=False,
            indent=2,
        )
        + "\n"
    )


if __name__ == "__main__":
    main()
