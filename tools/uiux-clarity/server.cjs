// Isolated UI fixture: no upstream requests, credentials or business data.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../..');
let settings = { store_name: '隔离测试门店', store_status: 'open', service_date: '2026-10-12', pickup_step_min: 30, pickup_point: '测试窗口', notice: '合成公告', meal_periods: [{code:'lunch',name:'午餐',cutoff_time:'11:30',pickup_from:'11:30',pickup_to:'13:30'},{code:'dinner',name:'晚餐',cutoff_time:'17:00',pickup_from:'17:00',pickup_to:'19:00'}], service_dates:[{date:'2026-10-12',status:'open'},{date:'2026-10-13',status:'open'}] };
let mode='normal', reads=0, writes=0, failNextRead=false, authorized=true, unauthorizedReads=0;
const controls = `<!doctype html><meta charset="utf-8"><title>隔离验证控制</title><h1>仅合成数据，无上游连接</h1>${['normal','slow','readback-failure','reject'].map(m=>`<button onclick="fetch('/fixture/mode?value=${m}',{method:'POST'}).then(refresh)">${m}</button>`).join(' ')}<button onclick="refresh()">读取计数</button><pre id="result"></pre><script>function refresh(){fetch('/fixture/status').then(r=>r.json()).then(v=>document.querySelector('#result').textContent=JSON.stringify(v,null,2))}refresh()</script>`;
const send=(res,status,body)=>{res.writeHead(status,{'Content-Type':'application/json','Cache-Control':'no-store'});res.end(JSON.stringify(body));};
http.createServer(async(req,res)=>{
 const u=new URL(req.url,'http://127.0.0.1');
 if(u.pathname==='/fixture'){res.writeHead(200,{'Content-Type':'text/html'});return res.end(controls);}
 if(u.pathname==='/fixture/mode'&&req.method==='POST'){mode=u.searchParams.get('value');if(mode==='normal')authorized=true;return send(res,200,{mode});}
 if(u.pathname==='/fixture/status')return send(res,200,{mode,reads,writes,unauthorizedReads});
 if(u.pathname==='/api/v1/admin/auth/logout'){authorized=false;return send(res,200,{});}
 if(u.pathname==='/api/v1/admin/auth/qrcode')return send(res,200,{login_id:'synthetic-login',poll_secret:'synthetic-only',qr_payload:'local-fixture-not-a-real-login',expires_at:new Date(Date.now()+60000).toISOString()});
 if(u.pathname==='/api/v1/admin/auth/poll')return send(res,200,{state:'WAITING'});
 if(u.pathname==='/api/v1/admin/me')return send(res,200,{id:'1',role:'OWNER',name:'合成账号',phone_masked:'***',enabled:true});
 if(u.pathname==='/api/v1/admin/settings'){
  if(req.method==='PUT'){
   writes++;
   let raw='';for await (const b of req)raw+=b;
   if(mode==='reject')return send(res,400,{error:{code:'INVALID_REQUEST',message:'隔离场景：保存被拒绝'}});
   const value=JSON.parse(raw);
   if(!Array.isArray(value.meal_periods)||value.meal_periods.map(p=>p.code).sort().join(',')!=='dinner,lunch')return send(res,400,{error:{code:'INVALID_REQUEST',message:'两个餐段必需'}});
   value.store_status=({'营业中':'open','休息中':'closed','已截单':'cutoff'})[value.store_status]||value.store_status;
   if(mode==='slow')await new Promise(r=>setTimeout(r,2500));
   settings={...settings,...value};
   if(mode==='readback-failure')failNextRead=true;
   return send(res,200,settings);
  }
  reads++;if(!authorized){unauthorizedReads++;return send(res,401,{error:{code:'UNAUTHENTICATED'}});}
  if(failNextRead){failNextRead=false;return send(res,503,{error:{code:'UNAVAILABLE',message:'隔离场景：读回失败'}});}
  return send(res,200,settings);
 }
 if(u.pathname==='/api/v1/admin/orders')return send(res,200,{orders:[],next_after_id:null});
 if(u.pathname==='/api/v1/admin/stats')return send(res,200,{today_revenue_cents:0,today_orders:0,pending_production:0});
 if(u.pathname.startsWith('/api/'))return send(res,404,{error:{code:'FIXTURE_ROUTE_UNSUPPORTED'}});
 const relative=u.pathname==='/'?'apps/web-admin/index.html':decodeURIComponent(u.pathname.slice(1));
 if(!relative.startsWith('apps/web-admin/') && relative!=='apps/wechat-miniprogram/assets/emblem.png')return send(res,404,{});
 const file=path.resolve(root,relative);
 if(!file.startsWith(root+path.sep)||!fs.existsSync(file)||!fs.statSync(file).isFile())return send(res,404,{});
 res.writeHead(200,{'Content-Type':({'.js':'application/javascript','.html':'text/html','.css':'text/css','.png':'image/png'})[path.extname(file)]||'application/octet-stream','Cache-Control':'no-store'});
 fs.createReadStream(file).pipe(res);
}).listen(0,'127.0.0.1',function(){console.log(`http://127.0.0.1:${this.address().port}/apps/web-admin/index.html#/settings`)});
