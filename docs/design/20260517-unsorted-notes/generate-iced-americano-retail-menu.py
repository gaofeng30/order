import base64
import json
import os
import time
from pathlib import Path

from openai import OpenAI
from PIL import Image, ImageDraw, ImageFilter, ImageFont


ROOT = Path(__file__).resolve().parent
ENV_FILE = Path("/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env")
PROMPT_DIR = ROOT / "prompts"
RAW_DIR = ROOT / "raw-3-2"
REF_DIR = ROOT / "references"

MODEL = "gpt-image-2"
GENERATE_SIZE = "1536x1024"
QUALITY = "medium"
FORMAT = "png"
FINAL_SIZE = (1980, 1080)

REF_IMAGES = [
    Path("/Users/marcusz/.cursor/projects/Users-marcusz-Projects-Test-Tool-order/assets/91108b2e3c037ac58c98fc1f4a68e6e5-4630beaf-d014-4501-9c6c-0eeaaff0ec44.png"),
    Path("/Users/marcusz/.cursor/projects/Users-marcusz-Projects-Test-Tool-order/assets/922e41a186efcb1a70e64c1f7e05c8ec-d62526cd-60f1-4214-abad-0c39a00c96ac.png"),
]

ITEM_ID = "01-iced-americano-retail-menu"
FINAL_FILE = "01-冰爽冰美式-便利零售-11x6-1080p.png"

MENU_ITEMS = [
    ("纸巾", "2元"),
    ("百岁山", "2元"),
    ("农夫山泉", "2元"),
    ("泉阳泉", "2元"),
    ("怡宝", "2元"),
    ("拉罐可口可乐330ml", "2.5元"),
    ("拉罐雪碧330ml", "2.5元"),
    ("拉罐芬达330ml", "2.5元"),
    ("拉罐美年达330ml", "2.5元"),
    ("可口可乐500ml", "3.5元"),
    ("雪碧500ml", "3.5元"),
    ("芬达500ml", "3.5元"),
    ("红牛250ml", "6元"),
    ("雀巢咖啡", "6元"),
    ("达利青梅绿茶450ml", "3元"),
    ("达利青梅绿茶1L", "4元"),
    ("茶π西柚茉莉花茶500ml", "5元"),
    ("茶π蜜桃乌龙茶500ml", "5元"),
    ("茶π柠檬红茶500ml", "5元"),
    ("康师傅冰红茶468ml", "3元"),
    ("东方树叶茉莉花茶500ml", "5元"),
    ("尖叫550ml", "5元"),
    ("秋林格瓦斯350ml", "3.5元"),
]


def load_env() -> None:
    for raw in ENV_FILE.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key, value.strip().strip("\"'"))


def build_prompt() -> str:
    return """基于两张输入参考图进行二次创作，生成一张现代咖啡店电视宣传海报的视觉底图，原始生成比例 3:2，后续会居中裁切为 11:6 横版电视画面。

参考图使用方式：
- 第一张参考图只学习冰美式的写实质感：深色高级咖啡广告氛围、咖啡液体从上方倾倒、飞溅水花、透明冰块、杯壁冷凝水、光影反射。不要保留任何手机截图界面、状态栏、时间、电量、播放按钮、下载按钮或品牌 logo。
- 第二张参考图只学习左右分栏信息结构：左侧大主视觉，右侧竖向菜单板。不要使用红金国风、灯笼、仙侠人物、毛笔字、祥云或国潮装饰。

画面要求：
- 11:6 横版电视海报安全构图，重要元素放在中间安全区，不贴边。
- 左侧约三分之二区域为冰美式主视觉区：一杯超大写实冰美式，透明杯体，无 logo，无品牌名，明显可见透明冰块；咖啡液体从上方倒入杯中并产生强烈飞溅，周围有少量冰块飞散、冷凝水珠、冷色高光，清爽提神。
- 右侧约三分之一区域预留一块干净、规整、无文字的菜单板区域，深棕或浅咖色高对比面板，用于后期叠加零售菜单。菜单板不要生成任何文字、编号、符号、装饰花纹或假字。
- 左侧标题和价格区域也请保持干净留白，不生成任何文字；后期会人工叠加中文标题和价格。
- 背景使用深咖色、黑棕色、冷调蓝灰高光，现代咖啡店广告风，高级、写实、商业感强、整洁不拥挤。

负面要求：
no logo, no brand name, no phone UI, no status bar, no time, no battery, no playback controls, no red-gold Chinese style, no lanterns, no fantasy character, no calligraphy headline, no tiny text, no fake text, no QR code, no watermark, no clutter, no numbered menu, no English text, no distorted Chinese characters.
"""


