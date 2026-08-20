/* 扫码核销 —— 对应 apps/wechat-miniprogram/pages/admin-verify
   PC 形态：大号取餐号输入框自动聚焦，回车提交。
   USB 扫码枪在系统里就是键盘输入设备（扫完自动补一个回车），
   因此这个输入框天然兼容扫码枪，无需额外驱动。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  const SIMS = ['A118', 'A126', 'A090'];
  let match = null;   // { o, err, rows }

  function render(el) {
    match = null;
    el.innerHTML =
      `<div class="vf-wrap">
         <div class="card vf-card">
           <div class="vf-scan">${I.svg('scan', 34, '#2a5fa6')}</div>
           <div class="vf-tip">等待扫描 · 取餐号</div>
           <input class="vf-inp tnum" id="vf-inp" placeholder="A118" maxlength="12" autocomplete="off">
           <button class="btn btn--blue btn--block" id="vf-go" style="margin-top:14px">核对订单</button>
           <div class="vf-sim">
             <span class="faint">演示用取餐号</span>
             ${SIMS.map(c => `<span class="chip blue" data-sim="${c}">${c}</span>`).join('')}
           </div>
         </div>
         <div id="vf-res"></div>
       </div>`;

    const inp = el.querySelector('#vf-inp');
    inp.focus();
    inp.addEventListener('input', () => { inp.value = inp.value.toUpperCase(); });
    inp.addEventListener('keydown', e => { if (e.key === 'Enter') tryVerify(el, inp.value); });
    el.querySelector('#vf-go').onclick = () => tryVerify(el, inp.value);
    el.querySelectorAll('[data-sim]').forEach(n => {
      n.onclick = () => { inp.value = n.dataset.sim; tryVerify(el, n.dataset.sim); };
    });
  }

  function tryVerify(el, code) {
    const c = (code || '').trim();
    if (!c) return;
    const o = Api.findOrderByCode(c);
    if (!o) {
      match = null;
      el.querySelector('#vf-res').innerHTML = '';
      window.Toast.show(`无效取餐号「${c}」`, { icon: 'warn' });
      return;
    }
    let err = '';
    if (o.status === '已完成') err = '该订单已核销';
    else if (o.status === '已退款') err = '该订单已退款，不可核销';
    else if (o.status !== '待取餐') err = '订单尚未备好';

    const rows = o.items.map(([id, q, p]) => {
      const m = window.Seed.itemById(id);
      return { name: m ? m.name : '已删除菜品', q, p, sub: p * q };
    });
    match = { o, err, rows };
    paintResult(el);
  }

  function paintResult(el) {
    const { o, err, rows } = match;
    el.querySelector('#vf-res').innerHTML =
      `<div class="card vf-detail fade-up">
         <div class="vf-dh">
           <div>
             <div class="vf-dcode tnum">${o.code}</div>
             <div class="faint tnum" style="font-size:12.5px">${o.no}</div>
           </div>
           <span class="grow"></span>
           ${T.pill(o.status)}
         </div>
         ${err ? `<div class="vf-err">${I.svg('warn', 16)}${err}</div>` : ''}
         <div class="vf-items">
           ${rows.map(r => `
             <div class="dt-item">
               <span class="grow">${T.esc(r.name)}</span>
               <span class="faint tnum">×${r.q}</span>
               <span class="tnum" style="width:56px;text-align:right">${T.money(r.sub)}</span>
             </div>`).join('')}
           <div class="dt-total"><span class="grow">合计</span>${T.money(o.total)}</div>
         </div>
         <div class="kv"><span class="k">联系人</span><span class="v">${T.esc(o.contact)} · <span class="tnum">${o.phone}</span></span></div>
         <div class="kv"><span class="k">口味备注</span><span class="v">${T.esc([o.flavor !== '—' ? o.flavor : '', o.note].filter(Boolean).join(' · ') || '—')}</span></div>
         <div class="vf-act">
           <button class="btn btn--line" data-cancel>取消</button>
           <button class="btn btn--primary" data-ok ${err ? 'disabled' : ''}>确认核销</button>
         </div>
       </div>`;

    el.querySelector('[data-cancel]').onclick = () => {
      match = null;
      el.querySelector('#vf-res').innerHTML = '';
      el.querySelector('#vf-inp').focus();
    };
    const ok = el.querySelector('[data-ok]');
    if (!err) ok.onclick = () => {
      // 核销 = 待取餐 → 已完成，与订单页的推进走同一条状态机
      Api.advanceOrder(o.id).then(() => {
        window.Toast.show('核销成功 · 看板营收/订单已更新', { icon: 'check' });
        match = null;
        el.querySelector('#vf-res').innerHTML = '';
        const inp = el.querySelector('#vf-inp');
        inp.value = '';
        inp.focus();
      }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
    };
  }

  window.Pages = window.Pages || {};
  window.Pages['verify'] = { sub: '扫码枪扫描取餐码，或手动输入取餐号后回车', render };
})();
