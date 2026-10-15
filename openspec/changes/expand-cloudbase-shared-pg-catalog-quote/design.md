# 固定业务命令与事实快照

## 唯一实施方案与当前阶段

以原native业务port为权威，索引为外部shared-pg-catalog-quote-port-map.md及实施brief。非Quote的Go保留现有Service/handler、selector、staffidentity.Resolve、quotepricing.Calculate、canonical digest和URL投影；Quote的源依赖派生迁入锁内SQL，Go仍以原算法独立校验返回聚合；存储适配只用普通服务UID固定CallRPC。不得逐query模拟Tx，不做HTTP跨调用session锁，不保留native fallback或新增公共配置。

17个非Quote RPC的name/type/named参数已批准冻结。其输入优先级仍须真实native差分证明，发现反例只修最小实现；只有必须改变ABI才向需求owner给具体反例。两个Quote最终ABI已按活性反例批准，SQL实现仍须差分验证。累计阶段47/最终目标49仅表示函数清单，不是验收结果。

## 已冻结17个非报价签名

```sql
order_catalog_browse_v1() RETURNS jsonb
order_catalog_detail_v1(p_product numeric,p_service_date date) RETURNS jsonb
order_catalog_categories_v1() RETURNS jsonb
order_catalog_products_v1(p_service_date date) RETURNS jsonb
order_catalog_product_v1(p_product numeric,p_service_date date) RETURNS jsonb
order_catalog_command_v1(p_user numeric,p_kind text,p_category numeric,p_product numeric,p_name_present boolean,p_name_hex text,p_description_hex text,p_price_cents bigint,p_meal_period_hex text,p_images jsonb,p_enabled boolean,p_listed boolean,p_service_date_hex text,p_sold_out boolean,p_ordered_ids numeric[],p_confirm_status text,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz) RETURNS jsonb
order_menu_snapshot_v1(p_service_date date) RETURNS jsonb
order_menu_pickup_facts_v1(p_date_a date,p_date_b date) RETURNS jsonb
order_storefront_get_v1() RETURNS jsonb
order_storefront_admin_v1() RETURNS jsonb
order_storefront_configure_v1(p_user numeric,p_store_status_hex text,p_notice_hex text,p_pickup_point_hex text,p_pickup_step_min integer,p_meal_periods jsonb,p_service_dates jsonb,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz) RETURNS jsonb
order_storefront_launch_read_v1() RETURNS jsonb
order_storefront_launch_configure_v1(p_user numeric,p_layer_present boolean,p_enabled boolean,p_image_key_hex text,p_image_key_json_hex text,p_center_x text,p_center_y text,p_width_ratio text,p_aspect_ratio text,p_confirm_status text,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz) RETURNS jsonb
order_image_confirmed_v1(p_target_key_hashes text[]) RETURNS jsonb
order_display_pricing_facts_v1(p_user numeric) RETURNS jsonb
order_merchant_soldout_replay_v1(p_user numeric,p_product numeric,p_service_date date,p_sold_out boolean,p_operation_hash text,p_request_id_hash text,p_request_digest text) RETURNS jsonb
order_merchant_soldout_command_v1(p_user numeric,p_product numeric,p_service_date date,p_sold_out boolean,p_now timestamptz,p_locator_found boolean,p_category numeric,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz) RETURNS jsonb
```

所有返回单jsonb闭合facts/envelope。无默认参数；未消费字段显式NULL，不接受任意SQL/table/where。读端口保持当前认证职责：公共缺bearer可匿名，坏bearer401；Admin PC认证仍原路由前置。

## Go构造及依赖

Catalog NewSharedPGRepository实现Reader；NewSharedPGAdminApplication(client,images,confirmed)实现AdminApplication。Menu NewSharedPGRepository实现两Reader并复用原meal selector。Storefront NewSharedPGRepository/ NewSharedPGAdminApplication(client,confirmed)分别读与管理。objectstore.NewSharedPGConfirmationReader仅实现Confirmed，不用假Record满足完整ConfirmationStore。staffdiscount.NewSharedPGPricing仅给原PricingApplication注入完整只读loader。merchantsoldout.NewSharedPGCommander保留原clock及SoldOutCommander。client为包内最小CallRPC接口，方法中写死RPC名。

## 参数表达与晚消费顺序


