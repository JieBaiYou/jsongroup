# JSONGroup - 基于分组的 JSON 序列化 Go 库

JSONGroup 是一个功能强大的 Go 语言库，用于按照字段标签的分组信息有选择地序列化结构体字段。这对于需要根据不同场景（如用户角色、权限级别等）输出不同字段集合的应用场景特别有用。

## 核心功能

- **按分组选择性序列化**：根据字段上的`groups:"group1,group2"`标签决定是否序列化该字段
- **支持嵌套结构**：正确处理嵌套结构体、匿名字段和多层嵌套
- **全面类型支持**：处理指针、切片、数组、map 等复杂类型
- **多分组逻辑**：支持"或"（默认）和"与"逻辑，灵活控制字段的选择条件
- **自定义顶层包装**：可选择是否添加顶层包装键
- **兼容标准 JSON 标签**：保留`json:"name,omitempty"`等标准功能
- **零值处理**：支持`omitempty`或强制输出 null 值
- **防御功能**：内置循环引用检测和递归深度限制，防止栈溢出
- **详细错误信息**：提供丰富的错误上下文，便于调试
- **性能优化**：内置类型缓存和 LRU 淘汰机制，优化反射性能

## 安装

```bash
go get github.com/JieBaiYou/jsongroup
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/JieBaiYou/jsongroup"
)

type User struct {
    ID       int    `json:"id" groups:"public,admin"`
    Name     string `json:"name" groups:"public,admin"`
    Email    string `json:"email" groups:"admin"`
    Password string `json:"password" groups:"internal"`
}

func main() {
    user := User{
        ID:       1,
        Name:     "张三",
        Email:    "zhangsan@example.com",
        Password: "secret123",
    }

    // 仅序列化public组字段
    publicJSON, _ := jsongroup.MarshalByGroups(user, "public")
    fmt.Println(string(publicJSON))
    // 输出: {"id":1,"name":"张三"}

    // 序列化admin组字段
    adminJSON, _ := jsongroup.MarshalByGroups(user, "admin")
    fmt.Println(string(adminJSON))
    // 输出: {"id":1,"name":"张三","email":"zhangsan@example.com"}

    // 序列化内部组字段
    internalJSON, _ := jsongroup.MarshalByGroups(user, "internal")
    fmt.Println(string(internalJSON))
    // 输出: {"password":"secret123"}

    // 序列化多个组（默认OR逻辑）
    combinedJSON, _ := jsongroup.MarshalByGroups(user, "public", "admin")
    fmt.Println(string(combinedJSON))
    // 输出: {"id":1,"name":"张三","email":"zhangsan@example.com"}
}
```

### 高级用法

```go
package main

import (
    "fmt"
    "github.com/JieBaiYou/jsongroup"
)

type Address struct {
    Street string `json:"street" groups:"admin,public"`
    City   string `json:"city" groups:"admin,public"`
    ZIP    string `json:"zip" groups:"admin"`
}

type User struct {
    ID      int     `json:"id" groups:"public,admin"`
    Name    string  `json:"name,omitempty" groups:"public,admin"`
    Email   string  `json:"email" groups:"admin"`
    Address Address `json:"address" groups:"public,admin"`
}

func main() {
    user := User{
        ID:    1,
        Name:  "张三",
        Email: "zhangsan@example.com",
        Address: Address{
            Street: "中关村大街1号",
            City:   "北京",
            ZIP:    "100080",
        },
    }

    // 使用AND逻辑 - 字段必须同时属于public和admin组
    opts := jsongroup.DefaultOptions().WithGroupMode(jsongroup.GroupModeAnd)
    andJSON, _ := jsongroup.MarshalByGroupsWithOptions(user, opts, "public", "admin")
    fmt.Println(string(andJSON))
    // 输出中只包含同时带有public和admin标签的字段

    // 添加顶层包装键
    wrapOpts := jsongroup.DefaultOptions().WithTopLevelKey("user")
    wrappedJSON, _ := jsongroup.MarshalByGroupsWithOptions(user, wrapOpts, "public")
    fmt.Println(string(wrappedJSON))
    // 输出: {"user":{"id":1,"name":"张三","address":{"street":"中关村大街1号","city":"北京"}}}

    // 设置nil值输出为null而不是跳过
    nullOpts := jsongroup.DefaultOptions().WithNullIfEmpty(true)
    emptyUser := User{ID: 1, Name: "张三"}
    nullJSON, _ := jsongroup.MarshalByGroupsWithOptions(emptyUser, nullOpts, "public")
    fmt.Println(string(nullJSON))
    // 输出: {"address":null,"id":1,"name":"张三"}

    // 设置最大递归深度，防止栈溢出
    safeOpts := jsongroup.DefaultOptions().WithMaxDepth(10)
    // 适用于处理复杂嵌套结构，防止无限递归
}
```

