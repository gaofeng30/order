import base64
import json
import os
import time
from pathlib import Path

from openai import OpenAI
from PIL import Image


ROOT = Path(__file__).resolve().parent
ENV_FILE = Path("/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env")
INPUT_IMAGE = Path(
    "/Users/marcusz/.cursor/projects/Users-marcusz-Projects-Test-Tool-order/assets/"
    "f15c45cf7633e2e3380eceef279e8e64-25740733-bd18-4e39-b64b-0dcfe58f1873.png"
)
PROMPT_FILE = ROOT / "prompts" / "main-edit.txt"
CANDIDATES_DIR = ROOT / "candidates"
OUTPUT_FILE = ROOT / "tv5-new-edition.png"
MANIFEST_FILE = ROOT / "manifest.json"

MODEL = "gpt-image-2"
SIZE = "1536x1024"
QUALITY = "high"
FORMAT = "png"


def load_env() -> None:
    if not ENV_FILE.exists():
        return
    for raw in ENV_FILE.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key, value.strip().strip("\"'"))


def image_size(path: Path) -> str:
    with Image.open(path) as image:
        return f"{image.width}x{image.height}"


def call_image_edit(prompt_text: str) -> bytes:
    client = OpenAI(api_key=os.environ["OPENAI_API_KEY"])
    with INPUT_IMAGE.open("rb") as image_file:
        response = client.images.edit(
            model=MODEL,
            image=image_file,
            prompt=prompt_text,
            n=1,
            size=SIZE,
            quality=QUALITY,
            output_format=FORMAT,
            timeout=420,
        )
    return base64.b64decode(response.data[0].b64_json)


def main() -> None:
    load_env()
    if "OPENAI_API_KEY" not in os.environ:
        raise RuntimeError("OPENAI_API_KEY is not set")
    if not INPUT_IMAGE.exists():
        raise FileNotFoundError(INPUT_IMAGE)

    CANDIDATES_DIR.mkdir(parents=True, exist_ok=True)
    prompt_text = PROMPT_FILE.read_text()
    started_at = time.time()

    candidate_file = CANDIDATES_DIR / "tv5-new-edition-candidate-01.png"
    print(f"Editing {INPUT_IMAGE.name} with {MODEL} ({QUALITY}, {SIZE}) ...", flush=True)
    candidate_file.write_bytes(call_image_edit(prompt_text))
    OUTPUT_FILE.write_bytes(candidate_file.read_bytes())

    elapsed = round(time.time() - started_at, 1)
    result = {
        "model": MODEL,
        "task": "TV5 herbal tea poster new 3:2 edition",
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "input_image": str(INPUT_IMAGE),
        "input_size": image_size(INPUT_IMAGE),
        "output_file": OUTPUT_FILE.name,
        "candidate_file": str(candidate_file.relative_to(ROOT)),
        "output_size": image_size(OUTPUT_FILE),
        "prompt_file": str(PROMPT_FILE.relative_to(ROOT)),
        "requested_size": SIZE,
        "quality": QUALITY,
        "output_format": FORMAT,
        "elapsed_seconds": elapsed,
        "manual_review_checklist": [
            "Main title reads exactly: 绥芬河市药食同源实验室",
            "Upper main card reads: 养生茶  15元/杯",
            "Lower main card reads: 清脂茶  15元/杯",
            "No forbidden text remains: 药食同源养生茶, 药食同源消脂茶, 消脂茶",
            "The two gourds contain visibly different brewed-tea ingredients",
            "Top and bottom areas remain natural background only",
        ],
    }
    MANIFEST_FILE.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(f"OK {OUTPUT_FILE.relative_to(ROOT)} ({elapsed}s)", flush=True)


if __name__ == "__main__":
    main()