- numeric ID/version：Go uint64 格式化十进制 string；SQL 验 integer 与 0..MaxUint64，何处允许0依 native kind 而定。bigint 钱/数量：Go int64，JSON 原整数，不用 float64；源数据库 NUMERIC 可能超过 int64，先精确读取后按 native 返回 Unavailable，不静默截断。公开金额/原始商品 cents 的 uint32 边界不混同 Quote 金融 int64。
- `*_hex` 是原 bytes 的小写 hex。Name 由原 catalogName NFKC+TrimSpace 生成，`p_name_present` 保留 UPDATE_CATEGORY 原 raw Name 是否为空；SQL 只在原 kind/锁序位置检查 byte 长度/解码并写。description/meal/date/notice 等晚消费字段用 hex，避免 REST 参数转换提前拒绝 NUL/非法UTF8，压过 owner/old receipt/target 优先级。
- Catalog p_images 是固定 ≤3 数组：`{object_key_hex:string,object_key_json_hex:string,url_json_hex:string,sort_order:integer}`。url_json_hex 是 Go json.Marshal(URL scalar) 的 UTF8 文本 hex，保留 native JSON 对非法UTF8替换及内部NUL在 receipt JSONB 写入处失败；不是新 URL 验证规则。object_key_hex保留原bytes用于原长度/重复检查，object_key_json_hex为Go json.Marshal(ObjectKey scalar)文本hex，与url_json_hex同样只在原晚阶段解JSON写DB/receipt；native非法UTF8首返回原bytes但持久化和replay为替换串，fresh CREATE/UPDATE产品的Go结果仅从原cmd.Images恢复首返回，replay必须读真实receipt，不能恢复raw。stored images只写object_key，原完整CatalogCommand Marshal摘要仍包含所有ignored字段。具体 scalar/JSON 验证需先 native NUL/invalidUTF8 Red，不能未经验证把 URL 转换提前。
- Configure p_meal_periods 固定数组：`{code_hex,cutoff_hex,pickup_from_hex,pickup_to_hex}`；Name 不参与业务写，但仍参与原完整 SettingsCommand 摘要。p_service_dates 为 `{date_hex,status_hex}` 数组，复制后按原 Go 日期排序；保留重复日期及原逐次 upsert，不去重、不删除遗漏日期。时间只在原 UPDATE 位置 cast，原 lunch/dinner 检查顺序不移到 owner 前。
- Launch 四个比例参数是精确 float64 的 round-trip 十进制文本（strconv.FormatFloat，非金融）；在原 UPDATE 位置 cast double precision。这样 direct Go port 的 NaN/Inf 不因 JSON 编码被提前统一拒绝，实际 constraint/receipt Marshal 错误仍按 native 顺序。`p_request_digest=NULL` 仅表示原 Go Marshal 已失败，SQL只能在原 Append 阶段报告 Unavailable，不能在 owner/constraint 之前消耗该状态；此边界必须先取 native 差分再冻结。`p_layer_present` 区分 nil 与 Enabled=false 的原 request digest，业务两者都清五项 NULL。HTTP 原有限数值校验不改。
- `p_confirm_status` 固定 CONFIRMED/NOT_CONFIRMED/UNAVAILABLE；由真实 Confirmed reader预读产生。Catalog 当前 owner→old receipt→确认结果；Launch 当前 owner→确认结果→singleton→写→receipt，没有早期 replay。不能统一流程。
- 商户 M receipt 元数据：scope仍SQL从当前 account派生 SHA256(merchant:<user>:<account>)；operation是Go SHA256(operation:<原key>)，request_id是SHA256(request:<原requestID>)或原空值NULL；request_digest是原完整 command json.Marshal 的 SHA256。不把 raw key/requestID 强塞PG text以新增NUL限制。收据解析返回原 `before_state_json::text`/`response_json::text` 给原Go decoder语义，不能强制这些原来宽松的 receipt 采用 Quote严格 decoder。
- Image confirmed 仅接受1..3合法Key经原ValidImageObjectKey核验后的 TargetKeyHash hex数组；duplicate输入仍按native map归并。SQL固定 action=object.image.confirm、target=IMAGE_OBJECT、entry_kind=COMMAND_RECEIPT，返回真实存在hash集合；不增owner过滤、不下载对象、不依据fixturebool判确认。
- Merchant soldout p_now是新命令上海今日/明日校验时刻；p_receipt_at独立是原第二次clock，UTC截微秒、zero才Unavailable。历史replay不消费now/date当前性。p_locator_found/p_category从 `order_catalog_product_v1` 的最小原始 locator facts 取得（只解析 found/category，不做 AdminProduct/PublicURL 投影），发生在原clock日期检查之后；锁内 category/product 再核一致，不把 JOIN附属事实验证新增为 locator 限制。

