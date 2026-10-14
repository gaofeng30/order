const http=require('node:http'),https=require('node:https'),fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'../apps/web-admin');
const upstream='https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com';
http.createServer((req,res)=>{
 const url=new URL(req.url,'http://127.0.0.1');
 if(url.pathname.startsWith('/api/v1/')){
  const headers={...req.headers};delete headers.host;delete headers.origin;delete headers.referer;headers['accept-encoding']='identity';
  const remote=https.request(upstream+url.pathname+url.search,{method:req.method,headers},response=>{res.writeHead(response.statusCode,{...response.headers,'cache-control':'no-store'});response.pipe(res);});
  remote.on('error',()=>{if(!res.headersSent)res.writeHead(502,{'content-type':'application/json'});res.end(JSON.stringify({error:{message:'无法连接 UAT 后台'}}));});req.pipe(remote);return;
 }
 const assetPath=url.pathname.replace(/^\/pc(?=\/|$)/,'') || '/';
 let file;try{file=path.resolve(root,'.'+decodeURIComponent(assetPath==='/'?'/index.html':assetPath));}catch{res.writeHead(400);res.end();return;}
 if(!file.startsWith(root+path.sep)){res.writeHead(403);res.end();return;}
 fs.readFile(file,(err,data)=>{if(err){res.writeHead(404);res.end();return;}const types={'.html':'text/html;charset=utf-8','.js':'text/javascript;charset=utf-8','.css':'text/css;charset=utf-8','.jpeg':'image/jpeg','.png':'image/png'};res.writeHead(200,{'content-type':types[path.extname(file)]||'application/octet-stream','cache-control':'no-store'});res.end(data);});
}).listen(3817,'127.0.0.1',()=>console.log('PC preview: http://127.0.0.1:3817/ — local UI, real UAT API'));
