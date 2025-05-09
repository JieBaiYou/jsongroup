# 版本历史

## v0.2.0 (2024-03-23)

### 代码优化

- 使用 Go 1.22+的 range over int 语法优化循环结构，提高代码可读性
- 使用 slices.Contains 替代手动 for 循环检查，简化代码
- 使用 any 替代 any 类型声明，符合最新 Go 语法建议
- 移除冗余的循环和条件检查，减少不必要计算
- 修复静态代码分析工具检测的问题(golint, go vet)

### 错误处理

- 修复测试代码中的潜在无效赋值问题
- 优化错误路径与上下文信息

### 类型支持

- 改进对特殊浮点值(NaN, Infinity)的处理
- 完善对复数类型的支持

### 文档完善

- 添加版本管理指南
- 更新 README，增加最新优化与增强说明
- 完善注释与文档字符串

## v0.1.0 (2024-03-22)

### 功能特性

- 实现基于标签分组的 JSON 序列化
- 支持嵌套结构体、匿名字段和复杂类型
- 实现 OR/AND 逻辑的多分组控制
- 支持自定义顶层包装键
- 处理指针、切片、数组、map 等各种 Go 类型
- 特殊处理 time.Time 类型
- 支持空值处理选项：忽略或输出 null

### 安全性增强

- 添加循环引用检测，防止无限递归
- 实现可配置的递归深度限制（默认 32 层）
- 完善的 panic 恢复和错误转换机制

### 性能优化

- 实现字段信息缓存，减少反射开销
- 添加 LRU 淘汰策略，防止内存泄漏
- 可配置的缓存大小限制
- 针对大型结构体的预分配优化

### 代码质量

- 遵循 Go 最佳实践，提高代码可维护性

### 错误处理

- 引入自定义错误类型，提供详细错误上下文
- 错误信息包含路径、值和错误类型
- 支持错误链和标准错误接口

### 测试覆盖

- 全面的测试套件，覆盖正常和边缘情况
- 详细的性能基准测试
- 验证在复杂数据结构下的表现
- 添加安全性和极限情况测试
