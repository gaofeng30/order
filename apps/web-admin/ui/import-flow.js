/* 批量导入三步流程外壳（PRD §6.13.1）—— 两个导入页共用。
   页面只提供模板定义与契约方法，不接触文件解析。 */
(function () {
  const T = window.Table, I = window.Icon;

  /* cfg: { title, hint, columns:[{name,required,note}], sample,
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
           <div class="fld-lb">第一步 · 模板列</div>
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
             <span class="faint">${state.file ? T.esc(state.file.name) : '只接受 .xlsx，单次最多 ' + window.Api.MAX_IMPORT_ROWS + ' 行'}</span>
           </div>
         </div>
         <div id="imp-result"></div>`;

      el.querySelector('[data-back]').onclick = () => window.App.go(cfg.backRoute);
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

  window.ImportFlow = { render };
})();
