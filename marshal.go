package jsongroup

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"time"
)

// serializeContext 序列化上下文，用于跟踪递归深度和循环引用
type serializeContext struct {
	// 当前路径，用于错误信息
	path string
	// 当前递归深度
	depth int
	// 已处理指针的地址映射，用于检测循环引用
	// key为指针地址，value为路径
	pointers map[uintptr]string
	// 序列化选项
	opts Options
}

// newContext 创建新的序列化上下文
func newContext(opts Options) *serializeContext {
	return &serializeContext{
		path:     "",
		depth:    0,
		pointers: make(map[uintptr]string),
		opts:     opts,
	}
}

// withPath 创建带新路径的上下文副本
func (ctx *serializeContext) withPath(segment string) *serializeContext {
	newPath := ctx.path
	if newPath == "" {
		newPath = segment
	} else {
		newPath = newPath + "." + segment
	}

	return &serializeContext{
		path:     newPath,
		depth:    ctx.depth,
		pointers: ctx.pointers,
		opts:     ctx.opts,
	}
}

// enterLevel 增加递归深度并检查限制
func (ctx *serializeContext) enterLevel() error {
	ctx.depth++
	if ctx.opts.MaxDepth > 0 && ctx.depth > ctx.opts.MaxDepth {
		return MaxDepthError(ctx.path, reflect.Value{}, ctx.opts.MaxDepth)
	}
	return nil
}

// leaveLevel 减少递归深度
func (ctx *serializeContext) leaveLevel() {
	ctx.depth--
}

// checkPointer 检查指针是否已被处理（循环引用检测）
func (ctx *serializeContext) checkPointer(ptr reflect.Value) error {
	if ctx.opts.DisableCircularCheck {
		return nil
	}

	if ptr.Kind() == reflect.Ptr && !ptr.IsNil() {
		addr := ptr.Pointer()
		if _, exists := ctx.pointers[addr]; exists {
			return CircularReferenceError(ctx.path, ptr)
		}
		ctx.pointers[addr] = ctx.path
	}
	return nil
}

// MarshalByGroups 用于按指定 groups 过滤字段并输出 JSON 字节
func MarshalByGroups(v any, groups ...string) ([]byte, error) {
	return MarshalByGroupsWithOptions(v, DefaultOptions(), groups...)
}

// MarshalByGroupsWithOptions 带更多可选配置的序列化函数
func MarshalByGroupsWithOptions(v any, opts Options, groups ...string) ([]byte, error) {
	// 捕获可能的panic并转换为错误
	defer func() {
		if r := recover(); r != nil {
			// 如果是标准错误则尝试包装
			if err, ok := r.(error); ok {
				panic(WrapJSONError(err, "Root"))
			}
			// 否则作为未知错误
			panic(ReflectionError("Root", fmt.Errorf("%v", r)))
		}
	}()

	if v == nil {
		return []byte("null"), nil
	}

	// 创建序列化上下文
	ctx := newContext(opts)

	// 获取值的中间表示
	data, err := valueToMap(ctx, reflect.ValueOf(v), groups, opts.GroupMode)
	if err != nil {
		// 包装可能的标准JSON错误
		return nil, WrapJSONError(err, "Root")
	}

	// 添加顶层包装键
	if opts.TopLevelKey != "" {
		wrappedData := make(map[string]any)
		wrappedData[opts.TopLevelKey] = data
		data = wrappedData
	}

	// 使用标准json包进行最终序列化
	jsonData, err := json.Marshal(data)
	if err != nil {
		// 包装标准JSON错误
		return nil, WrapJSONError(err, "Root")
	}

	return jsonData, nil
}

// MarshalToMap 将对象序列化为map[string]any形式
func MarshalToMap(v any, groups ...string) (map[string]any, error) {
	return MarshalToMapWithOptions(v, DefaultOptions(), groups...)
}