def call_image_api(prompt_text: str) -> bytes:
    client = OpenAI(api_key=os.environ["OPENAI_API_KEY"])
    files = [path.open("rb") for path in REF_IMAGES]
    try:
        response = client.images.edit(
            model=MODEL,
            image=files,
            prompt=prompt_text,
            n=1,
            size=GENERATE_SIZE,
            quality=QUALITY,
            output_format=FORMAT,
            timeout=360,
        )
    finally:
        for file in files:
            file.close()
    return base64.b64decode(response.data[0].b64_json)


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    candidates = [
        "/System/Library/Fonts/PingFang.ttc",
        "/System/Library/Fonts/STHeiti Medium.ttc" if bold else "/System/Library/Fonts/STHeiti Light.ttc",
        "/System/Library/Fonts/Supplemental/Songti.ttc",
        "/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
    ]
    for candidate in candidates:
        try:
            return ImageFont.truetype(candidate, size=size)
        except OSError:
            continue
    return ImageFont.load_default()


def fit_font(draw: ImageDraw.ImageDraw, text: str, max_width: int, start_size: int, min_size: int = 20) -> ImageFont.FreeTypeFont:
    current = start_size
    while current >= min_size:
        current_font = font(current)
        if draw.textbbox((0, 0), text, font=current_font)[2] <= max_width:
            return current_font
        current -= 1
    return font(min_size)


def draw_shadowed_text(
    draw: ImageDraw.ImageDraw,
    xy: tuple[int, int],
    text: str,
    text_font: ImageFont.FreeTypeFont,
    fill: str,
    shadow: str = "#120906",
    offset: tuple[int, int] = (4, 5),
) -> None:
    x, y = xy
    draw.text((x + offset[0], y + offset[1]), text, font=text_font, fill=shadow)
    draw.text((x, y), text, font=text_font, fill=fill)


def rounded_overlay(base: Image.Image, box: tuple[int, int, int, int], fill: tuple[int, int, int, int], radius: int) -> None:
    overlay = Image.new("RGBA", base.size, (0, 0, 0, 0))
    layer = ImageDraw.Draw(overlay)
    layer.rounded_rectangle(box, radius=radius, fill=fill)
    base.alpha_composite(overlay)


def prepare_canvas(raw_path: Path) -> Image.Image:
    image = Image.open(raw_path).convert("RGB")
    width, height = image.size
    crop_height = round(width * 6 / 11)
    top = (height - crop_height) // 2
    cropped = image.crop((0, top, width, top + crop_height))
    return cropped.resize(FINAL_SIZE, Image.Resampling.LANCZOS).convert("RGBA")


