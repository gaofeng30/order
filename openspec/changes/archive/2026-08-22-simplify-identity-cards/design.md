## 选定方案

### 按钮当壳，不当卡片

```xml
<button class="id-plain" open-type="getPhoneNumber" ...>
  <view class="id-card"> …图标 / 文字 / 箭头… </view>
</button>
```

`<button>` 只保留它唯一不可替代的能力 —— `open-type="getPhoneNumber"` 触发微信授权面板。样式上把它压成透明块：`display: block; width: 100%; margin: 0; padding: 0; background: transparent; text-align: left; font-size: inherit; color: inherit`，并用 `::after { border: none }` 去掉系统描边。

卡片的 flex 布局回到 `<view class="id-card">`，与用户端卡片用的是同一个类、同一份规则，因此两张卡必然一致 —— 不是「调到看起来一致」，而是没有第二份规则可以漂移。

上一版失败的根因是把这两件事压在了一个元素上：`.id-card` 要 `display: flex` 与 `padding: 40rpx`，`<button>` 默认要 `margin: auto` 与自己的内边距，二者在同一个盒子上互相干扰。分开之后各管各的。

### 门禁锁成因，不锁表现

断言「授权按钮上不得出现 `id-card` 类」。这比断言「卡片宽度是多少」有用：宽度断言在静态检查里做不到，而「布局类是否落在 button 上」是结构事实，可检、且正是这次出问题的根子。

### 描述文字整段删除

`.id-desc` 的样式规则一并删掉。留着一条没人引用的规则，下一个人会以为卡片还应该有副标题。

## 边界

- 不动授权逻辑：`onMerchantPhone` 的三条分支与文案保持上一个 change 的结论。
- 不动用户端卡片的跳转方式（普通 `bindtap`，不索取身份）。
- 不动 `.id-ico` / `.id-go` / `.id-name`：它们对两张卡是共用的。

## 会使旧验证失效的变更面

- 布局类重新落到 `<button>` 上。
- 卡片重新出现副标题。
