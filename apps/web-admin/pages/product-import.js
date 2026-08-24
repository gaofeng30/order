/* 菜品批量导入（PRD §6.13.2）—— 只新增不更新；分类不存在则自动新建；图片不进模板。 */
(function () {
  const Api = window.Api, T = window.Table;

  function render(el) {
    window.ImportFlow.render(el, {
      backRoute: 'products',
      maxRows: Api.MAX_IMPORT_ROWS,
      hint: '按模板整理菜品后上传 .xlsx。导入只新增不覆盖：同名菜品会被标为异常并跳过，改价请用菜品管理的批量调价。',
      sample: '商务双拼饭 | 32 | 今日套餐 | 全天 | 双主菜配例汤',
      templateRows: [['菜品名称', '售价', '分类', '餐段可售', '描述']],
      columns: [
        { name: '菜品名称', required: true },
        { name: '售价', required: true, note: '大于 0 的数值，单位元' },
        { name: '分类', required: true, note: '不存在的分类会自动新建，排在末尾且默认对用户端可见' },
        { name: '餐段可售', required: true, note: '只能填 全天 / 午餐 / 晚餐' },
        { name: '描述', required: false },
      ],
      preview: f => Api.previewProductImport(f),
      commit: t => Api.commitProductImport(t),
      extra: p => `
        ${p.newCategories && p.newCategories.length
          ? `<div class="imp-note">本次将新建分类：${p.newCategories.map(T.esc).join('、')}</div>` : ''}
        <div class="imp-note warn">图片不在模板中。导入的菜品先无图上架，请随后在菜品管理中逐个补图。</div>`,
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['product-import'] = { sub: '按模板批量新增菜品', render };
})();
