/* 员工折扣白名单（PRD §6.4）
   只有两个可填字段：手机号（唯一识别键）与姓名（§4.1 附加手机号双要素的第二要素）。
   状态由行内开关切换；加入时间自动，已绑定 / 累计消费 / 累计单量只读。
   全局折扣率在本页顶部维护：一个整数百分比，对所有命中名单的用户、所有商品统一生效。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let kw = '';

  function render(el) {
    el.innerHTML =
      `<div class="rate-card card card-pad" id="rate-host"></div>
       <div class="page-head">
         <span class="ph-s grow">命中名单的手机号按下方折扣率结算；停用保留记录但暂停折扣，用于离职人员。</span>
         <input class="inp staff-search" id="f-kw" placeholder="搜索手机号或姓名" value="${T.esc(kw)}">
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}添加员工</button>
       </div>
       <div class="tbl-wrap" id="staff-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    const search = el.querySelector('#f-kw');
    search.oninput = () => { kw = search.value; paintList(el); };
    paintRate(el);
    paintList(el);
  }

  /* ---------------- 全局折扣率 ---------------- */
  function paintRate(el) {
    Api.getDiscountRate().then(rate => {
      const host = el.querySelector('#rate-host');
      host.innerHTML =
        `<div class="fld-lb">全局折扣率 <span class="req">*</span></div>
         <div class="rate-row">
           <span class="rate-pre">员工实付</span>
           <input class="inp tnum rate-in" id="f-rate" type="number" min="1" max="100" step="1" value="${rate}">
           <span class="rate-suf">%</span>
           <button class="btn btn--line" data-save-rate>保存折扣率</button>
           <span class="faint rate-hint" id="rate-hint">${rateHint(rate)}</span>
         </div>
         <div class="fld-hint">对所有命中名单的用户、所有商品统一生效。逐商品先按单价乘折扣率四舍五入到分，再乘数量求和。修改只影响新报价，不回算历史订单。</div>`;

      const input = host.querySelector('#f-rate');
      input.oninput = () => { host.querySelector('#rate-hint').textContent = rateHint(Number(input.value)); };
      host.querySelector('[data-save-rate]').onclick = () => {
        Api.saveDiscountRate(Number(input.value))
          .then(saved => window.Toast.show(`折扣率已保存 · 员工实付 ${saved}%`, { icon: 'check' }))
          .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
      };
    });
  }

  function rateHint(rate) {
    if (!Number.isInteger(rate) || rate < 1 || rate > 100) return '需为 1 到 100 的整数';
    if (rate === 100) return '无折扣';
    const zhe = Math.round(rate / 10 * 10) / 10;
    return `约 ${String(zhe).replace(/\.0$/, '')} 折`;
  }

  /* ---------------- 名单 ---------------- */
  function paintList(el) {
    Api.listStaff(kw).then(list => {
      const host = el.querySelector('#staff-host');
      host.innerHTML = T.render({
        cols: [
          { t: '姓名', w: '120px', render: r => T.esc(r.name) },
          { t: '手机号', w: '140px', cls: 'num', render: r => `<span class="tnum">${T.esc(r.phone)}</span>` },
          { t: '状态', w: '96px', render: r => T.pill(r.enabled ? '启用' : '停用', r.enabled ? 'ok' : 'mute') },
          { t: '已绑定', w: '86px', render: r => r.bound ? T.pill('已绑定', 'ok') : `<span class="faint">未绑定</span>` },
          { t: '加入时间', w: '110px', render: r => `<span class="faint tnum">${T.esc(r.joinAt)}</span>` },
          { t: '累计消费', w: '96px', cls: 'num', render: r => T.money(r.spend) },
          { t: '累计单量', w: '86px', cls: 'num', render: r => `<span class="faint tnum">${r.orders}</span>` },
          { t: '操作', w: '196px', cls: 'act', render: r => `
            <button class="btn btn--sm btn--line" data-act="toggle" data-id="${r.id}">${r.enabled ? '停用' : '启用'}</button>
            <button class="btn btn--sm btn--ghost-blue" data-act="edit" data-id="${r.id}">编辑</button>
            <button class="btn btn--sm btn--line danger" data-act="del" data-id="${r.id}">删除</button>` },
        ],
        rows: list,
        empty: kw ? `没有匹配「${T.esc(kw)}」的员工` : '名单为空，点击「添加员工」或使用批量导入',
      });

      T.bind(host, {
        toggle(id) {
          const r = list.find(x => x.id === id);
          Api.setStaffEnabled(id, !r.enabled)
            .then(saved => { paintList(el); window.Toast.show(saved.enabled ? '已启用' : '已停用', { icon: 'check' }); })
            .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        },
        edit(id) { openEdit(el, list.find(x => x.id === id)); },
        del(id) {
          const r = list.find(x => x.id === id);
          window.Modal.confirm({
            title: '删除员工',
            body: `确认把「${T.esc(r.name)} · ${T.esc(r.phone)}」移出白名单？该手机号将不再享受员工折扣。停用可保留记录，删除不可恢复。`,
            okText: '删除', danger: true,
          }).then(yes => {
            if (!yes) return;
            Api.deleteStaff(id)
              .then(() => { paintList(el); window.Toast.show('已删除', { icon: 'check' }); })
              .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          });
        },
      });
    });
  }

  /* ---------------- 新增 / 编辑 ---------------- */
  function openEdit(el, r) {
    const isEdit = !!r;
    window.Drawer.open({
      title: isEdit ? '编辑员工' : '添加员工',
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">姓名 <span class="req">*</span></div>
           <input class="inp" id="f-name" value="${r ? T.esc(r.name) : ''}" placeholder="与身份证一致，用于附加手机号校验">
           <div class="fld-hint">用户在个人中心手工填写附加手机号时，需手机号与姓名同时命中同一条记录才生效。</div>
         </div>
         <div class="fld">
           <div class="fld-lb">手机号 <span class="req">*</span></div>
           <input class="inp tnum" id="f-phone" value="" placeholder="${r ? T.esc(r.phone) + ' · 留空不修改' : '11 位手机号，唯一识别键'}">
         </div>
         ${isEdit ? `<div class="card card-pad staff-ro">
           <div class="fld-lb">系统字段（只读）</div>
           <div class="ro-row"><span>状态</span><span>${r.enabled ? '启用' : '停用'} · 在列表中切换</span></div>
           <div class="ro-row"><span>加入时间</span><span class="tnum">${T.esc(r.joinAt)}</span></div>
           <div class="ro-row"><span>微信绑定</span><span>${r.bound ? '已绑定' : '未绑定'}</span></div>
           <div class="ro-row"><span>累计消费 / 单量</span><span class="tnum">${r.spend} 元 / ${r.orders} 单</span></div>
         </div>` : ''}`,
      footerHtml:
        `<button class="btn btn--line" data-a="cancel">取消</button>
         <button class="btn btn--primary" data-a="ok">${isEdit ? '保存' : '添加'}</button>`,
      onMount(root, close) {
        root.querySelector('[data-a="cancel"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = () => {
          Api.saveStaff({
            id: r ? r.id : '',
            name: root.querySelector('#f-name').value,
            phone: root.querySelector('#f-phone').value,
          }).then(() => {
            close();
            paintList(el);
            window.Toast.show(isEdit ? '已保存' : '已添加', { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['staff'] = { sub: '全局折扣率与员工名单', render };
})();
