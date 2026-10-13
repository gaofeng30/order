/* 请求进行中只允许一次提交；只恢复本实例按钮。 */
(function () {
  function submit(button, action, label) {
    let busy = false;
    return async () => {
      if (busy) return;
      busy = true; const text = button.textContent;
      button.disabled = true; button.textContent = label || '提交中…';
      try { await action(); }
      catch (error) { window.Toast.show(error.message || '操作失败，请重试', {icon:'warn'}); }
      finally { busy = false; button.disabled = false; button.textContent = text; }
    };
  }
  function copyButtons(root) {
    root.querySelectorAll('[data-copy]').forEach(button => {
      button.onclick = submit(button, async () => {
        if (!navigator.clipboard) throw new Error('当前浏览器无法复制，请手动复制号码');
        await navigator.clipboard.writeText(button.dataset.copy);
        window.Toast.show('已复制');
      }, '复制中…');
    });
  }
  window.FormUI = { submit, copyButtons };
})();
