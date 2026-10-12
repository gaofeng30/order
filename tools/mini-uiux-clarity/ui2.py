# Synthetic-only WeChat UI2 runner. Each call pins project; never upload this fixture.
import subprocess,json,sys,pathlib,base64
project=sys.argv[1]; group=sys.argv[2]; out=pathlib.Path(sys.argv[3])
fixture=pathlib.Path(project)
manifest=json.loads((fixture/'fixture-manifest.json').read_text())
assert manifest['warning']=='Not an uploadable candidate; no native payment evidence'
assert 'Synthetic fixture: all network APIs blocked' in (fixture/'utils/apiClient.js').read_text()
out.mkdir(parents=True);records=[]
def call(tool,**kwargs):
 args=['wechatide','-c','Codex',tool,'--project',project]
 for k,v in kwargs.items():args+=['--'+k.replace('_','-'),str(v)]
 record={'tool':tool,'args':kwargs,'command':args}
 records.append(record)
 def save(): (out/'calls.json').write_text(json.dumps(records,ensure_ascii=False,indent=2))
 def capture(stdout,stderr):
  for name,value in [('stdout',stdout),('stderr',stderr)]:
   raw=value if isinstance(value,bytes) else (value or '').encode('utf-8')
   record[name]=raw.decode('utf-8','replace');record[name+'_base64']=base64.b64encode(raw).decode('ascii')
 save()
 try:
  p=subprocess.run(args,capture_output=True,timeout=45)
 except subprocess.TimeoutExpired as error:
  capture(error.stdout,error.stderr);record.update(returncode=None,timed_out=True);save();raise
 except OSError as error:
  capture('',str(error));record.update(returncode=None,launch_error=True);save();raise
 capture(p.stdout,p.stderr);record.update(returncode=p.returncode,timed_out=False);save()
 if p.returncode!=0:raise RuntimeError('CLI exited '+str(p.returncode))
 raw=record['stdout'];start=raw.find('{');v=json.loads(raw[start:]);record['response']=v;save()
 if not v.get('ok'):raise RuntimeError(v)
 return v.get('result')

def ev(code):
 v=call('automation_evaluate',fn_source='function(){'+code+'}')
 while isinstance(v,dict) and 'result' in v:v=v['result']
 return v
def tap(selector):return call('automation_element_action',selector=selector,action='tap',wait_for_selector=selector)
def inp(key,value):return call('automation_element_action',selector='input[data-k="'+key+'"]',action='input',value=value)
def data():return call('automation_page_action',action='getData')['data']
def mode(v):ev('getApp().fixture.mode='+json.dumps(v)+';return true;')
def release(k):ev('getApp().fixture.release('+json.dumps(k)+');return true;')
def nav(action,url=None):return call('automation_navigate',action=action,**({'url':url} if url else {}))
def setup(m,selector):
 nav('reLaunch','/pages/fixture/fixture');ev('getApp().fixture.reset();getApp().fixture.mode='+json.dumps(m)+';return true;');tap(selector)
