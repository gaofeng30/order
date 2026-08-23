/* 工作台：KPI 与销量均为服务端派生读，不在浏览器持久化汇总事实。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  function render(el) {
    el.innerHTML = '<div class="card card-pad faint">正在读取经营数据…</div>';
    Promise.all([Api.dashboardStats(), Api.listOrders('全部')]).then(([stats, orders]) => {
      const live = orders.filter(o => ['已预约', '制作中', '待取餐'].includes(o.status));
      // 待制作数必须使用服务端全量统计；订单列表只是有限页，不能反推 KPI。
      const preparing = Number(stats.pending_production || 0);
      const rank = (stats.product_sales || stats.product_ranking || stats.productRanking || []).map(r => ({
        name: r.product_name || r.name,
        sold: Number(r.quantity || r.sold || 0),
      }));
      const max = Math.max(1, ...rank.map(r => r.sold));
      const kpis = [
        { k: '当日营收', v: '¥' + Api.yuan(stats.today_revenue_cents || 0), ic: 'wallet', c: '#467a32' },
        { k: '当日订单', v: String(stats.today_orders || stats.today_order_count || 0), ic: 'receipt', c: '#2a5fa6' },
        { k: '当月营收', v: '¥' + Api.yuan(stats.month_revenue_cents || 0), ic: 'chart', c: '#a4873f' },
        { k: '当月订单', v: String(stats.month_orders || stats.month_order_count || 0), ic: 'calendar', c: '#5e6354' },
        { k: '退款金额', v: '¥' + Api.yuan(stats.refund_cents || 0), ic: 'bell', c: '#a4873f' },
      ];

      el.innerHTML =
        `<div class="grid-4" style="grid-template-columns:repeat(5,minmax(0,1fr));margin-bottom:18px">
           ${kpis.map(k => `<div class="card card-pad kpi"><span class="kpi-ic" style="color:${k.c};background:${k.c}14">${I.svg(k.ic, 19, k.c)}</span><div class="kpi-v tnum">${k.v}</div><div class="kpi-k">${k.k}</div></div>`).join('')}
         </div>
         <div class="dash-cols">
           <div><div class="sec-h"><span class="t">实时订单</span><span class="more" data-more>全部订单 →</span></div><div id="live-host"></div></div>
           <div>
             <div class="sec-h"><span class="t">今日待办</span></div>
             <div class="card card-pad">${preparing ? `<div class="todo-row" data-lane="制作中"><i class="tdot" style="background:#d2483a"></i><b class="tnum">${preparing}</b><span>单待制作</span><span class="grow"></span>${I.svg('chevron', 15, '#8f9384')}</div>` : '<div class="faint" style="font-size:13px">暂无待办</div>'}</div>
             <div class="sec-h" style="margin-top:8px"><span class="t">销量排行</span></div>
             <div class="card card-pad">${rank.length ? rank.map((r, i) => `<div class="rank-row"><span class="rk-no${i < 3 ? ' top' : ''}">${i + 1}</span><div class="grow"><div class="rk-nm">${T.esc(r.name)}</div><div class="rk-track"><i style="width:${Math.round(r.sold / max * 100)}%"></i></div></div><span class="rk-sold tnum">${r.sold}</span></div>`).join('') : '<div class="faint">暂无销量数据</div>'}</div>
           </div>
         </div>`;
      renderLive(el, live);
      el.querySelector('[data-more]').onclick = () => window.App.go('orders');
      el.querySelectorAll('[data-lane]').forEach(n => { n.onclick = () => { window.__ordersLane = n.dataset.lane; window.App.go('orders'); }; });
    }).catch(e => {
      el.innerHTML = `<div class="card card-pad"><b>经营数据暂不可用</b><div class="faint" style="margin-top:8px">${T.esc(e.message)}</div><button class="btn btn--primary" style="margin-top:16px" data-retry>重试</button></div>`;
      el.querySelector('[data-retry]').onclick = () => render(el);
    });
  }

  function renderLive(el, live) {
    const host = el.querySelector('#live-host');
    host.innerHTML = T.render({
      cols: [
        { t: '取餐号', w: '78px', render: r => `<b class="tnum">${T.esc(r.code)}</b>` },
        { t: '菜品', render: r => `<span class="ellipsis" style="display:block">${T.esc(Api.itemsSummary(r.items))}</span>` },
        { t: '取餐', w: '92px', render: r => `<span class="tnum">${T.esc(r.pickupTime)}</span><br><span class="faint tnum" style="font-size:11.5px">${T.esc((r.pickupDate || '').slice(5))}</span>` },
        { t: '状态', w: '84px', render: r => T.pill(r.status) },
      ], rows: live, empty: '当前没有进行中的订单',
    });
    T.bind(host, {
      row(id) { window.__orderSel = id; window.App.go('orders'); },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages.dashboard = { sub: '今日经营概览与实时接单', render };
})();
