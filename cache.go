package jsongroup

import (
	"container/list"
	"reflect"
	"strings"
	"sync"
	"time"
)

// 全局字段信息缓存实例
var globalCache = newFieldCache()

// CacheStats 提供缓存使用统计信息
type CacheStats struct {
	CurrentSize int     // 当前缓存条目数
	MaxSize     int     // 最大缓存容量
	Hits        int64   // 缓存命中次数
	Misses      int64   // 缓存未命中次数
	HitRatio    float64 // 命中率（0-1之间）
}

// fieldInfo 表示结构体字段的元数据
type fieldInfo struct {
	// 字段索引路径
	Index []int
	// 字段原始名称
	Name string
	// JSON序列化名称
	JSONName string
	// 字段所属分组列表
	Groups []string
	// 是否忽略空值
	OmitEmpty bool
	// 是否忽略零值（Go 1.24新特性）
	OmitZero bool
	// 是否为匿名字段
	Anonymous bool
}

// cacheEntry 缓存条目，包含值和创建时间
type cacheEntry struct {
	// 创建时间，用于统计和清理策略
	createdAt time.Time
	// 缓存的字段信息列表
	value []fieldInfo
}

// fieldCache 结构体字段信息缓存
type fieldCache struct {
	// 保护缓存的互斥锁
	mu sync.RWMutex
	// 缓存映射：类型 -> 字段信息列表
	cache map[reflect.Type]*list.Element
	// 访问顺序列表，用于LRU淘汰
	evictList *list.List
	// 最大缓存条目数
	maxSize int
	// 缓存统计信息
	stats cacheStat
}

// cacheStat 缓存统计信息
type cacheStat struct {
	// 总访问次数
	hits int64
	// 缓存命中次数
	misses int64
	// 缓存淘汰次数
	evictions int64
}

// newFieldCache 创建字段缓存
func newFieldCache() *fieldCache {
	return &fieldCache{
		cache:     make(map[reflect.Type]*list.Element),
		evictList: list.New(),
		maxSize:   DefaultMaxCacheSize,
		stats:     cacheStat{},
	}
}

// GetCacheStats 返回当前缓存使用统计信息
func GetCacheStats() CacheStats {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	total := float64(globalCache.stats.hits + globalCache.stats.misses)
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(globalCache.stats.hits) / total
	}

	return CacheStats{
		CurrentSize: globalCache.evictList.Len(),
		MaxSize:     globalCache.maxSize,
		Hits:        globalCache.stats.hits,
		Misses:      globalCache.stats.misses,
		HitRatio:    hitRatio,
	}
}

// SetMaxCacheSize 设置全局缓存的最大容量
func SetMaxCacheSize(size int) {
	globalCache.SetMaxSize(size)
}

// SetMaxSize 设置缓存的最大容量
func (c *fieldCache) SetMaxSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxSize = size
	// 如果新的大小小于当前缓存条目数，需要进行淘汰
	for c.evictList.Len() > c.maxSize && c.maxSize > 0 {
		if err := c.evict(); err != nil {
			// 淘汰失败时停止，避免死循环
			break
		}
	}
}

// GetStats 获取缓存统计信息
func (c *fieldCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := float64(c.stats.hits + c.stats.misses)
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(c.stats.hits) / total
	}

	return CacheStats{
		CurrentSize: c.evictList.Len(),
		MaxSize:     c.maxSize,
		Hits:        c.stats.hits,
		Misses:      c.stats.misses,
		HitRatio:    hitRatio,
	}
}

// Clear 清空缓存
func (c *fieldCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[reflect.Type]*list.Element)
	c.evictList.Init()
	c.stats = cacheStat{}
}

// getFieldsInfo 获取类型的字段信息
// 优先从缓存获取，不存在则解析并加入缓存
func (c *fieldCache) getFieldsInfo(t reflect.Type, tagKey string) ([]fieldInfo, error) {
	// 快速检查非结构体类型
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	// 1. 首先尝试读取缓存 - 只读锁
	c.mu.RLock()
	if element, ok := c.cache[t]; ok {
		entry, valid := element.Value.(*cacheEntry)
		if valid && entry != nil {
			c.stats.hits++
			result := entry.value // 拷贝结果
			c.mu.RUnlock()

			// 异步更新LRU位置 - 减少持锁时间
			go func() {
				c.mu.Lock()
				c.evictList.MoveToFront(element)
				c.mu.Unlock()
			}()

			return result, nil
		}
	}
	c.mu.RUnlock() // 缓存未命中，释放读锁

	// 2. 解析字段信息 - 无锁操作
	fields, err := parseFields(t, tagKey)
	if err != nil {
		return nil, err
	}

	// 3. 缓存结果 - 写锁，但优化操作顺序
	c.mu.Lock()

	// 二次检查，可能在竞争条件下已被其他goroutine添加
	if element, ok := c.cache[t]; ok {
		entry, valid := element.Value.(*cacheEntry)
		if valid && entry != nil {
			c.evictList.MoveToFront(element)
			result := entry.value // 拷贝结果防止锁外修改
			c.mu.Unlock()
			return result, nil
		}
	}

	// 缓存管理逻辑
	if c.maxSize > 0 {
		// 提前批量淘汰，减少锁频率
		for c.evictList.Len() >= c.maxSize && c.evictList.Len() > 0 {
			_ = c.evict()
		}
	}

	// 添加新缓存
	entry := &cacheEntry{
		createdAt: time.Now(),
		value:     fields,
	}
	element := c.evictList.PushFront(entry)
	c.cache[t] = element
	c.stats.misses++

	// 拷贝结果防止锁外修改
	result := fields
	c.mu.Unlock()

	return result, nil
}

