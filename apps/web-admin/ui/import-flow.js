/* 批量导入三步流程外壳（PRD §6.13.1）—— 两个导入页共用。
   页面只提供模板定义与契约方法，不接触文件解析。 */
(function () {
  const T = window.Table, I = window.Icon;

  /* cfg: { title, hint, columns:[{name,required,note}], sample, templateRows,
            preview(file, images), commit(token), backRoute, extra(preview) } */
  function render(el, cfg) {
    const state={file:null,preview:null,busy:false,generation:0,result:null,batchId:null};
    const isProduct=cfg.backRoute==='products';
    function paint(){
      if(!el.isConnected)return;
      if(state.batchId){paintResult();return;}
      el.innerHTML=`<div class="page-head"><span class="ph-s grow">${cfg.hint}</span><button class="btn btn--line" data-back>返回列表</button></div>
        <div class="card card-pad imp-card"><div class="row gap12"><div class="fld-lb grow">第一步 · 模板列</div><button class="btn btn--line btn--sm" data-template>下载 .xlsx 模板</button></div><div class="imp-cols">${cfg.columns.map(c=>`<span class="imp-col${c.required?' req':''}">${T.esc(c.name)}${c.required?' *':''}</span>`).join('')}</div><div class="fld-hint">首行必须是表头，按列名匹配。未知列会在预览中提示。${cfg.sample?T.esc(cfg.sample):''}</div>${cfg.columns.filter(c=>c.note).map(c=>`<div class="imp-note">${T.esc(c.name)}：${T.esc(c.note)}</div>`).join('')}</div>
        <div class="card card-pad imp-card"><div class="fld-lb">第二步 · 选择文件</div><div class="imp-pick"><input type="file" accept=".xlsx" ${state.busy?'disabled':''}><span>${state.file?T.esc(state.file.name):`只接受 .xlsx，最多 ${cfg.maxRows} 行，文件不超过 10 MiB`}</span></div><div role="status">${state.busy?'处理中…':''}</div></div><div data-preview></div>`;
      el.querySelector('[data-back]').onclick=()=>window.App.go(cfg.backRoute);
      el.querySelector('[data-template]').onclick=()=>downloadTemplate(cfg);
      el.querySelector('input').onchange=async e=>{
        const file=e.target.files[0];if(!file)return;const version=++state.generation;
        state.file=file;state.preview=null;state.busy=true;paint();
        try{const preview=await cfg.preview(file);if(version===state.generation&&el.isConnected)state.preview=preview;}
        catch(e){if(version===state.generation&&el.isConnected)window.Toast.show(e.message,{icon:'warn'});}
        finally{if(version===state.generation){state.busy=false;paint();}}
      };
      const p=state.preview;if(!p)return;
      const total=p.added+p.updated;
      const hasProductRows=!isProduct || (Array.isArray(p.rows)&&p.rows.length>0&&p.rows.every(r=>r.product));
      el.querySelector('[data-preview]').innerHTML=`<div class="card card-pad imp-card"><div class="fld-lb">第三步 · 预览并确认导入</div><div class="imp-counts"><span>新增 <b>${p.added}</b> 条</span><span>更新 <b>${p.updated}</b> 条</span><span>异常 <b>${p.errors.length}</b> 条</span></div>${cfg.extra?cfg.extra(p):''}${p.ignoredColumns.length?`<p>已忽略未知列：${p.ignoredColumns.map(T.esc).join('、')}</p>`:''}${p.errors.length?`<details class="imp-errs" open><summary>异常行（将跳过）</summary>${p.errors.map(e=>`<div class="imp-err"><span>第 ${e.row} 行</span>${T.esc(e.reason)}</div>`).join('')}</details>`:''}<div data-product-preview></div><div class="imp-acts"><button class="btn btn--line" data-cancel ${state.busy?'disabled':''}>取消并重选文件</button><button class="btn btn--primary" data-commit ${!total||state.busy||!hasProductRows?'disabled':''}>${state.busy?'正在导入…':`确认导入 ${total} 条`}</button></div></div>`;
      if(isProduct){
        const host=el.querySelector('[data-product-preview]');
        if(!hasProductRows)host.innerHTML='<p class="text-danger">当前后端尚未提供可编辑预览，请更新导入服务后重新选择文件。</p>';
        else {
          host.innerHTML=T.render({cols:[
            {t:'图片',w:'72px',render:r=>{const image=(r.product.images||[])[0];return `<span class="thumb-frame"><span class="image-placeholder">暂无图片</span>${image?`<img class="thumb" src="${T.esc(window.Api.imgUrl(image.object_key))}" alt="" onerror="this.remove()">`:''}</span>`;}},
            {t:'菜品名称',render:r=>T.esc(r.product.name||'—')},
            {t:'分类',render:r=>T.esc(r.product.category||'—')},
            {t:'餐段',render:r=>T.esc(r.product.meal_input||'—')},
            {t:'售价（元）',render:r=>T.esc(r.product.price_input||'—')},
            {t:'描述',render:r=>T.esc(r.product.description||'—')},
            {t:'校验结果',render:r=>r.outcome==='ERROR'?`<span class="text-danger">${T.esc(r.reason)}</span>`:T.pill('待新增','ok')},
            {t:'操作',w:'180px',render:r=>`<button class="btn btn--sm btn--line" data-act="edit" data-id="${r.row}" ${state.busy?'disabled':''}>编辑</button><button class="btn btn--sm btn--line" data-act="images" data-id="${r.row}" ${state.busy?'disabled':''}>上传图片</button>`}
          ],rows:p.rows});
          T.bind(host,{edit:id=>editDraft(Number(id),false),images:id=>editDraft(Number(id),true)});
        }
      }
      el.querySelector('[data-cancel]').onclick=()=>{state.generation++;state.file=null;state.preview=null;paint();};
      el.querySelector('[data-commit]').onclick=async()=>{
        if(state.busy||!hasProductRows)return;state.busy=true;paint();
        try{const result=await cfg.commit(p.token);state.batchId=result.batchId;state.result=null;window.Toast.show(result.duplicate?'该批次已导入，正在读取结果':'导入完成');}
        catch(e){window.Toast.show(e.message,{icon:'warn'});}
        finally{state.busy=false;if(el.isConnected){paint();}}
      };
    }
    function editDraft(rowNumber, imagesOnly){
      if(state.busy)return;
      const preview=state.preview, version=state.generation;
      const source=preview.rows.find(r=>r.row===rowNumber);if(!source)return;
      const product=source.product;
      let images=(product.images||[]).map(i=>i.object_key),uploading=false;
      const initial=JSON.stringify(images);
      const field=(key,label,value)=>`<div class="fld"><label class="fld-lb" for="draft-${key}">${label}</label><input class="inp" id="draft-${key}" value="${T.esc(value||'')}"></div>`;
      window.Drawer.open({title:imagesOnly?'上传预览图片':'编辑导入预览',guardChanges:true,isDirty:()=>JSON.stringify(images)!==initial,
        bodyHtml:`${imagesOnly?`<p>${T.esc(product.name)}</p>`:field('name','菜品名称 *',product.name)+field('category','分类 *',product.category)+field('price','售价（元）*',product.price_input)+field('meal','餐段可售 *（全天 / 午餐 / 晚餐）',product.meal_input)+`<div class="fld"><label class="fld-lb" for="draft-description">描述</label><textarea class="txa" id="draft-description">${T.esc(product.description||'')}</textarea></div>`}<div data-draft-images></div><p class="fld-hint">保存仅更新预览，确认导入后才创建菜品。</p>`,
        footerHtml:'<button class="btn btn--line" data-cancel>取消</button><button class="btn btn--primary" data-save>保存到预览</button>',
        onMount(root,close){
          const button=root.querySelector('[data-save]');
          const editor=window.ImageEditor.mount(root.querySelector('[data-draft-images]'),images,value=>{images=value;},value=>{uploading=value;button.disabled=value;button.textContent=value?'上传中…':'保存到预览';});
          root.querySelector('[data-cancel]').onclick=close;
          button.onclick=window.FormUI.submit(button,async()=>{
            if(uploading||state.busy)return;
            const draft=Object.assign({},product,{images:images.map(object_key=>({object_key}))});
            if(!imagesOnly){
              draft.name=root.querySelector('#draft-name').value.trim();draft.category=root.querySelector('#draft-category').value.trim();
              draft.price_input=root.querySelector('#draft-price').value.trim();draft.meal_input=root.querySelector('#draft-meal').value.trim();draft.description=root.querySelector('#draft-description').value.trim();
            }
            const rows=preview.rows.map(r=>r.row===rowNumber?draft:r.product);
            const sheetRows=[['菜品名称','售价','分类','餐段可售','描述'],...rows.map(r=>[r.name||'',r.price_input||'',r.category||'',r.meal_input||'',r.description||''])];
            const imageMap={};rows.forEach((r,i)=>{if(r.images&&r.images.length)imageMap[i+2]=r.images;});
            const file=new File([workbook(sheetRows)],state.file.name,{type:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'});
            state.busy=true;editor.setLocked(true);
            root.querySelectorAll('input:not([type=file]),textarea').forEach(n=>n.disabled=true);
            try {
              const next=await cfg.preview(file,imageMap);
              if(!root.isConnected||!el.isConnected||version!==state.generation)return;
              if(!next.rows||!next.rows.every(r=>r.product))throw new Error('后端尚未支持可编辑预览，请更新服务后重试');
              state.preview=next;close(true);window.Toast.show('预览已更新，请核对后确认导入');
            } finally {state.busy=false;editor.setLocked(false);root.querySelectorAll('input:not([type=file]),textarea').forEach(n=>n.disabled=false);if(el.isConnected)paint();}
          },'正在校验…');
        }
      });
    }
    async function paintResult(){
      if(!el.isConnected)return;
      el.innerHTML=`<div class="page-head"><span class="ph-s grow">本次导入结果</span><button class="btn btn--line" data-all>查看全部${isProduct?'菜品':'员工'}</button></div><div data-results role="status">正在读取本次导入结果…</div>`;
      el.querySelector('[data-all]').onclick=()=>window.App.go(cfg.backRoute);
      try{
        const result=state.result||await window.Api.importResult(state.batchId);if(!el.isConnected)return;state.result=result;
        const host=el.querySelector('[data-results]');
        const rows=result.rows;if(Array.isArray(rows))rows.forEach(r=>{if(r.image===undefined&&r.images&&r.images[0])r.image=r.images[0].object_key;});if(!Array.isArray(rows))throw new Error('本批次结果暂不可用，请重试');
        host.innerHTML=`<p class="imp-note">本次成功导入 ${rows.length} 条，跳过 ${result.skipped_count||0} 条异常记录。</p>`+T.render({cols:isProduct?[
          {t:'图片',w:'72px',render:r=>`<span class="thumb-frame"><span class="image-placeholder">暂无图片</span>${r.image?`<img class="thumb" src="${T.esc(window.Api.imgUrl(r.image))}" alt="" onerror="this.remove()">`:''}</span>`},
          {t:'菜品名称',render:r=>T.esc(r.name)},{t:'分类',render:r=>T.esc(r.category)},{t:'餐段',render:r=>T.esc(window.Api.MEAL_LABEL[r.meal]||r.meal)},{t:'售价',render:r=>T.money((r.price_cents/100).toFixed(2))},{t:'操作',render:r=>`<button class="btn btn--sm btn--line" data-act="images" data-id="${T.esc(r.id)}">上传图片</button>`}
        ]:[{t:'姓名',render:r=>T.esc(r.name)},{t:'手机号',render:r=>T.esc(r.phone_masked)},{t:'结果',render:r=>T.pill(r.outcome==='ADD'?'新增':'更新','ok')}],rows});
        T.bind(host,{images(id,button){button.onclick=null;window.FormUI.submit(button,()=>window.ImageEditor.edit(id,async()=>{const product=await window.Api.getProduct(id);const row=rows.find(r=>r.id===id);row.image=product.imgs[0]||'';if(el.isConnected)paintResult();},()=>el.isConnected),'读取中…')();}});
      }catch(e){if(!el.isConnected)return;el.querySelector('[data-results]').innerHTML=`<p>${T.esc(e.message)}</p><button class="btn btn--line" data-retry>重新读取结果</button>`;el.querySelector('[data-retry]').onclick=paintResult;}
    }
    paint();
  }

  function downloadTemplate(cfg) {
    const rows = Array.isArray(cfg.templateRows) ? cfg.templateRows : [];
    if (rows.length !== 1 || !Array.isArray(rows[0]) || !rows[0].length) {
      return window.Toast.show('模板暂时不可用', { icon: 'warn' });
    }
    const bytes = workbook(rows);
    const blob = new Blob([bytes], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = cfg.backRoute === 'products' ? '菜品批量导入模板.xlsx' : '员工白名单批量导入模板.xlsx';
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }

  // 只生成表头工作簿；业务文件解析始终由服务端完成。
  function workbook(rows) {
    const files = [
      ['[Content_Types].xml', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`],
      ['_rels/.rels', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`],
      ['xl/workbook.xml', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="导入模板" sheetId="1" r:id="rId1"/></sheets></workbook>`],
      ['xl/_rels/workbook.xml.rels', `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`],
      ['xl/worksheets/sheet1.xml', sheet(rows)],
    ];
    return zip(files);
  }

  function sheet(rows) {
    const body = rows.map((row, y) => `<row r="${y + 1}">${row.map((value, x) => `<c r="${column(x)}${y + 1}" t="inlineStr"><is><t>${xml(value)}</t></is></c>`).join('')}</row>`).join('');
    return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${body}</sheetData></worksheet>`;
  }

  function column(index) {
    let value = index + 1, out = '';
    while (value) { value -= 1; out = String.fromCharCode(65 + (value % 26)) + out; value = Math.floor(value / 26); }
    return out;
  }

  function xml(value) {
    return String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&apos;');
  }

  function zip(files) {
    const encoder = new TextEncoder();
    const locals = [], centrals = [];
    let offset = 0;
    files.forEach(([name, value]) => {
      const nameBytes = encoder.encode(name), data = encoder.encode(value), checksum = crc32(data);
      const local = new Uint8Array(30 + nameBytes.length + data.length);
      const lv = new DataView(local.buffer);
      lv.setUint32(0, 0x04034b50, true); lv.setUint16(4, 20, true); lv.setUint16(8, 0, true);
      lv.setUint32(14, checksum, true); lv.setUint32(18, data.length, true); lv.setUint32(22, data.length, true); lv.setUint16(26, nameBytes.length, true);
      local.set(nameBytes, 30); local.set(data, 30 + nameBytes.length); locals.push(local);
      const central = new Uint8Array(46 + nameBytes.length);
      const cv = new DataView(central.buffer);
      cv.setUint32(0, 0x02014b50, true); cv.setUint16(4, 20, true); cv.setUint16(6, 20, true); cv.setUint16(10, 0, true);
      cv.setUint32(16, checksum, true); cv.setUint32(20, data.length, true); cv.setUint32(24, data.length, true); cv.setUint16(28, nameBytes.length, true); cv.setUint32(42, offset, true);
      central.set(nameBytes, 46); centrals.push(central); offset += local.length;
    });
    const centralSize = centrals.reduce((sum, item) => sum + item.length, 0);
    const end = new Uint8Array(22), ev = new DataView(end.buffer);
    ev.setUint32(0, 0x06054b50, true); ev.setUint16(8, files.length, true); ev.setUint16(10, files.length, true); ev.setUint32(12, centralSize, true); ev.setUint32(16, offset, true);
    const result = new Uint8Array(offset + centralSize + end.length);
    let cursor = 0;
    [...locals, ...centrals, end].forEach(part => { result.set(part, cursor); cursor += part.length; });
    return result;
  }

  function crc32(data) {
    let value = 0xffffffff;
    for (const byte of data) {
      value ^= byte;
      for (let bit = 0; bit < 8; bit += 1) value = (value >>> 1) ^ ((value & 1) ? 0xedb88320 : 0);
    }
    return (value ^ 0xffffffff) >>> 0;
  }

  window.ImportFlow = { render };
})();
