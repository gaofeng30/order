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
  const BIZ_LABEL = { open: '营业中', closed: '休息中', cutoff: '已截单' };
  let settingsWrite = null;
  let settingsDirty = false;

  function runSettingsWrite(action) {
    if (settingsWrite) return null;
    const pending = Promise.resolve().then(action).finally(() => {
      if (settingsWrite === pending) settingsWrite = null;
    });
    settingsWrite = pending;
    return pending;
  }

  function renderStatusPill() {
    const value = Api.storeView().status;
    const s = BIZ_LABEL[value] || value;
    const el = document.getElementById('tb-status');
    el.className = 'pill pill--' + Api.statusTone(s);
    el.innerHTML = `<i class="pd"></i>${s}`;
  }

  function renderAccountIdentity(account) {
    const name = account ? account.name : '未登录';
    const btn = document.getElementById('acct-btn');
    btn.querySelector('.acct-nm').textContent = name;
    btn.querySelector('.acct-av').textContent = account ? Array.from(name)[0] : '—';
    btn.title = name;
  }

  function renderAccount() {
    const view = Api.storeView();
    const mg = view.account || { name: '未登录', role: '—' };
    renderAccountIdentity(view.account);
    document.getElementById('acct-pop').innerHTML =
      `<div class="acct-card">
         <span class="ring r1"></span><span class="ring r2"></span>
         <div class="ac-in">
           <span class="ac-logo"><img src="assets/logo.jpeg" alt=""></span>
           <div>
             <div class="ac-nm">${view.name}</div>
             <div class="ac-sub">${view.name} · ${mg.role} ${mg.name}</div>
           </div>
         </div>
       </div>
       <div class="acct-sec">
         <div class="acct-lb">营业状态 · 点击立即生效</div>
         <div class="segs">
           ${BIZ.map(b => `<span class="seg${(BIZ_LABEL[view.status] || view.status) === b ? ' on' : ''}" data-biz="${b}">${b}</span>`).join('')}
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
        if (settingsWrite || settingsDirty) {
          window.Toast.show(settingsWrite ? '正在保存设置，请稍候' : '请先处理营业设置中的未保存修改', { icon: 'warn' });
          return;
        }
        const write = runSettingsWrite(() => Api.setStoreStatus(biz.dataset.biz));
        if (current === 'settings') go(current, true);
        write.then(s => {
          renderStatusPill();
          renderAccount();
          window.Toast.show(`已切换为「${s}」`, { icon: 'power' });
          if (current === 'dashboard') go(current, true);
        }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        return;
      }
      const goTo = e.target.closest('[data-go]');
      if (goTo) { pop.classList.remove('open'); location.hash = '#/' + goTo.dataset.go; return; }
      if (e.target.closest('[data-logout]')) {
        pop.classList.remove('open');
        Api.logout().then(showLogin).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
      }
    });
  }

  /* ---------------- 路由 ---------------- */
  let current = '';
  let loginVisible = false;
  let loginGeneration = 0, loginTimer = null, routeGeneration = 0;

  function showLogin() {
    if (loginVisible) return;
    loginVisible = true;
    renderAccountIdentity(null);
    current = '';
    window.Drawer.close(true); window.Modal.close();
    const pop = document.getElementById('acct-pop');
    pop.classList.remove('open'); pop.innerHTML = '';
    document.getElementById('tb-title').textContent = '登录';
    document.getElementById('tb-sub').textContent = '';
    document.getElementById('tb-status').textContent = '未登录';
    renderPCLogin();
  }

  window.addEventListener('pc-session-expired', showLogin);

  function parseHash() {
    const h = (location.hash || '').replace(/^#\/?/, '').trim();
    return window.Pages[h] ? h : 'dashboard';
  }

  async function go(route, force) {
    if (loginVisible) return;
    if (!force && route === current) return;
    const generation = ++routeGeneration;
    const pageBefore = window.Pages[current];
    const dirty = settingsDirty || (pageBefore && pageBefore.isDirty && pageBefore.isDirty());
    if (dirty && !await window.Modal.confirm({title:'退出编辑',body:'有未保存的修改，确定退出？',cancelText:'继续编辑',okText:'放弃修改'})) {
      if (generation === routeGeneration) history.replaceState(null, '', '#/' + current);
      return;
    }
    if (generation !== routeGeneration) return;
    if (!await window.Drawer.canClose()) {history.replaceState(null, '', '#/' + current); return;}
    if (generation !== routeGeneration) return;
    if (pageBefore && pageBefore.dispose) pageBefore.dispose();
    current = route;
    settingsDirty = false;
    window.Drawer.close(true);
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
  const navToggle = document.getElementById('nav-toggle');
  const narrow = matchMedia('(max-width: 1100px)');
  function collapseNav(collapsed) {
    document.body.classList.toggle('nav-collapsed', collapsed);
    navToggle.setAttribute('aria-expanded', String(!collapsed));
    navToggle.setAttribute('aria-label', collapsed ? '展开导航' : '收起导航');
    navToggle.textContent = collapsed ? '☰' : '‹';
  }
  collapseNav(narrow.matches);
  narrow.addEventListener('change', e => collapseNav(e.matches));
  navToggle.onclick = () => collapseNav(!document.body.classList.contains('nav-collapsed'));

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
      showLogin();
      return;
    }
    const host = document.getElementById('content');
    host.innerHTML = `<div class="card card-pad"><b>后台服务不可用</b><div class="faint" style="margin-top:8px">${e.message}</div><button class="btn btn--primary" style="margin-top:16px" onclick="location.reload()">重试</button></div>`;
    document.getElementById('tb-status').textContent = '不可用';
    renderAccount();
  });

  function renderPCLogin() {
    const generation = ++loginGeneration;
    clearTimeout(loginTimer);
    const host = document.getElementById('content');
    host.innerHTML = `<div class="card card-pad login-card"><b>主账号扫码登录</b><div class="faint" style="margin-top:8px">请用商户小程序扫码并授权手机号。二维码有效 2 分钟，登录后 12 小时内有效。</div><div id="pc-qr" class="login-qr"><span class="faint">正在生成二维码…</span></div><button class="btn btn--line" style="margin-top:16px" data-retry disabled>正在生成…</button></div>`;
    const retry = host.querySelector('[data-retry]'), code = host.querySelector('#pc-qr');
    const active = () => generation === loginGeneration && loginVisible && code.isConnected;
    retry.onclick = renderPCLogin;
    const expired = () => {
      if (!active()) return;
      const cover = document.createElement('div'); cover.className = 'qr-expired';
      cover.textContent = '二维码已过期，请刷新'; code.append(cover);
      retry.disabled = false; retry.textContent = '刷新二维码';
    };
    async function poll(login) {
      if (!active()) return;
      if (Date.now() >= Date.parse(login.expires_at)) {expired(); return;}
      try {
        const result = await Api.pollPCLogin(login.login_id, login.poll_secret, () => active() && Date.now() < Date.parse(login.expires_at));
        if (!active()) return;
        if (Date.now() >= Date.parse(login.expires_at) || result.state === 'EXPIRED') {expired(); return;}
        if (result.state === 'APPROVED') {location.reload(); return;}
        loginTimer = setTimeout(() => poll(login), 1500);
      } catch(e) {if (active()) {code.textContent = e.message; retry.disabled=false; retry.textContent='重新生成';}}
    }
    Api.beginPCLogin().then(login => {
      if (!active()) return;
      const canvas = document.createElement('canvas');
      canvas.setAttribute('aria-label','PC 登录二维码');
      window.PCQRCode.render(canvas, login.qr_payload, 228); code.replaceChildren(canvas);
      retry.disabled=false; retry.textContent='重新生成'; poll(login);
    }).catch(e => {if(active()){code.textContent=e.message;retry.disabled=false;retry.textContent='重新生成';}});
  }

  // 页面间跳转与外壳刷新
  window.App = {
    settingsWrite() { return settingsWrite; },
    runSettingsWrite,
    setSettingsDirty(value) { settingsDirty = !!value; },
    go(route) { location.hash = '#/' + route; },
    reload() { go(current, true); },
    refreshChrome() { if (!loginVisible) { renderStatusPill(); renderAccount(); } },
  };
})();
