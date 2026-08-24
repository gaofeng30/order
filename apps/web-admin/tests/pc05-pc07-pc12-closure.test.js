const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const source = name => fs.readFileSync(path.join(root, name), 'utf8');

const importFlow = source('ui/import-flow.js');
const productImport = source('pages/product-import.js');
const staffImport = source('pages/staff-import.js');
const products = source('pages/products.js');
const layer = source('pages/layer.js');

assert.match(importFlow, /data-template/,
  'PC11/PC12 must expose a visible template download action');
assert.match(importFlow, /application\/vnd\.openxmlformats-officedocument\.spreadsheetml\.sheet/,
  'the downloaded template must be a real xlsx MIME payload');
assert.match(importFlow, /\[Content_Types\]\.xml/,
  'the template must be an OpenXML workbook, not renamed CSV');
assert.match(productImport, /templateRows:\s*\[\s*\['菜品名称',\s*'售价',\s*'分类',\s*'餐段可售',\s*'描述'\]\s*\]/,
  'PC11 template columns must exactly match the canonical product import contract');
assert.match(staffImport, /templateRows:\s*\[\s*\['姓名',\s*'手机号'\]\s*\]/,
  'PC12 template must contain only the two canonical writable staff fields');

assert.match(products, /Promise\.all\(list\.map\(f => Api\.uploadImage\(f\)\)\)[\s\S]*?\.catch\(/,
  'PC05 image upload failure must be visibly handled');
assert.match(layer, /Api\.uploadImage\(f\)[\s\S]*?\.catch\(/,
  'PC08 image upload failure must be visibly handled');
assert.match(layer, /Api\.saveLayer\(cfg\)[\s\S]*?\.catch\(/,
  'PC08 save failure must not produce an unhandled false-success path');
assert.match(layer, /id="lay-img"[^>]+onerror="this\.hidden=true"/,
  'PC08 must hide a persisted object that becomes unreadable');

console.log('pc05-pc07-pc12 closure static contract: PASS');
