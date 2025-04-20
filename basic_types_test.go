package jsongroup

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// 所有基本类型的结构体
type AllBasicTypes struct {
	// 整数类型
	Int    int    `json:"int" groups:"integers,all"`
	Int8   int8   `json:"int8" groups:"integers,all"`
	Int16  int16  `json:"int16" groups:"integers,all"`
	Int32  int32  `json:"int32" groups:"integers,all"`
	Int64  int64  `json:"int64" groups:"integers,all"`
	Uint   uint   `json:"uint" groups:"integers,all"`
	Uint8  uint8  `json:"uint8" groups:"integers,all"`
	Uint16 uint16 `json:"uint16" groups:"integers,all"`
	Uint32 uint32 `json:"uint32" groups:"integers,all"`
	Uint64 uint64 `json:"uint64" groups:"integers,all"`

	// 浮点类型
	Float32 float32 `json:"float32" groups:"floats,all"`
	Float64 float64 `json:"float64" groups:"floats,all"`

	// 布尔类型
	Bool bool `json:"bool" groups:"bool,all"`

	// 字符串类型
	String string `json:"string" groups:"strings,all"`

	// 复数类型
	Complex64  complex64  `json:"complex64" groups:"complex,all"`
	Complex128 complex128 `json:"complex128" groups:"complex,all"`

	// 时间类型
	Time time.Time `json:"time" groups:"time,all"`
}

// 创建所有基本类型的测试对象
func createBasicTypes() AllBasicTypes {
	return AllBasicTypes{
		Int:        -42,
		Int8:       -8,
		Int16:      -16,
		Int32:      -32,
		Int64:      -64,
		Uint:       42,
		Uint8:      8,
		Uint16:     16,
		Uint32:     32,
		Uint64:     64,
		Float32:    3.1415,
		Float64:    math.Pi,
		Bool:       true,
		String:     "这是一个测试字符串",
		Complex64:  complex(float32(1.1), float32(2.2)),
		Complex128: complex(3.3, 4.4),
		Time:       time.Now(),
	}
}

// 测试基本类型的序列化
func TestBasicTypes(t *testing.T) {
	basic := createBasicTypes()

	// 测试不同分组
	testCases := []struct {
		name   string
		groups []string
	}{
		{"整数类型", []string{"integers"}},
		{"浮点类型", []string{"floats"}},
		{"布尔类型", []string{"bool"}},
		{"字符串类型", []string{"strings"}},
		{"复数类型", []string{"complex"}},
		{"时间类型", []string{"time"}},
		{"所有类型", []string{"all"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试默认选项
			data, err := MarshalByGroups(basic, tc.groups...)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			// 反序列化检查
			var result map[string]any
			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}

			// 验证分组是否正确
			if tc.name == "整数类型" {
				for _, key := range []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"} {
					if _, ok := result[key]; !ok {
						t.Errorf("期望包含字段 %s，但未找到", key)
					}
				}

				// 确保不包含其他分组的字段
				if _, ok := result["float32"]; ok {
					t.Errorf("不应包含非整数类型字段")
				}
			}

			t.Logf("结果: %s", string(data))
		})
	}
}

// 测试不同选项对基本类型的影响
func TestBasicTypesWithOptions(t *testing.T) {
	basic := createBasicTypes()

	// 测试不同选项
	testCases := []struct {
		name string
		opts *Options
	}{
		{"默认选项", New()},
		{"顶层键", New().WithTopLevelKey("data")},
		{"空值处理", New().WithNullIfEmpty(true)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := MarshalByGroupsWithOptions(basic, tc.opts, "all")
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			t.Logf("选项 '%s' 的结果: %s", tc.name, string(data))

			// 验证选项是否生效
			if tc.name == "顶层键" {
				var result map[string]any
				err = json.Unmarshal(data, &result)
				if err != nil {
					t.Fatalf("反序列化失败: %v", err)
				}

				// 检查顶层键是否存在
				if dataObj, ok := result["data"]; !ok {
					t.Errorf("顶层键 'data' 不存在")
				} else {
					// 检查顶层键下是否包含正确的数据
					dataMap, ok := dataObj.(map[string]any)
					if !ok {
						t.Errorf("'data' 不是一个对象")
					} else if _, ok := dataMap["int"]; !ok {
						t.Errorf("'data' 对象下缺少预期的字段")
					}
				}
			}
		})
	}
}

