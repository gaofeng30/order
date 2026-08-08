/* ============================================================
   绥安食品 商户端 PC 网页版 —— 路由与外壳
   零构建：按 <script> 顺序加载，页面模块挂在 window.Pages 上。
   ============================================================ */
(function () {
  const Seed = window.Seed;
  const Api = window.Api;
  const clone = v => JSON.parse(JSON.stringify(v));

  /* ---------------- 内存态数据源（对应小程序 app.js 的 globalData） ---------------- */
  function initStore() {
    window.__store = {
      store: clone(Seed.STORE),
      aOrders: clone(Seed.ADMIN_ORDERS),
      // 菜品多图：种子只有单图，统一收敛到 imgs 数组，img 保留为封面
      menu: clone(Seed.MENU).map(m => Object.assign(m, { imgs: m.img ? [m.img] : [] })),
      cats: clone(Seed.ADMIN_CATS),
      settings: clone(Seed.SETTINGS),
      layer: clone(Seed.LAYER_DEFAULTS),
      // 二期能力
      levels: clone(Seed.LEVELS),
      members: clone(Seed.MEMBERS),
      coupons: clone(Seed.COUPONS),
      couponUsed: clone(Seed.MY_COUPON_USED),
    };
  }

  /* ---------------- 导航（替代小程序 admin-profile 的入口聚合页） ---------------- */
  const NAV = [
    { g: '经营', items: [
      { r: 'dashboard', t: '工作台', ic: 'dash' },
      { r: 'orders', t: '订单管理', ic: 'receipt' },
      { r: 'verify', t: '扫码核销', ic: 'scan' },
    ] },
    { g: '菜品', items: [
      { r: 'products', t: '菜品管理', ic: 'bowl' },
      { r: 'categories', t: '分类管理', ic: 'tag' },
    ] },
    { g: '门店', items: [
      { r: 'settings', t: '营业设置', ic: 'settings' },
      { r: 'layer', t: '开屏图层', ic: 'layers' },
    ] },
    // p2：二期能力分组。侧边栏不挂标签，范围提示统一由顶栏副标题承担
    { g: '会员与营销', p2: true, items: [
      { r: 'levels', t: '会员等级', ic: 'layers' },
      { r: 'members', t: '会员名单', ic: 'user' },
      { r: 'members/import', t: '批量导入名单', ic: 'box' },
      { r: 'coupons', t: '优惠券', ic: 'ticket' },
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
      if (it) return { title: it.t, group: grp.g, p2: !!grp.p2 };
    }
    return { title: '工作台', group: '经营', p2: false };
  }

  /* ---------------- 顶栏：营业状态 + 账号下拉（原 admin-profile 商户中心） ---------------- */
  const BIZ = ['营业中', '休息中', '已截单'];

  function renderStatusPill() {
    const s = window.__store.store.status;
    const el = document.getElementById('tb-status');
    el.className = 'pill pill--' + Api.statusTone(s);
    el.innerHTML = `<i class="pd"></i>${s}`;
  }

  function renderAccount() {
    const st = window.__store;
    const mg = Seed.MANAGER;
    document.getElementById('acct-pop').innerHTML =
      `<div class="acct-card">
         <span class="ring r1"></span><span class="ring r2"></span>
         <div class="ac-in">
           <span class="ac-logo"><img src="../miniprogram/assets/emblem.png" alt=""></span>
           <div>
             <div class="ac-nm">${Seed.STORE.name}</div>
             <div class="ac-sub">${Seed.STORE.branch} · ${mg.role} ${mg.name}</div>
           </div>
         </div>
       </div>
       <div class="acct-sec">
         <div class="acct-lb">营业状态</div>
         <div class="segs">
           ${BIZ.map(b => `<span class="seg${st.store.status === b ? ' on' : ''}" data-biz="${b}">${b}</span>`).join('')}
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
        });
        return;
      }
      const goTo = e.target.closest('[data-go]');
      if (goTo) { pop.classList.remove('open'); location.hash = '#/' + goTo.dataset.go; return; }
      if (e.target.closest('[data-logout]')) {
        pop.classList.remove('open');
        // P0 原型不做鉴权：登录与账号体系是正式期的事，此处仅占位提示
        window.Toast.show('原型阶段未接入账号体系 · 登录鉴权为正式期范围', { icon: 'warn' });
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
      meta.p2 ? '二期能力 · 不在一期合同范围' : (page.sub || '');

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
  initStore();
  renderNav();
  renderAccount();
  bindAccount();
  renderStatusPill();

  window.addEventListener('hashchange', () => go(parseHash()));
  if (!location.hash) location.hash = '#/dashboard';
  go(parseHash(), true);

  // 页面间跳转与外壳刷新（会员导入完成后回名单页等）
  window.App = {
    go(route) { location.hash = '#/' + route; },
    reload() { go(current, true); },
    refreshChrome() { renderStatusPill(); renderAccount(); },
  };
})();
