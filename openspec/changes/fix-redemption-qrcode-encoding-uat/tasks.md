# 修复真实取餐二维码编码

基线 `7e7f2f9a75e87a4f311e0ff7e2496022d8569f82`，复用当前UAT writer worktree与分支。本writer仅拥有 `components/qrcode/qrcode.js`、`utils/vendor/qrcode-generator.js`、同vendor LICENSE及本目录工件。商户扫码页面由另一writer独占，不改页面、backend、token、订单状态、认证或资金；root统一提交和发布。

用户禁止测试套件，本切片只做已授权最小解码契约、语法/差异与工件内容检查；不恢复或编写测试用例。标准decoder不是模拟商户扫码，不调用核销API。真实手机UI3单独记录。

- [x] 1.1 Red：用户真实手机报告132取餐码NO_RECOGNITION。macOS系统Vision对已保存无遮挡截图检测0个标准QR。组件源码没有value属性，固定seed7伪随机21×21，没有数据编码/格式/纠错；页面虽传真实value，组件完全忽略。
- [x] 1.2 实现及本地decode Green：替换为官方npm qrcode-generator@2.0.4未修改纯JS源码，MIT许可与版权保留；value属性/observer、UTF8 Byte、自动version、M纠错、4module白边、整数物理像素、异步generation保护。真实组件draw的本地canvas指令栅格公开ASCII118字节，Vision检测1码，解码长度118/SHA256与输入精确一致。`node --check`组件/vendor与owned diff check通过。仅登记DECODE_CONTRACT_GREEN/STATIC_CHECKED。
- [x] 1.3 Refactor：不需邻近重构；保留canvas导出image和页面状态契约，不改共享扫码入口/服务端。未运行suite/Gate、未commit/IDE/Cloud/资金，未打印或额外持久化真实token。
- [ ] 2.1 UI3：root统一候选编译与预览；真实手机读取同132既有READY凭证并正常相机核销COMPLETED。当前PENDING，不以旧可见伪二维码或本地公开载荷替代。
- [ ] 2.2 UI3相邻：正常value/state变化仍显示正确凭证，COMPLETED后现有页面隐藏码；与并行扫码writer统一真实取消/重扫/幂等读回。当前PENDING。

公开主源：https://registry.npmjs.org/qrcode-generator/2.0.4 ，官方tarball `https://registry.npmjs.org/qrcode-generator/-/qrcode-generator-2.0.4.tgz` 内 `package/dist/qrcode.js` 原样vendor（56694字节，SHA25679ec86f82856005b1c887905cfccfcfbec3821ca61c7fd5a952faa5f778f791c）。npm metadata与源码头均MIT、Copyright(c)2009 Kazuhiko Arase；单独保存MIT许可正文，未执行npm安装。

完整证据位于外部 `uat-e2e-rehearsal/qr-generator-repair-receipt.json`，真实截图只保留既有用户用途工件。当前IMPLEMENTED / DECODE_CONTRACT_GREEN，非UI3 PASS、CANDIDATE或INDEPENDENT_VERIFIED。
