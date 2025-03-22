package jsongroup

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// MarshalByGroups 用于按指定 groups 过滤字段并输出 JSON 字节
func MarshalByGroups(v any, groups ...string) ([]byte, error) {
	return MarshalByGroupsWithOptions(v, DefaultOptions(), groups...)
}

// MarshalByGroupsWithOptions 带更多可选配置的序列化函数
func MarshalByGroupsWithOptions(v any, opts Options, groups ...string) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	// 获取值的中间表示
	data, err := valueToMap(reflect.ValueOf(v), opts, groups...)
	if err != nil {
		return nil, err
	}

	// 如果设置了顶层键
	if opts.TopLevelKey != "" {
		wrapped := make(map[string]any)
		wrapped[opts.TopLevelKey] = data
		data = wrapped
	}

	// 最终序列化为JSON
	return json.Marshal(data)
}

// MarshalToMap 将值序列化为map，便于二次处理
func MarshalToMap(v any, groups ...string) (map[string]any, error) {
	return MarshalToMapWithOptions(v, DefaultOptions(), groups...)
}

// MarshalToMapWithOptions 带选项的Map序列化
func MarshalToMapWithOptions(v any, opts Options, groups ...string) (map[string]any, error) {
	if v == nil {
		return nil, nil
	}

	result, err := valueToMap(reflect.ValueOf(v), opts, groups...)
	if err != nil {
		return nil, err
	}

	// 确保结果是map类型
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}

	// 如果结果不是map，创建一个包装
	rootType := reflect.TypeOf(v)
	for rootType.Kind() == reflect.Ptr {
		rootType = rootType.Elem()
	}

	key := opts.TopLevelKey
	if key == "" {
		key = strings.ToLower(rootType.Name())
	}

	return map[string]any{key: result}, nil
}

// valueToMap 将值转换为中间表示（map或其他类型）
func valueToMap(v reflect.Value, opts Options, groups ...string) (any, error) {
	// 获取基础值（解引用指针）
	v = indirect(v)
	if !v.IsValid() {
		return nil, nil
	}

	// 根据类型分别处理
	switch v.Kind() {
	case reflect.Struct:
		return structToMap(v, opts, groups...)
	case reflect.Map:
		return mapToMap(v, opts, groups...)
	case reflect.Slice, reflect.Array:
		return sliceToSlice(v, opts, groups...)
	default:
		// 基本类型直接返回
		return v.Interface(), nil
	}
}

// structToMap 将结构体转换为map
func structToMap(v reflect.Value, opts Options, groups ...string) (any, error) {
	t := v.Type()
	result := make(map[string]any)

	// 获取字段信息（从缓存或解析）
	fields := globalCache.getFieldsInfo(t, opts.TagKey)

	for _, field := range fields {
		// 获取字段值
		fieldValue := v.FieldByIndex(field.Index)

		// 检查字段是否属于指定分组
		if !shouldIncludeField(field, opts.GroupMode, groups...) {
			continue
		}

		// 处理内嵌匿名字段
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			// 递归处理匿名字段
			embedded, err := structToMap(fieldValue, opts, groups...)
			if err != nil {
				return nil, err
			}

			// 合并匿名字段的所有键
			if embeddedMap, ok := embedded.(map[string]any); ok {
				for k, v := range embeddedMap {
					result[k] = v
				}
			}
			continue
		}

		// 检查指针字段
		isNilPointer := fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil()

		// 如果启用了忽略nil指针选项，跳过nil指针字段
		if isNilPointer && opts.IgnoreNilPointers {
			continue
		}

		// 判断是否应该根据omitempty跳过字段
		isNilOrEmpty := isEmptyValue(fieldValue)
		if field.OmitEmpty && isNilOrEmpty && !opts.NullIfEmpty {
			continue
		}

		// 递归处理字段值
		var fieldInterface any
		var err error

		// 处理nil指针特殊情况
		if isNilPointer && opts.NullIfEmpty {
			fieldInterface = nil
		} else {
			fieldInterface, err = valueToMap(fieldValue, opts, groups...)
			if err != nil {
				return nil, err
			}
		}

		// 只有当字段值非nil或启用了NullIfEmpty时才添加
		// 确保嵌套递归时nil字段的正确处理
		if fieldInterface != nil || (isNilOrEmpty && opts.NullIfEmpty) {
			result[field.JSONName] = fieldInterface
		}
	}

	return result, nil
}

// mapToMap 处理map类型
func mapToMap(v reflect.Value, opts Options, groups ...string) (any, error) {
	if v.IsNil() {
		return nil, nil
	}

	resultMap := make(map[string]any)

	// 遍历map
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		mapVal := iter.Value()

		// 获取key的字符串表示
		var keyStr string
		if k.Kind() == reflect.String {
			keyStr = k.String()
		} else {
			// 非字符串键尝试转换为字符串
			keyStr = fmt.Sprint(k.Interface())
		}

		// 递归处理值
		valInterface, err := valueToMap(mapVal, opts, groups...)
		if err != nil {
			return nil, err
		}

		// 非nil值添加到结果
		if valInterface != nil || opts.NullIfEmpty {
			resultMap[keyStr] = valInterface
		}
	}

	return resultMap, nil
}

// sliceToSlice 处理切片和数组
func sliceToSlice(v reflect.Value, opts Options, groups ...string) (any, error) {
	// 对于空切片或数组，根据选项决定输出
	if v.Kind() == reflect.Slice && v.IsNil() {
		if opts.NullIfEmpty {
			return []any{}, nil
		}
		return nil, nil
	}

	length := v.Len()
	result := make([]any, 0, length)

	for i := 0; i < length; i++ {
		item := v.Index(i)

		// 递归处理元素
		itemInterface, err := valueToMap(item, opts, groups...)
		if err != nil {
			return nil, err
		}

		// 非nil值添加到结果
		if itemInterface != nil || opts.NullIfEmpty {
			result = append(result, itemInterface)
		}
	}

	return result, nil
}

// indirect 解引用指针，获取底层值
func indirect(v reflect.Value) reflect.Value {
	// 如果指针为nil，保持原样不解引用，由调用方处理
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return v
	}
	return indirect(v.Elem())
}

// isEmptyValue 判断值是否为空
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// shouldIncludeField 判断字段是否属于指定分组
func shouldIncludeField(field fieldInfo, mode GroupMode, groups ...string) bool {
	// 如果没有指定分组，则包含所有字段
	if len(groups) == 0 {
		return true
	}

	// 如果字段没有分组标签，则不包含
	if len(field.Groups) == 0 {
		return false
	}

	// 根据模式判断
	switch mode {
	case GroupModeOr:
		// 或模式：字段分组包含任意一个指定分组即可
		for _, g := range groups {
			for _, fg := range field.Groups {
				if g == fg {
					return true
				}
			}
		}
		return false

	case GroupModeAnd:
		// 与模式：字段分组必须包含所有指定分组
		for _, g := range groups {
			found := false
			for _, fg := range field.Groups {
				if g == fg {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	return false
}
