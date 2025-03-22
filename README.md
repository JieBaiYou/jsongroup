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
- **性能优化**：内置类型缓存，提升反射性能

## 安装

```bash
go get github.com/hs/jsongroup
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/hs/jsongroup"
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
    "github.com/hs/jsongroup"
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
    // ... 创建一个包含nil字段的结构 ...
    // 此时nil字段会序列化为null而不是被跳过
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

- **分组模式**：使用`WithGroupMode`设置 OR 或 AND 逻辑
- **顶层包装**：使用`WithTopLevelKey`添加顶层包装键
- **标签键定制**：使用`WithTagKey`自定义标签名（默认为"groups"）
- **空值处理**：使用`WithNullIfEmpty`配置 nil/空值的处理方式

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

## 性能考虑

JSONGroup 使用类型缓存来提高反射性能，适合在生产环境中使用。对于重复使用的结构体类型，只有首次访问时才会执行完整的反射解析，后续操作会直接从缓存获取结构信息。

## 许可证

MIT License

## 贡献

欢迎提交问题报告、功能请求和 Pull Request！
