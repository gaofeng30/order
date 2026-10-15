const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync(require('node:path').join(__dirname,'../../apps/web-admin/ui/image-editor.js'),'utf8');
async function runCrop(width,height,alpha=255){
 let mounted, output, opts;
 const nodes=new Map();
 const context={clearRect(){},drawImage(){},getImageData(){return {data:Uint8ClampedArray.from([80,90,100,alpha])};}};
 const canvas=()=>({width:640,height:640,getContext:()=>context,toBlob(cb,type,quality){opts={type,quality};cb(new Blob(['encoded'],{type}));}});
 const root={querySelector(s){if(!nodes.has(s))nodes.set(s,s==='canvas'?canvas():{});return nodes.get(s);}};
 const window={Modal:{revision:()=>1,open(o){mounted=o; o.onMount(root,()=>o.onDismiss('done'));}},Toast:{show(msg){throw new Error(msg);}}};
 class Image {naturalWidth=width;naturalHeight=height;set src(_){queueMicrotask(()=>this.onload());}}
 vm.runInNewContext(source,{window,URL:{createObjectURL:()=> 'blob:test',revokeObjectURL(){}},Image,document:{createElement(){return output=canvas();}},File,console});
 const pending=window.ImageEditor.crop(new File(['original'],'food.png',{type:'image/png'}),()=>true);
 await new Promise(setImmediate);
 assert.ok(mounted,'crop dialog opens');
 await nodes.get('[data-next]').onclick();
 await nodes.get('[data-next]').onclick();
 return {file:await pending,output,opts};
}
test('opaque photo upload is bounded and JPEG encoded',async()=>{
 const {file,output,opts}=await runCrop(4000,3000);
 assert.equal(output.width,640);assert.equal(output.height,640);
 assert.equal(file.type,'image/jpeg');assert.equal(file.name,'food.jpg');
 assert.equal(opts.quality,0.82);
});
test('small transparent image keeps alpha and is not enlarged',async()=>{
 const {file,output}=await runCrop(128,200,80);
 assert.equal(output.width,128);assert.equal(file.type,'image/png');
});
