package jsongroup

import (
	"encoding/json"
	"testing"
	"time"
)

// OmitZeroTest 用于测试omitzero标签
type OmitZeroTest struct {
	// 基本类型
	Int         int       `json:"int"`
	IntZero     int       `json:"intZero,omitzero"`
	IntValue    int       `json:"intValue,omitzero"`
	Float       float64   `json:"float"`
	FloatZero   float64   `json:"floatZero,omitzero"`
	FloatValue  float64   `json:"floatValue,omitzero"`
	Bool        bool      `json:"bool"`
	BoolFalse   bool      `json:"boolFalse,omitzero"`
	BoolTrue    bool      `json:"boolTrue,omitzero"`
	String      string    `json:"string"`
	StringEmpty string    `json:"stringEmpty,omitzero"`
	StringValue string    `json:"stringValue,omitzero"`
	Time        time.Time `json:"time"`
	TimeZero    time.Time `json:"timeZero,omitzero"`
	TimeValue   time.Time `json:"timeValue,omitzero"`

	// 集合类型
	SliceEmpty     []string          `json:"sliceEmpty,omitzero"`
	SliceWithItems []string          `json:"sliceWithItems,omitzero"`
	MapEmpty       map[string]string `json:"mapEmpty,omitzero"`
	MapWithItems   map[string]string `json:"mapWithItems,omitzero"`

	// 指针类型
	PtrNil   *string `json:"ptrNil,omitzero"`
	PtrValue *string `json:"ptrValue,omitzero"`

	// 对比omitempty
	EmptySlice  []string          `json:"emptySlice,omitempty"`
	EmptyMap    map[string]string `json:"emptyMap,omitempty"`
	EmptyString string            `json:"emptyString,omitempty"`
	ZeroInt     int               `json:"zeroInt,omitempty"`
}

func TestOmitZero(t *testing.T) {
	// 准备测试数据
	strValue := "测试值"
	timeValue := time.Date(2024, 4, 20, 12, 0, 0, 0, time.Local)

	test := OmitZeroTest{
		// 基本类型
		Int:         0,
		IntZero:     0,  // 应被省略
		IntValue:    42, // 不应被省略
		Float:       0.0,
		FloatZero:   0.0,  // 应被省略
		FloatValue:  3.14, // 不应被省略
		Bool:        false,
		BoolFalse:   false, // 应被省略
		BoolTrue:    true,  // 不应被省略
		String:      "",
		StringEmpty: "",  // 应被省略
		StringValue: "值", // 不应被省略
		Time:        time.Time{},
		TimeZero:    time.Time{}, // 应被省略
		TimeValue:   timeValue,   // 不应被省略

		// 集合类型
		SliceEmpty:     []string{},                    // 不应被省略 (与omitempty不同)
		SliceWithItems: []string{"项"},                 // 不应被省略
		MapEmpty:       map[string]string{},           // 不应被省略 (与omitempty不同)
		MapWithItems:   map[string]string{"key": "值"}, // 不应被省略

		// 指针类型
		PtrNil:   nil,       // 应被省略
		PtrValue: &strValue, // 不应被省略

		// 对比omitempty
		EmptySlice:  []string{},          // 应被省略
		EmptyMap:    map[string]string{}, // 应被省略
		EmptyString: "",                  // 应被省略
		ZeroInt:     0,                   // 应被省略
	}

	// 测试我们的库
	result, err := MarshalByGroups(test)
	if err != nil {
		t.Fatalf("测试失败: %v", err)
	}

	t.Logf("生成JSON: %s", result)

	// 解析生成的JSON
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("解析JSON失败: %v", err)
	}

	// 验证结果
	// 应存在的键
	expectedKeys := []string{
		"int", "intValue", "float", "floatValue", "bool", "boolTrue", "string", "stringValue",
		"time", "timeValue", "sliceEmpty", "sliceWithItems", "mapEmpty", "mapWithItems", "ptrValue",
	}

	// 应省略的键
	omittedKeys := []string{
		"intZero", "floatZero", "boolFalse", "stringEmpty", "timeZero", "ptrNil",
		"emptySlice", "emptyMap", "emptyString", "zeroInt",
	}

	// 检查应存在的键
	for _, key := range expectedKeys {
		if _, exists := resultMap[key]; !exists {
			t.Errorf("键 %s 应存在但未找到", key)
		}
	}

	// 检查应省略的键
	for _, key := range omittedKeys {
		if _, exists := resultMap[key]; exists {
			t.Errorf("键 %s 应被省略但实际存在", key)
		}
	}

	// 与标准库对比
	t.Log("与Go标准库JSON对比:")
	standardJSON, _ := json.Marshal(test)
	t.Logf("标准库JSON: %s", standardJSON)

	// 解析标准库生成的JSON
	var stdResultMap map[string]interface{}
	if err := json.Unmarshal(standardJSON, &stdResultMap); err != nil {
		t.Fatalf("解析标准库JSON失败: %v", err)
	}

	// 验证空集合类型的处理是否符合预期（与标准库的不同）
	collectionsExist := []string{"sliceEmpty", "mapEmpty"}
	for _, key := range collectionsExist {
		if _, ourExists := resultMap[key]; !ourExists {
			t.Errorf("我们的库: 集合 %s 应存在但未找到", key)
		}

		if _, stdExists := stdResultMap[key]; !stdExists {
			t.Logf("标准库: 集合 %s 被省略", key)
		}
	}
}