### 直接获取 map 结果

```go
// 如果需要在序列化为JSON前对数据做进一步处理
userMap, _ := jsongroup.MarshalToMap(user, "public")
// 现在可以修改map后再进行JSON序列化
userMap["extra_field"] = "额外信息"
finalJSON, _ := json.Marshal(userMap)
```

## 高级配置选项

JSONGroup 提供了多种配置选项来满足不同需求：

| 选项          | 方法                       | 默认值        | 说明                                |
| ------------- | -------------------------- | ------------- | ----------------------------------- |
| 分组模式      | `WithGroupMode`            | `GroupModeOr` | 设置字段选择的逻辑模式（OR 或 AND） |
| 顶层包装      | `WithTopLevelKey`          | `""`          | 添加顶层包装键                      |
| 标签键        | `WithTagKey`               | `"groups"`    | 自定义标签名                        |
| 空值处理      | `WithNullIfEmpty`          | `false`       | 配置 nil/空值的处理方式             |
| 忽略 nil 指针 | `WithIgnoreNilPointers`    | `true`        | 是否忽略所有 nil 指针字段           |
| 最大递归深度  | `WithMaxDepth`             | `32`          | 设置最大递归深度限制                |
| 循环引用检测  | `WithDisableCircularCheck` | `false`       | 是否禁用循环引用检测                |
| 缓存大小      | `WithMaxCacheSize`         | `1000`        | 设置字段缓存的最大条目数            |

### 安全性与健壮性

JSONGroup 内置多项安全保护机制，防止在处理复杂数据结构时出现问题：

1. **循环引用检测**：自动检测并处理循环引用结构，防止无限递归和栈溢出
2. **递归深度限制**：默认限制最大递归深度为 32 层，可自定义调整
3. **缓存大小限制**：使用 LRU 策略限制字段缓存大小，防止内存泄漏
4. **异常恢复机制**：捕获并转换反射操作的 panic 为标准错误
5. **详细错误上下文**：错误信息包含路径、类型、值等详细上下文

示例：

```go
// 处理包含循环引用的结构
type Node struct {
    Value int    `json:"value" groups:"public"`
    Next  *Node  `json:"next" groups:"public"`
}

// 创建循环引用
node1 := &Node{Value: 1}
node2 := &Node{Value: 2}
node1.Next = node2
node2.Next = node1

// 正常处理，不会导致栈溢出
result, err := jsongroup.MarshalByGroups(node1, "public")
if err != nil {
    // 错误会包含循环引用的详细信息
    fmt.Println(err) // 输出：检测到循环引用 at path 'Next.Next'
}
```

## 处理复杂嵌套结构

JSONGroup 能够正确处理复杂的嵌套结构：

```go
type Profile struct {
    Age  int    `json:"age" groups:"public"`
    Bio  string `json:"bio" groups:"public"`
    Role string `json:"role" groups:"admin"`
}

type ComplexUser struct {
    User    // 匿名嵌入
    Profile Profile            `json:"profile" groups:"public,admin"`
    Tags    []string           `json:"tags" groups:"public"`
    Meta    map[string]string  `json:"meta" groups:"admin"`
}
```

嵌套结构中的每个字段也会根据指定的分组进行筛选。

## 错误处理

JSONGroup 提供详细的错误信息，便于调试和处理各种异常情况：

