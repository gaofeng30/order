import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const POSTER_DIR = path.dirname(SCRIPT_DIR);
const PROMPTS_DIR = path.join(POSTER_DIR, "prompts");
const POSTERS_DIR = path.join(POSTER_DIR, "posters");
const ENV_FILE = "/Users/marcusz/Projects/Test Tool/draft-tool/openai-image-2/.env";

const IMAGE_MODEL = "gpt-image-2";
const SIZE = "1536x1024";
const QUALITY = "high";
const FORMAT = "png";

const TVS = [
  {
    index: 1,
    slug: "电视一-烤肠卤味档",
    title: "电视一 · 烤肠与卤味档",
    character: "太乙真人（《哪吒之魔童闹海》中的圆润长须神仙，骑乘飞猪，乐呵呵的老饕形象）",
    characterEnElements: "fire embers, smoky grill atmosphere, 仙气与烟火气并存",
    palette: "烧烤金棕 + 朱红 + 暖金",
    sideDishes: "烤肠、酱猪手、酱鸡胗、辣卤鸭脖、土豆丝卷饼",
    items: [
      ["皇小虎烤肠", "4元/根"],
      ["皇小虎蛋挞", "10元4个"],
      ["土豆丝卷饼（加肠 加卤蛋）", "7元/个"],
      ["俄罗斯经典大肉串", "12元/个"],
      ["酱猪手", "18元/袋（半个）"],
      ["酱猪耳", "35元/袋（1个）"],
      ["酱鸡手", "20元/袋（5个）"],
      ["酱鸡胗", "15元/袋"],
      ["辣卤鸭头", "6元/个"],
      ["辣卤鸭翅", "3元/个"],
      ["辣卤鸭脖", "5元/个"],
    ],
  },
  {
    index: 2,
    slug: "电视二-镇特产",
    title: "电视二 · 阜宁镇与绥芬河镇特产",
    character: "殷夫人（《哪吒之魔童闹海》中的温润母亲形象，铠甲外披柔软披风，手挽竹篮）",
    characterEnElements: "丰收暖光、田野金黄、桃花瓣轻飘",
    palette: "朱红 + 田园暖黄 + 翠绿",
    sideDishes: "翠绿黄瓜、糖葫芦、玉米、煎饼、鲜红草莓与西红柿",
    items: [
      ["阜宁镇黄瓜", "4元/盒"],
      ["阜宁镇煎饼", "糖煎饼10元/盒；大米煎饼5元/袋"],
      ["阜宁镇玉米", "2元/根"],
      ["阜宁镇糖葫芦", "3.5元/根"],
      ["阜宁镇绿豆糕", "5元/盒"],
      ["绥芬河镇草莓", "25元/斤"],
      ["绥芬河镇西红柿", "15元/斤"],
    ],
  },
  {
    index: 3,
    slug: "电视三-东北盖饭",
    title: "电视三 · 东北盖饭",
    character: "李靖（《哪吒之魔童闹海》中沉稳威严的陈塘关总兵，铠甲厚重，气场分量感十足）",
    characterEnElements: "厚重披风、暖光灶台、热气腾腾",
    palette: "深红 + 暖棕 + 金边",
    sideDishes: "猪脚饭砂锅、小鸡炖蘑菇砂锅、杀猪菜大碗",
    items: [
      ["猪脚饭", "15元/份"],
      ["小鸡炖蘑菇盖饭", "16元/份"],
      ["杀猪菜", "15元/份"],
    ],
  },
  {
    index: 4,
    slug: "电视四-荤素套餐",
    title: "电视四 · 荤素套餐",
    character: "哪吒（《哪吒之魔童闹海》中标志性双丸子头小哪吒，火尖枪、混天绫，与参考海报一致）",
    characterEnElements: "火尖枪、火焰飞动、混天绫飘扬",
    palette: "朱红 + 烈焰金 + 米黄",
    sideDishes: "色泽鲜亮的咕咾肉、白菜炖肉砂锅、酥脆煎饼",
    items: [
      ["一荤三素", "12元"],
      ["两荤两素", "15元"],
      ["一素三荤", "18元"],
    ],
  },
  {
    index: 5,
    slug: "电视五-茶咖",
    title: "电视五 · 茶饮与咖啡",
    character: "敖丙（《哪吒之魔童闹海》中的清冷龙太子，白发龙角，水袖与水流灵动）",
    characterEnElements: "水流飞溅、冰晶光、淡蓝色雾气与红金对冲",
    palette: "朱红主调 + 龙鳞蓝 + 暖金（保持系列一致性）",
    sideDishes: "玻璃杯装养生茶、柠檬茶、鲜果茶、咖啡拉花",
    items: [
      ["药食同源养生茶", "15元/杯"],
      ["药食同源消脂茶", "15元/杯"],
      ["蜂蜜柠檬茶", "5元/杯"],
      ["鲜果茶", "8元/杯"],
      ["美式咖啡", "10元/杯"],
    ],
  },
  {
    index: 6,
    slug: "电视六-零售饮料水",
    title: "电视六 · 零售与瓶装饮料",
    character: "敖光（《哪吒之魔童闹海》东海龙王，威严龙首披风，手持龙杖，气场磅礴）",
    characterEnElements: "海浪浮金、龙气翻涌、瓶罐如百宝陈列",
    palette: "朱红 + 深海蓝 + 金边",
    sideDishes: "成排瓶装水、玻璃瓶饮料、易拉罐汽水阵列",
    items: [
      ["纸巾", "—"],
      ["百岁山", "—"],
      ["农夫山泉", "—"],
      ["泉阳泉", "—"],
      ["怡宝", "—"],
      ["拉罐可口可乐330ml", "2.5元"],
      ["拉罐雪碧330ml", "2.5元"],
      ["拉罐芬达330ml", "2.5元"],
      ["拉罐美年达330ml", "2.5元"],
      ["可口可乐500ml", "3.5元"],
      ["雪碧500ml", "3.5元"],
      ["芬达500ml", "3.5元"],
      ["红牛250ml", "6元"],
      ["雀巢咖啡", "6元"],
      ["达利青梅绿茶450ml", "3元"],
      ["达利青梅绿茶1L", "4元"],
      ["茶π西柚茉莉花茶500ml", "5元"],
      ["茶π蜜桃乌龙茶500ml", "5元"],
      ["茶π柠檬红茶500ml", "5元"],
      ["康师傅冰红茶468ml", "3元"],
      ["东方树叶茉莉花茶500ml", "5元"],
      ["尖叫550ml", "5元"],
      ["秋林格瓦斯350ml", "3.5元"],
    ],
  },
  {
    index: 7,
    slug: "电视七-组合套餐",
    title: "电视七 · 组合套餐",
    character: "鹤童与鹿童（《哪吒之魔童闹海》中太乙真人门下的双童子，一红一青，组合呼应「组合套餐」）",
    characterEnElements: "双童并肩端餐盘、彩云托底、组合成对的对称气场",
    palette: "朱红 + 浅金 + 翠绿点缀",
    sideDishes: "卷饼、鸭翅、咖啡、猪脚饭、小鸡炖蘑菇盖饭、杀猪菜汤饭、柠檬水、烤肠、两荤两素盒饭",
    items: [
      ["套餐一（卷饼+鸭翅+咖啡）", "20元"],
      ["套餐二\n猪脚饭+柠檬水+烤肠 或\n小鸡炖蘑菇盖饭+柠檬水+烤肠 或\n杀猪菜汤饭+柠檬水+烤肠", "20元"],
      ["套餐三（盒饭两荤两素+咖啡）", "20元"],
    ],
  },
  {
    index: 8,
    slug: "电视八-甜品水果",
    title: "电视八 · 甜品与水果",
    character: "申小豹（《哪吒之魔童闹海》中萌系小豹妖，圆耳大眼，对甜食毫无抵抗力的童趣形象）",
    characterEnElements: "甜蜜光斑、奶油雾气、缤纷水果飞溅",
    palette: "朱红 + 蜜桃粉 + 奶白 + 金边",
    sideDishes: "水果拼盘、双皮奶、黑芝麻糊炖奶、彩色水果捞",
    items: [
      ["果切4拼", "23元/盒"],
      ["水果捞", "20元"],
      ["双皮奶", "15元/份"],
      ["黑芝麻糊炖奶", "15元/份"],
    ],
  },
];

