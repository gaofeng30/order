#!/usr/bin/env node
/* 订单卡片的取餐时间与取餐地点分行（PRD §5.8）。
   用法: node check_card_meta.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const MP = path.join(root, 'apps/wechat-miniprogram');
const fails = [];
const check = (label, fn) => {
  try { fn(); } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const read = rel => fs.readFileSync(path.join(MP, rel), 'utf8');
const walk = (d, out = []) => {
  for (const e of fs.readdirSync(d, { withFileTypes: true })) {
    if (e.name === 'node_modules' || e.name === '__pycache__' || e.name === 'tests') continue;
    const p = path.join(d, e.name);
    if (e.isDirectory()) walk(p, out); else out.push(p);
  }
  return out;
};

function mp() {
  for (const k of Object.keys(require.cache)) if (k.startsWith(MP + path.sep)) delete require.cache[k];
  let pageDef = null;
  global.Behavior = d => d; global.Component = d => d; global.App = () => {};
  global.Page = d => { pageDef = d; };
  global.wx = {
    getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, safeArea: { bottom: 778 } }),
    getSystemInfoSync() { return this.getWindowInfo(); },
    getMenuButtonBoundingClientRect: () => ({ top: 26, left: 278, height: 32 }),
    navigateTo() {}, redirectTo() {}, reLaunch() {}, navigateBack() {}, setClipboardData() {},
    request(r) { queueMicrotask(() => r.fail({ errMsg: 'offline' })); return { abort() {} }; },
  };
  const data = require(path.join(MP, 'utils/data.js'));
  const cl = v => JSON.parse(JSON.stringify(v));
  const globalData = {
    statusBarHeight: 20, navBarHeight: 44, navTotalHeight: 64, safeBottom: 0,
    aOrders: cl(data.ADMIN_ORDERS), orders: cl(data.USER_ORDERS), menu: cl(data.MENU),
    soldOut: cl(data.PRODUCT_SOLD_OUT_DATES), cart: {}, store: cl(data.STORE),
  };
  global.getApp = () => ({ globalData });
  const load = rel => {
    pageDef = null;
    require(path.join(MP, rel));
    const behaviors = (pageDef.behaviors || []).filter(Boolean);
    const page = {
      data: Object.assign({}, ...behaviors.map(b => cl(b.data || {})), cl(pageDef.data || {})),
      setData(patch, cb) {
        for (const [k, v] of Object.entries(patch || {})) {
          const parts = k.split('.');
          let cur = this.data;
          for (let i = 0; i < parts.length - 1; i += 1) cur = (cur[parts[i]] ||= {});
          cur[parts[parts.length - 1]] = cl(v);
        }
        if (cb) cb.call(this);
      },
      selectComponent: () => ({ show() {} }),
      createSelectorQuery: () => {
        const q = { select: () => q, selectAll: () => q, boundingClientRect: () => q, exec: cb => cb([]) };
        return q;
      },
    };
    for (const b of behaviors) Object.assign(page, b.methods || {});
    for (const [k, v] of Object.entries(pageDef)) if (k !== 'data' && k !== 'behaviors') page[k] = v;
    page.__behaviors = behaviors; page.__def = pageDef;
    return page;
  };
  const invoke = (page, hook, arg) => {
    for (const b of page.__behaviors) if (typeof b[hook] === 'function') b[hook].call(page, arg);
    if (typeof page.__def[hook] === 'function') return page.__def[hook].call(page, arg);
    return undefined;
  };
  return { data, globalData, load, invoke };
}
const openOrders = h => { const p = h.load('pages/orders/orders.js'); h.invoke(p, 'onLoad', {}); h.invoke(p, 'onShow'); return p; };

check('the card exposes time and place as separate fields', () => {
  const h = mp();
  const page = openOrders(h);
  assert.ok(page.data.list.length, 'the order list is empty');
  for (const row of page.data.list) {
    assert.equal(typeof row.timeText, 'string', `${row.id} has no time text`);
    assert.equal(typeof row.placeText, 'string', `${row.id} has no place text`);
    assert.ok(row.timeText.trim(), `${row.id} time text is empty`);
    assert.ok(row.placeText.trim(), `${row.id} place text is empty`);
  }
});

check('neither field carries the other information', () => {
  const h = mp();
  const page = openOrders(h);
  for (const row of page.data.list) {
    assert.ok(!row.timeText.includes(row.pickupPoint),
      `${row.id} time text still carries the pickup place: ${row.timeText}`);
    assert.ok(!row.placeText.includes(row.pickupTime),
      `${row.id} place text still carries the pickup time: ${row.placeText}`);
    assert.ok(!row.timeText.includes(' · '),
      `${row.id} time text still joins two facts: ${row.timeText}`);
  }
});

check('the place text is the order snapshot', () => {
  const h = mp();
  const page = openOrders(h);
  for (const row of page.data.list) {
    assert.equal(row.placeText, row.pickupPoint, `${row.id} place text is not the order snapshot`);
  }
  assert.doesNotMatch(read('pages/orders/orders.js'), /placeText:\s*[^,\n]*store\./,
    'the place text is read from store configuration instead of the order');
});

check('the time text names the pickup time', () => {
  const h = mp();
  const page = openOrders(h);
  for (const row of page.data.list) {
    assert.ok(row.timeText.includes(row.pickupTime),
      `${row.id} time text does not name the pickup time: ${row.timeText}`);
  }
});

check('the template renders two rows, each with its own icon', () => {
  const wxml = read('pages/orders/orders.wxml');
  const body = wxml.slice(wxml.indexOf('oc-body'), wxml.indexOf('oc-foot'));
  assert.match(body, /\{\{\s*item\.timeText\s*\}\}/, 'the template lost the time text');
  assert.match(body, /\{\{\s*item\.placeText\s*\}\}/, 'the template does not render the place text');
  const timeIdx = body.indexOf('item.timeText');
  const placeIdx = body.indexOf('item.placeText');
  assert.ok(timeIdx >= 0 && placeIdx >= 0 && timeIdx !== placeIdx, 'the two facts share one binding site');
  // 每条信息各自有图标：两段之间必须再出现一次 <icon
  const between = body.slice(Math.min(timeIdx, placeIdx), Math.max(timeIdx, placeIdx));
  assert.match(between, /<icon\b/, 'the two rows share a single icon');
});

check('info rows align their icon to the first line', () => {
  /* 取餐点是客户可配值，任何一行都可能因为配置变长而折行；
     此时 align-items: center 会让图标与两行都不对齐 —— 截图里就是这个现象。 */
  const wxss = read('pages/orders/orders.wxss').replace(/\/\*[\s\S]*?\*\//g, '');
  /* 按「选择器 { 声明 }」切分，不靠 indexOf —— `.oc-time icon` 这类后代选择器
     会让朴素查找抓到隔壁规则。 */
  const rules = [...wxss.matchAll(/([^{}]+)\{([^}]*)\}/g)]
    .map(m => ({ sel: m[1].trim(), body: m[2] }));
  const owns = cls => rules.filter(r =>
    r.sel.split(',').some(one => one.trim() === cls));
  for (const cls of ['.oc-time', '.oc-place']) {
    const mine = owns(cls);
    assert.ok(mine.length, `there is no style rule for ${cls}`);
    const aligned = mine.filter(r => /align-items:/.test(r.body));
    assert.ok(aligned.length, `${cls} declares no alignment`);
    for (const r of aligned) {
      assert.doesNotMatch(r.body, /align-items:\s*center/,
        `${cls} centres its icon; a wrapped line would leave it aligned to neither row`);
      assert.match(r.body, /align-items:\s*flex-start/,
        `${cls} does not align its icon to the first line`);
    }
  }
});

check('all javascript parses', () => {
  const files = walk(MP).filter(f => f.endsWith('.js'));
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`CARD_META_GATE=FAIL (${fails.length}/7)`);
  process.exit(1);
}
console.log('CARD_META_GATE=PASS');
