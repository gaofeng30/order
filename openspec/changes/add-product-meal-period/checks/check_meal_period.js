#!/usr/bin/env node
/* PC 菜品餐段可售门禁。用法: node check_meal_period.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const WA = path.join(root, 'apps/web-admin');
const fails = [], pending = [];
const check = (label, fn) => {
  try {
    const r = fn();
    if (r && typeof r.then === 'function') pending.push(r.catch(e => fails.push(`${label}: ${String(e.message).split('\n')[0]}`)));
  } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const MEALS = ['all', 'lunch', 'dinner'];
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const ctx = () => {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  return sb.window;
};

check('every seeded product declares a meal period', () => {
  const { Seed } = ctx();
  for (const p of Seed.MENU) {
    assert.equal(MEALS.includes(p.meal), true, `${p.id}.meal = ${p.meal}`);
  }
});

check('the contract requires and validates the meal period', () => {
  const w = ctx();
  w.__store = { menu: JSON.parse(JSON.stringify(w.Seed.MENU)), cats: JSON.parse(JSON.stringify(w.Seed.ADMIN_CATS)) };
  const base = { name: '门禁测试菜', price: 20, cat: w.Seed.MENU[0].cat, desc: '', imgs: [] };
  return w.Api.saveProduct(Object.assign({}, base, { meal: 'lunch' }))
    .then(created => {
      assert.equal(created.meal, 'lunch', 'saved product lost its meal period');
      return w.Api.saveProduct(Object.assign({}, base, { name: '缺餐段' }))
        .then(() => { throw new Error('accepted a product without a meal period'); }, () => {});
    })
    .then(() => w.Api.saveProduct(Object.assign({}, base, { name: '非法餐段', meal: '中午' }))
      .then(() => { throw new Error('accepted an invalid meal period'); }, () => {}));
});

check('the contract is the single source of meal-period labels', () => {
  // 标签由契约层提供，页面从 Api.MEAL_LABEL 渲染 —— 因此断言写在运行态，
  // 而不是去页面里找「全天」这三个字。
  const { Api } = ctx();
  assert.equal(JSON.stringify(Api.MEALS), JSON.stringify(MEALS));
  assert.equal(JSON.stringify(Api.MEAL_LABEL), JSON.stringify({ all: '全天', lunch: '午餐', dinner: '晚餐' }));
});

check('the product form and table expose the meal period', () => {
  const src = read('pages/products.js');
  assert.match(src, /id="f-meal"/, 'edit form has no meal-period control');
  assert.match(src, /餐段可售/, 'edit form does not label the meal-period field');
  assert.match(src, /Api\.MEALS\.map/, 'form does not render options from the contract');
  assert.match(src, /\{ t: '餐段'/, 'product table has no meal-period column');
  assert.match(src, /Api\.MEAL_LABEL\[r\.meal\]/, 'table does not render the label from the contract');
  assert.match(src, /meal: root\.querySelector\('#f-meal'\)\.value/, 'form does not submit the meal period');
});

/* 接管 specify-bulk-import 的 PRD 断言。
   那份归档门禁断言 §6.13 标着「开发方提出的范围新增，待客户确认」，
   而本 change 按项目负责人裁决把该标注改为「已确认纳入一期」——
   归档产物不修改，其断言集合由此处按新状态接管并补齐。 */
check('bulk-import PRD assertions, superseding the archived gate', () => {
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const need = t => { if (!prd.includes(t)) throw new Error(`PRD 缺少 ${JSON.stringify(t)}`); };
  const forbid = t => { if (prd.includes(t)) throw new Error(`PRD 不应出现 ${JSON.stringify(t)}`); };
  // 结构与格式决定（承自旧门禁）
  for (const t of ['### 6.13 批量导入（PC 后台）', '#### 6.13.1 通用流程', '#### 6.13.2 菜品批量导入',
                   '#### 6.13.3 员工白名单批量导入', '#### 6.13.4 不做批量导入的对象',
                   '`.xlsx`', '**不接受 CSV**', '**去重规则：只新增，不更新。**',
                   '**去重规则：按手机号覆盖更新。**', '**图片 MUST NOT 进入模板。**',
                   '**分类自动新建**', '**商户账号名单不提供批量导入**',
                   '解析 MUST 在服务端完成', '**PC 网页后台（12 页）**']) need(t);
  for (const t of ['CSV 批量导入', '导入页做显式检测并提示', '| CSV 为 GBK 编码 |']) forbid(t);
  // 新状态：范围已确认，且必须标明确认来自项目方而非客户
  forbid('开发方提出的范围新增，待客户确认');
  need('项目负责人于 2026-08-21 确认纳入一期范围');
  need('整份评审记录仍待客户书面签认');
  // P0 例外必须明确不构成生产契约
  need('**P0 原型例外**');
  need('不构成生产契约');
});

check('all javascript parses', () => {
  const files = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p); else if (e.name.endsWith('.js')) files.push(p);
    }
  })(WA);
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

Promise.all(pending).then(() => {
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('MEAL_PERIOD_GATE=FAIL'); process.exit(1); }
  console.log('MEAL_PERIOD_GATE=PASS');
});
