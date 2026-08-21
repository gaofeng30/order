/* 最小 .xlsx 夹具构造器：手写 ZIP（支持 stored 与 deflate-raw 两种压缩方式），
   用于在无第三方库的条件下验证契约层的解析实现。仅用于门禁，不进产物。 */
const zlib = require('node:zlib');

const CRC = (() => {
  const t = new Int32Array(256);
  for (let i = 0; i < 256; i++) { let c = i; for (let k = 0; k < 8; k++) c = c & 1 ? 0xEDB88320 ^ (c >>> 1) : c >>> 1; t[i] = c; }
  return buf => { let c = -1; for (const b of buf) c = t[(c ^ b) & 0xff] ^ (c >>> 8); return (c ^ -1) >>> 0; };
})();

function zip(entries) {
  const locals = [], central = [];
  let offset = 0;
  for (const { name, data, deflate } of entries) {
    const raw = Buffer.from(data, 'utf8');
    const body = deflate ? zlib.deflateRawSync(raw) : raw;
    const nameBuf = Buffer.from(name, 'utf8');
    const crc = CRC(raw);
    const lh = Buffer.alloc(30);
    lh.writeUInt32LE(0x04034b50, 0); lh.writeUInt16LE(20, 4); lh.writeUInt16LE(0, 6);
    lh.writeUInt16LE(deflate ? 8 : 0, 8); lh.writeUInt16LE(0, 10); lh.writeUInt16LE(0, 12);
    lh.writeUInt32LE(crc, 14); lh.writeUInt32LE(body.length, 18); lh.writeUInt32LE(raw.length, 22);
    lh.writeUInt16LE(nameBuf.length, 26); lh.writeUInt16LE(0, 28);
    locals.push(lh, nameBuf, body);

    const ch = Buffer.alloc(46);
    ch.writeUInt32LE(0x02014b50, 0); ch.writeUInt16LE(20, 4); ch.writeUInt16LE(20, 6);
    ch.writeUInt16LE(0, 8); ch.writeUInt16LE(deflate ? 8 : 0, 10);
    ch.writeUInt16LE(0, 12); ch.writeUInt16LE(0, 14);
    ch.writeUInt32LE(crc, 16); ch.writeUInt32LE(body.length, 20); ch.writeUInt32LE(raw.length, 24);
    ch.writeUInt16LE(nameBuf.length, 28); ch.writeUInt16LE(0, 30); ch.writeUInt16LE(0, 32);
    ch.writeUInt16LE(0, 34); ch.writeUInt16LE(0, 36); ch.writeUInt32LE(0, 38);
    ch.writeUInt32LE(offset, 42);
    central.push(ch, nameBuf);
    offset += lh.length + nameBuf.length + body.length;
  }
  const cd = Buffer.concat(central);
  const eocd = Buffer.alloc(22);
  eocd.writeUInt32LE(0x06054b50, 0); eocd.writeUInt16LE(0, 4); eocd.writeUInt16LE(0, 6);
  eocd.writeUInt16LE(entries.length, 8); eocd.writeUInt16LE(entries.length, 10);
  eocd.writeUInt32LE(cd.length, 12); eocd.writeUInt32LE(offset, 16); eocd.writeUInt16LE(0, 20);
  return Buffer.concat([Buffer.concat(locals), cd, eocd]);
}

const colName = i => { let s = '', n = i; while (n > 0) { const m = (n - 1) % 26; s = String.fromCharCode(65 + m) + s; n = (n - m - 1) / 26; } return s; };

/* rows: string[][]；全部走 sharedStrings，覆盖 t="s" 分支 */
function buildXlsx(rows, opts = {}) {
  const shared = [];
  const idx = v => { const i = shared.indexOf(v); return i >= 0 ? i : shared.push(v) - 1; };
  const sheetRows = rows.map((r, ri) => {
    const cells = r.map((v, ci) => {
      if (v === '' || v == null) return '';
      const ref = colName(ci + 1) + (ri + 1);
      if (opts.numericCols && opts.numericCols.includes(ci) && /^-?\d+(\.\d+)?$/.test(v)) {
        return `<c r="${ref}"><v>${v}</v></c>`;
      }
      return `<c r="${ref}" t="s"><v>${idx(String(v))}</v></c>`;
    }).join('');
    return `<row r="${ri + 1}">${cells}</row>`;
  }).join('');

  const sheet = `<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${sheetRows}</sheetData></worksheet>`;
  const sst = `<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="${shared.length}" uniqueCount="${shared.length}">${shared.map(s => `<si><t>${s.replace(/&/g, '&amp;').replace(/</g, '&lt;')}</t></si>`).join('')}</sst>`;
  const wb = `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`;
  const ct = `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`;
  return zip([
    { name: '[Content_Types].xml', data: ct, deflate: false },
    { name: 'xl/workbook.xml', data: wb, deflate: true },
    { name: 'xl/sharedStrings.xml', data: sst, deflate: true },
    { name: 'xl/worksheets/sheet1.xml', data: sheet, deflate: !opts.storedSheet },
  ]);
}

module.exports = { buildXlsx };