## 读快照与公共投影

Catalog Browse保留active空分类；Menu过滤后移除空分类。closed/missing date的Menu200 flags与Detail503不同，soldout Detail仍200；不得套同一valid模型。商品任一URL错误整体503，Storefront launch URL失败仅省略launch。管理Catalog/Storefront事实不套公共更严valid；nil价格、SQL NULL组、空数组分别保留。

Browse原LEFT JOIN产品字段全NULL与完整行区别；Detail/Menu完整facts合为单SELECT，明确内部一致读，不谎称native原RC为RR。DisplayPricing原RR必须一个SQL statement取user/settings/staff，不能多次VOLATILE内部SELECT；未绑手机号时缺discount仍可nil pricing。Storefront Admin继续数据库上海日期与未来已有31条，不改调用方today或补空日期。

员工primary优先；extra展示匹配width.Fold+NFC+ASCII空格删除，持久name_key用原NFKC+Unicode空白，不合并两个算法。金融按原单价half-up再乘quantity/求和，uint32展示价与int64报价不混，nil不是0。

## 不可拆原子写

Catalog：确认预读→OWNER UPDATE锁→旧receipt→确认错误→对应十kind行锁/写→receipt。完整原命令Marshal包含ignored字段；UPDATE_PRODUCT原response Listed=true但DB不改is_listed；PC售罄不能套商户今日明日限制。仅命名receipt重复完整回滚再重放，其他23505/23503/23514仍原Conflict。

Configure：原最少输入→owner→singleton→日期排序逐条upsert→lunch/dinner锁→原列/version→同事务Admin结果→receipt。无early replay，重复receipt必须回滚所有写再读历史；不删未提供日期、不改name/address/flavors。Launch确认错误早于原receipt，五列NULL组/NaNInf晚消费顺序按native。独立Go audit clock与DB业务clock不合并，旧路径不因新统一校验改变优先级。

MerchantSoldOut：shape→当前account SHARE和严格旧receipt；新命令才clock今日明日与locator。account→date UPDATE→category UPDATE→product UPDATE并核category→datedflag；日期无需is_open，noop仍写receipt。保留同一个原一次retry预算与最终当前授权receipt消解，不盲重试unknown transport；scope及receipt材料遵循本port，不能套履约结果模型。

## Quote最终路线：同一次锁内完成派生

```sql
order_quote_create_v1(p_user numeric,p_key_hash text,p_request_digest text,p_operation_hash text,p_request_id_hash text,p_input jsonb,p_captured_at timestamptz,p_receipt_at timestamptz) RETURNS jsonb
order_quote_read_v1(p_user numeric,p_quote numeric) RETURNS jsonb
```

取消source RPC、source_proof与caller candidate，不为保留20数目设置空入口。本轮新增17+2=19，连原30共49。Go继续原normalize、捕获一次capturedAt、原key/request/operation hash与公共投影；SQL在固定key锁→existing/replay→原源SHARE锁顺序后，完成全部源校验、身份解析、meal selector、价格、product/source和snapshot canonical digest，再一次提交header/items/receipt。没有SOURCE_CHANGED循环或跨调用锁，没有新增次数、超时、表或通用Tx框架。

key advisory ID仍原派生，5秒只约束原key锁获取，不新增整体deadline；capturedAt在等待之前冻结，expires=min(capturedAt+10min,pickupAt)。existing优先于任何当前源错误。原state/missing/null/字节/日期/金融异常顺序必须逐native保持。

为保留独立审计clock和历史receipt优先，create首次p_receipt_at=NULL：key/replay→完整源锁与派生/验证，若新请求合法才返回NEED_RECEIPT，零持久写。Go此时才采原receiptNow，第二次调用重新key/replay→完整锁内派生→提交；第二次以当时锁内源计算，不依赖第一次proof，因此源写不能迫使反复刷新。原业务错误和历史replay不受新clock提前阻断；未知提交不盲目自动重发。year1等clock边界按原port差分，不统一新增拒绝。

Create返回完整原始不可变聚合及Created；Go使用既有stored-shape、金额与canonical digest算法独立核对，不能只信SQL自报成功。Create replay继续原严格receipt/quote_id验证；Read只完整aggregate/owner隐藏/shape/digest，不额外要求receipt。