def add_text_layers(canvas: Image.Image) -> Image.Image:
    draw = ImageDraw.Draw(canvas)

    rounded_overlay(canvas, (70, 86, 760, 350), (13, 8, 6, 112), 34)
    draw_shadowed_text(draw, (112, 114), "冰爽冰美式", font(88, bold=True), "#F6F0DF")
    draw_shadowed_text(draw, (116, 222), "现做现饮  清凉提神", font(40), "#D8EEFF", shadow="#0B0806", offset=(3, 4))

    rounded_overlay(canvas, (88, 690, 720, 965), (30, 15, 8, 188), 42)
    draw.text((132, 724), "特惠价", font=font(58, bold=True), fill="#FFE6B0")
    draw_shadowed_text(draw, (132, 780), "10", font(154, bold=True), "#FFD064", shadow="#4A1B04", offset=(6, 7))
    draw.text((334, 842), "元/杯", font=font(72, bold=True), fill="#FFF2CC")

    board = (1188, 78, 1924, 1008)
    rounded_overlay(canvas, board, (39, 22, 14, 236), 36)
    draw.rounded_rectangle(board, radius=36, outline="#CFA86F", width=4)
    draw.text((1456, 112), "便利零售", font=font(72, bold=True), fill="#FFF0D0", anchor="ma")
    draw.line((1236, 206, 1876, 206), fill="#B98954", width=3)

    left_items = MENU_ITEMS[:12]
    right_items = MENU_ITEMS[12:]
    columns = [(1234, 244, 318), (1574, 244, 302)]
    row_step = 58
    row_font_size = 29
    price_font = font(29, bold=True)

    for col, items in zip(columns, [left_items, right_items]):
        x, y, col_width = col
        name_width = col_width - 112
        price_x = x + col_width
        for index, (name, price) in enumerate(items):
            row_y = y + index * row_step
            name_font = fit_font(draw, name, name_width, row_font_size, 22)
            draw.text((x, row_y), name, font=name_font, fill="#FFF7E8")
            draw.text((price_x, row_y), price, font=price_font, fill="#FFD889", anchor="ra")

    vignette = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    vignette_draw = ImageDraw.Draw(vignette)
    vignette_draw.rectangle((0, 0, 1980, 50), fill=(0, 0, 0, 85))
    vignette_draw.rectangle((0, 1030, 1980, 1080), fill=(0, 0, 0, 85))
    canvas.alpha_composite(vignette.filter(ImageFilter.GaussianBlur(18)))
    return canvas.convert("RGB")


def copy_references() -> list[str]:
    REF_DIR.mkdir(parents=True, exist_ok=True)
    copied = []
    for index, source in enumerate(REF_IMAGES, start=1):
        target = REF_DIR / f"reference-{index}.png"
        Image.open(source).save(target)
        copied.append(str(target.relative_to(ROOT)))
    return copied


def main() -> None:
    load_env()
    PROMPT_DIR.mkdir(parents=True, exist_ok=True)
    RAW_DIR.mkdir(parents=True, exist_ok=True)

    prompt_text = build_prompt()
    prompt_path = PROMPT_DIR / f"{ITEM_ID}.txt"
    raw_path = RAW_DIR / f"{ITEM_ID}.png"
    final_path = ROOT / FINAL_FILE
    prompt_path.write_text(prompt_text + "\n")
    references = copy_references()

    start = time.time()
    reused_raw = raw_path.exists()
    api_elapsed = None
    if reused_raw:
        print(f"Reusing {raw_path.relative_to(ROOT)} ...", flush=True)
    else:
        print(f"Generating {ITEM_ID} with {MODEL} ...", flush=True)
        api_start = time.time()
        raw_path.write_bytes(call_image_api(prompt_text))
        api_elapsed = round(time.time() - api_start, 1)
    final = add_text_layers(prepare_canvas(raw_path))
    final.save(final_path, quality=95)
    elapsed = round(time.time() - start, 1)
    print(f"OK {final_path.name} ({elapsed}s)", flush=True)

    (ROOT / "manifest-iced-americano-retail-menu.json").write_text(
        json.dumps(
            {
                "model": MODEL,
                "task": "Iced Americano hero visual with retail menu board",
                "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "generated_size": GENERATE_SIZE,
                "final_size": f"{FINAL_SIZE[0]}x{FINAL_SIZE[1]}",
                "final_aspect": "11:6",
                "quality": QUALITY,
                "items": [
                    {
                        "id": ITEM_ID,
                        "status": "ok",
                        "image_file": final_path.name,
                        "raw_file": str(Path("raw-3-2") / raw_path.name),
                        "prompt_file": str(Path("prompts") / prompt_path.name),
                        "reference_files": references,
                        "reused_raw": reused_raw,
                        "api_elapsed_seconds": api_elapsed,
                        "render_elapsed_seconds": elapsed,
                    }
                ],
            },
            ensure_ascii=False,
            indent=2,
        )
        + "\n"
    )


if __name__ == "__main__":
    main()
