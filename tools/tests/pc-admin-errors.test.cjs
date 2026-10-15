const {test}=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');
const path=require('node:path');
const source=fs.readFileSync(path.join(__dirname,'../../apps/web-admin/data/api.js'),'utf8');
for(const [code,message] of [
 ['INVALID_REQUEST','请检查必填项和填写格式'],
 ['CATALOG_CONFLICT','名称重复或数据已变更，请检查后重试'],
 ['IMPORT_UNAVAILABLE','导入服务暂时不可用，请稍后重试'],
])test(code+' has an actionable Chinese message',async()=>{
 const window={sessionStorage:{getItem:()=>null},fetch:async()=>({ok:false,status:400,headers:{get:()=> 'application/json'},json:async()=>({error:{code,message:'internal english error'}})})};
 vm.runInNewContext(source,{window,FormData,Map,Set,URL,console});
 await assert.rejects(()=>window.Api.listCategories(),e=>e.code===code&&e.message===message);
});
test('import result image editor reads the product detail envelope',async()=>{
 const product={id:'32',name:'三图测试菜品',category_id:'4',price_cents:1,meal_period:'all',images:[1,2,3].map(n=>({object_key:'image-'+n,url:'https://images.invalid/'+n+'.png'}))};
 const window={sessionStorage:{getItem:()=>null},fetch:async url=>({ok:true,status:200,headers:{get:()=> 'application/json'},json:async()=>url.includes('/products/')?{product}:{service_date:'2026-10-14'}})};
 vm.runInNewContext(source,{window,FormData,Map,Set,URL,console});
 const actual=await window.Api.getProduct('32');
 assert.equal(actual.name,product.name);
 assert.deepEqual(Array.from(actual.imgs),['image-1','image-2','image-3']);
 assert.equal(window.Api.imgUrl(actual.imgs[2]),'https://images.invalid/3.png');
});
