from pathlib import Path

from PIL import Image, ImageEnhance, ImageFilter


ROOT = Path(__file__).resolve().parent
RAW_DIR = ROOT / "raw-3-2"

SOURCE_RAW = RAW_DIR / "02-birch-snow-herbal.png"
PADDED_RAW = RAW_DIR / "02-birch-snow-herbal-more-padding.png"
PADDED_FINAL = ROOT / "02-birch-snow-herbal-more-padding.png"


def crop_to_11_6(image: Image.Image) -> Image.Image:
    width, height = image.size
    crop_height = round(width * 6 / 11)
    top = (height - crop_height) // 2
    return image.crop((0, top, width, top + crop_height))


def main() -> None:
    source = Image.open(SOURCE_RAW).convert("RGB")
    width, height = source.size

    background = source.resize((width, height), Image.Resampling.LANCZOS)
    background = background.filter(ImageFilter.GaussianBlur(34))
    background = ImageEnhance.Contrast(background).enhance(0.72)
    background = ImageEnhance.Brightness(background).enhance(1.08)

    content_width = 1120
    content_height = round(content_width * height / width)
    content = source.resize((content_width, content_height), Image.Resampling.LANCZOS)

    x = (width - content_width) // 2
    y = (height - content_height) // 2

    canvas = background.copy()
    shadow = Image.new("RGBA", (content_width + 34, content_height + 34), (0, 0, 0, 0))
    shadow_mask = Image.new("L", (content_width, content_height), 255)
    shadow.paste((24, 74, 104, 58), (17, 17), shadow_mask)
    shadow = shadow.filter(ImageFilter.GaussianBlur(18))
    canvas = canvas.convert("RGBA")
    canvas.alpha_composite(shadow, (x - 17, y - 17))
    canvas.paste(content.convert("RGBA"), (x, y))

    raw = canvas.convert("RGB")
    raw.save(PADDED_RAW)
    crop_to_11_6(raw).save(PADDED_FINAL)


if __name__ == "__main__":
    main()