def counts():return ev('const f=getApp().fixture;return {quote:f.quoteCalls,prepay:f.prepayCalls,confirm:f.confirmIDs,extra:f.extraCalls};')
def shot(name):call('simulator_screenshot',path=str(out/(name+'.png')))
def okay(name):print('PASS '+name,flush=True)
if group=='customer':
 setup('prepay-fail','button');inp('contact','合成联系人');inp('orderNote','合成整单备注');tap('.pay-btn');d=data();assert(d['subtotal_text'],d['discount_text'],d['payable_text'],d['items'][0]['item']['price_text'])==('0.42','0.08','0.34','0.17');assert d['editingLocked'];shot('quote')
 assert call('automation_element_action',selector='input[data-k="contact"]',action='property',name='disabled') is True
 tap('.rsv-picked');assert call('automation_runtime_info',action='currentPage')['currentPage']['path']=='pages/confirm/confirm'
 release('prepay');d=data();assert d['payable_text']=='0.20' and d['quote'] is None and not d['editingLocked'];okay('quote snapshot / prepay failure / edit lock')
 mode('selection-error');tap('.pay-btn');d=data();assert d['selectionUnavailable'] and counts()['prepay']==1
 assert '商品或取餐时间已不可用' in call('automation_element_action',selector='.quote-error',action='text');tap('.quote-error button');tap('.pickup-bar');tap('.pk-time[data-t="12:00"]');tap('.cart-go');d=data();assert d['pickup']['time']=='12:00' and d['form']['contact']=='合成联系人' and not d['quoteError'];okay('selection unavailable / edit and retained form')
 tap('.rsv-picked');tap('.cart-ico-wrap');tap('.sheet-clear');tap('.return-confirm');d=data();assert d['cartEmpty'] and d['form']['orderNote']=='合成整单备注';n=counts()['quote'];tap('.pay-btn');assert counts()['quote']==n;okay('empty cart visible return / no quote')
 setup('normal','button');inp('contact','保留姓名');inp('orderNote','保留备注');mode('menu-error');tap('.rsv-picked');assert data()['listState']=='error';tap('.return-confirm');assert data()['form']['contact']=='保留姓名';okay('menu load failure visible return')
 setup('normal','button');inp('contact','保留姓名');tap('.rsv-picked');call('automation_wx_api',action='mock',method='navigateBack',function_declaration='function(options){options.fail({errMsg:"navigateBack:fail synthetic"});}')
 try:
  tap('.return-confirm');assert call('automation_runtime_info',action='currentPage')['currentPage']['path']=='pages/menu/menu';assert ev("return getCurrentPages().slice(-1)[0].selectComponent('#toast').data.msg;")=='暂时无法返回确认页，请稍后重试'
 finally:call('automation_wx_api',action='restore',method='navigateBack')
 tap('.return-confirm');assert data()['form']['contact']=='保留姓名';okay('failed return preserves page and retained confirmation')
 setup('normal','button');inp('contact','保留姓名');mode('menu-hold');tap('.rsv-picked');assert data()['optionState']=='loading';tap('.return-confirm');release('menu-options');d=data();assert d['pickup']['time']=='11:30';assert ev('return getApp().globalData.pickup.time;')=='11:30';okay('late unloaded menu response cannot change pickup')
 setup('quote-hold','button');inp('contact','锁定姓名');inp('extraPhone','10000000000');tap('.pay-btn');assert data()['paymentState']=='quoting';assert call('automation_element_action',selector='input[data-k="contact"]',action='property',name='disabled') is True
 # Real page hide/show, preserving its transaction; only separate fixture page is navigated.
 nav('navigateTo','/pages/fixture/fixture');nav('navigateBack');assert data()['paymentState']=='quoting';release('quote');d=data();assert d['paymentState']=='pending' and d['editingLocked'];tap('.pay-btn');assert counts()['quote']==1 and counts()['prepay']==1 and counts()['confirm']==['88','88','88'];mode('confirm-error');tap('.pay-btn');assert data()['paymentState']=='error' and data()['editingLocked'];shot('pending-error');okay('background return / pending same prepayment / error retains lock')
if group in ['customer','customer_tail']:
 setup('extra-hold','button');inp('contact','合成姓名');inp('extraPhone','10000000000');tap('.field > view');tap('.pay-btn');assert counts()['quote']==0 and counts()['extra']==1;release('extra');assert not data()['extraPhoneSaving'];okay('extra phone save blocks quote')
 setup('normal','button:nth-child(3)');assert not data()['returnToConfirm'];tap('.cart-go');assert call('automation_runtime_info',action='currentPage')['currentPage']['path']=='pages/confirm/confirm';okay('ordinary menu checkout unchanged')
elif group=='merchant':
 setup('normal','button:nth-child(4)');d=data();assert d['listState']=='ready' and d['list'][0]['meta']['label']=='查看并备好';assert 'itemCount' not in d['list'][0];assert '2026-10-12' in call('automation_element_action',selector='.aorder-meta',action='text');shot('merchant-list');tap('.adv-btn');cur=call('automation_runtime_info',action='currentPage')['currentPage'];assert cur['path']=='pages/admin-order-detail/admin-order-detail' and cur['query']['id']=='1';okay('list fields and action targets same order')
 setup('orders-error','button:nth-child(4)');assert data()['listState']=='error';mode('orders-empty');tap('.empty button');assert data()['listState']=='empty';okay('error and retry to real empty')
 setup('orders-hold','button:nth-child(4)');assert data()['listState']=='loading';tap('.seg[data-l="制作中"]');release('orders-2');assert data()['list'][0]['id']=='2';release('orders-1');assert data()['list'][0]['id']=='2';okay('reverse lane responses ignore old result')
 mode('orders-empty');call('automation_element_action',selector='.search-in',action='input',value='999');d=data();assert d['kw']=='999' and d['listState']=='empty';assert '没有匹配的订单' in call('automation_element_action',selector='.empty',action='text');okay('search empty distinct from lane empty')
else:raise ValueError(group)
(out/'result.json').write_text(json.dumps({'result':'PASS','group':group,'project':project,'boundary':'Synthetic UI2, not native payment or UAT'},ensure_ascii=False,indent=2));print('ALL PASS',flush=True)
