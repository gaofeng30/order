import base64
import json
import os
import subprocess
import time
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parent
ENV_FILE = Path("/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env")
PROMPT_DIR = ROOT / "prompts"
RAW_DIR = ROOT / "raw-3-2"
OUT_DIR = ROOT

MODEL = "gpt-image-2"
GENERATE_SIZE = "1536x1024"
QUALITY = "medium"
FORMAT = "png"


def load_env() -> None:
    for raw in ENV_FILE.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key, value.strip().strip("\"'"))


def prompt(regional_feature: str, composition: str, accent: str) -> str:
    return f"""横屏商业餐饮海报，最终画面需要适合裁切成 11:6 宽屏。整体不是喜庆红色，不要红灯笼，不要红金春节风，不要哪吒风。

【整体风格】
东方健康、中药、药食同源实验室，高级商务蓝为底色：天蓝、湖蓝、青蓝、雾白、浅金少量点缀。画面清爽、可信、养生、现代东方，不要杂乱。

【左上标题】
图片左上方清晰写一行中文标题：「绥芬河市药食同源实验室」。字体不要用粗犷毛笔字，使用端正高级的宋体/楷体融合现代无衬线的标题字体，深青蓝或墨蓝色，字号大、清楚、易读。

【核心商品与文字】
主要目标是展示商品名称和价格，文字占比要更大、更显眼、更清楚。左侧中下区域放两只透明天然葫芦，葫芦形状必须像真实自然葫芦，上小下大，透明玻璃/琉璃材质，能清楚看到里面的中药材（人参片、黄芪、五味子、枸杞、菊花、陈皮、草本根茎）。

两只葫芦分别对应两个主打产品，放在左边相对位置，并配大字号商品卡：
01. 药食同源养生茶  15元/杯
02. 药食同源消脂茶  15元/杯
这两行必须最大、最醒目、优先保证可读。

右侧或右下角弱化展示两个普通饮品，字号可以小一些但仍清晰：
03. 蜂蜜柠檬茶  5元/杯
04. 鲜果茶  8元/杯
不要出现「美式咖啡」。

【人物 IP】
画面可出现一个东方诗人 IP：李白，青白色古风长袍，气质潇洒温和，手持酒葫芦或药葫芦，但人物不能抢过商品文字。人物可在右侧或背景中景，作为健康东方意境点缀。

【地方特色】
融入独特的黑龙江/绥芬河地方特色：{regional_feature}。这些元素作为背景或边缘装饰，不能压住文字和两只透明葫芦。

【构图要求】
{composition}
所有商品文字必须在画面安全区域内，字号大、横平竖直、不要变形、不要倒置、不要镜像。透明葫芦、药材和茶饮要精致真实，具有高端药膳实验室质感。

【颜色与质感】
商务蓝底，搭配{accent}，整体和谐、干净、有高级感。避免大面积红色和春节装饰。

【负面要求】
no red festive style, no lanterns, no Spring Festival look, no coffee, no extra product names, no English text, no logos, no QR code, no watermark, no messy typography, no tiny unreadable menu, no distorted Chinese characters, no mirrored text, no duplicated title.
"""


VARIANTS = [
    {
        "id": "01-suifenhe-port-lab",
        "title": "绥芬河口岸实验室风",
        "regional_feature": "绥芬河百年口岸、中东铁路记忆、淡蓝色俄式建筑轮廓、远处口岸国门剪影、轻微铁轨线条，体现中俄边境城市气质",
        "composition": "左上标题，左侧两只透明药材葫芦占画面 35%，两款主打产品文字紧贴葫芦；右侧李白与淡蓝俄式建筑背景；右下角小卡片放蜂蜜柠檬茶和鲜果茶。",
        "accent": "浅金线条和淡木色药柜纹理",
    },
    {
        "id": "02-birch-snow-herbal",
        "title": "白桦林寒地草本风",
        "regional_feature": "黑龙江白桦林、林海雪原、冷雾山林、雪地蓝影，并点缀五味子、刺五加、人参等寒地草本",
        "composition": "画面左半部分是两只透明药材葫芦和大字号价格，背景留白干净；右侧李白站在白桦林与蓝白雪雾中，普通饮品放右下角小区域。",
        "accent": "雾白、银蓝、少量草本金色",
    },
    {
        "id": "03-oriental-pharmacy",
        "title": "东方药房陈列风",
        "regional_feature": "绥芬河边城窗景、蓝色窗棂、东北寒地药材陈列、药柜抽屉、玻璃实验器皿，远处淡化国门和铁路线条",
        "composition": "像高级药食同源实验室的产品陈列大片：左侧两个透明葫芦并列为主视觉，商品名称和价格用大号蓝黑字放在葫芦旁；李白在右侧手持药葫芦，右侧低处摆蜂蜜柠檬茶和鲜果茶。",
        "accent": "青瓷蓝、米白、浅竹木色",
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
    with urllib.request.urlopen(req, timeout=360) as res:
        payload = json.loads(res.read().decode())
    return base64.b64decode(payload["data"][0]["b64_json"])


def crop_to_11_6(raw_path: Path, out_path: Path) -> None:
    subprocess.run(["sips", "-c", "838", "1536", str(raw_path), "--out", str(out_path)], check=True)


def main() -> None:
    load_env()
    PROMPT_DIR.mkdir(parents=True, exist_ok=True)
    RAW_DIR.mkdir(parents=True, exist_ok=True)

    results = []
    for item in VARIANTS:
        prompt_text = prompt(item["regional_feature"], item["composition"], item["accent"])
        prompt_path = PROMPT_DIR / f"{item['id']}.txt"
        raw_path = RAW_DIR / f"{item['id']}.png"
        out_path = OUT_DIR / f"{item['id']}.png"
        prompt_path.write_text(prompt_text + "\n")

        start = time.time()
        print(f"Generating {item['id']} ...", flush=True)
        raw_path.write_bytes(call_image_api(prompt_text))
        crop_to_11_6(raw_path, out_path)
        elapsed = round(time.time() - start, 1)
        print(f"OK {out_path.name} ({elapsed}s)", flush=True)
        results.append(
            {
                "id": item["id"],
                "title": item["title"],
                "status": "ok",
                "image_file": out_path.name,
                "raw_file": str(Path("raw-3-2") / raw_path.name),
                "prompt_file": str(Path("prompts") / prompt_path.name),
                "final_aspect": "11:6",
                "generated_size": GENERATE_SIZE,
                "quality": QUALITY,
                "elapsed_seconds": elapsed,
            }
        )

    (OUT_DIR / "manifest.json").write_text(
        json.dumps(
            {
                "model": MODEL,
                "task": "TV5 tea lab redesign, three candidate images",
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
