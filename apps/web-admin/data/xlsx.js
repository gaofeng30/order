/* 最小 .xlsx 读取器 —— 契约层内部实现，页面不得直接调用。
   PC 后台是零构建静态站点，不引第三方解析库；ZIP 由本文件手写解析，
   deflate 走浏览器与 Node 均内置的 DecompressionStream('deflate-raw')。

   生效 spec 与 PRD §6.13.1 规定解析 MUST 在服务端完成；本文件是该节
   标注的 P0 原型例外，用于在后端导入接口就位前验证流程、模板与校验规则，
   不构成生产契约。后端就位时连同 data/api.js 的导入实现一并替换。

   支持范围：Excel 生成的常规 .xlsx（stored 与 deflate 两种压缩方式、
   sharedStrings、inlineStr、数值单元格）。不支持加密、宏与多工作表选择，
   固定读取第一张工作表。 */
(function () {
  const dec = new TextDecoder('utf-8');

  /* ---------- ZIP ---------- */
  function findEOCD(view, len) {
    const max = Math.min(len, 0xffff + 22);
    for (let i = len - 22; i >= len - max; i--) {
      if (i >= 0 && view.getUint32(i, true) === 0x06054b50) return i;
    }
    throw new Error('不是有效的 .xlsx 文件');
  }

  function listEntries(buf) {
    const view = new DataView(buf);
    const len = buf.byteLength;
    const eocd = findEOCD(view, len);
    const count = view.getUint16(eocd + 10, true);
    let p = view.getUint32(eocd + 16, true);
    const out = {};
    for (let i = 0; i < count; i++) {
      if (view.getUint32(p, true) !== 0x02014b50) throw new Error('不是有效的 .xlsx 文件');
      const method = view.getUint16(p + 10, true);
      const compSize = view.getUint32(p + 20, true);
      const nameLen = view.getUint16(p + 28, true);
      const extraLen = view.getUint16(p + 30, true);
      const cmtLen = view.getUint16(p + 32, true);
      const local = view.getUint32(p + 42, true);
      const name = dec.decode(new Uint8Array(buf, p + 46, nameLen));
      out[name] = { method, compSize, local };
      p += 46 + nameLen + extraLen + cmtLen;
    }
    return out;
  }

  async function readEntry(buf, entry) {
    const view = new DataView(buf);
    if (view.getUint32(entry.local, true) !== 0x04034b50) throw new Error('不是有效的 .xlsx 文件');
    const nameLen = view.getUint16(entry.local + 26, true);
    const extraLen = view.getUint16(entry.local + 28, true);
    const start = entry.local + 30 + nameLen + extraLen;
    const raw = new Uint8Array(buf, start, entry.compSize);
    if (entry.method === 0) return dec.decode(raw);
    if (entry.method !== 8) throw new Error('不支持的压缩方式，请用 Excel 另存为 .xlsx');
    const stream = new Response(raw).body.pipeThrough(new DecompressionStream('deflate-raw'));
    return dec.decode(new Uint8Array(await new Response(stream).arrayBuffer()));
  }

  /* ---------- XML ---------- */
  const unescape = s => s
    .replace(/&lt;/g, '<').replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"').replace(/&apos;/g, "'")
    .replace(/&#(\d+);/g, (_, d) => String.fromCodePoint(Number(d)))
    .replace(/&amp;/g, '&');

  function sharedStrings(xml) {
    if (!xml) return [];
    return (xml.match(/<si\b[\s\S]*?<\/si>|<si\b[^>]*\/>/g) || []).map(si => {
      const parts = si.match(/<t\b[^>]*>([\s\S]*?)<\/t>/g) || [];
      return parts.map(t => unescape(t.replace(/<t\b[^>]*>([\s\S]*?)<\/t>/, '$1'))).join('');
    });
  }

  const colIndex = ref => {
    const m = /^([A-Z]+)/.exec(ref || '');
    if (!m) return 0;
    let n = 0;
    for (const ch of m[1]) n = n * 26 + (ch.charCodeAt(0) - 64);
    return n - 1;
  };

  function sheetRows(xml, sst) {
    const rows = [];
    for (const rowXml of xml.match(/<row\b[\s\S]*?<\/row>|<row\b[^>]*\/>/g) || []) {
      const rIdx = Number((/<row\b[^>]*\br="(\d+)"/.exec(rowXml) || [])[1] || rows.length + 1) - 1;
      const cells = [];
      for (const cXml of rowXml.match(/<c\b[\s\S]*?<\/c>|<c\b[^>]*\/>/g) || []) {
        const ref = (/\br="([A-Z]+\d+)"/.exec(cXml) || [])[1];
        const type = (/\bt="([^"]+)"/.exec(cXml) || [])[1];
        let val = '';
        if (type === 's') {
          const i = Number((/<v>([\s\S]*?)<\/v>/.exec(cXml) || [])[1]);
          val = sst[i] == null ? '' : sst[i];
        } else if (type === 'inlineStr') {
          val = (cXml.match(/<t\b[^>]*>([\s\S]*?)<\/t>/g) || [])
            .map(t => unescape(t.replace(/<t\b[^>]*>([\s\S]*?)<\/t>/, '$1'))).join('');
        } else {
          const m = /<v>([\s\S]*?)<\/v>/.exec(cXml);
          val = m ? unescape(m[1]) : '';
        }
        cells[colIndex(ref)] = String(val);
      }
      for (let i = 0; i < cells.length; i++) if (cells[i] == null) cells[i] = '';
      rows[rIdx] = cells;
    }
    for (let i = 0; i < rows.length; i++) if (!rows[i]) rows[i] = [];
    return rows;
  }

  /* ---------- 对外：读取第一张工作表的全部行 ---------- */
  async function readRows(file) {
    if (!file || !/\.xlsx$/i.test(file.name || '')) throw new Error('只接受 .xlsx 文件，请用 Excel 另存为 .xlsx');
    const buf = await file.arrayBuffer();
    const entries = listEntries(buf);
    const sheetName = Object.keys(entries).find(n => /^xl\/worksheets\/sheet\d+\.xml$/i.test(n));
    if (!sheetName) throw new Error('文件中没有可读取的工作表');
    const sstEntry = entries['xl/sharedStrings.xml'];
    const sst = sharedStrings(sstEntry ? await readEntry(buf, sstEntry) : '');
    return sheetRows(await readEntry(buf, entries[sheetName]), sst);
  }

  window.Xlsx = { readRows };
})();
