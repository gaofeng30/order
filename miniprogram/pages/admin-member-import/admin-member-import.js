/* 会员名单批量导入 —— 二期能力，不在一期合同范围

   小程序读不到手机本地文件，唯一途径是 wx.chooseMessageFile（从微信聊天选择），
   因此商户需先把 CSV 发到文件传输助手。Excel 默认另存的 CSV 为 GBK 编码，
   readFile 按 utf-8 读会整片乱码，这里做显式检测并给出可执行的提示。 */
const api = require('../../utils/api.js');

const TPL_COLS = [
  { k: '手机号', req: true }, { k: '姓名', req: true }, { k: '等级', req: true },
  { k: '单位', req: false }, { k: '部门', req: false }, { k: '工号', req: false },
];
const TPL_HEADER = TPL_COLS.map(c => c.k).join(',');
// 表头别名 → 内部字段
const ALIAS = {
  手机号: 'phone', 手机: 'phone', 电话: 'phone', 联系电话: 'phone',
  姓名: 'name', 名字: 'name',
  等级: 'levelName', 会员等级: 'levelName', 级别: 'levelName',
  单位: 'org', 公司: 'org',
  部门: 'dept', 科室: 'dept',
  工号: 'jobNo', 员工号: 'jobNo',
};
const ORDER = ['phone', 'name', 'levelName', 'org', 'dept', 'jobNo'];
const maskPhone = p => (p && p.length === 11 ? p.slice(0, 3) + '****' + p.slice(7) : p);

// 逐字符解析，支持引号包裹与字段内逗号
function splitLine(line) {
  const out = [];
  let cur = '';
  let q = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (q) {
      if (ch === '"') {
        if (line[i + 1] === '"') { cur += '"'; i++; } else q = false;
      } else cur += ch;
    } else if (ch === '"') q = true;
    else if (ch === ',') { out.push(cur.trim()); cur = ''; }
    else cur += ch;
  }
  out.push(cur.trim());
  return out;
}

function parseCsv(text) {
  const lines = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n').filter(l => l.trim());
  if (!lines.length) return { rows: [], bad: '文件是空的' };
  // 乱码检测：UTF-8 解码失败会产生替换字符
  if (text.indexOf('�') > -1) {
    return { rows: [], bad: '文件编码不是 UTF-8，中文已乱码。请在 Excel 中「另存为 → CSV UTF-8」后重试。' };
  }

  const first = splitLine(lines[0]);
  const hasHeader = first.some(c => ALIAS[c]);
  let map;
  if (hasHeader) {
    map = first.map(c => ALIAS[c] || '');
  } else {
    map = ORDER.slice(0, first.length);
  }
  if (map.indexOf('phone') < 0 || map.indexOf('name') < 0 || map.indexOf('levelName') < 0) {
    return { rows: [], bad: '识别不到「手机号 / 姓名 / 等级」三列，请按模板列顺序整理后重试。' };
  }

  const rows = [];
  for (let i = hasHeader ? 1 : 0; i < lines.length; i++) {
    const cells = splitLine(lines[i]);
    const r = { line: i + 1, raw: lines[i].slice(0, 60) };
    map.forEach((k, idx) => { if (k) r[k] = (cells[idx] || '').replace(/\s/g, k === 'phone' ? '' : ' ').trim(); });
    rows.push(r);
  }
  return { rows, bad: '' };
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    step: 'idle',          // idle | preview
    tplCols: TPL_COLS,
    levelNames: [],
    fileName: '',
    res: { adds: [], updates: [], errors: [] },
    rows: [],
    errOpen: true,
  },

  onLoad() {
    api.listLevels().then(ls => this.setData({ levelNames: ls.map(l => l.name) }));
  },

  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },

  copyTpl() {
    wx.setClipboardData({ data: TPL_HEADER, success: () => this.toast('表头已复制', 'copy') });
  },

  pick() {
    wx.chooseMessageFile({
      count: 1,
      type: 'file',
      extension: ['csv'],
      success: r => {
        const f = r.tempFiles[0];
        if (!f) return;
        if (!/\.csv$/i.test(f.name)) return this.toast('请选择 .csv 文件', 'warn');
        this.readFile(f);
      },
      fail: () => { /* 用户取消 */ },
    });
  },

  readFile(f) {
    wx.getFileSystemManager().readFile({
      filePath: f.path,
      encoding: 'utf-8',
      success: r => {
        const { rows, bad } = parseCsv(r.data);
        if (bad) return this.toast(bad, 'warn');
        if (!rows.length) return this.toast('没有解析到数据行', 'warn');
        api.previewImport(rows).then(res => this.showPreview(f.name, res));
      },
      fail: () => this.toast('文件读取失败，请重新选择', 'warn'),
    });
  },

  showPreview(fileName, res) {
    const rows = res.adds.map(r => Object.assign({}, r, { isNew: true }))
      .concat(res.updates.map(r => Object.assign({}, r, { isNew: false })))
      .map(r => {
        const org = [r.org, r.dept].filter(Boolean).join(' · ');
        return Object.assign({}, r, {
          phoneMask: maskPhone(r.phone),
          orgLine: org ? ' · ' + org : '',
        });
      });
    this.setData({ step: 'preview', fileName, res, rows, errOpen: true });
  },

  toggleErr() { this.setData({ errOpen: !this.data.errOpen }); },
  reset() { this.setData({ step: 'idle', fileName: '', rows: [], res: { adds: [], updates: [], errors: [] } }); },

  commit() {
    if (!this.data.rows.length) return this.toast('没有可导入的数据', 'warn');
    api.commitImport(this.data.rows.map(r => ({
      phone: r.phone, name: r.name, levelId: r.levelId, org: r.org, dept: r.dept, jobNo: r.jobNo,
    }))).then(r => {
      this.toast(`已导入 · 新增 ${r.added} 更新 ${r.updated}`);
      setTimeout(() => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-members/admin-members' }) }), 800);
    }).catch(err => this.toast(err.message, 'warn'));
  },
});
