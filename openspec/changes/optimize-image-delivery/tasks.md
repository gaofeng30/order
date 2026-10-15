# 任务
- [x] 基线：9 张封面 9.62 MB；同图处理 1.65 MB → 40.8 KB，现有 COS 能力可用。
- [x] Red：Go TestDisplayURLUsesSignedBoundedImageProcessing 失败；Node 裁剪 3000 !== 640。
- [x] Green：展示 URL 签入固定版本与处理参数；PC 有界裁剪与透明度判断。
- [x] 完成上传、旧图、受控网络及端上走查，范围和限制见 verification.md。
- [x] Writer 检查；8e2aef168f2eb49eef528183ecdfd5fdb0f8453c 在干净 detached worktree 独立验证通过。最终报告/测试增强候选另留精确 SHA 回执。
- [x] 部署 UAT：order-api-008 正常承载 100% 流量；两份 PC 静态文件已部署，真实浏览器完成上传和导入。
- [ ] GitHub 网页逐文件提交并核对完整树；最终发布回执在交付记录中提供，本文不预记成功。
- [x] Refactor：本次只改现有两个生成点，不增加变体表、迁移、后台任务或缓存框架。

命令与结果：
- Red：`go test ./services/api/internal/objectstore -run TestDisplayURLUsesSignedBoundedImageProcessing -count=1` → display URL must resize the exact confirmed version。
- Red：`node --test tools/tests/pc-image-crop.test.cjs` → 3000 !== 640；透明小图原契约测试已通过。
- Green：`go test ./services/api/...` 全部通过（无测试包不视为 E2E）；`node --test tools/tests/pc-image-crop.test.cjs tools/tests/pc-admin-errors.test.cjs` 6/6 PASS。

- Red/Green（走查）：实际 products.js 页面 VM 输入 3 次导致 1→4 次读取；修复后 1 次；上下架和重进页面仍重新读取。
- 最终针对性验证：`go test ./services/api/internal/objectstore -count=1`；`node --test tools/tests/pc-products-performance.test.cjs tools/tests/pc-image-crop.test.cjs tools/tests/pc-admin-errors.test.cjs`，7/7 PASS。
