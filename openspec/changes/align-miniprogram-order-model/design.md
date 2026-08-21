## 选定方案

### 一个格式化入口，页面永不做除法

新增 `utils/money.js`：

```js
function yuan(cents) { ... }   // 整数分 → '32.00'
```

`money` 组件加 `cents` 布尔属性；传 `cents` 时组件内部调 `yuan`。页面把整数分原样交给组件，**不做 `/ 100`**。

理由是 PC 端已经验证过的一条：一旦允许页面自己转换，四舍五入就会分散到 N 个地方，而 N 个地方迟早会有一个用 `toFixed(2)`（银行家舍入不一致）或先乘后除。把入口收成一个，门禁才能断言「没有第二处除法」。

不复用 `catalogStore.formatCents`：它在输入非法时抛 `CatalogError('CATALOG_UNAVAILABLE')`，那是目录接口的错误语义，用在渲染金额上会把一个显示问题伪装成网络故障。

### 剩余时间现算，不落库

删除 `minsToPickup` 字段，改为 `data.minsToPickup(o)`：

```
(pickupDate 相对当前营业日的天数) × 1440 + toMins(pickupTime) − NOW_MINS
```

`canCancelReserve(o)` 与订单详情的取消文案都走它。这样时钟一旦变成真的，判定自动跟着走。

`slotMins(off, t)` 已经是同一套算法的选择器版本（输入是 `{off, time}` 而非订单），两者共用 `toMins` 与 `NOW_MINS`，不再各写一份。

### 整单口味改为聚合行内口味

删掉 `flavor` / `flavors` 之后，展示信息不能凭空消失。做法与 PC `pages/orders.js` 一致：把 `items` 每行的口味与备注去重拼成一条 band，再接上 `orderNote`。

这不是「换个地方存」。口味本来就绑定在具体菜品上（§15.6.4），整单级字段是即时单时代的残留：一张单里两个菜要不同口味时，整单级字段根本表达不了。聚合展示是把已经正确的数据正确地显示出来。

### 折扣字段先立住，算价链路不动

`discountRate = 100`、`discountCut = 0`、`isStaff = false` 写死。身份识别链路还没接后端（§16.5），此刻实现折扣只能靠猜白名单命中与否。

写死不是占位符：它是当前真实业务状态的准确表达 —— 一期在手机号授权就位前，**所有人都是访客价**（§5.6「访客按原价结算」）。结算恒等式 `subtotal − discountCut === total` 在这组取值下同样成立且被门禁断言，接上折扣后无需改动断言本身。

## 边界

- 菜品价格仍是元。订单读的是自身的分级快照，不读菜品价，所以订单侧在本 change 之后已完全是分。两者的收敛属 §16.5 的另一件事。
- 不动 `catalogStore` / `catalogApi`：它们本来就是 `price_cents`。`confirm.pay()` 直接取 `price_cents` 写入订单，中间不再经过 `Number(price_text)` 这一次元的往返 —— 那次往返正是当前浮点数的来源。
- 不动状态机、泳道、搜索与取餐号口径。

## 会使旧验证失效的变更面

- 订单任一金额字段的单位变化。
- `minsToPickup` 重新变回字段。
- 出现第二处金额格式化实现或页面内除法。