// 测试特殊值序列化
func TestSpecialValues(t *testing.T) {
	// 创建包含特殊值的结构
	special := struct {
		// 极限值
		MaxInt     int64   `json:"max_int" groups:"limits,all"`
		MinInt     int64   `json:"min_int" groups:"limits,all"`
		MaxUint    uint64  `json:"max_uint" groups:"limits,all"`
		MaxFloat   float64 `json:"max_float" groups:"limits,all"`
		SmallFloat float64 `json:"small_float" groups:"limits,all"`

		// 特殊浮点值
		Infinity    float64 `json:"infinity" groups:"special,all"`
		NegInfinity float64 `json:"neg_infinity" groups:"special,all"`
		NaN         float64 `json:"nan" groups:"special,all"`

		// 特殊字符
		EmptyString string `json:"empty_string" groups:"strings,all"`
		Unicode     string `json:"unicode" groups:"strings,all"`
		EscapeChars string `json:"escape_chars" groups:"strings,all"`

		// 特殊时间
		ZeroTime   time.Time `json:"zero_time" groups:"time,all"`
		FutureTime time.Time `json:"future_time" groups:"time,all"`
	}{
		MaxInt:      math.MaxInt64,
		MinInt:      math.MinInt64,
		MaxUint:     math.MaxUint64,
		MaxFloat:    math.MaxFloat64,
		SmallFloat:  math.SmallestNonzeroFloat64,
		Infinity:    math.Inf(1),
		NegInfinity: math.Inf(-1),
		NaN:         math.NaN(),
		EmptyString: "",
		Unicode:     "你好世界 😊 🌍",
		EscapeChars: "引号\" 反斜杠\\ 制表符\t 换行符\n",
		ZeroTime:    time.Time{},
		FutureTime:  time.Now().AddDate(100, 0, 0), // 100年后
	}

	// 测试不同的分组和选项
	groupTests := []struct {
		name   string
		groups []string
	}{
		{"极限值", []string{"limits"}},
		{"特殊值", []string{"special"}},
		{"字符串", []string{"strings"}},
		{"时间", []string{"time"}},
		{"全部", []string{"all"}},
	}

	optionTests := []struct {
		name string
		opts *Options
	}{
		{"默认选项", New()},
		{"空值为null", New().WithNullIfEmpty(true)},
	}

	for _, gt := range groupTests {
		for _, ot := range optionTests {
			testName := fmt.Sprintf("%s-%s", gt.name, ot.name)
			t.Run(testName, func(t *testing.T) {
				data, err := MarshalByGroupsWithOptions(special, ot.opts, gt.groups...)
				if err != nil {
					t.Fatalf("序列化失败: %v", err)
				}

				t.Logf("结果: %s", string(data))

				// 针对特殊值进行验证
				if gt.name == "特殊值" {
					// NaN和无穷值在JSON中会被转为null或特定字符串
					if !strings.Contains(string(data), `"infinity":`) ||
						!strings.Contains(string(data), `"neg_infinity":`) ||
						!strings.Contains(string(data), `"nan":`) {
						t.Errorf("特殊浮点值未正确序列化")
					}
				}

				if gt.name == "字符串" && ot.name == "空值为null" {
					// 检查空字符串是否序列化为null
					if !strings.Contains(string(data), `"empty_string":null`) {
						t.Errorf("空字符串应序列化为null，但结果为: %s", string(data))
					}
				}
			})
		}
	}
}