// MarshalToMapWithOptions 带选项的Map序列化
func MarshalToMapWithOptions(v any, opts Options, groups ...string) (map[string]any, error) {
	// 捕获可能的panic并转换为错误
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				panic(WrapJSONError(err, "Root"))
			}
			panic(ReflectionError("Root", fmt.Errorf("%v", r)))
		}
	}()

	if v == nil {
		return nil, nil
	}

	// 创建序列化上下文
	ctx := newContext(opts)

	// 获取值的中间表示
	result, err := valueToMap(ctx, reflect.ValueOf(v), groups, opts.GroupMode)
	if err != nil {
		// 包装可能的标准JSON错误
		return nil, WrapJSONError(err, "Root")
	}

	// 转换为map[string]any
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}

	// 如果结果不是map，创建一个包含单个键的map
	tmp := make(map[string]any)
	tmp["value"] = result
	return tmp, nil
}

// valueToMap 将value转换成Map，根据分组和选项设置过滤字段
func valueToMap(ctx *serializeContext, v reflect.Value, groups []string, mode GroupMode) (any, error) {
	// 捕获潜在的panic并转换为我们的自定义错误
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				panic(WrapJSONError(err, ctx.path))
			}
			panic(ReflectionError(ctx.path, fmt.Errorf("%v", r)))
		}
	}()

	// 在处理nil指针前检查深度限制
	// 递归层级从0开始，当前深度为ctx.depth，同时考虑是否开启了深度检测
	if ctx.opts.MaxDepth > 0 && ctx.depth >= ctx.opts.MaxDepth {
		// 超出限制深度时，特殊处理nil指针和空值，允许返回null而不是错误
		// 这样就不会在处理嵌套结构时过早中断
		if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
			return nil, nil
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
			if v.Len() == 0 {
				if ctx.opts.NullIfEmpty {
					return nil, nil
				}
				if v.Kind() == reflect.Slice {
					return []any{}, nil
				}
				return map[string]any{}, nil
			}
		}
	}

	// 增加递归深度并检查限制
	if err := ctx.enterLevel(); err != nil {
		return nil, err
	}
	defer ctx.leaveLevel()

	// 使用reflect.Value的Kind方法获取底层类型
	kind := v.Kind()

	// 处理nil指针
	if (kind == reflect.Pointer || kind == reflect.Interface) && v.IsNil() {
		if ctx.opts.IgnoreNilPointers && kind == reflect.Pointer {
			return nil, errors.New("skip_field")
		}
		if ctx.opts.NullIfEmpty {
			return nil, nil
		}
		return nil, nil
	}

	// 检查循环引用
	if kind == reflect.Ptr || kind == reflect.Map || kind == reflect.Slice {
		if err := ctx.checkPointer(v); err != nil {
			return nil, err
		}
	}

	// 根据类型进行不同处理
	switch kind {
	case reflect.Ptr, reflect.Interface:
		// 递归处理指针和接口类型 - 前面已处理nil情况
		return valueToMap(ctx.withPath(""), v.Elem(), groups, mode)

	case reflect.Struct:
		// 特殊处理时间类型
		if v.Type() == reflect.TypeOf(time.Time{}) {
			t := v.Interface().(time.Time)
			if t.IsZero() && ctx.opts.NullIfEmpty {
				return nil, nil
			}
			return t, nil
		}
		// 处理结构体类型
		return structToMap(ctx, v, groups, mode)

	case reflect.Map:
		// 处理map类型
		if v.Len() == 0 && ctx.opts.NullIfEmpty {
			return nil, nil
		}
		return mapToMap(ctx, v, groups, mode)

	case reflect.Slice, reflect.Array:
		// 处理切片和数组类型
		if v.Len() == 0 {
			if ctx.opts.NullIfEmpty {
				// 对于nil切片，返回null
				if v.IsNil() {
					return nil, nil
				}
				// 对于非nil的空切片，返回空数组
				return []any{}, nil
			}
			// 默认处理
			return []any{}, nil
		}
		return sliceToSlice(ctx, v, groups, mode)

	case reflect.Complex64, reflect.Complex128:
		// 处理复数类型
		c := v.Complex()
		return complex128ToString(c), nil

	case reflect.String:
		// 处理字符串类型
		s := v.String()
		if s == "" && ctx.opts.NullIfEmpty {
			return nil, nil
		}
		return s, nil

	case reflect.Float32, reflect.Float64:
		// 处理浮点类型 - 特殊处理NaN和Inf
		f := v.Float()
		if isSpecialFloat(f) {
			return floatToString(f), nil
		}
		return f, nil

	default:
		// 处理基本类型
		return v.Interface(), nil
	}
}