// 测试omitzero与omitempty组合使用
type OmitCombinedTest struct {
	IntBoth        int      `json:"intBoth,omitempty,omitzero"`
	IntEmpty       int      `json:"intEmpty,omitempty"`
	IntZero        int      `json:"intZero,omitzero"`
	SliceEmptyBoth []string `json:"sliceEmptyBoth,omitempty,omitzero"`
	SliceEmptyOnly []string `json:"sliceEmptyOnly,omitempty"`
	SliceZeroOnly  []string `json:"sliceZeroOnly,omitzero"`
}

func TestOmitCombined(t *testing.T) {
	test := OmitCombinedTest{
		IntBoth:        0,
		IntEmpty:       0,
		IntZero:        0,
		SliceEmptyBoth: []string{},
		SliceEmptyOnly: []string{},
		SliceZeroOnly:  []string{},
	}

	result, err := MarshalByGroups(test)
	if err != nil {
		t.Fatalf("测试失败: %v", err)
	}

	t.Logf("组合标签JSON: %s", result)

	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("解析JSON失败: %v", err)
	}

	// 检查结果 - 所有int应被省略，sliceZeroOnly应保留
	expectedOmitted := []string{"intBoth", "intEmpty", "intZero", "sliceEmptyBoth", "sliceEmptyOnly"}
	expectedPresent := []string{"sliceZeroOnly"}

	for _, key := range expectedOmitted {
		if _, exists := resultMap[key]; exists {
			t.Errorf("键 %s 应被省略但实际存在", key)
		}
	}

	for _, key := range expectedPresent {
		if _, exists := resultMap[key]; !exists {
			t.Errorf("键 %s 应存在但未找到", key)
		}
	}
}

// 测试反序列化行为 - 确保omitzero不影响反序列化
func TestOmitZeroUnmarshal(t *testing.T) {
	jsonData := `{
		"int": 123,
		"intValue": 456,
		"bool": true,
		"stringValue": "测试"
	}`

	var test OmitZeroTest
	err := json.Unmarshal([]byte(jsonData), &test)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	// 验证值是否正确反序列化
	if test.Int != 123 {
		t.Errorf("Int应为123，实际为%d", test.Int)
	}

	if test.IntValue != 456 {
		t.Errorf("IntValue应为456，实际为%d", test.IntValue)
	}

	if !test.Bool {
		t.Errorf("Bool应为true")
	}

	if test.StringValue != "测试" {
		t.Errorf("StringValue应为'测试'，实际为'%s'", test.StringValue)
	}

	// 验证未在JSON中的标记了omitzero的字段仍为零值
	if test.IntZero != 0 {
		t.Errorf("IntZero应为0，实际为%d", test.IntZero)
	}

	if test.StringEmpty != "" {
		t.Errorf("StringEmpty应为''，实际为'%s'", test.StringEmpty)
	}
}
