package jsongroup

import (
	"encoding/json"
	"testing"
	"time"
)

// 简单结构体（用于嵌套）
type SimpleStruct struct {
	ID    int    `json:"id" groups:"basic,all"`
	Name  string `json:"name" groups:"basic,all"`
	Value bool   `json:"value" groups:"basic,all"`
}

// 复杂嵌套结构体
type ComplexStruct struct {
	// 基本类型
	ID   int    `json:"id" groups:"base,all"`
	Name string `json:"name" groups:"base,all"`

	// 嵌套结构
	Profile struct {
		Age     int      `json:"age" groups:"profile,all"`
		Gender  string   `json:"gender" groups:"profile,all"`
		Hobbies []string `json:"hobbies" groups:"profile,personal"`
	} `json:"profile" groups:"profile,all"`

	// 指针嵌套
	Address *struct {
		Street  string `json:"street" groups:"address,all"`
		City    string `json:"city" groups:"address,all"`
		Country string `json:"country" groups:"address,all"`
		ZipCode string `json:"zip_code" groups:"address,detail"`
	} `json:"address,omitempty" groups:"address,all"`

	// 数组和切片
	Tags   []string `json:"tags" groups:"collections,all"`
	Scores [3]int   `json:"scores" groups:"collections,all"`

	// 映射
	Metadata map[string]any `json:"metadata" groups:"meta,all"`

	// 时间
	CreatedAt time.Time  `json:"created_at" groups:"time,all"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" groups:"time,all"`
}

// 生成测试综合结构
func createTestComplex() ComplexStruct {
	now := time.Now()
	later := now.Add(24 * time.Hour)

	return ComplexStruct{
		ID:   12345,
		Name: "综合测试对象",
		Profile: struct {
			Age     int      `json:"age" groups:"profile,all"`
			Gender  string   `json:"gender" groups:"profile,all"`
			Hobbies []string `json:"hobbies" groups:"profile,personal"`
		}{
			Age:     30,
			Gender:  "男",
			Hobbies: []string{"读书", "旅行", "摄影"},
		},
		Address: &struct {
			Street  string `json:"street" groups:"address,all"`
			City    string `json:"city" groups:"address,all"`
			Country string `json:"country" groups:"address,all"`
			ZipCode string `json:"zip_code" groups:"address,detail"`
		}{
			Street:  "中关村大街1号",
			City:    "北京",
			Country: "中国",
			ZipCode: "100080",
		},
		Tags:   []string{"VIP", "活跃用户", "开发者"},
		Scores: [3]int{95, 87, 92},
		Metadata: map[string]any{
			"verified":    true,
			"login_count": 42,
			"preferences": map[string]any{
				"theme":         "dark",
				"notifications": true,
				"language":      "zh-CN",
			},
			"recent_logins": []string{"2023-06-01", "2023-06-05", "2023-06-10"},
		},
		CreatedAt: now,
		UpdatedAt: &later,
	}
}

// 测试复杂结构和各种选项组合
func TestComplexStructWithOptions(t *testing.T) {
	complex := createTestComplex()

	// 测试不同的分组
	testCases := []struct {
		name   string
		groups []string
	}{
		{"基础信息", []string{"base"}},
		{"个人资料", []string{"profile"}},
		{"地址信息", []string{"address"}},
		{"集合类型", []string{"collections"}},
		{"元数据", []string{"meta"}},
		{"时间信息", []string{"time"}},
		{"全部信息", []string{"all"}},
		{"多分组", []string{"base", "profile", "address"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试不同选项
			options := []struct {
				name string
				opts Options
			}{
				{"默认选项", DefaultOptions()},
				{"带顶层键", DefaultOptions().WithTopLevelKey("data")},
				{"AND逻辑", DefaultOptions().WithGroupMode(GroupModeAnd)},
				{"空值为null", DefaultOptions().WithNullIfEmpty(true)},
				{"忽略nil指针", DefaultOptions().WithIgnoreNilPointers(true)},
			}

			for _, opt := range options {
				t.Run(opt.name, func(t *testing.T) {
					// 序列化为JSON
					data, err := MarshalByGroupsWithOptions(complex, opt.opts, tc.groups...)
					if err != nil {
						t.Fatalf("序列化失败: %v", err)
					}

					// 验证可以解析回结构
					var result any
					err = json.Unmarshal(data, &result)
					if err != nil {
						t.Fatalf("反序列化失败: %v", err)
					}

					// 记录结果大小和部分内容
					t.Logf("结果大小: %d bytes", len(data))
					if len(data) < 1000 {
						t.Logf("结果内容: %s", string(data))
					} else {
						t.Logf("结果内容(截断): %s...", string(data[:500]))
					}
				})
			}
		})
	}
}
