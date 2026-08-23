/* ============================================================
   绥安食品 商户端 PC 网页版 —— 路由与外壳
   零构建：按 <script> 顺序加载，页面模块挂在 window.Pages 上。
   ============================================================ */
(function () {
  const Api = window.Api;

  /* ---------------- 导航（替代小程序 admin-profile 的入口聚合页） ---------------- */
  const NAV = [
    { g: '经营', items: [
      { r: 'dashboard', t: '工作台', ic: 'dash' },
      { r: 'orders', t: '订单管理', ic: 'receipt' },
      { r: 'finance', t: '财务与对账', ic: 'receipt' },
      { r: 'pending', t: '支付待处理', ic: 'bell' },
    ] },
    { g: '菜品', items: [
      { r: 'products', t: '菜品管理', ic: 'bowl' },
      { r: 'product-import', t: '菜品批量导入', ic: 'box' },
      { r: 'categories', t: '分类管理', ic: 'tag' },
    ] },
    { g: '名单', items: [
      { r: 'staff', t: '员工折扣白名单', ic: 'user' },
      { r: 'staff-import', t: '员工批量导入', ic: 'box' },
      { r: 'accounts', t: '商户账号名单', ic: 'store' },
    ] },
    { g: '门店', items: [
      { r: 'settings', t: '营业设置', ic: 'settings' },
      { r: 'layer', t: '开屏图层', ic: 'layers' },
    ] },
  ];

  function renderNav() {
    document.getElementById('side-nav').innerHTML = NAV.map(grp =>
      `<div class="side-group">
         <div class="side-gt">${grp.g}</div>
         ${grp.items.map(it =>
           `<a class="side-item" href="#/${it.r}" data-r="${it.r}">${window.Icon.svg(it.ic, 17)}<span>${it.t}</span></a>`
         ).join('')}
       </div>`).join('');
  }

  function navMeta(route) {
    for (const grp of NAV) {
      const it = grp.items.find(x => x.r === route);
      if (it) return { title: it.t, group: grp.g };
    }
    return { title: '工作台', group: '经营' };
  }

  /* ---------------- 顶栏：营业状态 + 账号下拉（原 admin-profile 商户中心） ---------------- */
  const BIZ = ['营业中', '休息中', '已截单'];

  function renderStatusPill() {
    const s = Api.storeView().status;
    const el = document.getElementById('tb-status');
    el.className = 'pill pill--' + Api.statusTone(s);
    el.innerHTML = `<i class="pd"></i>${s}`;
  }

  function renderAccount() {
    const view = Api.storeView();
    const mg = view.account || { name: '未登录', role: '—' };
    document.getElementById('acct-pop').innerHTML =
      `<div class="acct-card">
         <span class="ring r1"></span><span class="ring r2"></span>
         <div class="ac-in">
           <span class="ac-logo"><img src="../wechat-miniprogram/assets/emblem.png" alt=""></span>
           <div>
             <div class="ac-nm">${view.name}</div>
             <div class="ac-sub">${view.name} · ${mg.role} ${mg.name}</div>
           </div>
         </div>
       </div>
       <div class="acct-sec">
         <div class="acct-lb">营业状态</div>
         <div class="segs">
           ${BIZ.map(b => `<span class="seg${view.status === b ? ' on' : ''}" data-biz="${b}">${b}</span>`).join('')}
         </div>
       </div>
       <div class="acct-sec">
         <div class="acct-row" data-go="settings">${window.Icon.svg('settings', 16)}<span>营业设置</span></div>
         <div class="acct-row danger" data-logout>${window.Icon.svg('logout', 16)}<span>退出登录</span></div>
       </div>`;
  }

  function bindAccount() {
    const btn = document.getElementById('acct-btn');
    const pop = document.getElementById('acct-pop');

    btn.addEventListener('click', e => {
      e.stopPropagation();
      pop.classList.toggle('open');
    });
    document.addEventListener('click', e => {
      if (!pop.contains(e.target)) pop.classList.remove('open');
    });

    pop.addEventListener('click', e => {
      const biz = e.target.closest('[data-biz]');
      if (biz) {
        Api.setStoreStatus(biz.dataset.biz).then(s => {
          renderStatusPill();
          renderAccount();
          window.Toast.show(`已切换为「${s}」`, { icon: 'power' });
          if (current === 'dashboard' || current === 'settings') go(current, true);
        }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        return;
      }
      const goTo = e.target.closest('[data-go]');
      if (goTo) { pop.classList.remove('open'); location.hash = '#/' + goTo.dataset.go; return; }
      if (e.target.closest('[data-logout]')) {
        pop.classList.remove('open');
        Api.logout();
        window.Toast.show('登录已失效，请重新扫码登录', { icon: 'warn' });
        location.reload();
      }
    });
  }

  /* ---------------- 路由 ---------------- */
  let current = '';

  function parseHash() {
    const h = (location.hash || '').replace(/^#\/?/, '').trim();
    return window.Pages[h] ? h : 'dashboard';
  }

  function go(route, force) {
    if (!force && route === current) return;
    current = route;
    window.Drawer.close();
    window.Modal.close();

    document.querySelectorAll('.side-item').forEach(a =>
      a.classList.toggle('on', a.dataset.r === route));

    const page = window.Pages[route];
    const meta = navMeta(route);
    document.getElementById('tb-title').textContent = meta.title;
    // 页面标题只在顶栏出现一次；页内不再重复标题，只留操作按钮
    document.getElementById('tb-sub').textContent =
      page.sub || '';

    const el = document.getElementById('content');
    el.className = 'content' + (page.flush ? ' flush' : '');
    el.scrollTop = 0;

    /* 每次渲染换一个全新的容器，页面拿到的是这个容器而不是 #content 本身。
       页面的 Api 调用是异步的，用户在结果回来之前切走时，旧容器已从文档里摘除：
       此时旧回调仍能在自己的容器上 querySelector，写入的是一棵游离的树 —— 不抛异常、
       也不会污染新页面。否则 el.querySelector('#xxx') 会返回 null 并在赋值时崩溃。 */
    const root = document.createElement('div');
    root.className = 'page-root';
    el.replaceChildren(root);
    page.render(root);
    renderStatusPill();
  }

  /* ---------------- 启动 ---------------- */
  renderNav();
  bindAccount();
  document.getElementById('tb-status').textContent = '连接中';
  window.addEventListener('unhandledrejection', e => {
    e.preventDefault();
    const message = e.reason && e.reason.message ? e.reason.message : '请求失败，请重试';
    window.Toast.show(message, { icon: 'warn' });
  });

  window.addEventListener('hashchange', () => go(parseHash()));
  Api.bootstrap().then(() => {
    renderAccount();
    renderStatusPill();
    if (!location.hash) location.hash = '#/dashboard';
    go(parseHash(), true);
  }).catch(e => {
    if (e.status === 401) {
      renderPCLogin();
      return;
    }
    const host = document.getElementById('content');
    host.innerHTML = `<div class="card card-pad"><b>后台服务不可用</b><div class="faint" style="margin-top:8px">${e.message}</div><button class="btn btn--primary" style="margin-top:16px" onclick="location.reload()">重试</button></div>`;
    document.getElementById('tb-status').textContent = '不可用';
    renderAccount();
  });

  function renderPCLogin() {
    const host = document.getElementById('content');
    host.innerHTML = `<div class="card card-pad" style="max-width:560px;margin:56px auto;text-align:center"><b>主账号扫码登录</b><div class="faint" style="margin-top:8px">请用商户小程序扫码并授权手机号。登录挑战有效 2 分钟；会话固定 12 小时，不续期、不记住设备。</div><div id="pc-qr" class="card card-pad" style="margin:18px auto 0;min-height:244px;display:flex;align-items:center;justify-content:center"><span class="faint">正在创建登录挑战…</span></div><button class="btn btn--line" style="margin-top:16px" data-retry>重新生成</button></div>`;
    host.querySelector('[data-retry]').onclick = renderPCLogin;
    Api.beginPCLogin().then(login => {
      const code = host.querySelector('#pc-qr');
      if (!code) return;
      try {
        const canvas = document.createElement('canvas');
        canvas.setAttribute('aria-label', 'PC 登录二维码');
        window.PCQRCode.render(canvas, login.qr_payload, 228);
        code.replaceChildren(canvas);
      } catch (error) {
        code.innerHTML = `<span class="faint">二维码生成失败：${window.Toast.esc(error.message)}</span>`;
        return;
      }
      poll(login.login_id, login.poll_secret, login.expires_at);
    }).catch(e => {
      const code = host.querySelector('#pc-qr');
      if (code) code.textContent = e.message;
    });
  }

  function poll(loginID, pollSecret, expiresAt) {
    if (Date.now() >= Date.parse(expiresAt)) {
      window.Toast.show('登录挑战已过期，请重新生成', { icon: 'warn' });
      return;
    }
    Api.pollPCLogin(loginID, pollSecret).then(result => {
      if (result.state === 'APPROVED') { location.reload(); return; }
      setTimeout(() => poll(loginID, pollSecret, expiresAt), 1500);
    }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
  }

  // 页面间跳转与外壳刷新
  window.App = {
    go(route) { location.hash = '#/' + route; },
    reload() { go(current, true); },
    refreshChrome() { renderStatusPill(); renderAccount(); },
  };
})();
