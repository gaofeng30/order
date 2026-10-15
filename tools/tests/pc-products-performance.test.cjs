const {test}=require('node:test'), assert=require('node:assert/strict'), vm=require('node:vm'),fs=require('node:fs'),path=require('node:path');
test('product search reuses displayed list; entering page refreshes it',async()=>{
 let reads=0,lastRows=[],callbacks;const nodes=new Map();
 const node=()=>({innerHTML:'',value:'',querySelector:()=>null,querySelectorAll:()=>[],focus(){},setSelectionRange(){}});
 const el={isConnected:true,querySelector(s){if(!nodes.has(s))nodes.set(s,node());return nodes.get(s);},querySelectorAll:()=>[]};
 const products=[{id:'1',name:'牛肉饭',cat:'主食'},{id:'2',name:'鸡肉饭',cat:'主食'}];
 const window={Api:{setProductStatus:async()=>{},listCategories:async()=>[],listProducts:async()=>{reads++;return products;}},Icon:{svg:()=>''},Table:{esc:x=>x,render:({rows})=>{lastRows=rows;return '';},bind(_,handlers){callbacks=handlers;}},Toast:{show(){}}};
 vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../../apps/web-admin/pages/products.js'),'utf8'),{window,console,Set});
 window.Pages.products.render(el);await new Promise(setImmediate);assert.equal(reads,1);
 for(const value of ['牛','牛肉','牛肉饭']){nodes.get('#kw').value=value;nodes.get('#kw').oninput();await new Promise(setImmediate);}
 assert.equal(reads,1,'typing must not reread catalog and replace image URLs');assert.equal(lastRows.length,1);assert.equal(lastRows[0].id,'1');
 callbacks.shelf('1');await new Promise(setImmediate);assert.equal(reads,2,'a shelf write must refresh source data');
 window.Pages.products.render(el);await new Promise(setImmediate);assert.equal(reads,3,'re-enter page must refresh source data');
});
