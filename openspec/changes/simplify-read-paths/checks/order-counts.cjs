const fs=require('node:fs'),vm=require('node:vm'),path=require('node:path'),assert=require('node:assert/strict');
const calls=[];
let invalid=false;
const window={sessionStorage:{getItem:()=>null},fetch:async url=>{
 calls.push(url); assert.equal(url,'/api/v1/admin/orders/counts');
 return {ok:true,status:200,headers:{get:()=> 'application/json'},json:async()=>({counts:{RESERVED:2,PREPARING:3,READY_FOR_PICKUP:4,COMPLETED:1000,REFUNDING:0,REFUNDED:invalid?-1:1}})};
}};
vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../../../../apps/web-admin/data/api.js'),'utf8'),{window,URLSearchParams,FormData:class{},console});
(async()=>{
 const counts=await window.Api.readOrderCounts(); assert.equal(counts['全部'],1010); assert.equal(counts['已完成'],1000); assert.equal(calls.length,1);
 invalid=true; await assert.rejects(()=>window.Api.readOrderCounts()); assert.equal(window.Api.laneCounts()['全部'],null);
 console.log('PASS: one counts request, no order details; invalid counts remain unavailable');
})().catch(e=>{console.error(e);process.exitCode=1;});
