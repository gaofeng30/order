from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont


ROOT = Path(__file__).resolve().parent
TARGETS = [
    ROOT / "02-birch-snow-herbal-bg-expanded-safe-crop.png",
]


def font(size: int) -> ImageFont.FreeTypeFont:
    candidates = [
        "/System/Library/Fonts/PingFang.ttc",
        "/System/Library/Fonts/STHeiti Medium.ttc",
        "/System/Library/Fonts/Supplemental/Songti.ttc",
    ]
    for candidate in candidates:
        try:
            return ImageFont.truetype(candidate, size=size)
        except OSError:
            continue
    return ImageFont.load_default()


def text_width(draw: ImageDraw.ImageDraw, text: str, text_font: ImageFont.FreeTypeFont) -> int:
    bbox = draw.textbbox((0, 0), text, font=text_font)
    return bbox[2] - bbox[0]


def draw_title(draw: ImageDraw.ImageDraw) -> None:
    title = "绥芬河市药食同源实验室"
    title_font = font(60)
    x, y = 178, 50
    width = text_width(draw, title, title_font)

    # Light mist patch keeps the larger title clean without adding a hard box.
    patch = Image.new("RGBA", (width + 56, 88), (0, 0, 0, 0))
    patch_draw = ImageDraw.Draw(patch)
    patch_draw.rounded_rectangle((0, 0, width + 56, 86), radius=18, fill=(236, 248, 255, 224))
    patch = patch.filter(ImageFilter.GaussianBlur(5))
    return patch, (x - 24, y - 16), title, title_font, (x, y)


def draw_product_line(draw: ImageDraw.ImageDraw, y: int, name: str, height: int = 190) -> None:
    card_fill = (248, 244, 231, 246)
    draw.rounded_rectangle((300, y - 12, 952, y + height), radius=24, fill=card_fill)

    navy = "#082B5A"
    gold = "#C89624"
    name_font = font(82)
    number_font = font(88)
    unit_font = font(38)
    draw.text((328, y + 12), name, font=name_font, fill=navy)
    draw.text((682, y + 2), "15", font=number_font, fill=gold)
    draw.text((814, y + 42), "元/杯", font=unit_font, fill=gold)


def polish(path: Path) -> None:
    image = Image.open(path).convert("RGBA")
    draw = ImageDraw.Draw(image)

    patch, patch_xy, title, title_font, title_xy = draw_title(draw)
    image.alpha_composite(patch, patch_xy)
    draw = ImageDraw.Draw(image)
    draw.text((title_xy[0] + 3, title_xy[1] + 4), title, font=title_font, fill="#E8F4FF")
    draw.text(title_xy, title, font=title_font, fill="#0A2B59")

    draw_product_line(draw, 208, "养生茶")
    draw_product_line(draw, 438, "清脂茶", height=286)
    image.convert("RGB").save(path)


def main() -> None:
    for target in TARGETS:
        polish(target)
        print(f"Polished {target.name}")


if __name__ == "__main__":
    main()
