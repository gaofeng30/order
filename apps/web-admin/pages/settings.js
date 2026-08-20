/* 营业设置 —— 对应 apps/wechat-miniprogram/pages/admin-settings
   小程序端的时间/截单/取餐点为「建设中」占位，PC 端收敛为可编辑表单。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  const BIZ = ['营业中', '休息中', '已截单'];
  const BIZ_C = { 营业中: '#3d6b2f', 已截单: '#8a8f7c', 休息中: '#c98b3a' };

  function render(el) {
    Api.getSettings().then(s => {
      const pts = window.Seed.PICKUP_POINTS;
      el.innerHTML =
        `<div class="set-cols">
           <div>
             <div class="sec-h"><span class="t">营业状态</span></div>
             <div class="card card-pad">
               <div class="biz-switch" id="biz">
                 ${BIZ.map(b => `<span class="biz-seg${s.status === b ? ' on' : ''}" data-b="${b}"
                    ${s.status === b ? `style="background:${BIZ_C[b]};color:#fff"` : ''}>${b}</span>`).join('')}
               </div>
               <div class="fld-hint" style="margin-top:10px">切换后用户端首页与点单页会同步显示当前状态，「已截单」时不再接受新订单。</div>
             </div>

             <div class="sec-h" style="margin-top:8px"><span class="t">餐段与取餐时间</span></div>
             <div class="card card-pad">
               ${s.mealPeriods.map(p => `
                 <div class="fld-row">
                   <div class="fld"><div class="fld-lb">${T.esc(p.name)} 截单</div><input class="inp tnum" type="time" data-mp="${p.key}" data-k="cutoff" value="${p.cutoff}"></div>
                   <div class="fld"><div class="fld-lb">取餐自</div><input class="inp tnum" type="time" data-mp="${p.key}" data-k="from" value="${p.from}"></div>
                   <div class="fld"><div class="fld-lb">至</div><input class="inp tnum" type="time" data-mp="${p.key}" data-k="to" value="${p.to}"></div>
                 </div>`).join('')}
               <div class="fld">
                 <div class="fld-lb">取餐时间粒度（分钟）</div>
                 <input class="inp tnum" type="number" id="f-step" min="5" step="5" value="${s.pickupStepMin}">
                 <div class="fld-hint">取餐时间点由「取餐自 / 至」与该粒度推导。取餐时间是约定时刻，不是必须到场的窗口——商品备好后推送提醒。</div>
               </div>
               <div class="fld" style="margin-bottom:0">
                 <div class="fld-lb">取餐地点</div>
                 <select class="sel" id="f-pt">
                   ${pts.map(p => `<option value="${T.esc(p.name)}"${s.pickupPoint === p.name ? ' selected' : ''}>${T.esc(p.name)} · ${T.esc(p.addr)}</option>`).join('')}
                 </select>
                 <div class="fld-hint">取餐点显示在用户端首页与订单凭证上：${T.esc(window.Seed.STORE.pickupWindow)}</div>
               </div>
             </div>
           </div>

           <div>
             <div class="sec-h"><span class="t">门店公告</span><span class="more">展示在用户端首页</span></div>
             <div class="card card-pad">
               <textarea class="txa" id="f-notice" style="min-height:180px" placeholder="例如：今日卤味新鲜出锅，欢迎到店自提～">${T.esc(s.notice)}</textarea>
               <div class="fld-hint">公告为纯文本，用户端首页整段展示，建议 40 字以内。</div>
             </div>

             <div class="card card-pad set-note">
               ${I.svg('warn', 16, '#a4873f')}
               <div>截单时刻按餐段固定，餐段内全部取餐时间共用，不随取餐时间滚动。自动停止接单与取餐前 30 分钟自动开做的定时任务属一期后端范围。</div>
             </div>
           </div>
         </div>

         <div class="form-foot">
           <button class="btn btn--primary" id="save">保存设置</button>
         </div>`;

      let status = s.status;
      el.querySelectorAll('[data-b]').forEach(n => {
        n.onclick = () => {
          status = n.dataset.b;
          el.querySelectorAll('[data-b]').forEach(x => {
            const on = x.dataset.b === status;
            x.classList.toggle('on', on);
            x.setAttribute('style', on ? `background:${BIZ_C[status]};color:#fff` : '');
          });
        };
      });

      el.querySelector('#save').onclick = () => {
        const mealPeriods = s.mealPeriods.map(p => Object.assign({}, p));
        el.querySelectorAll('[data-mp]').forEach(n => {
          const mp = mealPeriods.find(x => x.key === n.dataset.mp);
          if (mp) mp[n.dataset.k] = n.value;
        });
        Api.saveSettings({
          status,
          pickupStepMin: Number(el.querySelector('#f-step').value),
          mealPeriods,
          pickupPoint: el.querySelector('#f-pt').value,
          notice: el.querySelector('#f-notice').value,
        }).then(() => {
          window.App.refreshChrome();
          window.Toast.show('设置已保存', { icon: 'check' });
        }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
      };
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['settings'] = { sub: '状态、餐段截单、取餐时间、取餐点与门店公告', render };
})();
