// Local packaging only. No upload, credentials or network calls.
const fs=require('node:fs');
const path=require('node:path');
const crypto=require('node:crypto');
const {execFileSync}=require('node:child_process');
const types={'.html':'text/html; charset=utf-8','.js':'application/javascript; charset=utf-8','.css':'text/css; charset=utf-8','.jpeg':'image/jpeg','.jpg':'image/jpeg','.png':'image/png','.svg':'image/svg+xml','.webp':'image/webp','.woff2':'font/woff2','.ico':'image/x-icon'};
function createPackage(source,out,meta){
 source=path.resolve(source);out=path.resolve(out);
 if(fs.existsSync(out))throw new Error('output already exists');
 if(out===source||out.startsWith(source+path.sep))throw new Error('output must be outside source');
 const buffers=new Map();
 function walk(dir,relative=''){
  for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
   const name=relative?relative+'/'+entry.name:entry.name;
   if(entry.isSymbolicLink())throw new Error('symlink resource: '+name);
   if(name==='tests')continue;
   if(entry.isDirectory())walk(path.join(dir,entry.name),name);
   else {if(!entry.isFile()||!types[path.extname(name)])throw new Error('unsupported resource: '+name);buffers.set(name,fs.readFileSync(path.join(dir,entry.name)));}
  }
 }
 walk(source);
 const html=buffers.get('index.html')?.toString('utf8');
 if(!html)throw new Error('missing index.html');
 const bases=[...html.matchAll(/<base\b[^>]*href\s*=\s*["']([^"']+)["'][^>]*>/gi)];
 if(bases.length!==1||bases[0][1]!=='/pc/')throw new Error('base must be exactly /pc/');
 function reference(ref,from){
  if(ref.startsWith('data:'))return;
  if(/^(?:[a-z]+:|\/\/)/i.test(ref)||ref.includes('\\'))throw new Error('external reference: '+ref);
  const url=new URL(ref,'https://package.invalid/pc/'+from);
  if(url.origin!=='https://package.invalid'||!url.pathname.startsWith('/pc/'))throw new Error('escaping reference: '+ref);
  const name=decodeURIComponent(url.pathname.slice(4));
  if(!buffers.has(name))throw new Error('missing resource: '+name);
 }
 for(const match of html.replace(/<base\b[^>]*>/gi,'').matchAll(/\b(?:src|href)\s*=\s*["']([^"']+)["']/gi))reference(match[1],'index.html');
 for(const [name,buf] of buffers){
  if(name.endsWith('.css'))for(const match of buf.toString('utf8').matchAll(/url\(\s*["']?([^"')]+)["']?\s*\)/gi))reference(match[1].trim(),name);
 }
 const names=[...buffers.keys()].sort((a,b)=>a.endsWith('.html')!==b.endsWith('.html')?(a.endsWith('.html')?1:-1):a.localeCompare(b));
 const files=names.map(name=>({path:name,destination:'/pc/'+name,bytes:buffers.get(name).length,sha256:crypto.createHash('sha256').update(buffers.get(name)).digest('hex'),contentType:types[path.extname(name)],cacheControl:'no-cache'}));
 const manifest={status:meta.draft?'DRAFT_NOT_DEPLOYABLE':'PACKAGED_NOT_VERIFIED',baseSHA:meta.baseSHA,candidateSHA:meta.draft?null:meta.baseSHA,files};
 fs.mkdirSync(out,{recursive:true});
 for(const name of names){const target=path.join(out,'pc',name);fs.mkdirSync(path.dirname(target),{recursive:true});fs.writeFileSync(target,buffers.get(name),{flag:'wx'});}
 fs.writeFileSync(path.join(out,'manifest.json'),JSON.stringify(manifest,null,2)+'\n',{flag:'wx'});
 return manifest;
}
if(require.main===module){
 try{
  const args=process.argv.slice(2),draft=args[0]==='--draft';if(draft)args.shift();
  if(args.length!==1)throw new Error('usage: node tools/pc-release-package.cjs [--draft] /absolute/new-output-directory');
  if(!path.isAbsolute(args[0]))throw new Error('absolute output directory required');
  const root=path.resolve(__dirname,'..');
  const git=(...a)=>execFileSync('git',a,{cwd:root,encoding:'utf8'}).trim();
  const baseSHA=git('rev-parse','HEAD');
  if(!draft&&git('status','--porcelain','--untracked-files=all'))throw new Error('dirty repository: release packaging requires exact clean candidate; use --draft for local inspection');
  const manifest=createPackage(path.join(root,'apps/web-admin'),args[0],{baseSHA,draft});
  console.log(JSON.stringify({status:manifest.status,baseSHA,candidateSHA:manifest.candidateSHA,files:manifest.files.length,manifest:path.join(args[0],'manifest.json')}));
 }catch(e){console.error(e.message);process.exitCode=1;}
}
module.exports={createPackage};