## Unicode、金融与canonical派生硬门

SQL局部固化映射由当前go.mod/go.sum锁定的golang.org/x/text/width遍历全部Unicode scalar生成，绑定模块版本、Unicode数据版本、生成命令及输出hash。只保留width.Fold实际发生变化的映射，不用NFKC代替width折叠，不把数学字母/圈字等兼容字符额外折叠。可嵌入QuoteSQL局部常量/生成片段；不新建业务表、不新增通用RPC或SECDEF helper。

两套姓名算法分别实现：展示员工比较为width.Fold→NFC→只删除ASCII空格；持久name_key验证为NFKC→删除Go unicode.IsSpace对应字符，并保留原非空/UTF8字节长度400规则。真实组合字符反例已否决内置normalize作为最终语义；NFC/NFKC由锁定Go有效数据及原segment/reorder/compose算法在QuoteSQL局部实现，含原uint16重组键行为。必须以conformance corpus、全scalar及组合/宽窄/非ASCII空格/兼容字/astral截断碰撞/边界实际差分，不靠常见中文样例豁免。所有候选包括disabled行先校验，才primary enabled优先、随后extra匹配；保留NULL组与重复phone拒绝。

金融SQL可用numeric防宿主溢出，但必须在原每一个乘加位置检查int64范围：unit*rate先验MaxInt64，即使除100后可表示也不得放行原Go会拒绝的中间溢出；除100余数>=50半入、再乘quantity、再逐行累加。原rate/settings约束、payable>=1及错误顺序不变，不能让PostgreSQL自动round代替整数算法。

Canonical逐字段严格按原digest.go：domain先长度BE64、字符串UTF8字节长度与原bytes、bytes长度、bool为8字节0/1、uint64全范围、int64二补码、items/flavors原序、image存在bool、微秒Unix时间。最大uint64高低32用精确div/mod，不用numeric除法round；不经JSONB重序列化hash。ProductSource与Snapshot两digest都与原Go逐字段Golden及真实native聚合比较，合法内部NUL/非法UTF8在原消费阶段处理。SQL不能接受用户派生价格或摘要替代其源锁内计算。

## 原型结论及新实现准入

亲读真实PG16原型：源码SHA464d017740bd008a4a45424f6e413af9e732b4d1537b9e92f366858115163f20，普通日志SHA52c36b790c03f25e64fee89ab7b8e1ab9655a5e1532d79bbed0c7e69f68870ed，4实验PASS只说明观察符合预期。7次变更后第8轮成功；无deadline32次实际变更仍32次零写SOURCE_CHANGED，第33次人工取消才结束；native持price SHARE锁完成聚合，实际writer等待后提交；samekey赢家提交后phone失效仍replay。该证据否决乐观刷新为最终等价方案，不是共享Quote功能PASS，也不是完整facts/HTTP/CloudBase权限验证。

新原子SQL必须实际复测：持续价格/状态writer被源锁阻塞且一次报价完成；ctx取消退出无部分写；key赢家优先；跨cutoff/capturedAt；Unicode/selector/金额/hash全native差分；16同异key/receipt故障/unknownack；普通CloudBaseJWT同完整函数验证。不得用人工停止churn或新timeout证明进展。未完成前Quote功能仍未验收，非Quote17可并行。

## 最小权限及付款后续

root按固定实际SQL列清点RLS/列INSERT/UPDATE/DELETE与序列USAGE，沿已知26表16seq重置、额外对象/列/SECDEFfailclosed；普通服务无DDL/bootstrap/管理凭据。非Quote阶段仅授实际需要能力，不为未实现Quote提前授写；最终预计新增8关系/3序列需实际核对，数量非PASS证据。

付款prepare未来必须一个固定事务含Finalize完整source锁→quote/items→prepayment lease/请求digest再commit，之后Go调用支付provider；materialize未来同事务含prepayment/observation、owner/digest/amount校验、序号/order/items。两caller-Tx端口本轮NOT_IMPLEMENTED，不做独立finalize提交或预建通用Tx框架。

8MiB client/15s单次transport为既有基础设施边界，不新增item/qty限制。直接Application<=65535行与HTTP64KiB不同，列真实fixture规模/字节数；超过受审资源范围报告实际限制，不暗拆快照或更改限制。所有本地PG16证据不冒CloudBasePG17；URL/微信替身不冒外部真实联调。

## 生成权威及完整Go规范化窄移植

