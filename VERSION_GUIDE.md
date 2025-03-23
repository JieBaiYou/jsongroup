# JSONGroup 版本管理指南

本文档提供了如何对 JSONGroup 库进行版本管理的详细步骤。

## 语义化版本简介

我们采用语义化版本 (Semantic Versioning)，格式为 `vX.Y.Z`：

- **X**: 主版本号 - 当进行不兼容的 API 更改时增加
- **Y**: 次版本号 - 当添加向后兼容的新功能时增加
- **Z**: 修订版本号 - 当进行向后兼容的 bug 修复时增加

## 版本发布步骤

### 1. 更新 CHANGELOG.md

确保 CHANGELOG.md 中包含本次版本的所有变更内容。

### 2. 提交变更并创建标签

```bash
# 添加并提交 CHANGELOG.md
git add CHANGELOG.md
git commit -m "更新版本日志: vX.Y.Z"

# 创建带注释的版本标签
git tag -a vX.Y.Z -m "版本X.Y.Z发布说明"

# 查看当前标签列表
git tag -l
```

### 3. 推送标签到远程仓库

```bash
# 推送单个标签
git push origin vX.Y.Z

# 或推送所有标签
git push origin --tags
```

### 4. 在 GitHub 上创建 Release

1. 访问仓库的 Releases 页面: https://github.com/JieBaiYou/jsongroup/releases
2. 点击 "Draft a new release"
3. 选择您刚创建的标签
4. 填写发布标题和描述（可从 CHANGELOG.md 复制）
5. 点击 "Publish release"

## 目前的版本

**当前版本**: v0.1.0 (2024-03-22)

主要特性和改进:

- 实现基于标签分组的 JSON 序列化
- 支持嵌套结构体和复杂类型处理
- 循环引用检测和深度限制
- 性能优化与字段缓存
- 现代化 Go 语法和最佳实践
- 完整的测试覆盖

## 下一步版本计划

**计划版本**: v0.2.0

可能的改进方向:

- 更多性能优化
- 改进错误处理
- 添加新序列化选项
- 支持更多特殊类型

## 版本兼容性注意事项

- 保持次版本和修订版本的向后兼容性
- 主版本变更时需更新 go.mod 和导入路径
- 主版本为 0 时表示 API 不稳定，可能发生变化
