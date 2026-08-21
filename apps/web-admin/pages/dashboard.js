/* 工作台 —— 对应 apps/wechat-miniprogram/pages/admin-dashboard
   PC 形态：KPI 四张卡一行铺开 + 左「实时订单」右「待办 / 销量排行」，一屏不滚。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  // KPI 口径与小程序端一致（当前为固定演示值，正式期由 GET /admin/stats 下发）
  const KPI = [
    { k: '当日营收', v: '¥1,286', ic: 'wallet', c: '#467a32' },
    { k: '当日订单', v: '38', ic: 'receipt', c: '#2a5fa6' },
    { k: '当月营收', v: '¥32,840', ic: 'chart', c: '#a4873f' },
    { k: '当月订单', v: '906', ic: 'calendar', c: '#5e6354' },
  ];

  function build() {
    const orders = window.__store.aOrders;
    const pend = orders.filter(o => o.status === '制作中').length;

    const todos = [];
    if (pend) todos.push({ n: pend, label: '单待制作', dot: '#d2483a', to: '制作中' });

    const live = orders.filter(o => ['已预约', '制作中', '待取餐'].includes(o.status));

    const max = window.Seed.RANK[0].sold;
    const rank = window.Seed.RANK.map(r => {
      const m = window.Seed.itemById(r.id);
      return { id: r.id, name: m ? m.name : r.id, sold: r.sold, pct: Math.round(r.sold / max * 100) };
    });

    return { todos, live, rank };
  }

  function render(el) {
    const { todos, live, rank } = build();

    el.innerHTML =
      `<div class="grid-4" style="margin-bottom:18px">
         ${KPI.map(k => `
           <div class="card card-pad kpi">
             <span class="kpi-ic" style="color:${k.c};background:${k.c}14">${I.svg(k.ic, 19, k.c)}</span>
             <div class="kpi-v tnum">${k.v}</div>
             <div class="kpi-k">${k.k}</div>
           </div>`).join('')}
       </div>

       <div class="dash-cols">
         <div>
           <div class="sec-h">
             <span class="t">实时订单</span>
             <span class="more" data-more>全部订单 →</span>
           </div>
           <div id="live-host"></div>
         </div>

         <div>
           <div class="sec-h"><span class="t">今日待办</span></div>
           <div class="card card-pad">
             ${todos.length ? todos.map(t => `
               <div class="todo-row"${t.to ? ` data-lane="${t.to}"` : ''}>
                 <i class="tdot" style="background:${t.dot}"></i>
                 <b class="tnum">${t.n}</b><span>${t.label}</span>
                 ${t.to ? `<span class="grow"></span>${I.svg('chevron', 15, '#8f9384')}` : ''}
               </div>`).join('') : '<div class="faint" style="font-size:13px">暂无待办</div>'}
           </div>

           <div class="sec-h" style="margin-top:8px"><span class="t">销量排行</span></div>
           <div class="card card-pad">
             ${rank.map((r, i) => `
               <div class="rank-row" data-rank="${r.name}">
                 <span class="rk-no${i < 3 ? ' top' : ''}">${i + 1}</span>
                 <div class="grow">
                   <div class="rk-nm">${T.esc(r.name)}</div>
                   <div class="rk-track"><i style="width:${r.pct}%"></i></div>
                 </div>
                 <span class="rk-sold tnum">${r.sold}</span>
               </div>`).join('')}
           </div>
         </div>
       </div>`;

    renderLive(el, live);

    el.querySelector('[data-more]').onclick = () => window.App.go('orders');
    el.querySelectorAll('[data-lane]').forEach(n => {
      n.onclick = () => { window.__ordersLane = n.dataset.lane; window.App.go('orders'); };
    });
    el.querySelectorAll('[data-rank]').forEach(n => {
      n.onclick = () => window.Toast.show(`「${n.dataset.rank}」明细 · 建设中`, { icon: 'chart' });
    });
  }

  function renderLive(el, live) {
    const host = el.querySelector('#live-host');
    host.innerHTML = T.render({
      cols: [
        { t: '取餐号', w: '78px', render: r => `<b class="tnum">${r.code}</b>` },
        { t: '菜品', render: r => `<span class="ellipsis" style="display:block">${T.esc(Api.itemsSummary(r.items))}</span>` },
        { t: '取餐', w: '92px', render: r => `<span class="tnum">${r.pickupTime}</span><br><span class="faint tnum" style="font-size:11.5px">${r.pickupDate.slice(5)}</span>` },
        { t: '状态', w: '84px', render: r => T.pill(r.status) },
        { t: '', w: '88px', cls: 'act', render: r => {
          const m = Api.advanceMeta(r.status);
          return `<button class="btn btn--sm ${m.cls}" data-act="adv" data-id="${r.id}">${m.label}</button>`;
        } },
      ],
      rows: live,
      empty: '当前没有进行中的订单',
    });

    T.bind(host, {
      adv(id) {
        // 用本次渲染的容器，不要去抓 #content：切页后那已是另一棵树（见 app.js go()）
        Api.advanceOrder(id).then(r => {
          render(el);
          window.Toast.show(`已${r.act}「${r.code}」`, { icon: 'check' });
        }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
      },
      row(id) { window.__orderSel = id; window.App.go('orders'); },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['dashboard'] = { sub: '今日经营概览与实时接单', render };
})();
