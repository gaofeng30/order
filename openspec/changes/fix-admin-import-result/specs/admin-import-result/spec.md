## ADDED Requirements
### Requirement: 已提交导入结果可读取
后台 SHALL 通过已登记的数据库 RPC 读取当前主账号已经提交的导入结果，且不得因客户端漏登记而返回服务不可用。

#### Scenario: 三图导入后读取结果
- **WHEN** 主账号确认导入含三张已上传图片的菜品
- **THEN** 提交成功后可读取批次结果且刷新后菜品图片保持一致

#### Scenario: 普通客户端分发
- **WHEN** 后端调用 order_import_result_v1
- **THEN** 真实客户端发送对应 RPC 请求并解码响应
