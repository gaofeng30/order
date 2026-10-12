const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const path=require('node:path');
const file=path.resolve(__dirname,'../../apps/wechat-miniprogram/pages/admin-orders/admin-orders.js');
function deferred(){let resolve,reject;const promise=new Promise((a,b)=>{resolve=a;reject=b});return{promise,resolve,reject};}
function harness(orders){let definition;const navigation=[];const identity={merchant:{bound:true,role:'OWNER'}};
 const stubs={'../../utils/apiClient.js':{},'../../utils/merchantStore.js':{orders},'../../utils/storefrontStore.js':{load:async()=>({businessStatus:'open'})},'../../utils/identityStore.js':{load:async()=>identity},'../../utils/util.js':{nav:{go:(...args)=>navigation.push(args)}},'../../utils/navBehavior.js':{}};
 vm.runInNewContext(fs.readFileSync(file,'utf8'),{Page:v=>definition=v,require:id=>stubs[id]});
 const page={...definition,data:structuredClone(definition.data),setData(v){Object.assign(this.data,v)}};
 return{page,navigation};
}
const order=id=>({id,state:'PREPARING',status:'制作中',pickupDate:'2026-10-12',pickupTime:'11:30',materialized_at:'2026-10-12T01:00:00Z',available_actions:['READY']});
test('旧查询不能覆盖新泳道，动作仍导航同一订单',async()=>{
 const old=deferred(),fresh=deferred();let calls=0;const{page,navigation}=harness(()=>++calls===1?old.promise:fresh.promise);
 const first=page.load();const second=page.switchLane({currentTarget:{dataset:{l:'制作中'}}});
 fresh.resolve([order('2')]);await second;old.resolve([order('1')]);await first;
 assert.equal(page.data.list[0].id,'2');assert.equal(page.data.list[0].meta.label,'查看并备好');
 page.advance({currentTarget:{dataset:{id:'2'}}});assert.equal(navigation[0][0],'admin-order-detail');assert.equal(navigation[0][1].id,'2');
});

test('过期错误与卸载后的响应不改写页面',async()=>{
 const old=deferred(),fresh=deferred();let calls=0;const{page}=harness(()=>++calls===1?old.promise:fresh.promise);
 const first=page.load();const second=page.load();fresh.resolve([order('3')]);await second;
 old.reject(Error('synthetic old failure'));await first;assert.equal(page.data.listState,'ready');assert.equal(page.data.list[0].id,'3');
 const pending=deferred();const{page:gone}=harness(()=>pending.promise);const work=gone.load();gone.onUnload();pending.resolve([order('4')]);await work;assert.equal(gone.data.listState,'loading');
});
test('当前读取失败可重试，空结果不冒充加载失败',async()=>{
 let calls=0;const{page}=harness(async()=>{if(++calls===1)throw Error('synthetic failure');return []});
 await page.load();assert.equal(page.data.listState,'error');await page.load();assert.equal(page.data.listState,'empty');
});