// evict 根据LRU淘汰策略删除一个缓存条目
func (c *fieldCache) evict() error {
	// 从列表尾部获取最近最少使用的条目
	if c.evictList.Len() == 0 {
		return nil // 缓存为空，无需淘汰
	}

	// 获取最久未使用的条目
	element := c.evictList.Back()
	if element == nil {
		return nil // 无可淘汰条目
	}

	// 获取键值并从列表中移除
	c.evictList.Remove(element)

	// 安全地转换条目
	entry, ok := element.Value.(*cacheEntry)
	if !ok || entry == nil {
		return CacheOverflowError("字段缓存", c.maxSize)
	}

	// 找到对应的类型并从映射中移除
	found := false
	for typ, elem := range c.cache {
		if elem == element {
			delete(c.cache, typ)
			found = true
			c.stats.evictions++
			break
		}
	}

	if !found {
		// 无法找到映射中对应的条目，这是不应该发生的
		return CacheOverflowError("字段缓存", c.maxSize)
	}

	return nil
}

// parseFields 解析结构体字段信息
func parseFields(t reflect.Type, tagKey string) ([]fieldInfo, error) {
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	var fields []fieldInfo
	var err error

	// 捕获panic以提供友好的错误信息
	defer func() {
		if r := recover(); r != nil {
			// 转换为标准错误
			err = &Error{
				Type:    ErrTypeReflection,
				Message: "解析结构体字段时发生panic",
				Value:   r,
			}
			// 这里不能直接返回err，因为defer的返回值无法影响外部函数
			// 但至少记录了panic，并防止程序崩溃
			fields = nil
		}
	}()

	// 处理所有字段
	for i := range t.NumField() {
		field := t.Field(i)

		// 跳过非导出字段
		if !field.IsExported() {
			continue
		}

		// 获取tag标签
		jsonTag := field.Tag.Get("json")
		groupsTag := field.Tag.Get(tagKey)

		// 解析JSON标签
		jsonName, omitEmpty, omitZero := parseJSONTag(field.Name, jsonTag)
		if jsonName == "-" {
			continue // 忽略标记为"-"的字段
		}

		// 解析分组标签
		groups := parseGroupsTag(groupsTag)

		// 处理匿名嵌套字段
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// 递归处理嵌套字段
			nestedFields, nestedErr := parseFields(field.Type, tagKey)
			if nestedErr != nil {
				return nil, nestedErr
			}

			// 添加嵌套字段，保持正确的索引路径
			for _, nf := range nestedFields {
				indexPath := append([]int{i}, nf.Index...)

				fields = append(fields, fieldInfo{
					Index:     indexPath,
					Name:      field.Name + "." + nf.Name,
					JSONName:  nf.JSONName,
					Groups:    nf.Groups,
					OmitEmpty: nf.OmitEmpty,
					OmitZero:  nf.OmitZero,
					Anonymous: nf.Anonymous,
				})
			}
		} else {
			// 普通字段
			fields = append(fields, fieldInfo{
				Index:     []int{i},
				Name:      field.Name,
				JSONName:  jsonName,
				Groups:    groups,
				OmitEmpty: omitEmpty,
				OmitZero:  omitZero,
				Anonymous: field.Anonymous,
			})
		}
	}

	return fields, err
}

// parseJSONTag 解析JSON标签
func parseJSONTag(fieldName, jsonTag string) (string, bool, bool) {
	if jsonTag == "" {
		return fieldName, false, false
	}

	parts := strings.Split(jsonTag, ",")
	name := parts[0]
	if name == "" {
		name = fieldName
	}

	// 检查omitempty和omitzero选项
	omitEmpty := false
	omitZero := false
	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			omitEmpty = true
		} else if opt == "omitzero" {
			omitZero = true
		}
	}

	return name, omitEmpty, omitZero
}

// parseGroupsTag 解析分组标签
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
