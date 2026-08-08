/* 批量导入名单 —— 对应 miniprogram/pages/admin-member-import（二期能力）

   这是 PC 端相对小程序端能力差最大的一页：
   PRD §6.12 写明「小程序读不到手机本地文件，唯一途径是 wx.chooseMessageFile，
   商户需先把 CSV 发到微信聊天再选择」。电脑端可以直接把文件拖进来。

   Excel 默认另存的 CSV 为 GBK 编码，按 UTF-8 读取会整片乱码，
   这里做显式检测并给出可执行的提示（规则与小程序端一致）。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  const TPL_COLS = [
    { k: '手机号', req: true }, { k: '姓名', req: true }, { k: '等级', req: true },
    { k: '单位', req: false }, { k: '部门', req: false }, { k: '工号', req: false },
  ];
  const TPL_HEADER = TPL_COLS.map(c => c.k).join(',');
  // 表头别名 → 内部字段
  const ALIAS = {
    手机号: 'phone', 手机: 'phone', 电话: 'phone', 联系电话: 'phone',
    姓名: 'name', 名字: 'name',
    等级: 'levelName', 会员等级: 'levelName', 级别: 'levelName',
    单位: 'org', 公司: 'org',
    部门: 'dept', 科室: 'dept',
    工号: 'jobNo', 员工号: 'jobNo',
  };
  const ORDER = ['phone', 'name', 'levelName', 'org', 'dept', 'jobNo'];
  const maskPhone = p => (p && p.length === 11 ? p.slice(0, 3) + '****' + p.slice(7) : p);

  // 逐字符解析，支持引号包裹与字段内逗号
  function splitLine(line) {
    const out = [];
    let cur = '';
    let q = false;
    for (let i = 0; i < line.length; i++) {
      const ch = line[i];
      if (q) {
        if (ch === '"') {
          if (line[i + 1] === '"') { cur += '"'; i++; } else q = false;
        } else cur += ch;
      } else if (ch === '"') q = true;
      else if (ch === ',') { out.push(cur.trim()); cur = ''; }
      else cur += ch;
    }
    out.push(cur.trim());
    return out;
  }

  function parseCsv(text) {
    const lines = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n').filter(l => l.trim());
    if (!lines.length) return { rows: [], bad: '文件是空的' };
    // 乱码检测：UTF-8 解码失败会产生替换字符
    if (text.indexOf('�') > -1) {
      return { rows: [], bad: '文件编码不是 UTF-8，中文已乱码。请在 Excel 中「另存为 → CSV UTF-8」后重试。' };
    }

    const first = splitLine(lines[0]);
    const hasHeader = first.some(c => ALIAS[c]);
    let map;
    if (hasHeader) map = first.map(c => ALIAS[c] || '');
    else map = ORDER.slice(0, first.length);

    if (map.indexOf('phone') < 0 || map.indexOf('name') < 0 || map.indexOf('levelName') < 0) {
      return { rows: [], bad: '识别不到「手机号 / 姓名 / 等级」三列，请按模板列顺序整理后重试。' };
    }

    const rows = [];
    for (let i = hasHeader ? 1 : 0; i < lines.length; i++) {
      const cells = splitLine(lines[i]);
      const r = { line: i + 1, raw: lines[i].slice(0, 60) };
      map.forEach((k, idx) => { if (k) r[k] = (cells[idx] || '').replace(/\s/g, k === 'phone' ? '' : ' ').trim(); });
      rows.push(r);
    }
    return { rows, bad: '' };
  }

  let state = { step: 'idle', fileName: '', res: null, rows: [], errOpen: true };

  function render(el) {
    state = { step: 'idle', fileName: '', res: null, rows: [], errOpen: true };
    paint(el);
  }

  function paint(el) {
    Api.listLevels().then(levels => {
      el.innerHTML =
        `<div class="page-head">
           <span class="ph-s grow">手机号为唯一键：名单中已存在则更新，不存在则新增。异常行会被跳过，不影响其余数据。</span>
           <button class="btn btn--line" data-back>${I.svg('back', 16)}返回名单</button>
         </div>
         <div id="imp-host"></div>`;

      el.querySelector('[data-back]').onclick = () => window.App.go('members');

      if (state.step === 'idle') paintIdle(el, levels);
      else paintPreview(el);
    });
  }

  function paintIdle(el, levels) {
    el.querySelector('#imp-host').innerHTML =
      `<div class="imp-cols">
         <div>
           <div class="drop big" id="drop">
             ${I.svg('upload', 30, '#8f9384')}
             <div class="drop-t">把 CSV 文件拖到这里</div>
             <div class="faint" style="font-size:12.5px">或点击选择本机文件 · 仅支持 .csv</div>
           </div>
           <input type="file" id="file" accept=".csv,text/csv" hidden>
           <div class="card card-pad set-note" style="margin-top:16px">
             ${I.svg('warn', 16, '#a4873f')}
             <div>Excel 默认另存的 CSV 是 GBK 编码，导入会整片乱码。请用「另存为 → <b>CSV UTF-8</b>」。</div>
           </div>
         </div>

         <div>
           <div class="sec-h">
             <span class="t">模板列</span>
             <span class="more" id="dl">下载模板 CSV ↓</span>
           </div>
           <div class="tbl-wrap">
             <table class="tbl">
               <thead><tr><th style="width:96px">列名</th><th style="width:64px">必填</th><th>说明</th></tr></thead>
               <tbody>
                 ${TPL_COLS.map(c => `<tr>
                   <td><b>${c.k}</b></td>
                   <td>${c.req ? '<span class="pill pill--warn">必填</span>' : '<span class="faint">选填</span>'}</td>
                   <td class="faint">${hint(c.k)}</td>
                 </tr>`).join('')}
               </tbody>
             </table>
           </div>
           <div class="fld-hint" style="margin-top:10px">
             表头支持别名识别（手机／电话／联系电话 等）；无表头时按上表列顺序解析。<br>
             当前可用等级名：<b>${levels.map(l => T.esc(l.name)).join('、')}</b>，等级名必须与之完全一致。<br>
             去重规则：按手机号覆盖更新，覆盖姓名与等级，保留加入时间、累计消费与微信绑定关系。
           </div>
         </div>
       </div>`;

    const file = el.querySelector('#file');
    const drop = el.querySelector('#drop');
    drop.onclick = () => file.click();
    file.onchange = () => { if (file.files[0]) accept(el, file.files[0]); file.value = ''; };

    ['dragenter', 'dragover'].forEach(ev =>
      drop.addEventListener(ev, e => { e.preventDefault(); drop.classList.add('over'); }));
    ['dragleave', 'drop'].forEach(ev =>
      drop.addEventListener(ev, e => { e.preventDefault(); drop.classList.remove('over'); }));
    drop.addEventListener('drop', e => { const f = e.dataTransfer.files[0]; if (f) accept(el, f); });

    el.querySelector('#dl').onclick = downloadTemplate;
  }

  function hint(k) {
    return {
      手机号: '11 位手机号，唯一识别键',
      姓名: '会员真实姓名',
      等级: '必须与后台已有等级名完全一致',
      单位: '所属单位，用于筛选',
      部门: '所属科室',
      工号: '内部编号',
    }[k] || '';
  }

  // PC 端下载模板文件（小程序端只能复制表头到剪贴板）
  function downloadTemplate() {
    const sample = [TPL_HEADER, '13800006620,林建国,中级会员,县前管理处,综合科,XQ0107'].join('\n');
    // BOM 让 Excel 双击打开时按 UTF-8 解析，避免商户拿到模板反而乱码
    const blob = new Blob(['﻿' + sample], { type: 'text/csv;charset=utf-8' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = '会员名单导入模板.csv';
    a.click();
    URL.revokeObjectURL(a.href);
    window.Toast.show('模板已下载', { icon: 'check' });
  }

  function accept(el, f) {
    if (!/\.csv$/i.test(f.name)) return window.Toast.show('请选择 .csv 文件', { icon: 'warn' });
    const reader = new FileReader();
    reader.onload = () => {
      const { rows, bad } = parseCsv(String(reader.result));
      if (bad) return window.Toast.show(bad, { icon: 'warn', duration: 5000 });
      if (!rows.length) return window.Toast.show('没有解析到数据行', { icon: 'warn' });
      Api.previewImport(rows).then(res => {
        state.step = 'preview';
        state.fileName = f.name;
        state.res = res;
        state.rows = res.adds.map(r => Object.assign({}, r, { isNew: true }))
          .concat(res.updates.map(r => Object.assign({}, r, { isNew: false })));
        state.errOpen = true;
        paint(el);
      });
    };
    reader.onerror = () => window.Toast.show('文件读取失败，请重新选择', { icon: 'warn' });
    reader.readAsText(f, 'utf-8');
  }

  function paintPreview(el) {
    const { fileName, res, rows } = state;
    el.querySelector('#imp-host').innerHTML =
      `<div class="imp-file">
         ${I.svg('box', 17, '#2a5fa6')}<b>${T.esc(fileName)}</b>
         <span class="grow"></span>
         <button class="btn btn--sm btn--line" data-reset>重新选择</button>
       </div>

       <div class="grid-3" style="margin:14px 0 18px">
         <div class="card card-pad kpi">
           <span class="kpi-ic" style="color:#467a32;background:#467a3214">${I.svg('plus', 19, '#467a32')}</span>
           <div class="kpi-v tnum">${res.adds.length}</div><div class="kpi-k">新增</div>
         </div>
         <div class="card card-pad kpi">
           <span class="kpi-ic" style="color:#2a5fa6;background:#2a5fa614">${I.svg('refresh', 19, '#2a5fa6')}</span>
           <div class="kpi-v tnum">${res.updates.length}</div><div class="kpi-k">更新</div>
         </div>
         <div class="card card-pad kpi">
           <span class="kpi-ic" style="color:#d2483a;background:#d2483a14">${I.svg('warn', 19, '#d2483a')}</span>
           <div class="kpi-v tnum">${res.errors.length}</div><div class="kpi-k">异常（将跳过）</div>
         </div>
       </div>

       ${res.errors.length ? `
       <div class="err-box">
         <div class="err-head" data-toggle>
           ${I.svg('warn', 16, '#d2483a')}
           <b>${res.errors.length} 行有问题，导入时会跳过</b>
           <span class="grow"></span>
           <span class="faint">${state.errOpen ? '收起' : '展开'}</span>
         </div>
         ${state.errOpen ? `<div class="err-list">
           ${res.errors.map(e => `<div class="err-row">
             <span class="err-line tnum">第 ${e.line} 行</span>
             <span class="err-why">${T.esc(e.reason)}</span>
             <span class="err-raw ellipsis faint">${T.esc(e.raw)}</span>
           </div>`).join('')}
         </div>` : ''}
       </div>` : ''}

       <div class="sec-h" style="margin-top:14px"><span class="t">将写入的 ${rows.length} 条</span></div>
       <div id="pv-host"></div>

       <div class="form-foot">
         <button class="btn btn--line" data-reset2>取消</button>
         <button class="btn btn--primary" data-commit ${rows.length ? '' : 'disabled'}>确认导入 ${rows.length} 条</button>
       </div>`;

    el.querySelector('#pv-host').innerHTML = T.render({
      cols: [
        { t: '行号', w: '64px', render: r => `<span class="faint tnum">${r.line}</span>` },
        { t: '', w: '64px', render: r => r.isNew
          ? '<span class="pill pill--ok">新增</span>' : '<span class="pill pill--info">更新</span>' },
        { t: '姓名', w: '110px', render: r => `<b>${T.esc(r.name)}</b>` },
        { t: '手机号', w: '130px', render: r => `<span class="tnum">${maskPhone(r.phone)}</span>` },
        { t: '等级', w: '110px', render: r => T.esc(r.levelName) },
        { t: '单位 / 部门', render: r => {
          const s = [r.org, r.dept].filter(Boolean).join(' · ');
          return s ? T.esc(s) : '<span class="faint">—</span>';
        } },
        { t: '工号', w: '100px', render: r => r.jobNo ? `<span class="tnum">${T.esc(r.jobNo)}</span>` : '<span class="faint">—</span>' },
      ],
      rows,
      empty: '没有可导入的数据',
    });

    const reset = () => { state.step = 'idle'; paint(el); };
    el.querySelector('[data-reset]').onclick = reset;
    el.querySelector('[data-reset2]').onclick = reset;

    const tg = el.querySelector('[data-toggle]');
    if (tg) tg.onclick = () => { state.errOpen = !state.errOpen; paint(el); };

    el.querySelector('[data-commit]').onclick = () => {
      if (!rows.length) return window.Toast.show('没有可导入的数据', { icon: 'warn' });
      Api.commitImport(rows.map(r => ({
        phone: r.phone, name: r.name, levelId: r.levelId, org: r.org, dept: r.dept, jobNo: r.jobNo,
      }))).then(r => {
        window.Toast.show(`已导入 · 新增 ${r.added} 更新 ${r.updated}`, { icon: 'check' });
        setTimeout(() => window.App.go('members'), 800);
      }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
    };
  }

  window.Pages = window.Pages || {};
  window.Pages['members/import'] = { render };
})();
