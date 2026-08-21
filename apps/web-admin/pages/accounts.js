/* 商户账号名单（PRD §4.4）—— 决定谁能进商户端与 PC 后台。
   与员工折扣白名单是两份互不影响的名单：这份管「能不能登录」，那份管「打不打折」。
   PRD §6.13.4 明确不提供批量导入：账号数量极少，而误操作会直接影响谁能进后台。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let kw = '';

  function render(el) {
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">本名单决定谁能登录，与「员工折扣白名单」互不影响 —— 那份只决定顾客结算时打不打折。主账号可登录 PC 后台并拥有全部权限；子账号只能用小程序的订单、核销、菜品三个页面。</span>
         <input class="inp staff-search" id="f-kw" placeholder="搜索手机号或姓名" value="${T.esc(kw)}">
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}添加账号</button>
       </div>
       <div class="tbl-wrap" id="acc-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    const search = el.querySelector('#f-kw');
    search.oninput = () => { kw = search.value; paint(el); };
    paint(el);
  }

  function paint(el) {
    Api.listMerchantAccounts(kw).then(list => {
      const owners = list.filter(a => a.role === 'owner' && a.enabled).length;
      const host = el.querySelector('#acc-host');
      host.innerHTML = T.render({
        cols: [
          { t: '姓名', w: '130px', render: a => T.esc(a.name) },
          { t: '手机号', w: '140px', cls: 'num', render: a => `<span class="tnum">${T.esc(a.phone)}</span>` },
          { t: '角色', w: '96px', render: a => T.pill(Api.ROLE_LABEL[a.role], a.role === 'owner' ? 'ok' : 'info') },
          { t: '可用范围', w: '210px', render: a => `<span class="faint">${a.role === 'owner' ? 'PC 后台 + 小程序商户端' : '仅小程序：订单 / 核销 / 菜品'}</span>` },
          { t: '状态', w: '90px', render: a => T.pill(a.enabled ? '启用' : '停用', a.enabled ? 'ok' : 'mute') },
          { t: '微信绑定', w: '96px', render: a => a.boundOpenId ? T.pill('已绑定', 'ok') : `<span class="faint">未绑定</span>` },
          { t: '操作', w: '196px', cls: 'act', render: a => `
            <button class="btn btn--sm btn--line" data-act="toggle" data-id="${a.id}">${a.enabled ? '停用' : '启用'}</button>
            <button class="btn btn--sm btn--ghost-blue" data-act="edit" data-id="${a.id}">编辑</button>
            <button class="btn btn--sm btn--line danger" data-act="del" data-id="${a.id}">删除</button>` },
        ],
        rows: list,
        empty: kw ? `没有匹配「${T.esc(kw)}」的账号` : '名单为空',
      });

      if (!kw && owners <= 1) {
        host.insertAdjacentHTML('afterend',
          `<div class="card card-pad imp-note warn" id="acc-warn">当前只有 1 个启用的主账号。它不能被停用、删除或降级 —— 否则将没有人能登录 PC 后台。建议再添加一个主账号作为备份。</div>`);
      }

      T.bind(host, {
        toggle(id) {
          const a = list.find(x => x.id === id);
          Api.setMerchantAccountEnabled(id, !a.enabled)
            .then(saved => { render(el); window.Toast.show(saved.enabled ? '已启用' : '已停用', { icon: 'check' }); })
            .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        },
        edit(id) { openEdit(el, list.find(x => x.id === id)); },
        del(id) {
          const a = list.find(x => x.id === id);
          window.Modal.confirm({
            title: '删除商户账号',
            body: `确认删除「${T.esc(a.name)} · ${T.esc(a.phone)}」？该手机号将无法再登录商户端${a.role === 'owner' ? '与 PC 后台' : ''}。${a.boundOpenId ? '已绑定的微信也会解绑。' : ''}停用可保留记录，删除不可恢复。`,
            okText: '删除', danger: true,
          }).then(yes => {
            if (!yes) return;
            Api.deleteMerchantAccount(id)
              .then(() => { render(el); window.Toast.show('已删除', { icon: 'check' }); })
              .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          });
        },
      });
    });
  }

  function openEdit(el, a) {
    const isEdit = !!a;
    window.Drawer.open({
      title: isEdit ? '编辑商户账号' : '添加商户账号',
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">姓名 <span class="req">*</span></div>
           <input class="inp" id="f-name" value="${a ? T.esc(a.name) : ''}" placeholder="便于识别是谁，例如 后厨老陈">
         </div>
         <div class="fld">
           <div class="fld-lb">手机号 <span class="req">*</span></div>
           <input class="inp tnum" id="f-phone" value="${a ? T.esc(a.phone) : ''}" placeholder="11 位手机号，用于微信登录时比对">
           <div class="fld-hint">商户首次在小程序点「商户登录」时授权手机号，命中本名单才获得商户端权限；PC 后台扫码登录比对同一份名单。</div>
         </div>
         <div class="fld">
           <div class="fld-lb">角色 <span class="req">*</span></div>
           <select class="sel" id="f-role">
             ${Api.ROLES.map(r => `<option value="${r}"${a && a.role === r ? ' selected' : ''}>${Api.ROLE_LABEL[r]}</option>`).join('')}
           </select>
           <div class="fld-hint">主账号：PC 后台全部功能 + 小程序商户端。子账号：只能用小程序的订单、核销、菜品三个页面，不能登录 PC。</div>
         </div>
         ${isEdit ? `<div class="card card-pad staff-ro">
           <div class="fld-lb">系统字段（只读）</div>
           <div class="ro-row"><span>状态</span><span>${a.enabled ? '启用' : '停用'} · 在列表中切换</span></div>
           <div class="ro-row"><span>微信绑定</span><span>${a.boundOpenId ? '已绑定 · 由小程序端建立' : '未绑定'}</span></div>
         </div>` : ''}`,
      footerHtml:
        `<button class="btn btn--line" data-a="cancel">取消</button>
         <button class="btn btn--primary" data-a="ok">${isEdit ? '保存' : '添加'}</button>`,
      onMount(root, close) {
        root.querySelector('[data-a="cancel"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = () => {
          Api.saveMerchantAccount({
            id: a ? a.id : '',
            name: root.querySelector('#f-name').value,
            phone: root.querySelector('#f-phone').value,
            role: root.querySelector('#f-role').value,
          }).then(() => {
            close();
            render(el);
            window.Toast.show(isEdit ? '已保存' : '已添加', { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['accounts'] = { sub: '谁能登录商户端与 PC 后台', render };
})();
