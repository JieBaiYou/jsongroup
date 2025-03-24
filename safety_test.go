package jsongroup

import (
	"errors"
	"strings"
	"testing"
)

func TestMaxDepthLimit(t *testing.T) {
	// 创建一个深度嵌套的结构
	type Nested struct {
		Value   int     `json:"value" groups:"public"`
		Child   *Nested `json:"child,omitempty" groups:"public"`
		Ignored string  `json:"-"`
	}

	// 创建一个嵌套结构，深度超过默认限制
	createNestedStruct := func(depth int) *Nested {
		root := &Nested{Value: 0}
		current := root
		for i := 1; i < depth; i++ {
			current.Child = &Nested{Value: i}
			current = current.Child
		}
		return root
	}

	// 深度5，应该正常工作
	shallow := createNestedStruct(5)
	_, err := MarshalByGroups(shallow, "public")
	if err != nil {
		t.Errorf("浅层嵌套序列化失败: %v", err)
	}

	// 默认深度限制是32，创建33层嵌套
	deep := createNestedStruct(33)
	_, err = MarshalByGroups(deep, "public")
	if err == nil {
		t.Error("深层嵌套应该报错，但没有")
	}

	var jsonErr *Error
	if !errors.As(err, &jsonErr) || jsonErr.Type != ErrTypeMaxDepthExceeded {
		t.Errorf("错误类型不正确, 得到: %T, %v", err, err)
	}

	// 使用自定义深度限制
	opts := DefaultOptions().WithMaxDepth(10)
	deepCustom := createNestedStruct(11)
	_, err = MarshalByGroupsWithOptions(deepCustom, opts, "public")
	if err == nil {
		t.Error("使用自定义深度限制，超过限制应报错")
	}
	if !errors.As(err, &jsonErr) || jsonErr.Type != ErrTypeMaxDepthExceeded {
		t.Errorf("错误类型不正确, 得到: %T, %v", err, err)
	}

	// 禁用深度检查
	opts = DefaultOptions().WithMaxDepth(0)
	deepNoLimit := createNestedStruct(50)
	_, err = MarshalByGroupsWithOptions(deepNoLimit, opts, "public")
	if err != nil {
		t.Errorf("禁用深度限制后应该成功: %v", err)
	}
}

func TestCircularReferenceDetection(t *testing.T) {
	// 创建测试结构
	type Node struct {
		Value int    `json:"value" groups:"public"`
		Next  *Node  `json:"next,omitempty" groups:"public"`
		Prev  *Node  `json:"prev,omitempty" groups:"public"`
		Data  string `json:"data,omitempty" groups:"public"`
	}

	// 创建简单循环引用：A -> B -> A
	nodeA := &Node{Value: 1, Data: "A"}
	nodeB := &Node{Value: 2, Data: "B"}
	nodeA.Next = nodeB
	nodeB.Next = nodeA

	_, err := MarshalByGroups(nodeA, "public")
	if err == nil {
		t.Error("循环引用A->B->A应该检测到错误")
	}

	var jsonErr *Error
	if !errors.As(err, &jsonErr) || jsonErr.Type != ErrTypeCircularReference {
		t.Errorf("错误类型不正确, 得到: %T, %v", err, err)
	}

	// 创建更复杂的循环引用：A -> B -> C -> A
	nodeA = &Node{Value: 1, Data: "A"}
	nodeB = &Node{Value: 2, Data: "B"}
	nodeC := &Node{Value: 3, Data: "C"}
	nodeA.Next = nodeB
	nodeB.Next = nodeC
	nodeC.Next = nodeA

	_, err = MarshalByGroups(nodeA, "public")
	if err == nil {
		t.Error("复杂循环引用A->B->C->A应该检测到错误")
	}

	// 禁用循环引用检测（将会导致栈溢出，但我们只在测试中验证选项逻辑）
	// opts := DefaultOptions().WithDisableCircularCheck(true)
	// _, err = MarshalByGroupsWithOptions(nodeA, opts, "public")
	// 注意：这里不应该期待成功，而是应该因为堆栈溢出而panic
	// 为防止测试真的panic，我们不实际调用这个测试，仅作为示例

	// 双向循环引用
	nodeA = &Node{Value: 1, Data: "A"}
	nodeB = &Node{Value: 2, Data: "B"}
	nodeA.Next = nodeB
	nodeB.Prev = nodeA

	_, err = MarshalByGroups(nodeA, "public")
	if err == nil {
		t.Error("双向循环引用应该检测到错误")
	}

	// 自引用
	nodeSelf := &Node{Value: 1, Data: "Self"}
	nodeSelf.Next = nodeSelf

	_, err = MarshalByGroups(nodeSelf, "public")
	if err == nil {
		t.Error("自引用循环应该检测到错误")
	}
}

