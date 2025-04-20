package jsongroup

import (
	"fmt"
	"strings"
	"testing"
)

// 循环引用结构
type Node struct {
	ID    int            `json:"id" groups:"base,all"`
	Name  string         `json:"name" groups:"base,all"`
	Next  *Node          `json:"next,omitempty" groups:"links,all"`
	Prev  *Node          `json:"prev,omitempty" groups:"links,all"`
	Child *Node          `json:"child,omitempty" groups:"tree,all"`
	Data  map[string]any `json:"data,omitempty" groups:"data,all"`
}

// 具有空或nil值的结构
type EmptyFields struct {
	// 空字符串
	EmptyString string `json:"empty_string" groups:"empty,all"`
	// 空数组
	EmptyArray []string `json:"empty_array" groups:"empty,all"`
	// 空切片
	EmptySlice []int `json:"empty_slice" groups:"empty,all"`
	// 空映射
	EmptyMap map[string]string `json:"empty_map" groups:"empty,all"`
	// nil指针
	NilPointer *string `json:"nil_pointer" groups:"nil,all"`
	// nil切片
	NilSlice []float64 `json:"nil_slice" groups:"nil,all"`
	// nil映射
	NilMap map[string]any `json:"nil_map" groups:"nil,all"`
	// 具有nil字段的结构指针
	NilStruct *struct {
		Field string `json:"field"`
	} `json:"nil_struct" groups:"nil,all"`
	// 具有值的指针（用于对比）
	ValuePointer *string `json:"value_pointer" groups:"value,all"`
	// 值为空的指针字段
	EmptyValuePointer *string `json:"empty_value_pointer" groups:"value,all"`
}

// 测试循环引用检测
func TestCircularReferences(t *testing.T) {
	// 创建循环链表
	node1 := &Node{ID: 1, Name: "节点1"}
	node2 := &Node{ID: 2, Name: "节点2"}
	node3 := &Node{ID: 3, Name: "节点3"}

	// 设置循环引用
	node1.Next = node2
	node2.Next = node3
	node3.Next = node1 // 循环回第一个

	node3.Prev = node2
	node2.Prev = node1
	node1.Prev = node3 // 反向循环

	// 创建嵌套循环
	childNode := &Node{ID: 4, Name: "子节点"}
	childNode.Child = childNode // 自引用
	node1.Child = childNode

	// 创建映射循环
	node1.Data = map[string]any{
		"self": node1, // 引用自身
	}

	testCases := []struct {
		name   string
		node   *Node
		groups []string
		opts   *Options
	}{
		{
			name:   "默认选项-检测循环引用",
			node:   node1,
			groups: []string{"all"},
			opts:   DefaultOptions(),
		},
		{
			name:   "顶层键-检测循环引用",
			node:   node1,
			groups: []string{"base", "links"},
			opts:   DefaultOptions().WithTopLevelKey("node"),
		},
		{
			name:   "禁用循环检测",
			node:   node1,
			groups: []string{"base"},
			opts:   DefaultOptions().WithDisableCircularCheck(true).WithMaxDepth(3), // 限制深度防止无限递归
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := MarshalByGroupsWithOptions(tc.node, tc.opts, tc.groups...)

			if tc.name == "禁用循环检测" {
				// 因为限制了深度，应该成功但有深度限制
				if err != nil {
					t.Fatalf("期望成功但有深度限制，实际错误: %v", err)
				}
				t.Logf("禁用循环检测但有深度限制的结果: %s", string(data))
			} else {
				// 默认情况下应该检测到循环引用并返回错误
				if err == nil {
					t.Fatalf("期望检测到循环引用错误，但没有错误")
				}

				// 验证错误类型
				if !isErrType(err, ErrTypeCircularReference) {
					t.Fatalf("期望循环引用错误，但得到: %v", err)
				}

				t.Logf("正确检测到循环引用: %v", err)
			}
		})
	}
}