// structToMap 将结构体转换为map
func structToMap(ctx *serializeContext, v reflect.Value, groups []string, mode GroupMode) (any, error) {
	// 估计map容量
	t := v.Type()
	numField := t.NumField()
	result := make(map[string]any, numField)

	// 获取字段信息（从缓存或解析）
	fields, err := globalCache.getFieldsInfo(t, ctx.opts.TagKey)
	if err != nil {
		return nil, ReflectionError(ctx.path, err)
	}

	for _, field := range fields {
		// 创建新上下文，包含字段路径
		fieldCtx := ctx.withPath(field.Name)

		// 获取字段值
		fieldValue := v.FieldByIndex(field.Index)

		// 检查字段是否属于指定分组
		if !shouldIncludeField(field, mode, groups...) {
			continue
		}

		// 处理内嵌匿名字段
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			// 递归处理匿名字段
			embedded, err := structToMap(fieldCtx, fieldValue, groups, mode)
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

		// 检查是否为nil指针
		isNilPointer := fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil()

		// 处理nil指针的跳过逻辑
		if isNilPointer {
			if ctx.opts.IgnoreNilPointers {
				continue
			}
			if ctx.opts.NullIfEmpty {
				result[field.JSONName] = nil
				continue
			}
		}

		// 判断是否应该根据omitempty跳过字段
		isNilOrEmpty := isEmptyValue(fieldValue)
		if field.OmitEmpty && isNilOrEmpty && !ctx.opts.NullIfEmpty {
			continue
		}

		// 递归处理字段值
		fieldInterface, err := valueToMap(fieldCtx, fieldValue, groups, mode)
		if err != nil {
			// 跳过已标记为需要忽略的字段
			if err.Error() == "skip_field" {
				continue
			}
			return nil, err
		}

		// 只有当字段值非nil或启用了NullIfEmpty时才添加
		if fieldInterface != nil || (isNilOrEmpty && ctx.opts.NullIfEmpty) {
			result[field.JSONName] = fieldInterface
		}
	}

	return result, nil
}

// mapToMap 处理map类型
func mapToMap(ctx *serializeContext, v reflect.Value, groups []string, mode GroupMode) (any, error) {
	// nil和空map检查在valueToMap已处理

	// 预分配合理容量的map
	size := v.Len()
	resultMap := make(map[string]any, size)

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

		// 为map元素创建上下文
		itemCtx := ctx.withPath(keyStr)

		// 递归处理值
		valInterface, err := valueToMap(itemCtx, mapVal, groups, mode)
		if err != nil {
			return nil, err
		}

		// 非nil值添加到结果
		if valInterface != nil || ctx.opts.NullIfEmpty {
			resultMap[keyStr] = valInterface
		}
	}

	return resultMap, nil
}

// sliceToSlice 处理切片和数组
func sliceToSlice(ctx *serializeContext, v reflect.Value, groups []string, mode GroupMode) (any, error) {
	// 空切片检查在valueToMap已处理

	// 预分配合理容量的切片
	length := v.Len()
	result := make([]any, 0, length)

	for i := range length {
		item := v.Index(i)

		// 为数组元素创建上下文
		itemCtx := ctx.withPath(fmt.Sprintf("[%d]", i))

		// 递归处理元素
		itemInterface, err := valueToMap(itemCtx, item, groups, mode)
		if err != nil {
			return nil, err
		}

		// 非nil值添加到结果
		if itemInterface != nil || ctx.opts.NullIfEmpty {
			result = append(result, itemInterface)
		}
	}

	return result, nil
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
			if slices.Contains(field.Groups, g) {
				return true
			}
		}
		return false

	case GroupModeAnd:
		// 与模式：字段分组必须包含所有指定分组
		for _, g := range groups {
			if !slices.Contains(field.Groups, g) {
				return false
			}
		}
		return true
	}

	return false
}

// complex128ToString 将复数转换为字符串表示
func complex128ToString(c complex128) string {
	return fmt.Sprintf("%g", c)
}

// isSpecialFloat 检查浮点数是否为NaN或Infinite
func isSpecialFloat(f float64) bool {
	return math.IsNaN(f) || math.IsInf(f, 0)
}

// floatToString 将特殊浮点数转换为字符串
func floatToString(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	return fmt.Sprintf("%g", f)
}