```go
result, err := jsongroup.MarshalByGroups(complexObject, "public")
if err != nil {
    switch e := err.(type) {
    case *jsongroup.Error:
        // 可以访问错误的详细信息
        fmt.Printf("错误类型: %v\n", e.Type)
        fmt.Printf("错误路径: %s\n", e.Path)
        fmt.Printf("错误消息: %s\n", e.Message)
    default:
        fmt.Printf("未知错误: %v\n", err)
    }
}
```

## 性能考虑

JSONGroup 使用多种策略优化性能：

1. **类型缓存**：缓存结构体字段信息，减少重复反射开销
2. **LRU 淘汰机制**：控制缓存大小，平衡内存占用和性能
3. **容量预分配**：为 map 和 slice 预分配合理容量，减少扩容开销
4. **延迟初始化**：只在实际需要时进行计算和分配

## 测试与验证

JSONGroup 包含全面的测试套件，确保库的功能性和可靠性：

### 测试文件及其作用

1. **marshal_test.go**：核心功能测试

   - 测试基本的序列化逻辑
   - 验证分组功能（OR/AND 逻辑）
   - 验证顶层键包装功能
   - 测试复杂的嵌套结构

2. **basic_types_test.go**：基本数据类型支持测试

   - 测试所有 Go 基本类型的正确序列化（整数、浮点数、布尔、字符串等）
   - 验证时间类型处理
   - 测试特殊值处理（NaN、无穷大等）

3. **complex_types_test.go**：复杂数据结构测试

   - 测试嵌套结构体、切片、映射的正确序列化
   - 验证多级嵌套数据处理
   - 确保正确应用分组逻辑到复杂结构

4. **edge_cases_test.go**：边缘情况测试

   - 测试循环引用检测
   - 验证 nil 指针和空值处理
   - 测试递归深度限制功能

5. **safety_test.go**：安全性和鲁棒性测试

   - 验证异常处理和错误恢复机制
   - 测试深度限制和防御性功能
   - 检查内存和资源使用

6. **benchmark_test.go**：性能测试
   - 与标准 JSON 库性能对比
   - 测试不同规模数据的序列化性能
   - 验证缓存机制的有效性

### 运行测试

```bash
# 运行所有测试
go test github.com/JieBaiYou/jsongroup

# 运行性能测试
go test github.com/JieBaiYou/jsongroup -bench=.

# 运行带内存分析的性能测试
go test github.com/JieBaiYou/jsongroup -bench=. -benchmem

# 运行特定测试文件
go test github.com/JieBaiYou/jsongroup -run=TestBasicTypes
```

## 最近优化与增强

最近的版本包含以下重要优化和增强：

1. **安全性增强**：

   - 添加循环引用检测，防止无限递归
   - 实现可配置的递归深度限制（默认 32 层）
   - 完善的 panic 恢复和错误转换机制
   - 修复测试代码中的潜在无效赋值问题，提高代码质量

2. **性能优化**：

   - 改进字段缓存，添加 LRU 淘汰策略
   - 可配置的缓存大小限制，防止内存泄漏
   - 针对大型结构体的预分配优化
   - 使用 Go 1.22+ 的 range over int 语法优化循环结构，提高代码可读性
   - 移除冗余的循环和条件检查，减少不必要的计算

3. **功能完善**：

   - 增加忽略 nil 指针的选项（默认开启）
   - 改进空值处理逻辑，支持 null 和忽略两种模式
   - 支持更多基本类型（包括复数）
   - 处理特殊浮点值（NaN 和 Infinity）
   - 使用 slices.Contains 替代手动循环，简化代码

4. **错误处理**：

   - 引入自定义错误类型，提供详细错误上下文
   - 错误信息包含路径、值和错误类型
   - 支持错误链和标准错误接口

5. **代码质量**：

   - 全面使用 Go 1.22+ 的新特性进行代码现代化
   - 使用 any 代替 interface{} 类型声明，符合最新 Go 语法建议
   - 消除静态代码分析工具（如 go vet 和 golangci-lint）检测的问题
   - 遵循最佳实践，提高代码可维护性

6. **测试覆盖**：
   - 全面的测试套件，覆盖正常和边缘情况
   - 详细的性能基准测试
   - 验证在复杂数据结构下的表现
   - 添加安全性和极限情况测试

## 许可证

MIT License

## 贡献

欢迎提交问题报告、功能请求和 Pull Request！
