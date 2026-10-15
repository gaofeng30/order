const {test}=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');
const path=require('node:path');
const source=fs.readFileSync(path.join(__dirname,'../../apps/web-admin/data/api.js'),'utf8');
const date='2026-10-15';
const order=(id,status='制作中')=>({id:String(id),status,pickup_date:date,meal_period:'lunch',items:[{product_id:'1',name:'米饭',quantity:1,flavors:[]}]});
function setup(pages){const urls=[];const window={sessionStorage:{getItem:()=>null},fetch:async url=>{urls.push(url);assert(pages.length,'unexpected extra request');return {ok:true,status:200,headers:{get:()=> 'application/json'},json:async()=>pages.shift()};}};vm.runInNewContext(source,{window,FormData,Map,Set,URL,console});return {api:window.Api,urls};}
test('preparation reads all pages and filters only after validating the complete page',async()=>{
 const {api,urls}=setup([{orders:[order(1),order(2,'待取餐')],next_after_id:'2'},{orders:[order(3,'已预约')],next_after_id:null}]);
 assert.deepEqual(Array.from(await api.preparationOrders(date),o=>o.id),['1','3']);
 assert.match(urls[1],/after_id=2/);
});
for(const [name,pages] of [
 ['missing cursor',[{orders:[order(1)]}]],
 ['cursor beyond last row',[{orders:[order(1)],next_after_id:'2'},{orders:[],next_after_id:null}]],
 ['empty page with continuation',[{orders:[],next_after_id:'2'},{orders:[],next_after_id:null}]],
 ['unordered rows',[{orders:[order(2),order(1)],next_after_id:null}]],
 ['row below previous cursor',[{orders:[order(2)],next_after_id:'2'},{orders:[order(1)],next_after_id:null}]],
 ['duplicate across pages',[{orders:[order(1)],next_after_id:'1'},{orders:[order(1)],next_after_id:null}]],
 ['failure after first page',[{orders:[order(1)],next_after_id:'1'},{error:'failure'}]],
])test('preparation rejects incomplete batch: '+name,async()=>{
 const {api}=setup(pages);await assert.rejects(()=>api.preparationOrders(date),/订单/);
});
