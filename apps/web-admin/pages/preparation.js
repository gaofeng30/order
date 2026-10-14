/* 备餐管理：只汇总未备好订单，状态变更复用订单备好命令。 */
(function () {
  const Api = window.Api, T = window.Table;
  const keys = new Map();
  let activeBatch = false;
  function render(el) {
    let date = Api.today(), meal = 'lunch', orders = [], selected = new Set(), generation = 0;
    let loading = false, error = '', result = null;
    el.innerHTML = `<div class="toolbar prep-toolbar">
      <label>取餐日期 <input class="inp" type="date" data-date value="${T.esc(date)}"></label>
      <label>餐段 <select class="inp" data-meal><option value="lunch">午餐</option><option value="dinner">晚餐</option></select></label>
      <button class="btn btn--line" data-refresh>刷新</button>
      <span class="faint" data-progress role="status"></span></div>
      <p class="faint prep-note">汇总已预约、制作中的未备好订单，不含待取餐、已完成和退款订单。仅制作中订单可标记备好；请确认整份餐已装齐。</p>
      <div data-prep-content></div>`;
    const content = el.querySelector('[data-prep-content]');
    const current = () => orders.filter(o => String(o.mealPeriod).toLowerCase() === meal);
    const eligible = o => o.status === '制作中' && Array.isArray(o.available_actions) && o.available_actions.includes('MARK_READY');
    const lock = () => {
      el.querySelector('[data-date]').disabled = activeBatch;
      el.querySelector('[data-meal]').disabled = activeBatch;
      el.querySelector('[data-refresh]').disabled = loading || activeBatch;
    };
    function paint() {
      lock();
      if (loading) { content.innerHTML = '<div class="card card-pad" role="status">正在读取订单与菜品汇总…</div>'; return; }
      if (error) { content.innerHTML = `<div class="card card-pad" role="alert">${T.esc(error)}，请点击刷新重试。</div>`; return; }
      const list = current().sort((a, b) => a.pickupTime.localeCompare(b.pickupTime) || a.id.localeCompare(b.id));
      const ready = list.filter(eligible);
      const summaries = new Map();
      for (const order of list) for (const item of order.items) {
        const label = item.name || item.product_name;
        const detail = [...(item.flavors || []), item.note].filter(Boolean).join(' · ');
        const key = JSON.stringify([item.product_id, label, detail]);
        const entry = summaries.get(key) || { name: label, detail, quantity: 0 };
        entry.quantity += item.quantity;
        summaries.set(key, entry);
      }
      const disabled = activeBatch ? 'disabled' : '';
      content.innerHTML = `${result ? `<div class="card card-pad prep-result" role="status">${T.esc(result)}</div>` : ''}
        <section class="card card-pad prep-summary"><h3>待备菜品汇总 <span class="faint">${list.length} 笔订单</span></h3>
        <p class="faint">数量来自整单尚未备好的订单，不代表每道菜的实际制作进度；整单备注请在下方核对。</p>
        <div class="prep-summary-grid">${[...summaries.values()].map(s => `<div class="prep-summary-item"><div><b>${T.esc(s.name)}</b>${s.detail ? `<div class="faint">${T.esc(s.detail)}</div>` : ''}</div><b class="tnum">${s.quantity} 份</b></div>`).join('') || '<span class="faint">该餐段暂无待备菜品</span>'}</div></section>
        <div class="toolbar prep-actions"><label class="select-all"><input type="checkbox" data-all ${ready.length && ready.every(o => selected.has(o.id)) ? 'checked' : ''} ${!ready.length || activeBatch ? 'disabled' : ''}>全选可备好订单</label>
        <span>已选 ${selected.size} 笔 · 可备好 ${ready.length} 笔</span><span class="sp"></span>
        <button class="btn btn--line" data-selected ${!selected.size || activeBatch ? 'disabled' : ''}>已选订单备好</button>
        <button class="btn btn--primary" data-entire ${!ready.length || activeBatch ? 'disabled' : ''}>本餐段全部备好</button></div>
        <p class="faint prep-note">已预约订单计入汇总，距取餐时间 30 分钟内进入制作中后才可备好。全部备好仅处理当前已加载且符合条件的订单。</p>
        <div class="tbl-wrap"><table class="tbl prep-table"><thead><tr><th>选择</th><th>取餐号</th><th>菜品与备注</th><th>取餐时间</th><th>状态</th></tr></thead><tbody>${list.map(o => `<tr>
        <td><input type="checkbox" data-pick="${o.id}" aria-label="选择取餐号 ${T.esc(o.code)}" ${selected.has(o.id) ? 'checked' : ''} ${!eligible(o) ? 'disabled' : disabled}></td>
        <td><b class="tnum">${T.esc(o.code)}</b></td><td>${o.items.map(i => `<div>${T.esc(i.name || i.product_name)} × ${i.quantity}<span class="faint"> ${T.esc([...(i.flavors || []), i.note].filter(Boolean).join(' · '))}</span></div>`).join('')}${o.orderNote ? `<div class="ord-band">整单备注：${T.esc(o.orderNote)}</div>` : ''}</td>
        <td class="tnum">${T.esc(o.pickupTime)}</td><td>${T.pill(o.status)}</td></tr>`).join('') || '<tr><td colspan="5" class="tbl-empty">该日期、餐段暂无未备好订单</td></tr>'}</tbody></table></div>`;
      const all = content.querySelector('[data-all]');
      all.indeterminate = selected.size > 0 && !ready.every(o => selected.has(o.id));
      all.onchange = () => { selected = new Set(all.checked ? ready.map(o => o.id) : []); paint(); };
      content.querySelectorAll('[data-pick]').forEach(n => { n.onchange = () => { if(n.checked) selected.add(n.dataset.pick); else selected.delete(n.dataset.pick); paint(); }; });
      content.querySelector('[data-selected]').onclick = () => submit(ready.filter(o => selected.has(o.id)));
      content.querySelector('[data-entire]').onclick = () => submit(ready);
    }
    async function load() {
      if (activeBatch) { paint(); return; }
      const request = ++generation;
      selected.clear(); orders = []; error = ''; loading = true; paint();
      try {
        if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) throw new Error('请选择取餐日期');
        const values = await Api.preparationOrders(date);
        if (request !== generation || !el.isConnected) return;
        orders = values;
      } catch (e) { if(request === generation) error = e.message; }
      finally { if(request === generation && el.isConnected) { loading = false; paint(); } }
    }
    async function submit(batch) {
      if (activeBatch || !batch.length) return;
      const captured = generation, capturedDate = date, capturedMeal = meal;
      activeBatch = true; paint();
      const yes = await window.Modal.confirm({ title: '确认批量备好', okText: `确认备好 ${batch.length} 笔`,
        body: `<p>${T.esc(capturedDate)} · ${Api.MEAL_LABEL[capturedMeal]}，共 <b>${batch.length}</b> 笔订单。</p><p>请确认这些订单的全部菜品已装齐。成功后订单进入待取餐，沿用现有取餐凭证与通知流程。</p>` });
      if (!yes || !el.isConnected || captured !== generation) { activeBatch = false; if(el.isConnected) paint(); return; }
      let succeeded = 0;
      const failures = [];
      try {
        // 按明确 ID 顺序提交，不扫描新订单，不同时打出大量写请求。
        for (const order of batch) {
          if (!keys.has(order.id)) keys.set(order.id, Api.idempotencyKey());
          try { await Api.markOrderReady(order.id, keys.get(order.id)); succeeded++; selected.delete(order.id); }
          catch (e) {
            failures.push({ code: order.code, message: e.code === 'NETWORK_UNAVAILABLE' ? '连接中断，结果待确认；刷新后核对' : e.message });
            if (e.status === 401 || e.status === 403) {
              const remaining = batch.length - succeeded - failures.length;
              if (remaining) failures.push({ code: '其余 ' + remaining + ' 笔', message: '权限或登录状态变化，未继续提交' });
              break;
            }
          }
          if (el.isConnected) el.querySelector('[data-progress]').textContent = `正在处理 ${succeeded + failures.length} / ${batch.length} 笔`;
        }
      } finally { activeBatch = false; }
      result = `本次 ${batch.length} 笔：已备好 ${succeeded} 笔，其余 ${batch.length - succeeded} 笔未确认成功。`;
      if (el.isConnected) { el.querySelector('[data-progress]').textContent = ''; await load(); }
      if (failures.length) window.Modal.open({ title: '部分订单未确认备好', bodyHtml: `<p>${T.esc(result)}</p><ul>${failures.map(f => `<li>${T.esc(f.code)}：${T.esc(f.message)}</li>`).join('')}</ul>`, footerHtml:'<button class="btn btn--line" data-close>关闭</button>', onMount(root, close) { root.querySelector('[data-close]').onclick = close; } });
      else window.Toast.show(`已备好 ${succeeded} 笔订单`, { icon: 'check' });
    }
    el.querySelector('[data-date]').onchange = e => { date = e.target.value; result = null; load(); };
    el.querySelector('[data-meal]').onchange = e => { meal = e.target.value; selected.clear(); result = null; paint(); };
    el.querySelector('[data-refresh]').onclick = () => load();
    load();
  }
  window.Pages['preparation'] = { sub: '按日期、餐段汇总菜品与批量备好订单', render };
})();
