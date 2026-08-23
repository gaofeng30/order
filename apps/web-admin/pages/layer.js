/* 开屏图层 —— 对应 apps/wechat-miniprogram/pages/admin-layer
   上传透明 PNG → 在等比缩放的手机预览上拖拽定位、滑杆调大小 →
   保存后展示在「业务选择页 brand」与「身份选择页 launch」（两页共用一套配置）。
   PC 端优势：装饰图原本就存在电脑里，可直接拖拽上传。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  // PC 屏放得下，预览按真机 1:1 渲染：商户判断装饰图位置时所见即所得，
  // 不必在缩放过的画面上估算。坐标仍以屏宽/屏高比例存储，与缩放比无关。
  const SCREEN_W = 375, SCREEN_H = 667, SCALE = 1;
  const PW = Math.round(SCREEN_W * SCALE);
  const PH = Math.round(SCREEN_H * SCALE);
  const DEFAULT_LAYER = { src: '', enabled: false, size: 0.3, cx: 0.5, cy: 0.35, ar: 1, v: 1 };

  let cfg = null;      // 当前编辑态
  let mock = 'brand';  // 预览背景
  let dirtyRemove = false;

  function render(el) {
    Api.getLayer().then(c => {
      cfg = Object.assign({}, c);
      dirtyRemove = false;
      paint(el);
    });
  }

  function paint(el) {
    el.innerHTML =
      `<div class="layer-cols">
         <div class="lay-stage">
           <div class="segs" style="margin-bottom:14px">
             <span class="seg${mock === 'brand' ? ' on' : ''}" data-m="brand">业务选择页</span>
             <span class="seg${mock === 'launch' ? ' on' : ''}" data-m="launch">身份选择页</span>
           </div>
           <div class="phone" id="phone" style="width:${PW}px;height:${PH}px">
             ${mockBg()}
             ${cfg.src ? `<img class="lay-img" id="lay-img" src="${Api.imgUrl(cfg.src)}" draggable="false">` : ''}
           </div>
           <div class="faint" style="font-size:12px;margin-top:10px">按 iPhone 逻辑分辨率 ${SCREEN_W}×${SCREEN_H} 1:1 预览 · 拖动图片调整位置</div>
         </div>

         <div class="lay-panel">
           <div class="sec-h"><span class="t">图片</span></div>
           <div class="card card-pad">
             <div class="drop" id="drop">
               ${I.svg('upload', 22, '#8f9384')}
               <div class="drop-t">拖拽 PNG 到此处，或点击选择</div>
               <div class="faint" style="font-size:11.5px">仅支持透明背景 PNG</div>
             </div>
             ${cfg.src ? `<button class="btn btn--danger btn--block btn--sm" id="rm" style="margin-top:12px">移除图层</button>` : ''}
             <input type="file" id="file" accept="image/png" hidden>
           </div>

           <div class="sec-h" style="margin-top:8px"><span class="t">位置与大小</span></div>
           <div class="card card-pad">
             <div class="fld">
               <div class="fld-lb">显示宽度 <span class="grow"></span><b class="tnum">${Math.round(cfg.size * 100)}%</b></div>
               <input type="range" class="rng" id="size" min="15" max="60" value="${Math.round(cfg.size * 100)}">
               <div class="fld-hint">占屏幕宽度的比例。在左侧预览上直接拖动图片可调整位置。</div>
             </div>
             <div class="kv"><span class="k">中心 X</span><span class="v tnum" id="vx">${cfg.cx.toFixed(2)}</span></div>
             <div class="kv"><span class="k">中心 Y</span><span class="v tnum" id="vy">${cfg.cy.toFixed(2)}</span></div>
           </div>

           <div class="sec-h" style="margin-top:8px"><span class="t">启用</span></div>
           <div class="card card-pad row gap12">
             <button class="sw${cfg.enabled ? ' on' : ''}" id="en"></button>
             <span class="grow" style="font-size:13px">${cfg.enabled ? '已启用，用户端开屏会显示该图层' : '未启用，用户端开屏不显示'}</span>
           </div>

           <div class="card card-pad set-note" style="margin-top:16px">
             ${I.svg('warn', 16, '#a4873f')}
             <div>
               图层同时作用于「业务选择页」与「身份选择页」，两页共用一套位置与大小，不能分别配置。<br>
               图片需为透明背景 PNG，否则会挡住底部按钮。上传后先在左侧确认两个页面都不遮挡文字，再保存。
             </div>
           </div>

           <div class="form-foot" style="padding-left:0;padding-right:0">
             <button class="btn btn--line" id="reset">还原</button>
             <button class="btn btn--primary" id="save">保存图层</button>
           </div>
         </div>
       </div>`;

    layoutImg(el);
    bind(el);
  }

  /* 仿真稿逐字移植自 apps/wechat-miniprogram/pages/admin-layer 的 mock（样式取自 brand.wxss / launch.wxss），
     rpx 一律按 1rpx = 0.5px 换算。预览是 1:1，所以这里的 px 与真机 px 一一对应。
     必须用真实页面内容而不是占位块：这一页的用途就是判断装饰图会不会压住 Logo 与按钮。
     徽标 emblem.png 是抠好的透明 PNG，与手机端一致垫白色圆角卡，不裸贴在深蓝底上。 */
  const EMBLEM = '../wechat-miniprogram/assets/emblem.png';

  function mockBg() {
    if (mock === 'launch') {
      return `<div class="mock mock-dark mk-launch">
          <span class="ring r1"></span><span class="ring r2"></span><span class="ring r3"></span>
          <div class="m-brand">
            <div class="m-brand-logo"><img class="m-brand-emblem" src="${EMBLEM}" alt=""></div>
            <div>
              <div class="m-brand-name serif">绥安食品</div>
              <div class="m-brand-sub">在地手作 · 到店自提</div>
            </div>
          </div>
          <div class="m-hero">
            <div class="m-welcome">欢迎光临</div>
            <div class="m-big serif"><div>在地手作</div><div>到店自提</div></div>
            <div class="m-lead"><div>古法慢工的绥安味道，</div><div>循味而来，各得其门。</div></div>
          </div>
          <div class="m-cards">
            <div class="m-id primary">
              <div class="m-id-ico">${I.svg('bag', 28, '#ffffff')}</div>
              <div class="m-id-body"><div class="m-id-name serif">用户端</div><div class="m-id-desc">浏览菜单 · 在线点单 · 到店扫码取餐</div></div>
              <div class="m-id-go primary">${I.svg('arrowRight', 18, '#3d6b2f')}</div>
            </div>
            <div class="m-id">
              <div class="m-id-ico ghost">${I.svg('store', 28, '#ffffff')}</div>
              <div class="m-id-body"><div class="m-id-name serif">商户端</div><div class="m-id-desc">接单 · 制作 · 核销 · 菜品与经营管理</div></div>
              <div class="m-id-go">${I.svg('arrowRight', 18, '#ffffff')}</div>
            </div>
            <div class="m-ver">${T.esc(Api.storeView().name)}　小程序 v1.0</div>
          </div>
        </div>`;
    }
    return `<div class="mock mock-dark mk-brand">
        <span class="ring r1"></span><span class="ring r3"></span>
        <div class="m-logo-mid">
          <div class="m-logo-card"><img class="m-logo-emblem" src="${EMBLEM}" alt=""></div>
        </div>
        <div class="m-opts">
          <div class="m-opt ready">
            <div class="m-opt-ico">${I.svg('bowl', 27, '#ffffff')}</div>
            <div class="m-opt-body"><div class="m-opt-name serif">绥安食品</div><div class="m-opt-desc">到店点单 · 预约取餐</div></div>
            <div class="m-opt-go">${I.svg('arrowRight', 18, '#3d6b2f')}</div>
          </div>
          <div class="m-opt off">
            <div class="m-opt-ico">${I.svg('washer', 27, 'rgba(255,255,255,.5)')}</div>
            <div class="m-opt-body"><div class="m-opt-name serif">绥安洗衣</div><div class="m-opt-desc">即将上线</div></div>
          </div>
          <div class="m-opt off">
            <div class="m-opt-ico">${I.svg('car', 27, 'rgba(255,255,255,.5)')}</div>
            <div class="m-opt-body"><div class="m-opt-name serif">绥安洗车</div><div class="m-opt-desc">即将上线</div></div>
          </div>
          <div class="m-ver">绥安集团 · 本地生活服务　v1.0</div>
        </div>
      </div>`;
  }

  // 由 size 与中心比例计算图片几何，并钳制在预览画布内（与小程序 geometry() 同规则）
  function layoutImg(el) {
    const img = el.querySelector('#lay-img');
    if (!img) return;
    const w = cfg.size * PW;
    const h = w * cfg.ar;
    let x = cfg.cx * PW - w / 2;
    let y = cfg.cy * PH - h / 2;
    x = Math.max(0, Math.min(x, PW - w));
    y = Math.max(0, Math.min(y, PH - h));
    // 钳制后回写比例，保证保存值与所见一致
    cfg.cx = (x + w / 2) / PW;
    cfg.cy = (y + h / 2) / PH;
    img.style.width = w + 'px';
    img.style.height = h + 'px';
    img.style.left = x + 'px';
    img.style.top = y + 'px';
    const vx = el.querySelector('#vx'), vy = el.querySelector('#vy');
    if (vx) vx.textContent = cfg.cx.toFixed(2);
    if (vy) vy.textContent = cfg.cy.toFixed(2);
  }

  function bind(el) {
    el.querySelectorAll('[data-m]').forEach(n => {
      n.onclick = () => { mock = n.dataset.m; paint(el); };
    });

    const file = el.querySelector('#file');
    const drop = el.querySelector('#drop');
    drop.onclick = () => file.click();
    file.onchange = () => { if (file.files[0]) accept(el, file.files[0]); file.value = ''; };

    ['dragenter', 'dragover'].forEach(ev =>
      drop.addEventListener(ev, e => { e.preventDefault(); drop.classList.add('over'); }));
    ['dragleave', 'drop'].forEach(ev =>
      drop.addEventListener(ev, e => { e.preventDefault(); drop.classList.remove('over'); }));
    drop.addEventListener('drop', e => {
      const f = e.dataTransfer.files[0];
      if (f) accept(el, f);
    });

    const size = el.querySelector('#size');
    size.oninput = () => {
      cfg.size = Number(size.value) / 100;
      size.previousElementSibling.querySelector('b').textContent = size.value + '%';
      layoutImg(el);
    };

    el.querySelector('#en').onclick = () => {
      if (!cfg.enabled && !cfg.src) return window.Toast.show('请先上传图片', { icon: 'warn' });
      cfg.enabled = !cfg.enabled;
      paint(el);
    };

    const rm = el.querySelector('#rm');
    if (rm) rm.onclick = () => {
      window.Modal.confirm({
        title: '移除图层', body: '保存后将删除图片与位置设置。', okText: '移除', danger: true,
      }).then(yes => {
        if (!yes) return;
        // 与小程序一致：移除只是暂存状态，点「保存」才真正生效
        dirtyRemove = true;
        cfg = Object.assign({}, DEFAULT_LAYER);
        paint(el);
        window.Toast.show('已移除，保存后生效');
      });
    };

    el.querySelector('#reset').onclick = () => render(el);

    el.querySelector('#save').onclick = () => {
      const done = () => window.Toast.show('图层已保存', { icon: 'check' });
      if (dirtyRemove && !cfg.src) {
        Api.clearLayer().then(() => { dirtyRemove = false; done(); });
        return;
      }
      Api.saveLayer(cfg).then(() => { dirtyRemove = false; done(); });
    };

    bindDrag(el);
  }

  function accept(el, f) {
    if (f.type !== 'image/png') return window.Toast.show('请选择透明背景的 PNG 图片', { icon: 'warn' });
    Api.uploadImage(f).then(url => {
      const probe = new Image();
      probe.onload = () => {
        cfg.src = url;
        cfg.ar = probe.height / probe.width;
        cfg.enabled = true;
        dirtyRemove = false;
        if (cfg.cx === undefined) { cfg.cx = DEFAULT_LAYER.cx; cfg.cy = DEFAULT_LAYER.cy; }
        paint(el);
      };
      probe.onerror = () => window.Toast.show('图片读取失败', { icon: 'warn' });
      probe.src = url;
    });
  }

  // 预览上拖动图片：按下记录偏移，移动时改 left/top，松开折算回中心比例
  function bindDrag(el) {
    const img = el.querySelector('#lay-img');
    if (!img) return;
    img.addEventListener('pointerdown', e => {
      e.preventDefault();
      img.setPointerCapture(e.pointerId);
      img.classList.add('moving');
      const w = img.offsetWidth, h = img.offsetHeight;
      const dx = e.clientX - img.offsetLeft;
      const dy = e.clientY - img.offsetTop;

      const move = ev => {
        const x = Math.max(0, Math.min(ev.clientX - dx, PW - w));
        const y = Math.max(0, Math.min(ev.clientY - dy, PH - h));
        img.style.left = x + 'px';
        img.style.top = y + 'px';
        cfg.cx = (x + w / 2) / PW;
        cfg.cy = (y + h / 2) / PH;
        el.querySelector('#vx').textContent = cfg.cx.toFixed(2);
        el.querySelector('#vy').textContent = cfg.cy.toFixed(2);
      };
      const up = () => {
        img.classList.remove('moving');
        img.removeEventListener('pointermove', move);
        img.removeEventListener('pointerup', up);
      };
      img.addEventListener('pointermove', move);
      img.addEventListener('pointerup', up);
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['layer'] = { sub: '业务选择页与身份选择页的装饰图片，两页共用一套配置', render };
})();
