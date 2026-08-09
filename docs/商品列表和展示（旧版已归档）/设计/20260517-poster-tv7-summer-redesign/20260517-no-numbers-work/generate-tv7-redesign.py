import base64
import json
import os
import subprocess
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


ROOT = Path(__file__).resolve().parent
ENV_FILE = Path("/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env")
PROMPT_DIR = ROOT / "prompts"
RAW_DIR = ROOT / "raw-3-2"

MODEL = "gpt-image-2"
GENERATE_SIZE = "1536x1024"
QUALITY = "medium"
FORMAT = "png"
MAX_WORKERS = 2


def load_env() -> None:
    for raw in ENV_FILE.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key, value.strip().strip("\"'"))


def build_prompt(style: str, layout: str, local_feature: str) -> str:
    return f"""横屏餐饮套餐海报，最终需要从 1536x1024 居中裁切成 11:6 宽屏。生成时必须为裁切预留上下白边/蓝色留白：所有重要文字、人物脸、菜单板、价格都放在中间安全区，距离画面上边缘至少 150px，距离下边缘至少 130px。

【整体方向】
重新设计电视七「组合套餐」海报。不要红灯笼，不要春节喜庆红色，不要大面积红色背景。改为夏天、清爽、山、海浪、水纹、浅蓝天空、湖蓝色、青绿色、米白色。整体明亮、干净、清爽、有商业餐饮质感。

【标题】
画面上方安全区内可以保留较小但清晰的品牌标题「味在绥安」，不要金色大毛笔占满画面。标题使用清爽蓝白渐变或深湖蓝字体，不能贴边，不能被裁切。

【菜单文字是主体】
这张图最重要的是让顾客看清套餐名称和价格。菜单板必须占画面 55% 到 65%，比人物和菜品更显眼。菜单板使用米白/浅蓝白卡片，深蓝大字，价格用更粗更大字号。菜品图片只能占 15% 到 20%，作为边缘点缀，不要抢菜单。

菜单内容必须大字号、清晰、规整，按以下格式展示。图片里不要出现 01、02、03 或任何编号/序号：
套餐一（卷饼+鸭翅+咖啡）  20元
套餐二  20元
    猪脚饭+柠檬水+烤肠 或
    小鸡炖蘑菇盖饭+柠檬水+烤肠 或
    杀猪菜汤饭+柠檬水+烤肠
套餐三（盒饭两荤两素+咖啡）  20元

「套餐二」下方三行必须换行展示，前两行末尾必须有「或」，最后一行不加「或」。文字必须横平竖直、足够大、不要重叠、不要变形、不要镜像。

【画面元素】
保留少量对应菜品：卷饼、鸭翅、咖啡、猪脚饭、小鸡炖蘑菇盖饭、杀猪菜汤饭、柠檬水、烤肠、两荤两素盒饭；这些只作为底部或侧边小面积点缀。
可以保留两个童子/组合 IP 的感觉，但人物只作为右侧或边缘小角色，不能抢菜单板。

【夏日山水特色】
{local_feature}

【视觉风格】
{style}

【版式】
{layout}

【负面要求】
no red lanterns, no Spring Festival style, no large red background, no product numbers, no numbered list, no 01 02 03 labels, no tiny menu, no unreadable Chinese text, no cropped title, no text outside safe area, no mirrored text, no QR code, no English text, no logo, no watermark, no oversized food photos, no menu hidden by characters.
"""


VARIANTS = [
    {
        "id": "01-summer-mountain-wave-menu",
        "style": "夏日湖蓝商业海报风：浅蓝天空、青绿色远山、白色海浪线条、清透水纹，整体清爽明亮。",
        "layout": "菜单板放在左中到中间，占画面约 62%；右侧小人物和小菜品围绕菜单边缘；底部只有一排小尺寸菜品，不超过画面 18%。",
        "local_feature": "背景融入东北夏季青山、绥芬河边城清凉蓝天、抽象海浪纹和水波纹，像夏日山海风。",
    },
    {
        "id": "02-blue-coast-clean-board",
        "style": "高级商务蓝餐饮价目牌风：湖蓝渐变背景、米白大菜单板、深蓝字体、少量橙黄食物点缀形成对比。",
        "layout": "菜单板居中偏左，超大字号；标题在上方安全区小而清楚；食物缩小成右下角和左下角点缀；人物 IP 只在右侧半身小比例出现。",
        "local_feature": "背景是夏天山脉剪影、浅色海浪和蓝白水花，加入东北林区青山轮廓，不使用红色灯笼。",
    },
]


def call_image_api(prompt_text: str) -> bytes:
    body = json.dumps(
        {
            "model": MODEL,
            "prompt": prompt_text,
            "n": 1,
            "size": GENERATE_SIZE,
            "quality": QUALITY,
            "output_format": FORMAT,
        },
        ensure_ascii=False,
    ).encode()
    req = urllib.request.Request(
        "https://api.openai.com/v1/images/generations",
        data=body,
        headers={
            "Authorization": f"Bearer {os.environ['OPENAI_API_KEY']}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    last_error = None
    for attempt in range(1, 4):
        try:
            with opener.open(req, timeout=360) as res:
                payload = json.loads(res.read().decode())
            return base64.b64decode(payload["data"][0]["b64_json"])
        except Exception as error:
            last_error = error
            if attempt < 3:
                time.sleep(8 * attempt)
    raise last_error


def crop_to_11_6(raw_path: Path, out_path: Path) -> None:
    subprocess.run(["sips", "-c", "838", "1536", str(raw_path), "--out", str(out_path)], check=True)


def main() -> None:
    load_env()
    PROMPT_DIR.mkdir(parents=True, exist_ok=True)
    RAW_DIR.mkdir(parents=True, exist_ok=True)

    def process_item(item: dict) -> dict:
        prompt_text = build_prompt(item["style"], item["layout"], item["local_feature"])
        prompt_path = PROMPT_DIR / f"{item['id']}.txt"
        raw_path = RAW_DIR / f"{item['id']}.png"
        out_path = ROOT / f"{item['id']}.png"
        prompt_path.write_text(prompt_text + "\n")

        print(f"Generating {item['id']} ...", flush=True)
        start = time.time()
        raw_path.write_bytes(call_image_api(prompt_text))
        crop_to_11_6(raw_path, out_path)
        elapsed = round(time.time() - start, 1)
        print(f"OK {out_path.name} ({elapsed}s)", flush=True)
        return {
            "id": item["id"],
            "status": "ok",
            "image_file": out_path.name,
            "raw_file": str(Path("raw-3-2") / raw_path.name),
            "prompt_file": str(Path("prompts") / prompt_path.name),
            "final_aspect": "11:6",
            "generated_size": GENERATE_SIZE,
            "quality": QUALITY,
            "elapsed_seconds": elapsed,
        }

    results = []
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = [executor.submit(process_item, item) for item in VARIANTS]
        for future in as_completed(futures):
            results.append(future.result())
    results.sort(key=lambda item: item["id"])

    (ROOT / "manifest.json").write_text(
        json.dumps(
            {
                "model": MODEL,
                "task": "TV7 summer mountain wave redesign, two candidates",
                "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "final_aspect": "11:6",
                "items": results,
            },
            ensure_ascii=False,
            indent=2,
        )
        + "\n"
    )


if __name__ == "__main__":
    main()
