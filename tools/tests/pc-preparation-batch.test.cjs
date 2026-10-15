const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const source=fs.readFileSync(path.join(__dirname,'../../apps/web-admin/pages/preparation.js'),'utf8');
const flush=async()=>{for(let i=0;i<30;i++)await Promise.resolve();};
const deferred=()=>{let resolve;const promise=new Promise(r=>resolve=r);return {promise,resolve};};
function node(){const children=new Map();return {isConnected:true,innerHTML:'',querySelector(s){if(!children.has(s))children.set(s,node());return children.get(s);},querySelectorAll(){return [];}};}
const order=(id,status='制作中',meal='lunch',actions=['MARK_READY'])=>({id:String(id),status,mealPeriod:meal,available_actions:actions,code:'000'+id,pickupDate:'2026-10-15',pickupTime:'12:00',items:[{product_id:'1',name:'米饭',quantity:1,flavors:[]}]});
async function setup({orders=[order(1),order(2)],mark=async()=>{},confirm=async()=>true}={}){
 const calls=[],events={};let serial=0,reads=0,confirms=0;
 const window={Pages:{},addEventListener:(name,handler)=>events[name]=handler,Api:{today:()=> '2026-10-15',MEAL_LABEL:{lunch:'午餐',dinner:'晚餐'},preparationOrders:async()=>{reads++;return orders;},idempotencyKey:()=> 'key-'+ ++serial,markOrderReady:async(id,key)=>{calls.push({id,key});return mark(id,key);}},Table:{esc:s=>s||'',pill:s=>s},Modal:{confirm:async options=>{confirms++;return confirm(options);}}};
 vm.runInNewContext(source,{window,Map,Set,Date});const el=node();window.Pages.preparation.render(el);await flush();
 const content=el.querySelector('[data-prep-content]');
 return {window,el,calls,events,button:()=>content.querySelector('[data-entire]'),result:()=>el.querySelector('[data-result]').innerHTML,reads:()=>reads,confirms:()=>confirms};
}
test('whole meal batch excludes reserved, other meal and orders without server permission',async()=>{
 const h=await setup({orders:[order(1),order(2,'已预约'),order(3,'制作中','dinner'),order(4,'制作中','lunch',[])]});
 await h.button().onclick();await flush();assert.deepEqual(h.calls.map(c=>c.id),['1']);assert.match(h.result(),/已备好 1 笔/);assert.equal(h.reads(),2);
});
test('duplicate clicks during confirmation and in-flight request submit once',async()=>{
 const approval=deferred(),request=deferred();const h=await setup({confirm:()=>approval.promise,mark:()=>request.promise});
 const first=h.button().onclick();await h.button().onclick();assert.equal(h.confirms(),1);approval.resolve(true);await flush();
 await h.button().onclick();assert.equal(h.calls.length,1);assert.equal(h.confirms(),1);
 request.resolve();await first;assert.deepEqual(h.calls.map(c=>c.id),['1','2']);
});
test('network unknown can be retried with same order idempotency key',async()=>{
 let fail=true;const h=await setup({orders:[order(1)],mark:async()=>{if(fail)throw Object.assign(new Error('offline'),{code:'NETWORK_UNAVAILABLE'});}});
 await h.button().onclick();await flush();assert.match(h.result(),/结果待确认/);assert.match(h.result(),/已备好 0 笔/);
 fail=false;await h.button().onclick();assert.equal(h.calls.length,2);assert.equal(h.calls[0].key,h.calls[1].key);assert.match(h.result(),/已备好 1 笔/);
});
for(const status of [401,403])test('permission '+status+' stops remaining orders',async()=>{
 const h=await setup({mark:async()=>{throw Object.assign(new Error('denied'),{status});}});
 await h.button().onclick();assert.deepEqual(h.calls.map(c=>c.id),['1']);assert.match(h.result(),/未处理 1 笔/);assert.match(h.result(),/未提交/);
});
test('expired session cancels pending confirmation',async()=>{
 const approval=deferred();const h=await setup({confirm:()=>approval.promise});const work=h.button().onclick();h.events['pc-session-expired']();approval.resolve(true);await work;assert.equal(h.calls.length,0);
});
test('expired session prevents remaining requests and stale result in new view',async()=>{
 const request=deferred();const h=await setup({mark:()=>request.promise});const work=h.button().onclick();await flush();assert.equal(h.calls.length,1);
 h.events['pc-session-expired']();h.window.Pages.preparation.render(h.el);await flush();request.resolve();await work;assert.equal(h.calls.length,1);assert.equal(h.result(),'');
});
