/* 备餐管理：未备好需求汇总；会话内保留筛选、批量进度与结果。 */
(function () {
  const Api = window.Api, T = window.Table;
  const keys = new Map();
  let filters = null, batch = null, confirming = false, epoch = 0, view = null;
  const busy = () => confirming || !!(batch && !batch.done);
  function notify() { if (view) view.update(); }
  window.addEventListener('pc-session-expired', () => {
    epoch++; filters = null; batch = null; confirming = false; keys.clear();
    window.__preparationDate = null;
  });

  async function runBatch(job) {
    for (let index = 0; index < job.orders.length; index++) {
      if (job.epoch !== epoch) return;
      const order = job.orders[index];
      if (!keys.has(order.id)) keys.set(order.id, Api.idempotencyKey());
      try { await Api.markOrderReady(order.id, keys.get(order.id)); job.succeeded++; }
      catch (e) {
        if (job.epoch !== epoch) return;
        job.failures.push({ order, message: e.code === 'NETWORK_UNAVAILABLE' ? '连接中断，结果待确认；请核对订单状态' : e.message });
        if (e.status === 403 || e.status === 401) {
          job.unprocessed = job.orders.slice(index + 1).map(order => ({ order, message: '权限或登录状态变化，未提交' }));
          break;
        }
      }
      job.processed = index + 1; notify();
    }
    if (job.epoch !== epoch) return;
    job.processed = job.succeeded + job.failures.length;
    job.done = true; notify();
  }

  function render(el) {
    filters = filters || { date: Api.today(), meal: 'lunch' };
    if (window.__preparationDate) { filters.date = window.__preparationDate; window.__preparationDate = null; }
    if (batch && !batch.done) filters = { date: batch.date, meal: batch.meal };
    let date = filters.date, meal = filters.meal, orders = [], selected = new Set();
    let generation = 0, loading = false, error = '', updatedAt = '', refreshedJob = batch && batch.done ? batch : null;
    const localEpoch = epoch;
    el.innerHTML = `<div class="toolbar prep-toolbar">
      <label>取餐日期 <input class="inp" type="date" data-date value="${T.esc(date)}"></label>
      <label>餐段 <select class="inp" data-meal><option value="lunch">午餐</option><option value="dinner">晚餐</option></select></label>
      <button class="btn btn--line" data-refresh>刷新</button>
      <span class="faint" data-updated role="status">尚未更新</span></div>
      <p class="faint prep-note">汇总已预约、制作中的未备好订单，不含待取餐、已完成和退款订单。仅制作中订单可标记备好；请确认整份餐已装齐。</p>
      <div data-result></div><div data-prep-content></div>`;
    el.querySelector('[data-meal]').value = meal;
    const content = el.querySelector('[data-prep-content]');
    const connected = () => el.isConnected && localEpoch === epoch;
    const current = () => orders.filter(o => String(o.mealPeriod).toLowerCase() === meal);
    const eligible = o => o.status === '制作中' && Array.isArray(o.available_actions) && o.available_actions.includes('MARK_READY');
    function resultPanel() {
      const host = el.querySelector('[data-result]');
      if (!batch) { host.innerHTML = ''; return; }
      const job = batch;
      const issues = job.failures.concat(job.unprocessed);
      host.innerHTML = `<section class="card card-pad prep-result" aria-live="polite">
        <b>${T.esc(job.date)} · ${Api.MEAL_LABEL[job.meal]} · 本批 ${job.orders.length} 笔</b>
        <p>${job.done ? '处理结束' : `正在处理 ${job.processed} / ${job.orders.length} 笔`}：已备好 ${job.succeeded} 笔，未确认成功 ${job.failures.length} 笔，${job.done ? '未处理' : '尚未处理'} ${job.orders.length - job.succeeded - job.failures.length} 笔。</p>
        ${issues.length ? `<ul>${issues.map((f, i) => `<li>取餐号 ${T.esc(f.order.code)}：${T.esc(f.message)} <button class="btn btn--sm btn--line" data-locate="${i}" ${busy() ? 'disabled' : ''}>定位订单</button></li>`).join('')}</ul>` : ''}
      </section>`;
      host.querySelectorAll('[data-locate]').forEach(button => { button.onclick = () => locate(issues[Number(button.dataset.locate)].order, job); });
    }
    async function locate(order, job) {
      if (busy()) return;
      if (date !== job.date || meal !== job.meal) {
        date = job.date; meal = job.meal; filters = { date, meal };
        el.querySelector('[data-date]').value = date; el.querySelector('[data-meal]').value = meal;
        await load();
      }
      if (!connected()) return;
      const row = [...content.querySelectorAll('[data-order]')].find(n => n.dataset.order === order.id);
      if (row) {
        content.querySelectorAll('.prep-located').forEach(n => n.classList.remove('prep-located'));
        row.classList.add('prep-located'); row.scrollIntoView({ block: 'center' }); row.focus({ preventScroll: true });
      } else {
        // 订单可能已成功、退款或被另一设备处理，按订单号跨状态核对。
        window.__orderTarget = { id: order.id, no: order.no, date: order.pickupDate, status: '全部' };
        window.App.go('orders');
      }
    }
    function paint() {
      const locked = busy();
      el.querySelector('[data-date]').disabled = locked;
      el.querySelector('[data-meal]').disabled = locked;
      el.querySelector('[data-refresh]').disabled = loading || locked;
      el.querySelector('[data-updated]').textContent = updatedAt ? `更新于 ${updatedAt}${error ? ' · 本次刷新失败' : ''}` : '尚未更新';
      resultPanel();
      if (loading) { content.innerHTML = '<div class="card card-pad" role="status">正在读取订单与菜品汇总…</div>'; return; }
      if (error) { content.innerHTML = `<div class="card card-pad" role="alert">${T.esc(error)}，请点击刷新重试。</div>`; return; }
      const list = current().sort((a, b) => a.pickupTime.localeCompare(b.pickupTime) || a.id.localeCompare(b.id));
      const ready = list.filter(eligible), summaries = new Map(), groups = new Map();
      for (const order of list) {
        if (!groups.has(order.pickupTime)) groups.set(order.pickupTime, []);
        groups.get(order.pickupTime).push(order);
        for (const item of order.items) {
          const name = item.name || item.product_name, key = JSON.stringify([item.product_id, name]);
          const entry = summaries.get(key) || { name, quantity: 0 };
          entry.quantity += item.quantity; summaries.set(key, entry);
        }
      }
      function row(o) {
        return `<tr data-order="${o.id}" tabindex="-1"><td><input type="checkbox" data-pick="${o.id}" aria-label="选择取餐号 ${T.esc(o.code)}" ${selected.has(o.id) ? 'checked' : ''} ${!eligible(o) || locked ? 'disabled' : ''}></td>
          <td><b class="tnum">${T.esc(o.code)}</b></td><td>${o.items.map(i => `<div>${T.esc(i.name || i.product_name)} × ${i.quantity}<span class="prep-remark"> ${T.esc([...(i.flavors || []), i.note].filter(Boolean).join(' · '))}</span></div>`).join('')}${o.orderNote ? `<div class="ord-band prep-remark">整单备注：${T.esc(o.orderNote)}</div>` : ''}</td>
          <td class="tnum">${T.esc(o.pickupTime)}</td><td>${T.pill(o.status)}</td></tr>`;
      }
      content.innerHTML = `<section class="card card-pad prep-summary"><h3>未备好订单 · 菜品汇总 <span class="faint">${list.length} 笔订单</span></h3>
        <p class="faint">数量来自整单尚未备好的订单，不代表每道菜的实际制作进度；备注请在下方核对。</p>
        <div class="prep-summary-grid">${[...summaries.values()].map(s => `<div class="prep-summary-item"><b>${T.esc(s.name)}</b><b class="tnum">${s.quantity} 份</b></div>`).join('') || '<span class="faint">该餐段暂无待备菜品</span>'}</div></section>
        <div class="toolbar prep-actions"><label class="select-all"><input type="checkbox" data-all ${ready.length && ready.every(o => selected.has(o.id)) ? 'checked' : ''} ${!ready.length || locked ? 'disabled' : ''}>全选可备好订单</label>
        <span>已选 ${selected.size} 笔 · 可备好 ${ready.length} 笔</span><span class="sp"></span>
        <button class="btn btn--line" data-selected ${!selected.size || locked ? 'disabled' : ''}>已选订单备好</button>
        <button class="btn btn--primary" data-entire ${!ready.length || locked ? 'disabled' : ''}>本餐段全部备好</button></div>
        <p class="faint prep-note">已预约订单计入汇总，距取餐时间 30 分钟内进入制作中后才可备好。全部备好仅处理当前已加载且符合条件的订单。</p>
        <div class="tbl-wrap"><table class="tbl prep-table"><thead><tr><th>选择</th><th>取餐号</th><th>菜品与备注</th><th>取餐时间</th><th>状态</th></tr></thead>
        ${[...groups.entries()].map(([time, group]) => `<tbody><tr class="prep-time-group"><th colspan="5" scope="rowgroup">${T.esc(time)} 取餐 · ${group.length} 笔</th></tr>${group.map(row).join('')}</tbody>`).join('') || '<tbody><tr><td colspan="5" class="tbl-empty">该日期、餐段暂无未备好订单</td></tr></tbody>'}</table></div>`;
      const all = content.querySelector('[data-all]');
      all.indeterminate = selected.size > 0 && !ready.every(o => selected.has(o.id));
      all.onchange = () => { selected = new Set(all.checked ? ready.map(o => o.id) : []); paint(); };
      content.querySelectorAll('[data-pick]').forEach(n => { n.onchange = () => { if(n.checked) selected.add(n.dataset.pick); else selected.delete(n.dataset.pick); paint(); }; });
      content.querySelector('[data-selected]').onclick = () => submit(ready.filter(o => selected.has(o.id)));
      content.querySelector('[data-entire]').onclick = () => submit(ready);
    }
    async function load() {
      const request = ++generation;
      selected.clear(); orders = []; error = ''; loading = true; paint();
      try {
        if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) throw new Error('请选择取餐日期');
        const values = await Api.preparationOrders(date);
        if (request !== generation || !connected()) return;
        orders = values; updatedAt = new Date().toLocaleString('zh-CN', { hour12: false });
      } catch (e) { if(request === generation) error = e.message; }
      finally { if(request === generation && connected()) { loading = false; paint(); } }
    }
    async function submit(ordersToReady) {
      if (busy() || !ordersToReady.length) return;
      confirming = true; paint();
      const yes = await window.Modal.confirm({ title: '确认批量备好', okText: `确认备好 ${ordersToReady.length} 笔`,
        body: `<p>${T.esc(date)} · ${Api.MEAL_LABEL[meal]}，共 <b>${ordersToReady.length}</b> 笔订单。</p><p>请确认这些订单的全部菜品已装齐。成功后订单进入待取餐，沿用现有取餐凭证与通知流程。</p>` });
      if (localEpoch !== epoch) return;
      confirming = false;
      if (!yes || !connected()) { notify(); return; }
      batch = { date, meal, orders: ordersToReady.slice(), succeeded: 0, processed: 0, failures: [], unprocessed: [], done: false, epoch };
      selected.clear(); notify();
      await runBatch(batch);
    }
    view = { update() {
      if (!connected()) return;
      if (batch && batch.done && batch !== refreshedJob) { refreshedJob = batch; load(); }
      else if (batch && !batch.done) resultPanel();
      else paint();
    } };
    el.querySelector('[data-date]').onchange = e => { date = e.target.value; filters = { date, meal }; updatedAt = ''; load(); };
    el.querySelector('[data-meal]').onchange = e => { meal = e.target.value; filters = { date, meal }; selected.clear(); paint(); };
    el.querySelector('[data-refresh]').onclick = () => load();
    load();
  }
  window.Pages['preparation'] = { sub: '按日期、餐段汇总菜品与批量备好订单', render, dispose() { view = null; } };
})();
