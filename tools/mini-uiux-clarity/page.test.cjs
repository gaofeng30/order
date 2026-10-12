const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const root = path.resolve(__dirname, '../..');
const tick = () => new Promise(resolve => setImmediate(resolve));
function deferred() { let resolve,reject; const promise=new Promise((a,b)=>{resolve=a;reject=b;}); return {promise,resolve,reject}; }
function pageHarness(transactionOverrides={}, phoneOverrides={}) {
 let definition, quoteCalls=0, prepayCalls=0;
 const original=[{item:{id:'1',name:'合成菜品',price_text:'0.10',original_price_text:'0.10'},q:2,flavors:['少饭'],note:'合成备注',line_total_text:'0.20'}];
 const selected={date:'2026-10-12',time:'11:30',mealPeriod:'lunch'};
 const quote={id:'7',store:{address:'合成地址'},pickup:{point:'合成窗口'},original_subtotal_cents:42,discount_cents:8,payable_cents:34,items:[{line_number:1,product_id:'1',name:'报价菜品',original_unit_price_cents:21,discounted_unit_price_cents:17,quantity:2,payable_subtotal_cents:34,flavors:['少饭'],note:'合成备注'}]};
 const cart={list:()=>structuredClone(original),count:()=>2,totalCents:()=>20,entry:()=>({product:original[0].item,qty:2,flavors:[],note:''}),clear(){},setPrefs(){}};
 const calls={navigate:[],extra:0,confirm:[]};
 const transaction={createQuote:async()=>{quoteCalls++;return quote},createPrepayment:async()=>{prepayCalls++;throw Error('synthetic prepay failure')},confirm:async(id)=>{calls.confirm.push(id);return {state:'PENDING'}},requestPayment:async()=>false,...transactionOverrides};
 const stubs={ '../../utils/apiClient.js':{newIdempotencyKey:()=> 'synthetic-key'}, '../../utils/phoneStore.js':{status:async()=>({bound:true,maskedPhone:'***'}),setExtra:async()=>{calls.extra++;return{extraPhone:{masked_phone:'***'}}},...phoneOverrides}, '../../utils/transactionStore.js':transaction,'../../utils/catalogStore.js':{formatCents:v=>(v/100).toFixed(2)}, '../../utils/util.js':{cart,pickup:{get:()=>selected,label:()=> '10-12 午餐 11:30'},nav:{tabTo:(...v)=>calls.navigate.push(['tabTo',...v]),go:(...v)=>calls.navigate.push(['go',...v]),replace:(...v)=>calls.navigate.push(['replace',...v])}}, '../../utils/navBehavior.js':{} };
 vm.runInNewContext(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/pages/confirm/confirm.js'),'utf8'),{Page:v=>definition=v,require:id=>{if(!(id in stubs))throw Error(id);return stubs[id]},getApp:()=>({globalData:{storefrontFlavors:[]}}),getCurrentPages:()=>[{route:'pages/confirm/confirm'}],wx:{navigateTo:config=>calls.navigate.push(['navigateTo',config.url])},console});
 const page={...definition,data:structuredClone(definition.data),setData(values){for(const [key,value]of Object.entries(values)){const parts=key.split('.');let target=this.data;for(const part of parts.slice(0,-1))target=target[part];target[parts.at(-1)]=value}},selectComponent:()=>({show(){}})};
 page.onLoad();page.setData({phoneState:'bound',form:{contact:'合成联系人',orderNote:'合成整单备注',extraPhone:''}});
 return {page,quote,calls,cart,selected,getCounts:()=>({quoteCalls,prepayCalls})};
}
test('权威报价展示同一快照，预支付失败后回到预估',async()=>{
 const prepay=deferred();const {page}=pageHarness({createPrepayment:()=>prepay.promise});
 const payment=page.pay();await tick();
 assert.equal(page.data.subtotal_text,'0.42');assert.equal(page.data.discount_text,'0.08');assert.equal(page.data.payable_text,'0.34');
 assert.equal(page.data.items[0].item.price_text,'0.17');assert.equal(page.data.items[0].line_total_text,'0.34');assert.equal(page.data.items[0].item.name,'报价菜品');
 prepay.reject(Error('synthetic prepay failure'));await payment;
 assert.equal(page.data.quote,null);assert.equal(page.data.payable_text,'0.20');assert.equal(page.data.items[0].item.price_text,'0.10');assert.equal(page.data.payLabel,'预估合计');
});
module.exports={pageHarness,deferred,tick};
test('不可售原因持续可见且没有预支付，表单保留',async()=>{
 const error=Object.assign(Error('selection'),{code:'QUOTE_SELECTION_UNAVAILABLE'});
 const {page,getCounts}=pageHarness({createQuote:async()=>{throw error}});
 await page.pay();
 assert.match(page.data.quoteError,/商品或取餐时间已不可用/);assert.equal(page.data.selectionUnavailable,true);
 assert.equal(getCounts().prepayCalls,0);assert.equal(page.data.form.orderNote,'合成整单备注');
});
test('核价与已有预支付冻结编辑和附加手机号写入',async()=>{
 const prepay=deferred();const {page,calls}=pageHarness({createPrepayment:()=>prepay.promise});
 page.setData({'form.extraPhone':'10000000000'});const payment=page.pay();await tick();
 page.onInput({currentTarget:{dataset:{k:'contact'}},detail:{value:'不应覆盖'}});
 assert.equal(page.data.form.contact,'合成联系人');
 await page.saveExtraPhone();assert.equal(calls.extra,0);
 await page.onShow();assert.equal(page.data.subtotal_text,'0.42');
 prepay.resolve({id:'88',wx_request_payment:{}});await payment;
 assert.equal(page.data.paymentState,'pending');page.onInput({currentTarget:{dataset:{k:'orderNote'}},detail:{value:'不应覆盖'}});
 assert.equal(page.data.form.orderNote,'合成整单备注');await page.pay();assert.deepEqual(calls.confirm,['88','88','88']);
});

test('返回修改保留确认页，返回后刷新选择而不丢表单',async()=>{
 const {page,calls,selected}=pageHarness();page.editPickup();
 assert.deepEqual(calls.navigate,[['navigateTo','/pages/menu/menu?returnToConfirm=1']]);
 selected.time='12:00';await page.onShow();
 assert.equal(page.data.pickup.time,'12:00');assert.equal(page.data.form.contact,'合成联系人');assert.equal(page.data.form.orderNote,'合成整单备注');
 let menuDefinition;let backs=0;let newConfirms=0;
 vm.runInNewContext(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/pages/menu/menu.js'),'utf8'),{
  Page:v=>menuDefinition=v,require:id=>id.endsWith('util.js')?{nav:{go:()=>newConfirms++},cart:{},pickup:{get:()=>selected}}:{},
  getCurrentPages:()=>[{route:'pages/confirm/confirm'},{route:'pages/menu/menu'}],wx:{navigateBack:()=>backs++}
 });
 const menu={...menuDefinition,data:{canCheckout:true},refresh(){},setData(v){Object.assign(this.data,v)}};
 menu.onLoad({returnToConfirm:'1'});menu.goConfirm();assert.equal(backs,1);assert.equal(newConfirms,0);
 menu.onLoad({});menu.goConfirm();assert.equal(newConfirms,1);
});