function buildItemList(items) {
  return items
    .map(([name, price], i) => `  ${String(i + 1).padStart(2, "0")}. ${name}  ——  ${price}`)
    .join("\n");
}

function buildPrompt(tv) {
  const itemList = buildItemList(tv.items);
  const itemCount = tv.items.length;
  const layoutHint =
    itemCount > 12
      ? "价目板分两栏紧凑排版，行高紧凑但保持清晰可读，禁止重叠或溢出"
      : itemCount > 6
        ? "价目板单栏或双栏均可，每行宽松对齐"
        : "价目板单栏大字号排版，条目舒朗可读";

  return [
    "横屏 3:2 中式国潮节庆海报，整体红金配色，写实国潮插画质感（仿油画 + 工笔细节），风格延续同系列哪吒主题海报。",
    "",
    "【背景】暗红到朱红的金色描边渐变；左上至中上区域有一行大型立体金色毛笔字「味在绥安」（流金泼墨笔触，有金属高光与浮雕描边），周围金色烟雾流光。除此之外画面中不允许出现任何其他中文、英文、数字、广告语、店招或副标题。",
    "",
    "【环境元素】左右各悬挂一组红色宫灯，下方铺设祥云金线浮雕带，远景淡金色山形纹饰；不允许出现「绥安」以外的任何招牌字。",
    "",
    `【主角】右侧约 35% 画面绘制：${tv.character}。神态与服饰严格匹配电影官方设定；动态构图，${tv.characterEnElements}。`,
    "",
    `【调色】${tv.palette}，整体与参考海报色温一致。`,
    "",
    `【价目板】中下方放置一块米黄色 / 奶油色木纹边的价目板（圆角矩形，金色描边与卷草纹边角），${layoutHint}。表头不需要任何标题字，左列为品名（深红字），右列为价格（深红字）。务必严格、原样、完整呈现以下 ${itemCount} 条条目，顺序、字符、单位都不得改动：`,
    "",
    itemList,
    "",
    `【辅助物】在主角与价目板四周点缀该档口代表性食物的写实特写：${tv.sideDishes}；造型实拍质感、热气与油亮反光。`,
    "",
    "【布局规则】「味在绥安」标题不被遮挡；价目板与角色互不压字；价目板四周留呼吸空间；条目较多时自动缩小字号但保持清晰；禁止文字重叠、溢出、断字、镜像或倒置。",
    "",
    "【负面】no extra slogans, no English captions, no watermarks, no logos, no QR codes, no malformed faces or hands, no duplicated dishes, no garbled / mirrored / upside-down characters, no other Chinese signage besides 「味在绥安」 and the price-board items.",
  ].join("\n");
}

