/* Table —— 后台列表的统一渲染。
   PC 端把小程序的卡片流换成表格：同样的数据，信息密度提升约 3 倍。 */
(function () {
  const esc = s => window.Toast.esc(s);

  /* render({
       cols:  [{ t: 表头, k?: 字段名, w?: 列宽, cls?: 'num'|'act', render?: row => html }],
       rows:  对象数组,
       key:   row => 唯一键（写入 tr 的 data-id）
       empty: 空态文案
       rowCls: row => 附加 class
     }) → HTML 字符串 */
  function render(cfg) {
    const cols = cfg.cols;
    const rows = cfg.rows || [];
    const head = cols.map(c =>
      `<th class="${c.cls || ''}"${c.w ? ` style="width:${c.w}"` : ''}>${c.t}</th>`).join('');

    if (!rows.length) {
      return `<div class="tbl-wrap"><table class="tbl"><thead><tr>${head}</tr></thead></table>
              <div class="tbl-empty">${esc(cfg.empty || '暂无数据')}</div></div>`;
    }

    const body = rows.map(r => {
      const id = cfg.key ? cfg.key(r) : (r.id || '');
      const cls = [cfg.rowCls ? cfg.rowCls(r) : '', cfg.onRow ? 'clickable' : ''].filter(Boolean).join(' ');
      const tds = cols.map(c => {
        const v = c.render ? c.render(r) : esc(r[c.k] == null ? '' : r[c.k]);
        return `<td class="${c.cls || ''}">${v}</td>`;
      }).join('');
      return `<tr data-id="${esc(id)}"${cls ? ` class="${cls}"` : ''}>${tds}</tr>`;
    }).join('');

    return `<div class="tbl-wrap"><table class="tbl">
              <thead><tr>${head}</tr></thead><tbody>${body}</tbody></table></div>`;
  }

  /* 行内点击代理：把 tr[data-id] 与 [data-act] 的点击分派给回调。
     data-act 的点击不会冒泡触发行点击（编辑/删除按钮不应同时选中行）。 */
  function bind(root, handlers) {
    root.addEventListener('click', e => {
      const actEl = e.target.closest('[data-act]');
      const tr = e.target.closest('tr[data-id]');
      const id = tr ? tr.dataset.id : (actEl ? actEl.dataset.id : '');
      if (actEl) {
        e.stopPropagation();
        const fn = handlers[actEl.dataset.act];
        if (fn) fn(actEl.dataset.id || id, actEl);
        return;
      }
      if (tr && handlers.row) handlers.row(id, tr);
    });
  }

  // 状态胶囊
  function pill(text, tone) {
    return `<span class="pill pill--${tone || window.Api.statusTone(text)}"><i class="pd"></i>${esc(text)}</span>`;
  }

  // 金额
  function money(n, blue) {
    return `<span class="price${blue ? ' price-blue' : ''}"><span class="cur">¥</span>${n}</span>`;
  }

  window.Table = { render, bind, pill, money, esc };
})();
