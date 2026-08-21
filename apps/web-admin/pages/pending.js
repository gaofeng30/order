/* 支付待处理（PRD §7.3）

   这里的每一条都是「顾客已经付了钱，但系统没能给他建单」。钱在微信账户里，
   订单不存在 —— 对顾客而言就是付了钱没饭吃，对商户而言是一笔悬着的资损。

   条目从哪来：后端定时任务扫描已发起支付但未生成订单的预支付记录，调微信查询
   接口核对；查得已支付则自动幂等补建。**只有自动补建也失败的才落到这里。**
   所以这页不是实时的，它是自动兜底之后剩下的人工出口。

   两个出口，对应 §7.3 的「发起退款或人工建单」：
   - 人工建单：阻塞原因解除后重试。原因没解除必须拒绝 —— 建了单也做不出菜。
   - 退款作废：全额原路退回（§7.7），进退款中等微信确认。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  function render(el) {
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">
           顾客已付款但系统未建单的条目。由后端定时任务核对微信支付结果后自动补建，
           <b>只有自动补建也失败的才会出现在这里</b>，需主账号人工处理。这些条目不是订单：没有状态，也没有取餐号。
         </span>
       </div>
       <div class="tbl-wrap" id="pd-host"></div>
       <div class="card card-pad imp-note" style="margin-top:14px">
         <b>人工建单</b>会按 §7.4 排产（距取餐不足 30 分钟直接进「制作中」）并分配取餐号；阻塞原因未解除时会被拒绝，
         请先到菜品管理恢复销售状态再重试。<b>退款作废</b>为原路全额退回，订单不会生成，退款进入「退款中」，
         到账以微信为准，可在财务与对账页跟进。两个动作都不可撤销。
       </div>`;
    paint(el);
  }

  function paint(el) {
    Api.listPendingPayments().then(list => {
      el.querySelector('#pd-host').innerHTML = T.render({
        cols: [
          { t: '预支付单号', w: '162px', render: p =>
              `<span class="tnum">${T.esc(p.outTradeNo)}</span><br><span class="faint tnum" style="font-size:11.5px">${T.esc(p.txnId)}</span>` },
          { t: '支付时间', w: '150px', render: p => `<span class="tnum">${T.esc(p.paidAt)}</span>` },
          { t: '金额', w: '96px', cls: 'num', render: p => T.money(Api.yuan(p.amount), true) },
          { t: '顾客', w: '132px', render: p =>
              `${T.esc(p.contact)}<br><span class="faint tnum" style="font-size:11.5px">${T.esc(p.phone)}</span>` },
          { t: '意向取餐', w: '132px', render: p =>
              `<span class="tnum">${T.esc(p.pickupDate)}</span><br><span class="faint tnum" style="font-size:11.5px">${T.esc(p.pickupTime)} ${p.mealPeriod === 'lunch' ? '午餐' : '晚餐'}</span>` },
          { t: '菜品', render: p => `<span class="ellipsis" style="display:block">${T.esc(Api.itemsSummary(p.items))}</span>` },
          { t: '未建单原因', w: '210px', render: p =>
              `${T.pill(p.cause, 'warn')}<br><span class="faint" style="font-size:11.5px">${T.esc(p.causeDetail || '')}</span>` },
          { t: '操作', w: '170px', cls: 'act', render: p =>
              `<button class="btn btn--sm btn--line" data-act="build" data-id="${p.id}">人工建单</button>
               <button class="btn btn--sm btn--line danger" data-act="void" data-id="${p.id}">退款作废</button>` },
        ],
        rows: list,
        key: p => p.id,
        empty: '没有待处理的支付条目',
      });

      T.bind(el.querySelector('#pd-host'), {
        build(id) { openBuild(el, list.find(p => p.id === id)); },
        void(id) { openVoid(el, list.find(p => p.id === id)); },
      });
    });
  }

  /* 建单确认。先把当前的阻塞判定读出来展示 —— 让主账号在点之前就知道能不能成，
     而不是点完看到一句红字。原因仍在时直接禁用确认按钮并给出解法。 */
  function openBuild(el, p) {
    if (!p) return;
    const why = Api.blockingReason(p);
    window.Modal.open({
      title: '人工建单',
      width: '480px',
      bodyHtml:
        `<div class="kv"><span class="k">预支付单号</span><span class="v tnum">${T.esc(p.outTradeNo)}</span></div>
         <div class="kv"><span class="k">已收金额</span><span class="v">${T.money(Api.yuan(p.amount))}</span></div>
         <div class="kv"><span class="k">意向取餐</span><span class="v tnum">${T.esc(p.pickupDate)} ${T.esc(p.pickupTime)}</span></div>
         <div class="kv"><span class="k">菜品</span><span class="v">${T.esc(Api.itemsSummary(p.items))}</span></div>
         <div class="imp-note ${why ? 'warn' : ''}" style="margin-top:12px">
           ${why ? `<b>暂时不能建单：</b>${T.esc(why)}`
                 : '将按原支付金额与取餐信息生成订单，分配取餐号，并按取餐时间自动排产。该动作不可撤销。'}
         </div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="cancel">取消</button>` +
        (why ? '' : `<button class="btn btn--blue" data-a="ok">确认建单</button>`),
      onMount(root, done) {
        root.querySelector('[data-a="cancel"]').onclick = done;
        const ok = root.querySelector('[data-a="ok"]');
        if (ok) ok.onclick = () => {
          Api.rebuildOrder(p.id).then(o => {
            done();
            paint(el);
            window.Toast.show(`已建单「${o.code}」· ${o.status}`, { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  function openVoid(el, p) {
    if (!p) return;
    const me = Api.currentAccount();
    window.Modal.open({
      title: '退款作废',
      width: '460px',
      bodyHtml:
        `<div class="kv"><span class="k">预支付单号</span><span class="v tnum">${T.esc(p.outTradeNo)}</span></div>
         <div class="kv"><span class="k">退款金额</span><span class="v">${T.money(Api.yuan(p.amount))} <span class="faint" style="font-size:12px">全额，不可修改</span></span></div>
         <div class="kv"><span class="k">操作人</span><span class="v">${T.esc(me ? me.name : '—')}</span></div>
         <div style="margin-top:12px">
           <div class="fld-lb">作废原因<span class="req">*</span></div>
           <input class="inp" id="pv-why" placeholder="例如：取餐时间已过，无法补建" maxlength="40" style="width:100%">
         </div>
         <div class="imp-note warn" style="margin-top:12px">
           作废后<b>不再生成订单</b>，款项原路全额退回。退款进入「退款中」，只有微信确认成功才是「已退款」，
           可在财务与对账页跟进。该动作不可撤销。
         </div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="cancel">取消</button>
         <button class="btn btn--danger" data-a="ok">确认作废并退款</button>`,
      onMount(root, done) {
        const why = root.querySelector('#pv-why');
        why.focus();
        root.querySelector('[data-a="cancel"]').onclick = done;
        root.querySelector('[data-a="ok"]').onclick = () => {
          Api.refundPendingPayment(p.id, why.value).then(v => {
            done();
            paint(el);
            window.Toast.show(`已作废「${v.outTradeNo}」并发起全额退款`, { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['pending'] = { sub: '已收款但未建单的条目', render };
})();
