/* 会员名单 —— 对应 apps/wechat-miniprogram/pages/admin-members + admin-member-edit（二期能力）
   PC 形态：表格 + 搜索 / 等级筛选 + 右侧编辑抽屉。
   手机号是唯一识别键，与用户在小程序授权的手机号比对后自动生效。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let kw = '';
  let levelId = '';

  function render(el) {
    el.innerHTML =
      `<div class="toolbar">
         <div class="search">
           ${I.svg('search', 15)}
           <input class="inp" id="kw" placeholder="搜索手机号或姓名" value="${T.esc(kw)}">
         </div>
         <select class="sel" id="lv" style="width:180px"></select>
         <span class="faint" id="cnt" style="font-size:12.5px"></span>
         <span class="sp"></span>
         <button class="btn btn--line" data-import>${I.svg('box', 16)}批量导入</button>
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}新增会员</button>
       </div>
       <div id="tbl-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    el.querySelector('[data-import]').onclick = () => window.App.go('members/import');
    const k = el.querySelector('#kw');
    k.oninput = () => { kw = k.value; paint(el, true); };

    paint(el);
  }

  function paint(el, keepFocus) {
    Api.listLevels().then(levels => {
      const sel = el.querySelector('#lv');
      sel.innerHTML = `<option value="">全部等级</option>` +
        levels.map(l => `<option value="${l.id}"${levelId === l.id ? ' selected' : ''}>${T.esc(l.name)}（${(l.discount / 10).toFixed(1)} 折）</option>`).join('');
      sel.onchange = () => { levelId = sel.value; paint(el); };

      const lvName = id => (levels.find(l => l.id === id) || { name: '—' }).name;

      Api.listMembers({ kw, levelId }).then(res => {
        el.querySelector('#cnt').textContent = `共 ${res.total} 人`;

        const host = el.querySelector('#tbl-host');
        host.innerHTML = T.render({
          cols: [
            { t: '姓名', w: '92px', render: r => `<b>${T.esc(r.name)}</b>` },
            { t: '手机号', w: '116px', render: r => `<span class="tnum">${r.phone}</span>` },
            { t: '等级', w: '96px', render: r => `<span class="pill pill--info">${T.esc(lvName(r.levelId))}</span>` },
            { t: '单位 / 部门', render: r => {
              const s = [r.org, r.dept].filter(Boolean).join(' · ');
              return s ? T.esc(s) : '<span class="faint">—</span>';
            } },
            { t: '工号', w: '80px', render: r => r.jobNo ? `<span class="tnum">${T.esc(r.jobNo)}</span>` : '<span class="faint">—</span>' },
            { t: '加入时间', w: '104px', render: r => `<span class="faint tnum">${r.joinAt}</span>` },
            // 累计消费与单量都是服务端只读统计，合成一列，腾出宽度给单位/部门
            { t: '累计', w: '124px', render: r =>
              `${T.money(r.spend)}<span class="faint tnum" style="margin-left:6px">${r.orders} 单</span>` },
            // 「微信绑定」与「权益状态」是两个独立开关，并排展示比拆两列省 80px
            { t: '状态', w: '150px', render: r =>
              (r.bound ? '<span class="pill pill--ok"><i class="pd"></i>已绑定</span>'
                       : '<span class="pill pill--mute"><i class="pd"></i>待授权</span>') +
              (r.enabled ? '<span class="pill pill--ok" style="margin-left:4px"><i class="pd"></i>生效</span>'
                         : '<span class="pill pill--mute" style="margin-left:4px"><i class="pd"></i>停用</span>') },
            { t: '操作', w: '104px', cls: 'act', render: r =>
              `<button class="btn btn--sm btn--ghost-blue" data-act="edit" data-id="${r.id}">编辑</button>
               <button class="ibtn danger" data-act="del" data-id="${r.id}">${I.svg('trash', 16)}</button>` },
          ],
          rows: res.list,
          empty: (kw || levelId) ? '没有匹配的会员' : '名单还是空的，可手动新增或批量导入',
        });

        T.bind(host, {
          edit(id) { openEdit(el, res.list.find(m => m.id === id)); },
          del(id) {
            const m = res.list.find(x => x.id === id);
            window.Modal.confirm({
              title: '移出名单',
              body: `确认把「${T.esc(m.name)}」移出会员名单？该手机号将失去会员折扣与券，历史订单不受影响。`,
              okText: '移出', danger: true,
            }).then(yes => {
              if (!yes) return;
              Api.deleteMember(id).then(() => { paint(el); window.Toast.show('已移出名单', { icon: 'check' }); })
                .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
            });
          },
        });

        if (keepFocus) {
          const k = el.querySelector('#kw');
          k.focus();
          k.setSelectionRange(k.value.length, k.value.length);
        }
      });
    });
  }

  function openEdit(el, m) {
    const isEdit = !!m;
    Api.listLevels().then(levels => {
      window.Drawer.open({
        title: isEdit ? '编辑会员' : '新增会员',
        tag: '二期',
        bodyHtml:
          `<div class="fld">
             <div class="fld-lb">手机号 <span class="req">*</span></div>
             <input class="inp tnum" id="f-phone" value="${m ? m.phone : ''}" placeholder="11 位手机号" maxlength="11">
             <div class="fld-hint">手机号是唯一识别键。用户在小程序授权手机号后自动命中名单并生效，无需额外操作。</div>
           </div>
           <div class="fld">
             <div class="fld-lb">姓名 <span class="req">*</span></div>
             <input class="inp" id="f-name" value="${m ? T.esc(m.name) : ''}">
           </div>
           <div class="fld">
             <div class="fld-lb">会员等级 <span class="req">*</span></div>
             <select class="sel" id="f-lv">
               <option value="">请选择等级</option>
               ${levels.map(l => `<option value="${l.id}"${m && m.levelId === l.id ? ' selected' : ''}>${T.esc(l.name)}（${(l.discount / 10).toFixed(1)} 折）</option>`).join('')}
             </select>
           </div>
           <div class="fld-row">
             <div class="fld"><div class="fld-lb">单位</div><input class="inp" id="f-org" value="${m ? T.esc(m.org || '') : ''}"></div>
             <div class="fld"><div class="fld-lb">部门</div><input class="inp" id="f-dept" value="${m ? T.esc(m.dept || '') : ''}"></div>
             <div class="fld"><div class="fld-lb">工号</div><input class="inp" id="f-job" value="${m ? T.esc(m.jobNo || '') : ''}"></div>
           </div>
           <div class="fld">
             <div class="fld-lb">备注</div>
             <textarea class="txa" id="f-remark" style="min-height:64px">${m ? T.esc(m.remark || '') : ''}</textarea>
           </div>
           <div class="fld row gap12">
             <button class="sw${!m || m.enabled ? ' on' : ''}" id="f-en"></button>
             <span style="font-size:13px">权益生效</span>
             <span class="grow"></span>
             <span class="fld-hint" style="margin:0">停用后保留在名单中，但不再享受折扣与券</span>
           </div>
           ${isEdit ? `
           <div class="sec-h" style="margin-top:6px"><span class="t">只读信息</span></div>
           <div class="card card-pad">
             <div class="kv"><span class="k">加入时间</span><span class="v tnum">${m.joinAt}</span></div>
             <div class="kv"><span class="k">微信绑定</span><span class="v">${m.bound ? '已绑定' : '待用户授权手机号'}</span></div>
             <div class="kv"><span class="k">累计消费</span><span class="v tnum">¥${m.spend}</span></div>
             <div class="kv"><span class="k">累计单量</span><span class="v tnum">${m.orders}</span></div>
           </div>
           <div class="fld-hint">累计消费与单量由服务端统计，是商户手动调级的依据；按消费自动升降级列入后期演进。</div>` : ''}`,
        footerHtml:
          `<span class="grow"></span>
           <button class="btn btn--line" data-a="c">取消</button>
           <button class="btn btn--primary" data-a="ok">保存</button>`,
        onMount(root, close) {
          const en = root.querySelector('#f-en');
          en.onclick = () => en.classList.toggle('on');
          root.querySelector('[data-a="c"]').onclick = close;
          root.querySelector('[data-a="ok"]').onclick = () => {
            Api.saveMember({
              id: m ? m.id : '',
              phone: root.querySelector('#f-phone').value,
              name: root.querySelector('#f-name').value,
              levelId: root.querySelector('#f-lv').value,
              org: root.querySelector('#f-org').value,
              dept: root.querySelector('#f-dept').value,
              jobNo: root.querySelector('#f-job').value,
              remark: root.querySelector('#f-remark').value,
              enabled: en.classList.contains('on'),
            }).then(() => {
              close(); paint(el);
              window.Toast.show(isEdit ? '已保存' : '已加入名单', { icon: 'check' });
            }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          };
        },
      });
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['members'] = { render };
})();
