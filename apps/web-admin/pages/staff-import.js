/* 员工白名单批量导入（PRD §6.13.3）—— 按手机号覆盖更新，保留状态与统计字段。 */
(function () {
  const Api = window.Api;

  function render(el) {
    window.ImportFlow.render(el, {
      backRoute: 'staff',
      maxRows: Api.MAX_STAFF_IMPORT_ROWS,
      hint: '按模板整理员工后上传 .xlsx。手机号已在名单中的会被覆盖更新，其状态、加入时间、微信绑定与累计统计一律保留 —— 导入不会把已停用的员工重新启用。',
      sample: '林建国 | 13800006620',
      templateRows: [['姓名', '手机号']],
      columns: [
        { name: '姓名', required: true, note: '需与用户保存的默认取餐姓名一致；主手机号或附加手机号任意一个匹配本行号码即可享员工折扣' },
        { name: '手机号', required: true, note: '唯一识别键，11 位手机号' },
      ],
      preview: f => Api.previewStaffImport(f),
      commit: t => Api.commitStaffImport(t),
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['staff-import'] = { sub: '按模板批量维护员工名单', render };
})();
