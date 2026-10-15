# 图片交付性能

## 问题与目标
菜单小图直接读取原始 PNG。UAT 9 张封面合计 9,618,639 bytes，两张反馈慢图分别为 1,653,353 / 1,590,703 bytes。后台裁剪按原始分辨率输出 PNG，使问题持续出现。

## 唯一实施路线
- 使用现有 COS 下载时图片处理：最长边 640px、不放大小图、WebP quality 80；处理参数与固定 versionId 一起签名。原始对象、内容哈希、上传验证和绑定不变。
- PC 菜品裁剪最长边 640px；不透明图输出 JPEG quality .82，透明图保留 PNG。
- 已有图片自动使用同一展示处理规则，不迁移、不覆盖原对象、不修改商品数据。
- 范围内走查小程序首页/菜单/详情/购物车和商户页面、PC 主要页面请求与图片读取；只有可证实的问题纳入修复。

## 最小成功标准
新旧图片地址可解码；原版本绑定与签名约束不变；反馈两张菜品图传输量下降至少 90%；受控限速下实际下载改善。PC 三图上传、保存和重开、透明图通过。模拟器菜单和详情图片展示通过。证据标明本机、限速、模拟器与真机差异。

## 平台验证
现有私有 COS 对相同固定版本 imageMogr2 请求返回 200 image/webp，1,653,353 bytes → 40,820 bytes。无需开通新资源。腾讯云基础处理属于按量服务，官方说明每月基础处理免费额度 10TB；不使用高级压缩或购买套餐。
来源：https://cloud.tencent.com/document/api/1246/109193 、https://cloud.tencent.cn/document/product/460/47483 。

## 影响与回滚
仅展示派生内容改变，原始文件始终保留。回滚本次 API 和 PC 静态文件即可恢复原展示与上传编码。小程序源码不变，不以此声明体验版上传或真机验收通过。

## 质量与所有权
- base_sha: 5509ed8d2dd6ce01b22f1ca55ba7960552b8d503
- owner: 本任务 writer；没有并行 writer。
- owned paths: services/api/internal/objectstore/cos_versioned.go、read_path_test.go；apps/web-admin/ui/image-editor.js；tools/tests/pc-image-crop.test.cjs；openspec/changes/optimize-image-delivery/**。
- gate_type: W1；目标 UI2（PC 浏览器 UI1、微信开发者工具 UI2），实际等级随 verification.md 更新。无支付/账号权限/商品状态业务变更。
- Review: code-review 两轴无运行代码阻断；规范轴要求补本段声明，已补。开屏与上传确认地址同样派生，纳入验证。

## 走查追加的同范围修复
PC 菜品搜索/分类本来在本地过滤，却每次重新调用完整 listProducts。真实页面代码反馈循环 `node --test tools/tests/pc-products-performance.test.cjs`：输入 3 次使读取 1→4（Red）；复用本页读取 Promise 后为 1（Green）；重新进入页面仍读取，保存/上下架/排序路径仍调用刷新。追加 owned paths：apps/web-admin/pages/products.js、tools/tests/pc-products-performance.test.cjs。