async function loadEnvFile(filePath) {
  const text = await fs.readFile(filePath, "utf8").catch((e) => {
    if (e.code === "ENOENT") return "";
    throw e;
  });
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const m = line.match(/^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$/);
    if (!m) continue;
    const [, key, rawValue] = m;
    if (process.env[key]) continue;
    process.env[key] = rawValue.replace(/^['"]|['"]$/g, "");
  }
}

function parseArgs(argv) {
  const args = { only: null, dryRun: false, concurrency: 4, outputSubdir: "" };
  for (let i = 0; i < argv.length; i += 1) {
    const a = argv[i];
    const next = () => {
      i += 1;
      if (i >= argv.length) throw new Error(`Missing value after ${a}`);
      return argv[i];
    };
    if (a === "--only") {
      args.only = next()
        .split(",")
        .map((s) => Number(s.trim()))
        .filter((n) => Number.isInteger(n) && n >= 1 && n <= 8);
    } else if (a === "--dry-run") args.dryRun = true;
    else if (a === "--concurrency") args.concurrency = Math.max(1, Number(next()) || 1);
    else if (a === "--output-subdir") args.outputSubdir = next().replace(/[\\/]+/g, "-").trim();
    else if (a === "-h" || a === "--help") {
      console.log("Usage: node generate-posters.mjs [--only 1,3,7] [--dry-run] [--concurrency 4] [--output-subdir name]");
      process.exit(0);
    } else throw new Error(`Unknown option: ${a}`);
  }
  return args;
}

async function callImageApi(prompt) {
  const key = process.env.OPENAI_API_KEY;
  if (!key) throw new Error("OPENAI_API_KEY not found in env or .env file.");

  const body = {
    model: IMAGE_MODEL,
    prompt,
    n: 1,
    size: SIZE,
    quality: QUALITY,
    output_format: FORMAT,
  };

  const res = await fetch("https://api.openai.com/v1/images/generations", {
    method: "POST",
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  const text = await res.text();
  let json;
  try {
    json = JSON.parse(text);
  } catch {
    json = { raw: text };
  }
  if (!res.ok) {
    const err = new Error(json?.error?.message || text || res.statusText);
    err.status = res.status;
    throw err;
  }
  const image = json.data?.[0];
  if (!image?.b64_json) throw new Error("Image response missing b64_json");
  return {
    buffer: Buffer.from(image.b64_json, "base64"),
    metadata: {
      created: json.created,
      size: json.size || SIZE,
      quality: json.quality || QUALITY,
      output_format: json.output_format || FORMAT,
      usage: json.usage,
    },
  };
}

async function generateWithRetry(prompt) {
  try {
    return await callImageApi(prompt);
  } catch (e) {
    if (e.status === 429 || e.status === 500 || e.status === 503) {
      await new Promise((r) => setTimeout(r, 2000));
      try {
        return await callImageApi(prompt);
      } catch (e2) {
        if (e2.status === 429 || e2.status === 500 || e2.status === 503) {
          await new Promise((r) => setTimeout(r, 4000));
          return await callImageApi(prompt);
        }
        throw e2;
      }
    }
    throw e;
  }
}

async function processOne(tv, args) {
  const baseName = `${String(tv.index).padStart(2, "0")}-${tv.slug}`;
  const outputDir = args.outputDir || POSTERS_DIR;
  const promptDir = args.promptDir || PROMPTS_DIR;
  const promptPath = path.join(promptDir, `${baseName}.txt`);
  const imagePath = path.join(outputDir, `${baseName}.${FORMAT}`);
  const prompt = buildPrompt(tv);

  await fs.writeFile(promptPath, prompt + "\n");

  if (args.dryRun) {
    console.log(`[${tv.index}] dry-run prompt written: ${promptPath}`);
    return { index: tv.index, slug: tv.slug, status: "dry-run", prompt_file: path.basename(promptPath) };
  }

  const startedAt = Date.now();
  console.log(`[${tv.index}] generating ${tv.title} ...`);
  try {
    const { buffer, metadata } = await generateWithRetry(prompt);
    await fs.writeFile(imagePath, buffer);
    const elapsed = ((Date.now() - startedAt) / 1000).toFixed(1);
    console.log(`[${tv.index}] OK ${baseName}.${FORMAT} (${elapsed}s)`);
    return {
      index: tv.index,
      slug: tv.slug,
      status: "ok",
      prompt_file: path.basename(promptPath),
      image_file: path.basename(imagePath),
      elapsed_seconds: Number(elapsed),
      metadata,
    };
  } catch (e) {
    console.error(`[${tv.index}] FAIL ${tv.title}: ${e.message}`);
    return {
      index: tv.index,
      slug: tv.slug,
      status: "error",
      prompt_file: path.basename(promptPath),
      error: e.message,
    };
  }
}

async function runWithConcurrency(items, limit, fn) {
  const results = new Array(items.length);
  let cursor = 0;
  const workers = Array.from({ length: Math.min(limit, items.length) }, async () => {
    while (true) {
      const i = cursor++;
      if (i >= items.length) return;
      results[i] = await fn(items[i]);
    }
  });
  await Promise.all(workers);
  return results;
}

async function main() {
  await loadEnvFile(ENV_FILE);
  const args = parseArgs(process.argv.slice(2));
  args.outputDir = args.outputSubdir ? path.join(POSTERS_DIR, args.outputSubdir) : POSTERS_DIR;
  args.promptDir = args.outputSubdir ? path.join(args.outputDir, "prompts") : PROMPTS_DIR;
  await fs.mkdir(args.promptDir, { recursive: true });
  await fs.mkdir(args.outputDir, { recursive: true });

  const targets = args.only ? TVS.filter((t) => args.only.includes(t.index)) : TVS;
  console.log(
    `Generating ${targets.length} poster(s) at ${SIZE} ${QUALITY} (concurrency=${args.concurrency}${args.dryRun ? ", dry-run" : ""}).`,
  );

  const results = await runWithConcurrency(targets, args.concurrency, (tv) => processOne(tv, args));

  const manifestPath = path.join(args.outputDir, "manifest.json");
  let existing = {};
  try {
    existing = JSON.parse(await fs.readFile(manifestPath, "utf8"));
  } catch {
    existing = { images: [] };
  }
  const byIndex = new Map((existing.images || []).map((r) => [r.index, r]));
  for (const r of results) byIndex.set(r.index, r);
  const manifest = {
    image_model: IMAGE_MODEL,
    size: SIZE,
    quality: QUALITY,
    format: FORMAT,
    updated_at: new Date().toISOString(),
    images: Array.from(byIndex.values()).sort((a, b) => a.index - b.index),
  };
  await fs.writeFile(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);

  const okCount = results.filter((r) => r.status === "ok" || r.status === "dry-run").length;
  console.log(`Done: ${okCount}/${results.length} succeeded. Manifest: ${manifestPath}`);
  const failed = results.filter((r) => r.status === "error");
  if (failed.length) {
    console.log(`Retry failed: node scripts/generate-posters.mjs --only ${failed.map((r) => r.index).join(",")}`);
  }
}

main().catch((e) => {
  console.error(e.stack || e.message);
  process.exit(1);
});
