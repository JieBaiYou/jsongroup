package jsongroup

import (
	"reflect"
	"strings"
	"sync"
)

// 全局字段信息缓存
var globalCache = newFieldCache()

// fieldInfo 表示结构体字段的元数据
type fieldInfo struct {
	Index      []int    // 字段索引路径
	Name       string   // 原始字段名称
	JSONName   string   // JSON 序列化名称
	Groups     []string // 字段所属分组
	OmitEmpty  bool     // 是否忽略空值
	Anonymous  bool     // 是否为匿名字段
}

// fieldCache 用于缓存已解析的字段信息
type fieldCache struct {
	mu    sync.RWMutex
	cache map[reflect.Type][]fieldInfo
}

// newFieldCache 创建新的字段缓存
func newFieldCache() *fieldCache {
	return &fieldCache{
		cache: make(map[reflect.Type][]fieldInfo),
	}
}

// getFieldsInfo 获取指定类型的字段信息，如缓存中不存在则解析
func (c *fieldCache) getFieldsInfo(t reflect.Type, tagKey string) []fieldInfo {
	c.mu.RLock()
	fields, ok := c.cache[t]
	c.mu.RUnlock()

	if ok {
		return fields
	}

	// 解析字段
	fields = parseFields(t, tagKey)

	// 写入缓存
	c.mu.Lock()
	c.cache[t] = fields
	c.mu.Unlock()

	return fields
}

// parseFields 解析结构体字段的元数据（包括标签、索引等）
func parseFields(t reflect.Type, tagKey string) []fieldInfo {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []fieldInfo

	// 处理所有结构体字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 忽略未导出字段
		if !field.IsExported() {
			continue
		}

		// 获取tags
		jsonTag := field.Tag.Get("json")
		groupsTag := field.Tag.Get(tagKey)

		// 解析json标签，支持omitempty选项
		jsonName, omitEmpty := parseJSONTag(field.Name, jsonTag)

		// 如果json标签为"-"，则忽略此字段
		if jsonName == "-" {
			continue
		}

		// 解析groups标签
		groups := parseGroupsTag(groupsTag)

		// 处理匿名嵌套结构体字段
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// 获取嵌套字段的信息
			nestedFields := parseFields(field.Type, tagKey)

			// 将嵌套字段添加到主列表
			for _, nf := range nestedFields {
				// 特殊处理嵌套字段的索引路径
				indexPath := append([]int{i}, nf.Index...)

				fields = append(fields, fieldInfo{
					Index:     indexPath,
					Name:      field.Name + "." + nf.Name,
					JSONName:  nf.JSONName,
					Groups:    nf.Groups,
					OmitEmpty: nf.OmitEmpty,
					Anonymous: nf.Anonymous,
				})
			}
		} else if field.Anonymous {
			// 非结构体的匿名字段
			fields = append(fields, fieldInfo{
				Index:     []int{i},
				Name:      field.Name,
				JSONName:  jsonName,
				Groups:    groups,
				OmitEmpty: omitEmpty,
				Anonymous: true,
			})
		} else {
			// 普通字段
			fields = append(fields, fieldInfo{
				Index:     []int{i},
				Name:      field.Name,
				JSONName:  jsonName,
				Groups:    groups,
				OmitEmpty: omitEmpty,
				Anonymous: false,
			})
		}
	}

	return fields
}

// parseJSONTag 解析json标签
func parseJSONTag(fieldName, jsonTag string) (string, bool) {
	if jsonTag == "" {
		return fieldName, false
	}

	parts := strings.Split(jsonTag, ",")
	name := parts[0]
	if name == "" {
		name = fieldName
	}

	// 检查是否设置了omitempty选项
	omitEmpty := false
	if len(parts) > 1 {
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				omitEmpty = true
				break
			}
		}
	}

	return name, omitEmpty
}

// parseGroupsTag 解析分组标签，支持逗号分隔的多个分组
func parseGroupsTag(groupsTag string) []string {
	if groupsTag == "" {
		return nil
	}

	parts := strings.Split(groupsTag, ",")
	groups := make([]string, 0, len(parts))

	for _, part := range parts {
		g := strings.TrimSpace(part)
		if g != "" {
			groups = append(groups, g)
		}
	}

	return groups
}
