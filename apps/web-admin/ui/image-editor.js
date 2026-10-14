/* 菜品图片：正方形裁剪、预览、上传。开屏图层不走此流程。 */
(function () {
  async function crop(file, isCurrent) {
    if (!/^image\//.test(file.type)) throw new Error('请选择图片文件');
    const revision = window.Modal.revision();
    const url = URL.createObjectURL(file), source = new Image();
    try { await new Promise((resolve,reject) => { source.onload = resolve; source.onerror = () => reject(new Error('图片读取失败，请重新选择')); source.src = url; }); }
    catch (e) { URL.revokeObjectURL(url); throw e; }
    if (!isCurrent() || revision !== window.Modal.revision()) { URL.revokeObjectURL(url); return undefined; }
    return new Promise(resolve => {
      let cx = source.naturalWidth / 2, cy = source.naturalHeight / 2, zoom = 1, preview = false, closed = false;
      const side = () => Math.min(source.naturalWidth, source.naturalHeight) / zoom;
      window.Modal.open({ stack:true, title:'裁剪菜品图片',width:'460px',
        bodyHtml:'<p class="crop-hint">拖动图片调整保留区域，缩放后预览。</p><canvas class="crop-canvas" width="640" height="640" aria-label="正方形图片裁剪预览"></canvas><label class="crop-controls">缩放<input type="range" min="1" max="4" step="0.01" value="1" aria-label="图片缩放"></label>',
        footerHtml:'<button class="btn btn--line" data-cancel>取消</button><button class="btn btn--line" data-back hidden>返回调整</button><button class="btn btn--primary" data-next>预览</button>',
        onDismiss(reason) { closed = true; URL.revokeObjectURL(url); resolve(reason === 'replaced' ? undefined : null); },
        onMount(root,done) {
          const canvas=root.querySelector('canvas'),context=canvas.getContext('2d');
          const draw = () => { const size=side(); cx=Math.max(size/2,Math.min(cx,source.naturalWidth-size/2));cy=Math.max(size/2,Math.min(cy,source.naturalHeight-size/2));context.clearRect(0,0,640,640);context.drawImage(source,cx-size/2,cy-size/2,size,size,0,0,640,640); };
          const setPreview = value => { preview=value;root.querySelector('.crop-controls').hidden=value;root.querySelector('[data-back]').hidden=!value;root.querySelector('[data-next]').textContent=value?'确认上传':'预览';root.querySelector('.crop-hint').textContent=value?'确认后上传此区域的图片。':'拖动图片调整保留区域，缩放后预览。'; };
          root.querySelector('input').oninput=e=>{zoom=Number(e.target.value);draw();};
          let drag=null;
          canvas.onpointerdown=e=>{if(preview)return;canvas.setPointerCapture(e.pointerId);drag={x:e.clientX,y:e.clientY,cx,cy};};
          canvas.onpointermove=e=>{if(!drag)return;const ratio=side()/canvas.getBoundingClientRect().width;cx=drag.cx-(e.clientX-drag.x)*ratio;cy=drag.cy-(e.clientY-drag.y)*ratio;draw();};
          canvas.onpointerup=canvas.onpointercancel=canvas.onlostpointercapture=()=>{drag=null;};
          root.querySelector('[data-cancel]').onclick=done;root.querySelector('[data-back]').onclick=()=>setPreview(false);
          root.querySelector('[data-next]').onclick=async()=>{
            if(!preview){setPreview(true);return;}
            const button=root.querySelector('[data-next]');button.disabled=true;
            try {
              const output=document.createElement('canvas'),size=side();output.width=output.height=Math.min(640,Math.max(1,Math.round(size)));
              const ctx=output.getContext('2d');
              ctx.drawImage(source,cx-size/2,cy-size/2,size,size,0,0,output.width,output.height);
              // Preserve alpha for transparent artwork; compress opaque food photos.
              const pixels=ctx.getImageData(0,0,output.width,output.height).data;
              let transparent=false;
              for(let i=3;i<pixels.length;i+=4){if(pixels[i]!==255){transparent=true;break;}}
              const type=transparent?'image/png':'image/jpeg',extension=transparent?'.png':'.jpg';
              const blob=await new Promise(resolve=>output.toBlob(resolve,type,0.82));
              if(closed)return;
              if(!blob)throw new Error('图片生成失败，请重试');
              resolve(new File([blob],file.name.replace(/\.[^.]+$/,'')+extension,{type}));done();
            } catch(e){window.Toast.show(e.message,{icon:'warn'});button.disabled=false;}
          };
          draw();
        }
      });
    });
  }
  function mount(host, initial, changed, busyChanged) {
    let images=initial.slice(),busy=false,locked=false;
    const paint=()=>{
      host.innerHTML=`<div class="img-grid">${images.map((src,i)=>`<div class="img-cell"><img src="${window.Table.esc(window.Api.imgUrl(src))}" alt="菜品图片"><button class="img-cover" data-cover="${i}" ${busy||locked?'disabled':''}>${i===0?'封面':'设为封面'}</button><button class="img-del" data-remove="${i}" aria-label="移除第${i+1}张图片" ${busy||locked?'disabled':''}>×</button></div>`).join('')}${images.length<3?`<button class="img-cell add" data-add ${busy||locked?'disabled':''}>${busy?'上传中…':'添加图片'}</button>`:''}</div><input type="file" accept="image/*" multiple hidden><div class="fld-hint">最多 3 张，第一张为封面。点击添加图片，裁剪并预览后上传。${busy?'正在处理图片，请稍候。':''}</div>`;
      host.querySelectorAll('img').forEach(img=>img.onerror=()=>{img.hidden=true;const tip=document.createElement('span');tip.className='image-placeholder';tip.textContent='暂无图片';img.parentElement.prepend(tip);});
      host.querySelectorAll('[data-remove]').forEach(b=>b.onclick=()=>{images.splice(Number(b.dataset.remove),1);changed(images.slice());paint();});
      host.querySelectorAll('[data-cover]').forEach(b=>b.onclick=()=>{images.unshift(images.splice(Number(b.dataset.cover),1)[0]);changed(images.slice());paint();});
      const add=host.querySelector('[data-add]'),file=host.querySelector('input');if(add)add.onclick=()=>file.click();
      file.onchange=async()=>{
        if(busy||locked)return;
        const files=Array.from(file.files);if(!files.length)return;
        const capacity=3-images.length;
        if(files.length>capacity){window.Toast.show(`最多 3 张，还可添加 ${capacity} 张，请重新选择`,{icon:'warn'});file.value='';return;}
        busy=true;busyChanged(true);paint();
        try { let modalRevision=window.Modal.revision(); for(const f of files){if(modalRevision !== window.Modal.revision())break;const chosen=await crop(f,()=>host.isConnected);if(!host.isConnected || chosen === undefined)break;modalRevision=window.Modal.revision();if(!chosen)continue;const key=await window.Api.uploadImage(chosen);if(!host.isConnected)break;images.push(key);changed(images.slice());paint();} }
        catch(e){window.Toast.show(e.message||'上传失败，请重试',{icon:'warn'});}
        finally{busy=false;busyChanged(false);if(host.isConnected)paint();}
      };
    };
    paint();
    return {setLocked(value){locked=value;if(host.isConnected)paint();}};
  }
  async function edit(productID, saved, isCurrent) {
    const revision=window.Modal.revision();
    const p=await window.Api.getProduct(productID);
    if(!isCurrent() || revision !== window.Modal.revision())return;
    let images=p.imgs.slice(),busy=false;
    window.Modal.open({title:`上传图片 · ${p.name}`,bodyHtml:'<div data-images></div>',footerHtml:'<button class="btn btn--line" data-cancel>取消</button><button class="btn btn--primary" data-save>保存图片</button>',onMount(root,done){
      const button=root.querySelector('[data-save]');const editor=mount(root.querySelector('[data-images]'),images,value=>{images=value;},value=>{busy=value;button.disabled=value;button.textContent=value?'上传中…':'保存图片';});
      root.querySelector('[data-cancel]').onclick=done;
      button.onclick=window.FormUI.submit(button,async()=>{if(busy)return;editor.setLocked(true);try{await window.Api.saveProduct(Object.assign({},p,{imgs:images}));done();window.Toast.show('图片已保存');if(saved)await saved();}finally{editor.setLocked(false);}},'保存中…');
    }});
  }
  window.ImageEditor={crop,mount,edit};
})();