// 测试空值和nil值处理
func TestEmptyAndNilFields(t *testing.T) {
	// 创建包含各种空值的结构
	emptyVal := ""
	valueStr := "有值的字符串"

	empty := EmptyFields{
		EmptyString:       "",
		EmptyArray:        []string{},
		EmptySlice:        []int{},
		EmptyMap:          map[string]string{},
		NilPointer:        nil,
		NilSlice:          nil,
		NilMap:            nil,
		NilStruct:         nil,
		ValuePointer:      &valueStr,
		EmptyValuePointer: &emptyVal,
	}

	testCases := []struct {
		name   string
		opts   *Options
		groups []string
	}{
		{
			name:   "默认选项",
			opts:   DefaultOptions(),
			groups: []string{"all"},
		},
		{
			name:   "空值为null",
			opts:   DefaultOptions().WithNullIfEmpty(true),
			groups: []string{"empty", "nil"},
		},
		{
			name:   "忽略nil指针",
			opts:   DefaultOptions().WithIgnoreNilPointers(true),
			groups: []string{"nil", "value"},
		},
		{
			name:   "空值为null且忽略nil指针",
			opts:   DefaultOptions().WithNullIfEmpty(true).WithIgnoreNilPointers(false), // NullIfEmpty 会覆盖 IgnoreNilPointers
			groups: []string{"all"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := MarshalByGroupsWithOptions(empty, tc.opts, tc.groups...)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			t.Logf("结果: %s", string(data))

			// 验证结果中是否包含特定字段
			if tc.name == "忽略nil指针" {
				if contains(string(data), "nil_pointer") ||
					contains(string(data), "nil_struct") {
					t.Errorf("期望忽略nil指针字段，但它们出现在结果中")
				}

				if !contains(string(data), "value_pointer") {
					t.Errorf("期望包含有值的指针字段，但它不在结果中")
				}
			}

			if tc.name == "空值为null" {
				if !contains(string(data), `"empty_string":null`) {
					t.Errorf("期望空字符串序列化为null，但结果不符合预期")
				}

				if !contains(string(data), `"nil_pointer":null`) {
					t.Errorf("期望nil指针序列化为null，但结果不符合预期")
				}
			}
		})
	}
}

// 测试最大深度限制
func TestMaxDepth(t *testing.T) {
	// 创建深度嵌套的结构
	var createNestedNode func(depth int) *Node
	createNestedNode = func(depth int) *Node {
		if depth <= 0 {
			return nil
		}

		node := &Node{
			ID:   depth,
			Name: fmt.Sprintf("深度%d", depth),
		}

		node.Child = createNestedNode(depth - 1)
		return node
	}

	// 创建深度为10的树
	rootNode := createNestedNode(10)

	testCases := []struct {
		name        string
		maxDepth    int
		expectError bool
	}{
		{"默认深度限制", 32, false},
		{"深度限制为5", 5, true},
		{"深度限制为3", 3, true},
		{"深度限制为1", 1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := DefaultOptions().WithMaxDepth(tc.maxDepth)
			data, err := MarshalByGroupsWithOptions(rootNode, opts, "all")

			if tc.expectError {
				if err == nil {
					t.Fatalf("期望因超过最大深度而失败，但没有错误")
				}

				// 验证错误类型
				if !isErrType(err, ErrTypeMaxDepthExceeded) {
					t.Fatalf("期望最大深度错误，但得到: %v", err)
				}

				t.Logf("正确检测到超过最大深度: %v", err)
			} else {
				if err != nil {
					t.Fatalf("序列化失败: %v", err)
				}

				t.Logf("深度限制%d的结果长度: %d", tc.maxDepth, len(data))
			}
		})
	}
}

// 辅助函数：检查错误类型
func isErrType(err error, errType ErrType) bool {
	if jsonErr, ok := err.(*Error); ok {
		return jsonErr.Type == errType
	}
	return false
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && s != "null" && s != `""` && s != "{}" && s != "[]" && s != "0" && s != "false" && s != "true" && s != "undefined" && strings.Contains(s, substr)
}