最终路线不使用PG内置normalize承担NFC/NFKC语义。实际先有4935样例长nonstarter 3/3差异，再有全scalar组合差分暴露原Go combine将两rune截uint16作为32bit键：U+10041/U+20041+0301映00C1，U+11F41+0301映1F45，PG结果不同。后者不是修改库版本或拒绝字符的理由；这是本次必须保留的native可观察行为。先前CGJ+PGnormalize方案已被否决，相关实验仅Red证据。

common writer唯一owned sharedpg_unicode_generate.go为buildignore生成器，绑定Go1.26.5、x/text v0.34.0及有效norm/width/Go Unicode15.0.0。允许生成时reflect.Uint读取Properties私有nLead/flags，必须断言模块/source SHA、有效buildtag、字段存在/uint8形状、公式范围及全scalar；不unsafe、不改库、不运行时反射。宽度映射、form-specific分解与CCC/组合属性、stream-safe lead/trail、重组map以锁定库为权威。recompMapPacked可用Go AST解析有效tables15.0.0.go常量串，校验每条8bytes、键/值范围与重复规则，不从未来buildtag未启用表取值、不手填反例例外。

QuoteSQL局部执行原NFC/NFKC的segment、分解、稳定CCC重排、阻塞与compose/Hangul逻辑，重组键明确同uint16(a)<<16|uint16(b)。不能把一般Unicode规范算法冒充原Go实现，也不能只在最终字符串补某几个astral碰撞。限定两种现有姓名规范化，无新通用Unicode API/RPC/业务表/SECDEF。由quote_sql_writer独占quote.sql整合生成片段，root独占Go/codec/tests，common仅生成器与外部数据/片段。

Stream-safe仍按原输入顺序、各form Properties计数：nLead=p.nLead，nTrail=p.flags&3；sum=count+nLead。sum>30时在本rune前插CGJ、count=nTrail；否则nLead0时count=nTrail，非零count=sum。Jamo V/T的CCC0不代表nLead0；不能先分解重排序后插CGJ。widthFold先于NFC规范化；name_key单独NFKC，最终ASCII空格/GoUnicode空白删除顺序保持。

输出须含版本/source/data SHA manifest，两次生成相同；嵌入SQL片段可从权威生成器精确再生。Go Unicode15官方NormalizationTest数据SHA fb9ac8cc154a80cad6caac9897af55a4e75176af6f4e2bb6edc2bf8b1d57f326、4935cases、全scalar组合及astral低16bit collision、30/31/60 nonstarter、不同CCC跨CGJ、分解增长、Jamo/Hangul、组合排除/阻塞、width折叠均须真实PG16和目标PG17执行对照原Go。primitive Green不等于Quote整体Green，身份/金额/hash/锁并发及云业务仍独立验收。

Catalog/Storefront NULL receipt材料预检保持各原优先级：Catalog owner/oldreceipt/confirm/target/input先于NEED_MATERIAL；Configure/Launch无early replay，最终写与重复receipt整批rollback保持。第一次零持久写、Go仅需材料分支采内部clock、第二次重检；不新增IsZero拒绝，不宣称后续constraint失败前clock采样次数全等，不搭试写框架。

Launch最终14参：p_image_key_hex后新增p_image_key_json_hex。前者保持原BYTEA存储，后者为Go json.Marshal(ImageObjectKey scalar)UTF8文本hex，仅原晚receipt阶段解码；首次成功返回原layer bytes，重放读取真实receipt替换串。公开Get/AdminRead仍原convert_from拒非法UTF8。nil确认只用于原native边界差分，生产继续真实Confirmed依赖；不能凭此放开图片确认规则。


## 当前部署与验证状态

最终部署以完整InstallSQL为authority，使用仅省29条逐字不变历史函数的delta；19旧history Query原样，正式管理迁移task-ae6113f2成功。PG16旧态升级/回滚2PASS；UAT最终49函数body/20RLS/8seq完整Check true，无源漂移。该编码修正没有改变业务源码/原44ledger，没有repair历史或绕过正式migration。细项hash、失败尝试与平台边界见tasks及external delta-deployment-binding.json。

Quote两RPC和非Quote17入口已完成真实普通服务JWT+原Go/HTTP的限定业务矩阵；provider/URL synthetic。Unicode PG16与PG17完整固定Go算法差分通过，不能将原语结果扩大为完整UAT。五写族故障/恢复与lostack、权限撤销及cleanup已闭合。Root Writer Gate总包已完成；候选/独验与main集成仍未完成。