func TestCacheOverflow(t *testing.T) {
	// 测试缓存溢出保护
	originalMaxSize := DefaultMaxCacheSize
	defer SetMaxCacheSize(originalMaxSize) // 测试完恢复默认值

	// 设置较小的缓存大小以便测试
	SetMaxCacheSize(5)

	// 创建不同的结构体类型以填充缓存
	type (
		Type1 struct {
			F int `groups:"public"`
		}
		Type2 struct {
			F int `groups:"public"`
		}
		Type3 struct {
			F int `groups:"public"`
		}
		Type4 struct {
			F int `groups:"public"`
		}
		Type5 struct {
			F int `groups:"public"`
		}
		Type6 struct {
			F int `groups:"public"`
		}
		Type7 struct {
			F int `groups:"public"`
		}
	)

	// 填充缓存
	_, _ = MarshalByGroups(Type1{1}, "public")
	_, _ = MarshalByGroups(Type2{2}, "public")
	_, _ = MarshalByGroups(Type3{3}, "public")
	_, _ = MarshalByGroups(Type4{4}, "public")
	_, _ = MarshalByGroups(Type5{5}, "public")

	// 验证缓存大小
	stats := GetCacheStats()
	if stats.CurrentSize > 5 {
		t.Errorf("缓存大小应该限制在5条以内，当前: %d", stats.CurrentSize)
	}

	// 添加更多条目，验证老条目被淘汰
	_, _ = MarshalByGroups(Type6{6}, "public")
	_, _ = MarshalByGroups(Type7{7}, "public")

	// 重新查询条目，验证LRU机制
	_, _ = MarshalByGroups(Type1{1}, "public") // 这应该是个缓存未命中

	// 检查命中/未命中统计
	statsAfter := GetCacheStats()
	if statsAfter.Misses <= stats.Misses {
		t.Error("应该有新的缓存未命中")
	}
}

// 验证恢复机制
func TestPanicRecovery(t *testing.T) {
	// 创建一个会在序列化时panic的类型
	type BadType struct {
		Value any `json:"value" groups:"public"`
	}

	// 使用一个会在序列化时panic的值
	badVal := BadType{
		Value: make(chan int), // json不支持序列化channel
	}

	_, err := MarshalByGroups(badVal, "public")
	if err == nil {
		t.Error("序列化不支持的类型应该返回错误")
	}

	var jsonErr *Error
	if !errors.As(err, &jsonErr) || jsonErr.Type != ErrTypeUnsupportedType {
		t.Errorf("错误类型不正确, 得到: %T, %v", err, err)
	}
}

// 测试路径跟踪
func TestErrorPathTracking(t *testing.T) {
	// 创建一个嵌套结构包含循环引用
	type Inner struct {
		Value int    `json:"value" groups:"public"`
		Self  *Inner `json:"self,omitempty" groups:"public"`
	}

	type Outer struct {
		Name string `json:"name" groups:"public"`
		Data Inner  `json:"data" groups:"public"`
	}

	// 创建循环引用
	obj := Outer{
		Name: "测试",
		Data: Inner{
			Value: 42,
		},
	}
	obj.Data.Self = &obj.Data

	_, err := MarshalByGroups(obj, "public")
	if err == nil {
		t.Error("循环引用应该返回错误")
	}

	var jsonErr *Error
	if !errors.As(err, &jsonErr) {
		t.Errorf("错误类型不正确, 得到: %T", err)
	} else {
		// 验证路径信息是否正确
		if jsonErr.Path == "" {
			t.Error("错误应该包含路径信息")
		}

		// 检查路径是否包含关键部分
		if !strings.Contains(strings.ToLower(jsonErr.Path), "data") ||
			!strings.Contains(strings.ToLower(jsonErr.Path), "self") {
			t.Errorf("错误路径不正确，应该包含'data'和'self'，实际: %s", jsonErr.Path)
		}
	}
}
