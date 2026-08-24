/* 批量导入三步流程外壳（PRD §6.13.1）—— 两个导入页共用。
   页面只提供模板定义与契约方法，不接触文件解析。 */
(function () {
  const T = window.Table, I = window.Icon;

  /* cfg: { title, hint, columns:[{name,required,note}], sample, templateRows,
            preview(file), commit(token), backRoute, extra(preview) } */
  function render(el, cfg) {
    let state = { file: null, preview: null, busy: false };

    function paint() {
      el.innerHTML =
        `<div class="page-head">
           <span class="ph-s grow">${cfg.hint}</span>
           <button class="btn btn--line" data-back>返回列表</button>
         </div>
         <div class="card card-pad imp-card">
           <div class="row gap12">
             <div class="fld-lb grow">第一步 · 模板列</div>
             <button class="btn btn--line btn--sm" data-template>下载 .xlsx 模板</button>
           </div>
           <div class="imp-cols">
             ${cfg.columns.map(c => `<span class="imp-col${c.required ? ' req' : ''}">${T.esc(c.name)}${c.required ? ' *' : ''}</span>`).join('')}
           </div>
           <div class="fld-hint">首行必须是表头，按列名匹配，与列的先后顺序无关。未知列会被忽略并在预览中列出。${cfg.sample ? ` 示例：${T.esc(cfg.sample)}` : ''}</div>
           ${cfg.columns.filter(c => c.note).map(c => `<div class="imp-note">${T.esc(c.name)}：${T.esc(c.note)}</div>`).join('')}
         </div>
         <div class="card card-pad imp-card">
           <div class="fld-lb">第二步 · 选择文件</div>
           <div class="imp-pick">
             <input type="file" id="f-file" accept=".xlsx">
             <span class="faint">${state.file ? T.esc(state.file.name) : '只接受 .xlsx，单次最多 ' + (cfg.maxRows || window.Api.MAX_IMPORT_ROWS) + ' 行，文件不超过 10 MiB'}</span>
           </div>
         </div>
         <div id="imp-result"></div>`;

      el.querySelector('[data-back]').onclick = () => window.App.go(cfg.backRoute);
      el.querySelector('[data-template]').onclick = () => downloadTemplate(cfg);
      el.querySelector('#f-file').onchange = e => {
        const f = e.target.files && e.target.files[0];
        if (!f) return;
        state.file = f; state.preview = null;
        cfg.preview(f)
          .then(p => { state.preview = p; paint(); paintResult(); })
          .catch(err => { state.preview = null; paint(); window.Toast.show(err.message, { icon: 'warn' }); });
      };
      paintResult();
    }

    function paintResult() {
      const host = el.querySelector('#imp-result');
      const p = state.preview;
      if (!host) return;
      if (!p) { host.innerHTML = ''; return; }
      const total = p.added + p.updated;
      host.innerHTML =
        `<div class="card card-pad imp-card">
           <div class="fld-lb">第三步 · 确认导入</div>
           <div class="imp-counts">
             <span class="imp-cnt add">新增 <b class="tnum">${p.added}</b> 条</span>
             <span class="imp-cnt upd">更新 <b class="tnum">${p.updated}</b> 条</span>
             <span class="imp-cnt err${p.errors.length ? ' on' : ''}">异常 <b class="tnum">${p.errors.length}</b> 条</span>
           </div>
           ${cfg.extra ? cfg.extra(p) : ''}
           ${p.ignoredColumns && p.ignoredColumns.length
             ? `<div class="imp-note">已忽略未知列：${p.ignoredColumns.map(T.esc).join('、')}</div>` : ''}
           ${p.errors.length
             ? `<details class="imp-errs" open><summary>异常 ${p.errors.length} 条（可跳过后继续导入）</summary>
                  ${p.errors.map(e => `<div class="imp-err"><span class="tnum">第 ${e.row} 行</span>${T.esc(e.reason)}</div>`).join('')}
                </details>` : ''}
           <div class="imp-acts">
             <button class="btn btn--line" data-cancel>取消并重选文件</button>
             <button class="btn btn--primary" data-ok${total ? '' : ' disabled'}>
               ${p.errors.length ? `跳过异常行，导入 ${total} 条` : `确认导入 ${total} 条`}
             </button>
           </div>
           ${total ? '' : '<div class="fld-hint">没有可导入的行，请修正后重新选择文件。</div>'}
         </div>`;

      host.querySelector('[data-cancel]').onclick = () => { state.file = null; state.preview = null; paint(); };
      const ok = host.querySelector('[data-ok]');
      if (ok && total) ok.onclick = () => {
        if (state.busy) return;
        state.busy = true;
        cfg.commit(p.token)
          .then(r => {
            state.busy = false;
            if (r.duplicate) { window.Toast.show('该批次已导入过，未重复写入', { icon: 'warn' }); return; }
            window.Toast.show(`导入完成 · 新增 ${r.added} 条${r.updated ? ` · 更新 ${r.updated} 条` : ''}`, { icon: 'check' });
            window.App.go(cfg.backRoute);
          })
          .catch(err => { state.busy = false; window.Toast.show(err.message, { icon: 'warn' }); });
      };
    }

    paint();
  }

  function downloadTemplate(cfg) {
    const rows = Array.isArray(cfg.templateRows) ? cfg.templateRows : [];
    if (rows.length !== 1 || !Array.isArray(rows[0]) || !rows[0].length) {
      return window.Toast.show('模板暂时不可用', { icon: 'warn' });
    }
    const bytes = workbook(rows);
    const blob = new Blob([bytes], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = cfg.backRoute === 'product-import' ? '菜品批量导入模板.xlsx' : '员工白名单批量导入模板.xlsx';
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }

  // 只生成表头工作簿；业务文件解析始终由服务端完成。
  function workbook(rows) {
    const files = [
      ['[Content_Types].xml', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`],
      ['_rels/.rels', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`],
      ['xl/workbook.xml', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="导入模板" sheetId="1" r:id="rId1"/></sheets></workbook>`],
      ['xl/_rels/workbook.xml.rels', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`],
      ['xl/worksheets/sheet1.xml', sheet(rows)],
    ];
    return zip(files);
  }

  function sheet(rows) {
    const body = rows.map((row, y) => `<row r="${y + 1}">${row.map((value, x) => `<c r="${column(x)}${y + 1}" t="inlineStr"><is><t>${xml(value)}</t></is></c>`).join('')}</row>`).join('');
    return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${body}</sheetData></worksheet>`;
  }

  function column(index) {
    let value = index + 1, out = '';
    while (value) { value -= 1; out = String.fromCharCode(65 + (value % 26)) + out; value = Math.floor(value / 26); }
    return out;
  }

  function xml(value) {
    return String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&apos;');
  }

  function zip(files) {
    const encoder = new TextEncoder();
    const locals = [], centrals = [];
    let offset = 0;
    files.forEach(([name, value]) => {
      const nameBytes = encoder.encode(name), data = encoder.encode(value), checksum = crc32(data);
      const local = new Uint8Array(30 + nameBytes.length + data.length);
      const lv = new DataView(local.buffer);
      lv.setUint32(0, 0x04034b50, true); lv.setUint16(4, 20, true); lv.setUint16(8, 0, true);
      lv.setUint32(14, checksum, true); lv.setUint32(18, data.length, true); lv.setUint32(22, data.length, true); lv.setUint16(26, nameBytes.length, true);
      local.set(nameBytes, 30); local.set(data, 30 + nameBytes.length); locals.push(local);
      const central = new Uint8Array(46 + nameBytes.length);
      const cv = new DataView(central.buffer);
      cv.setUint32(0, 0x02014b50, true); cv.setUint16(4, 20, true); cv.setUint16(6, 20, true); cv.setUint16(10, 0, true);
      cv.setUint32(16, checksum, true); cv.setUint32(20, data.length, true); cv.setUint32(24, data.length, true); cv.setUint16(28, nameBytes.length, true); cv.setUint32(42, offset, true);
      central.set(nameBytes, 46); centrals.push(central); offset += local.length;
    });
    const centralSize = centrals.reduce((sum, item) => sum + item.length, 0);
    const end = new Uint8Array(22), ev = new DataView(end.buffer);
    ev.setUint32(0, 0x06054b50, true); ev.setUint16(8, files.length, true); ev.setUint16(10, files.length, true); ev.setUint32(12, centralSize, true); ev.setUint32(16, offset, true);
    const result = new Uint8Array(offset + centralSize + end.length);
    let cursor = 0;
    [...locals, ...centrals, end].forEach(part => { result.set(part, cursor); cursor += part.length; });
    return result;
  }

  function crc32(data) {
    let value = 0xffffffff;
    for (const byte of data) {
      value ^= byte;
      for (let bit = 0; bit < 8; bit += 1) value = (value >>> 1) ^ ((value & 1) ? 0xedb88320 : 0);
    }
    return (value ^ 0xffffffff) >>> 0;
  }

  window.ImportFlow = { render };
})();
