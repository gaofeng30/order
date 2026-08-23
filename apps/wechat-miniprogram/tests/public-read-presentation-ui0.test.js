const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const ROOT = path.join(__dirname, '..');

function source(file) {
  return fs.readFileSync(path.join(ROOT, file), 'utf8');
}

test('menu renders the frozen cover and shows staff and original unit prices', () => {
  const wxml = source('pages/menu/menu.wxml');
  assert.match(wxml, /wx:if="\{\{m\.cover\}\}"[^>]*src="\{\{m\.cover\.url\}\}"/);
  assert.match(wxml, /wx:else class="dish-cover dish-cover--empty"/);
  assert.match(wxml, /wx:if="\{\{m\.isStaffPrice\}\}"[^>]*>员工价/);
  assert.match(wxml, /original_price_text/);
  assert.match(wxml, /ci\.item\.isStaffPrice/);
});

test('detail and confirm visibly retain the original price when a staff price applies', () => {
  const detail = source('pages/detail/detail.wxml');
  const confirm = source('pages/confirm/confirm.wxml');
  assert.match(detail, /wx:if="\{\{m\.isStaffPrice\}\}"[^>]*>员工价/);
  assert.match(detail, /m\.original_price_text/);
  assert.match(confirm, /ci\.item\.isStaffPrice/);
  assert.match(confirm, /ci\.item\.original_price_text/);
  assert.match(confirm, /ci\.item\.price_text/);
});
