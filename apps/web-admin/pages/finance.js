/* 财务与对账（PRD §6.11）

   这页的全部价值是「我这里的数和微信商户平台的交易账单对得上」。差一分就是废的，
   所以两条归集口径必须写在明处，页面也照着说给管理员听：

   - 收款按**支付日期**归集（微信账单以交易时间为准），不是营业日期。预约单可以
     今天付、明天取，两个日期在同一行里都展示，避免管理员自己脑补。
   - 退款按**退款日期**归集，不是原订单的支付日期。跨日退款在微信账单里出现在到账
     那天。所以「今天的退款」里可能有昨天的订单，这是对的。

   后台按账单日拉取并比对微信账单，对账证据写入统一审计；本页仍只展示系统事实，
   不把本地汇总误标成微信已核平。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let today = '';                       // Api.bootstrap 完成后才有服务端营业日
  let range = { from: '', to: '' };
  let tab = 'pay';                       // pay | refund

  function render(el) {
    today = Api.today();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(today)) {
      el.innerHTML = '<div class="card card-pad"><b>财务日期暂不可用</b><div class="faint" style="margin-top:8px">请稍后重试，系统不会用本机日期代替服务端营业日。</div></div>';
      return;
    }
    if (!range.from || !range.to) range = { from: today, to: today };
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">收款按<b>支付日期</b>归集，与微信商户平台「交易账单」同口径；退款按<b>退款到账日期</b>归集，因此今天的退款里可能有前几天的订单。</span>
       </div>
       <div class="card card-pad" style="display:flex;align-items:center;gap:10px;margin-bottom:16px">
         <span class="faint">日期</span>
         <input class="inp" type="date" id="f-from" value="${range.from}">
         <span class="faint">至</span>
         <input class="inp" type="date" id="f-to" value="${range.to}">
         <button class="btn btn--line" data-quick="today">今天</button>
         <button class="btn btn--line" data-quick="7">近 7 天</button>
         <span class="grow"></span>
         <button class="btn btn--primary" data-export>${I.svg('box', 16)}导出明细</button>
       </div>
       <div id="sum-host"></div>
       <div class="page-head" style="margin-top:14px">
         <div class="segs" id="fin-tabs"></div>
       </div>
       <div class="tbl-wrap" id="fin-host"></div>
       <div class="card card-pad imp-note" style="margin-top:14px">
         对账口径：<b>净额 = 实收合计 − 退款合计</b>，均按金额相加；一期只有原路全额退款，每笔退款金额等于原订单实付。
         把净额与微信商户平台同一日期区间的「交易账单」核对即可。
         <b>已收款未建单的条目不计入实收合计</b>（它们没有订单），因此微信账单会比实收合计多出这部分 —— 差额见「支付待处理」页，处理完即回归一致。
         后台按账单日自动拉取并比对微信账单，对账结果写入统一审计；<b>本页数字只汇总本系统事实，不单独代表已与微信核平</b>。
       </div>`;

    const from = el.querySelector('#f-from'), to = el.querySelector('#f-to');
    const apply = () => {
      if (from.value && to.value && from.value > to.value) {
        window.Toast.show('起始日期不能晚于结束日期', { icon: 'warn' });
        from.value = range.from; to.value = range.to; return;
      }
      range = { from: from.value, to: to.value };
      paint(el);
    };
    from.onchange = apply;
    to.onchange = apply;

    el.querySelectorAll('[data-quick]').forEach(b => {
      b.onclick = () => {
        const d = new Date(today + 'T00:00:00');
        if (b.dataset.quick === '7') d.setDate(d.getDate() - 6);
        range = { from: d.toISOString().slice(0, 10), to: today };
        from.value = range.from; to.value = range.to;
        paint(el);
      };
    });

    el.querySelector('[data-export]').onclick = () => doExport();
    paint(el);
  }

  function paint(el) {
    Api.financeSummary(range).then(s => paintSummary(el, s));
    paintTabs(el);
    if (tab === 'pay') paintPayments(el); else paintRefunds(el);
  }

  /* 金额的符号必须在 ¥ 之前：Api.yuan 只管数字，负号跟在 ¥ 后面（¥-12.00）
     是错的排版。退款与折扣天然是减项，为零时不带负号。 */
  function amt(cents, minus) {
    const n = Number(cents) || 0;
    if (n < 0) return '−¥' + Api.yuan(-n);
    return (minus && n > 0 ? '−¥' : '¥') + Api.yuan(n);
  }

  function paintSummary(el, s) {
    const card = (k, v, sub, ic, c) =>
      `<div class="card card-pad kpi">
         <span class="kpi-ic" style="color:${c};background:${c}14">${I.svg(ic, 19, c)}</span>
         <div class="kpi-v tnum">${v}</div>
         <div class="kpi-k">${k}</div>
         <div class="kpi-k faint" style="margin-top:2px">${sub}</div>
       </div>`;
    el.querySelector('#sum-host').innerHTML =
      `<div class="grid-4">
         ${card('实收合计', amt(s.gross), s.count + ' 笔收款', 'receipt', '#3f6b46')}
         ${card('退款合计', amt(s.refundAmount, true),
                s.refundCount + ' 笔' + (s.pendingCount ? ` · ${s.pendingCount} 笔未到账` : ''), 'bell', '#a4873f')}
         ${card('净额', amt(s.net), s.unbuiltCount
                  ? `另有 ${s.unbuiltCount} 笔已收款未建单 ¥${Api.yuan(s.unbuiltAmount)}，未计入`
                  : '与微信交易账单核对此项', 'chart', '#2f5d8a')}
         ${card('员工折扣', amt(s.discountCut, true), s.staffCount + ' 笔按员工价', 'user', '#7a6a52')}
       </div>`;
  }

  function paintTabs(el) {
    Promise.all([Api.listPayments(range), Api.listRefunds(range)]).then(([p, r]) => {
      el.querySelector('#fin-tabs').innerHTML =
        [['pay', '收款明细', p.length], ['refund', '退款记录', r.length]]
          .map(([k, t, n]) => `<button class="seg${tab === k ? ' on' : ''}" data-tab="${k}">${t}<span class="cnt">${n}</span></button>`).join('');
      el.querySelectorAll('[data-tab]').forEach(b => {
        b.onclick = () => { tab = b.dataset.tab; paint(el); };
      });
    });
  }

  function paintPayments(el) {
    Api.listPayments(range).then(list => {
      el.querySelector('#fin-host').innerHTML = T.render({
        cols: [
          { t: '订单号', w: '150px', render: o => `<span class="tnum">${T.esc(o.no)}</span><br><span class="faint tnum" style="font-size:11.5px">取餐号 ${T.esc(o.code)}</span>` },
          { t: '支付时间', w: '160px', render: o => `<span class="tnum">${T.esc(o.paidAt)}</span>` },
          { t: '微信交易号', w: '200px', render: o => `<span class="tnum" style="font-size:12px">${T.esc(o.txnId)}</span>` },
          { t: '原价', w: '92px', cls: 'num', render: o => `<span class="faint tnum">¥${Api.yuan(o.subtotal)}</span>` },
          { t: '折扣', w: '104px', cls: 'num', render: o => o.discountCut
              ? `<span class="tnum">−¥${Api.yuan(o.discountCut)}</span><br><span class="faint tnum" style="font-size:11.5px">员工价 ${o.discountRate}%</span>`
              : '<span class="faint">—</span>' },
          { t: '实付', w: '104px', cls: 'num', render: o => T.money(Api.yuan(o.total), true) },
          { t: '营业日期', w: '130px', render: o => `<span class="tnum">${T.esc(o.pickupDate)}</span><br><span class="faint tnum" style="font-size:11.5px">${T.esc(o.pickupTime)} ${o.mealPeriod === 'lunch' ? '午餐' : '晚餐'}</span>` },
          { t: '状态', w: '90px', render: o => T.pill(o.status) },
        ],
        rows: list,
        empty: '所选日期没有收款记录',
      });
    });
  }

  function paintRefunds(el) {
    Api.listRefunds(range).then(list => {
      el.querySelector('#fin-host').innerHTML = T.render({
        cols: [
          { t: '退款单号', w: '210px', render: r => `<span class="tnum" style="font-size:12px">${T.esc(r.no)}</span>` },
          { t: '原订单', w: '150px', render: r => `<span class="tnum">${T.esc(r.orderNo)}</span><br><span class="faint tnum" style="font-size:11.5px">付于 ${T.esc(r.paidAt.slice(0, 10))}</span>` },
          { t: '微信交易号', w: '190px', render: r => `<span class="tnum" style="font-size:12px">${T.esc(r.txnId)}</span>` },
          { t: '退款金额', w: '110px', cls: 'num', render: r => T.money(Api.yuan(r.amount)) },
          { t: '退款状态', w: '96px', render: r => T.pill(r.status) },
          { t: '操作人', w: '90px', render: r => T.esc(r.operator) },
          { t: '退款时间', w: '160px', render: r => `<span class="tnum">${T.esc(r.at)}</span>` },
          { t: '原因', render: r => `<span class="faint ellipsis">${T.esc(r.reason || '—')}</span>` },
        ],
        rows: list,
        key: r => r.no,
        empty: '所选日期没有退款记录',
      });
    });
  }

  /* 导出走 Blob + a[download]。契约层只产出文本，落盘方式属页面职责；
     接后端后改为服务端出文件、页面拿 URL，本函数是唯一要改的地方。 */
  function doExport() {
    Api.buildPaymentExport(range).then(csv => {
      const name = `收款明细_${range.from}_至_${range.to}.csv`;
      const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
      const a = document.createElement('a');
      a.href = url; a.download = name;
      document.body.appendChild(a); a.click(); a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 0);
      window.Toast.show(`已导出「${name}」`, { icon: 'check' });
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['finance'] = { sub: '按支付日期与微信交易账单核对', render };
})();