test('附加手机号仍在保存时不启动报价',async()=>{
 const pending=deferred();const {page,getCounts}=pageHarness({}, {setExtra:()=>pending.promise});
 page.setData({'form.extraPhone':'10000000000'});const saving=page.saveExtraPhone();await tick();
 await page.pay();assert.equal(getCounts().quoteCalls,0);
 pending.resolve({extraPhone:{masked_phone:'***'}});await saving;
});

test('修改商品后清除上一次选择失败，不残留旧支付状态',async()=>{
 const {page}=pageHarness({createQuote:async()=>{throw Object.assign(Error('selection'),{code:'QUOTE_SELECTION_UNAVAILABLE'})}});
 await page.pay();assert.equal(page.data.selectionUnavailable,true);
 page.setData({czItem:{id:'1'}});page.onCzConfirm({detail:{qty:1,flavors:[],note:''}});
 assert.equal(page.data.selectionUnavailable,false);assert.equal(page.data.quoteError,'');assert.equal(page.data.paymentState,'idle');
});

test('修改菜单即使不可结算也有显式返回，普通菜单无返回入口',()=>{
 let definition;vm.runInNewContext(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/pages/menu/menu.js'),'utf8'),{Page:v=>definition=v,require:()=>({}),getCurrentPages:()=>[{route:'pages/confirm/confirm'},{route:'pages/menu/menu'}]});
 const page={...definition,data:{canCheckout:false},refresh(){},setData(v){Object.assign(this.data,v)}};
 page.onLoad({returnToConfirm:'1'});assert.equal(page.data.returnToConfirm,true);
 assert.match(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/pages/menu/menu.wxml'),'utf8'),/bindtap="returnToConfirmation"/);
 page.onLoad({});assert.equal(page.data.returnToConfirm,false);
});

test('返回确认后迟到的菜单响应或失败不能修改共享取餐选择',async()=>{
 for(const reject of [false,true]){
  let definition;const pending=deferred();let selection={date:'2026-10-12',mealPeriod:'lunch',time:'11:30'};
  vm.runInNewContext(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/pages/menu/menu.js'),'utf8'),{Page:v=>definition=v,require:id=>id.endsWith('reservationMenuStore.js')?{loadPickupOptions:()=>pending.promise,firstAvailable:()=>({date:'2026-10-12',mealPeriod:'lunch',time:'12:00'})}:id.endsWith('util.js')?{pickup:{get:()=>selection,set:v=>selection=v,label:()=>''}}:{}});
  const page={...definition,data:{},setData(v){Object.assign(this.data,v)},loadMenu:async()=>true};const work=page.loadOptionsAndMenu();
  if(page.onUnload)page.onUnload();
  if(reject)pending.reject(Error('synthetic failure'));else pending.resolve({dates:[{date:'2026-10-12',available:true,mealPeriods:[{mealPeriod:'lunch',available:true,pickupTimes:['12:00']}]}]});
  await work;assert.equal(selection&&selection.time,'11:30');
 }
});

test('显式返回失败只提示，不重启小程序丢弃原确认表单',()=>{
 let menuDefinition,navbar;const events=[];const pages=[{route:'pages/confirm/confirm',form:{contact:'合成保留姓名'}},{route:'pages/menu/menu'}];
 const wx={navigateBack:opts=>{events.push('back');opts.fail({errMsg:'synthetic navigation failure'})},reLaunch:()=>{events.push('relaunch');pages.length=0}};
 vm.runInNewContext(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/pages/menu/menu.js'),'utf8'),{Page:v=>menuDefinition=v,require:()=>({}),getCurrentPages:()=>pages,wx});
 const menu={...menuDefinition,_returnToConfirm:true,selectComponent:()=>({show:()=>events.push('toast')})};
 if(menu.returnToConfirmation)menu.returnToConfirmation();else{
  vm.runInNewContext(fs.readFileSync(path.join(root,'apps/wechat-miniprogram/components/navbar/navbar.js'),'utf8'),{Component:v=>navbar=v,wx});navbar.methods.onBack.call({data:{exit:false}});
 }
 assert.equal(pages.length,2);assert.equal(pages[0].form.contact,'合成保留姓名');assert.deepEqual(events,['back','toast']);
});
