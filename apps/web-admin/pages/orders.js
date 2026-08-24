/* 订单管理 —— 对应 apps/wechat-miniprogram/pages/admin-orders + admin-order-detail
   PC 形态：左侧订单表格 + 右侧常驻详情面板，选中即显示，不跳页。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let lane = '已预约';
  let selId = '';
  /* §6.7 的「未取餐」是查询口径不是状态：营业日已过仍在 待取餐 的单。
     因此它做成一个可叠加在 待取餐 泳道上的开关，不进 LANES。 */
  let uncollected = false;
  /* PC 扫码核销页已删（§15.5.3，扫码在手机上做）。手工核销的定位并到这里：
     按取餐号 / 订单号 / 手机号搜索，跨泳道 —— 要核销时并不知道单子在哪个状态。 */
  let kw = '';
  let date = '';

  function render(el) {
    // 由工作台带过来的泳道 / 选中项
    if (window.__ordersLane) { lane = window.__ordersLane; window.__ordersLane = ''; }
    if (window.__orderSel) { selId = window.__orderSel; window.__orderSel = ''; }

    el.innerHTML =
      `<div class="split-l">
         <div class="toolbar">
           <div class="segs" id="lanes"></div>
           <span class="sp"></span>
           <span id="unc-host"></span>
           <input class="inp" id="f-date" type="date" value="${T.esc(date)}" aria-label="营业日期">
           <input class="inp" id="f-kw" placeholder="取餐号 / 订单号 / 手机号" value="${T.esc(kw)}" style="width:210px">
         </div>
         <div id="tbl-host"></div>
       </div>
       <aside class="split-r" id="detail"></aside>`;

    const search = el.querySelector('#f-kw');
    search.oninput = () => { kw = search.value; paint(el); };
    const dateInput = el.querySelector('#f-date');
    if (dateInput) dateInput.onchange = () => { date = dateInput.value; paint(el); };
    paint(el);
  }

  function paint(el) {
    const counts = Api.laneCounts();
    paintUncollected(el);
    paintHint(el);
    el.querySelector('#lanes').innerHTML = Api.LANES.map(l =>
      `<span class="seg${l === lane ? ' on' : ''}" data-lane="${l}">${l}<span class="cnt">${counts[l]}</span></span>`
    ).join('');
    el.querySelectorAll('[data-lane]').forEach(n => {
      /* 点泳道即退出搜索态：两者都决定列表内容，同时生效会让人不知道自己在看什么 */
      n.onclick = () => {
        lane = n.dataset.lane;
        if (lane !== '待取餐') uncollected = false;
        kw = '';
        const box = el.querySelector('#f-kw');
        if (box) box.value = '';
        paint(el);
      };
    });

    const q = kw.trim();
    (q ? Api.searchOrders(q, { date }) : Api.listOrders(lane, { uncollected, date })).then(list => {
      // 选中项不在当前泳道时，回落到本泳道第一条
      if (!list.some(o => o.id === selId)) selId = list.length ? list[0].id : '';

      const host = el.querySelector('#tbl-host');
      host.innerHTML = T.render({
        // 左表只做可扫读的索引：订单号 / 联系人 / 金额 都在右侧详情面板里，
        // 分栏布局的意义就在于不重复展示同一份信息。
        cols: [
          { t: '取餐号', w: '72px', render: r => `<b class="tnum">${r.code}</b>` },
          { t: '菜品', render: r => {
            const band = [...new Set(r.items.flatMap(it => [it[5], it[6]]).filter(Boolean)), r.orderNote]
              .filter(Boolean).join(' · ');
            return `<div class="ellipsis">${T.esc(Api.itemsSummary(r.items))}</div>` +
                   (band ? `<div class="ord-band ellipsis">${T.esc(band)}</div>` : '');
          } },
          { t: '取餐', w: '92px', render: r => `<span class="tnum">${r.pickupTime}</span><br><span class="faint tnum" style="font-size:11.5px">${r.pickupDate.slice(5)}</span>` },
          { t: '状态', w: '84px', render: r => T.pill(r.status) },
        ],
        rows: list,
        empty: kw.trim() ? `没有匹配「${T.esc(kw.trim())}」的订单` : `「${lane}」暂无订单`,
        rowCls: r => (r.id === selId ? 'sel' : ''),
        onRow: true,
      });

      T.bind(host, {
        row(id) { selId = id; paint(el); },
      });

      detail(el, selId);
    });
  }

  /* 跨营业日取餐号的提示。搜不到就以为没有，是 §6.6 那条规则最容易造成的误判 ——
     单子明明在，只是在别的营业日，而取餐号按营业日重复使用。 */
  function paintHint(el) {
    let host = el.querySelector('#kw-hint');
    const msg = kw.trim() ? Api.codeHint(kw) : '';
    if (!msg) { if (host) host.remove(); return; }
    if (!host) {
      el.querySelector('.toolbar').insertAdjacentHTML('afterend',
        `<div class="card card-pad imp-note warn" id="kw-hint" style="margin-bottom:12px"></div>`);
      host = el.querySelector('#kw-hint');
    }
    host.textContent = msg;
  }

  /* 「未取餐」筛选。只在 待取餐 泳道下有意义 —— 其余状态谈不上"未取"。 */
  function paintUncollected(el) {
    const host = el.querySelector('#unc-host');
    const n = Api.uncollectedCount();
    if (lane !== '待取餐') {
      host.innerHTML = `<span class="faint" style="font-size:12.5px">选中左侧订单，右侧显示完整单据</span>`;
      return;
    }
    host.innerHTML =
      `<span class="seg${uncollected ? ' on' : ''}" data-unc>未取餐<span class="cnt">${n}</span></span>
       <span class="faint" style="font-size:12.5px;margin-left:10px">营业日已结束仍未核销的单，可事后核销或发起退款</span>`;
    const btn = host.querySelector('[data-unc]');
    if (btn) btn.onclick = () => { uncollected = !uncollected; paint(el); };
  }

  function detail(el, id) {
    const host = el.querySelector('#detail');
    const o = id ? Api.findOrder(id) : null;
    if (!o) {
      host.innerHTML = `<div class="dt-empty">${I.svg('receipt', 30, '#c7cab8')}<div>选择左侧订单查看详情</div></div>`;
      return;
    }

    /* 口味与备注绑定在 items 行内（PRD §15.6.2），整单级只有 orderNote。
       小计按折后单价算 —— 逐行折后价之和恒等于 o.total，展示与结算不会各说各话。 */
    // 名称取订单自身的快照，不回查商品表（§15.6.2）
    const rows = o.items.map(item => {
      const [, iname, q, , discountedUnitPrice, flavor, note] = item;
      // 当前 admin HTTP 投影的第 4 位是行应付小计；旧视觉蓝本的 7 位数组才含
      // 折后单价。两种公开形状在页面边界收敛，不回查商品或重算历史价格。
      const sub = item.length >= 5 ? discountedUnitPrice * q : item[3];
      return { name: iname, q, sub, band: [flavor, note].filter(Boolean).join(' · ') };
    });
    host.innerHTML =
      `<div class="dt-hero">
         <span class="ring r1"></span>
         <div class="dt-code tnum">${o.code}</div>
         <div class="dt-no tnum">${o.no}</div>
         <span class="pill pill--ondark"><i class="pd"></i>${o.status}</span>
       </div>
       <div class="dt-body">
         <div class="sec-h"><span class="t">菜品明细</span></div>
         <div class="card card-pad">
           ${rows.map(r => `
             <div class="dt-item">
               <span class="grow">${T.esc(r.name)}${r.band ? `<span class="ord-band">${T.esc(r.band)}</span>` : ''}</span>
               <span class="faint tnum">×${r.q}</span>
               <span class="tnum" style="width:64px;text-align:right">${T.money(Api.yuan(r.sub))}</span>
             </div>`).join('')}
           ${o.discountCut ? `<div class="dt-item"><span class="grow faint">员工折扣 ${o.discountRate}%</span><span class="tnum faint">-${T.money(Api.yuan(o.discountCut))}</span></div>` : ''}
           <div class="dt-total"><span class="grow">实付</span>${T.money(Api.yuan(o.total))}</div>
         </div>

         <div class="sec-h"><span class="t">取餐与备注</span></div>
         <div class="card card-pad">
           <div class="kv"><span class="k">联系人</span><span class="v">${T.esc(o.contact)}</span></div>
           <div class="kv"><span class="k">手机号</span><span class="v tnum">${o.phone}</span></div>
           <div class="kv"><span class="k">支付时间</span><span class="v tnum">${o.paidAt}</span></div>
           <div class="kv"><span class="k">取餐时间</span><span class="v tnum">${o.pickupDate} ${o.pickupTime}</span></div>
           <div class="kv"><span class="k">整单备注</span><span class="v">${T.esc(o.orderNote || '—')}</span></div>
           <div class="kv"><span class="k">取餐点</span><span class="v">${T.esc(o.pickupPoint)}</span></div>
           <div class="kv"><span class="k">微信交易号</span><span class="v tnum">${T.esc(o.txnId)}</span></div>
         </div>
       </div>
       <div class="dt-foot">
         <span class="grow"></span>
         ${Api.canRefund(o.status) ? `<button class="btn btn--line danger" data-refund>发起退款</button>` : ''}
         <span class="faint" style="font-size:12.5px">履约操作请使用商户小程序</span>
       </div>`;

    const refundBtn = host.querySelector('[data-refund]');
    if (refundBtn) refundBtn.onclick = () => openRefund(el, o);
  }

  /* 退款确认层。三件事必须让主账号看见后再点：
     - 退多少（§7.7 只有全额，金额不可改，所以直接显示而不是给输入框）
     - 谁在退（操作人会记进财务对账，追责靠它）
     - 为什么退（必填，同样进对账）
     以及一句后果：退款发起后不可撤销，且到账要以微信为准。 */
  function openRefund(el, o) {
    const me = Api.currentAccount();
    window.Modal.open({
      title: '发起退款',
      width: '460px',
      bodyHtml:
        `<div class="kv"><span class="k">订单</span><span class="v tnum">${T.esc(o.no)} · 取餐号 ${T.esc(o.code)}</span></div>
         <div class="kv"><span class="k">退款金额</span><span class="v">${T.money(Api.yuan(o.total))} <span class="faint" style="font-size:12px">全额，不可修改</span></span></div>
         <div class="kv"><span class="k">操作人</span><span class="v">${T.esc(me ? me.name : '—')}</span></div>
         <div style="margin-top:12px">
           <div class="fld-lb">退款原因<span class="req">*</span></div>
           <input class="inp" id="rf-why" placeholder="例如：客户未取餐 / 菜品临时售罄" maxlength="40" style="width:100%">
         </div>
         <div class="imp-note warn" style="margin-top:12px">
           退款发起后<b>不可撤销</b>。订单先进入「退款中」，只有微信确认退款成功才会变为「已退款」；
           退款失败时订单停在「退款中」，需在财务与对账页跟进。
         </div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="cancel">取消</button>
         <button class="btn btn--danger" data-a="ok">确认退款</button>`,
      onMount(root, done) {
        const why = root.querySelector('#rf-why');
        why.focus();
        root.querySelector('[data-a="cancel"]').onclick = done;
        root.querySelector('[data-a="ok"]').onclick = () => {
          Api.refundOrder(o.id, why.value).then(r => {
            done();
            /* 跳到退款中并选中该单。uncollected 必须一并清掉 ——
               它是挂在 待取餐 上的筛选，带到别的泳道会把列表筛成空的。 */
            lane = '退款中'; uncollected = false; selId = r.id;
            paint(el);
            window.Toast.show(`已对「${r.code}」发起全额退款，等待微信到账`, { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['orders'] = { sub: '订单查询、未取餐筛选与全额退款', flush: true, render };
})();
