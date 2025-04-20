package jsongroup

import (
	"testing"
)

// 创建一个复杂结构体带有多种类型
type EnhancedBenchStruct struct {
	ID        int                    `json:"id" groups:"basic"`
	Name      string                 `json:"name" groups:"basic"`
	Tags      []string               `json:"tags" groups:"basic"`
	Metadata  map[string]interface{} `json:"metadata" groups:"extended"`
	SubStruct struct {
		Field1 int     `json:"field1" groups:"basic"`
		Field2 string  `json:"field2" groups:"basic"`
		Field3 float64 `json:"field3" groups:"extended"`
	} `json:"sub_struct" groups:"basic,extended"`
}

// 生成测试数据
func createEnhancedTestStruct() EnhancedBenchStruct {
	result := EnhancedBenchStruct{
		ID:   12345,
		Name: "TestStruct",
		Tags: []string{"tag1", "tag2", "tag3"},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
			"nested": map[string]interface{}{
				"nestedKey": "nestedValue",
			},
		},
	}

	result.SubStruct.Field1 = 100
	result.SubStruct.Field2 = "SubField"
	result.SubStruct.Field3 = 3.14159

	return result
}

// 缓存预热 - 使用多种不同结构体类型
func BenchmarkCacheWarmup(b *testing.B) {
	// 重置缓存
	globalCache.Clear()

	// 多种不同结构类型
	types := []interface{}{
		EnhancedBenchStruct{},
		BenchUser{},
		BenchComplexUser{},
		struct {
			A int    `json:"a" groups:"test"`
			B string `json:"b" groups:"test"`
		}{},
		struct {
			X float64 `json:"x" groups:"coords"`
			Y float64 `json:"y" groups:"coords"`
			Z float64 `json:"z" groups:"coords"`
		}{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 循环使用所有类型
		typeIndex := i % len(types)
		_, err := MarshalByGroups(types[typeIndex], "test", "basic", "coords")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 测试缓存命中率 - 重复使用相同结构体
func BenchmarkCacheHitRate(b *testing.B) {
	// 预热缓存
	globalCache.Clear()
	data := createEnhancedTestStruct()
	_, _ = MarshalByGroups(data, "basic")

	// 记录开始时的命中统计
	startStats := GetCacheStats()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalByGroups(data, "basic")
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	// 计算命中率
	endStats := GetCacheStats()
	hitsAdded := endStats.Hits - startStats.Hits
	missesAdded := endStats.Misses - startStats.Misses

	// 报告命中率，但不影响基准测试结果
	if hitsAdded+missesAdded > 0 {
		hitRate := float64(hitsAdded) / float64(hitsAdded+missesAdded) * 100
		b.Logf("Cache Hit Rate: %.2f%% (Hits: %d, Misses: %d)",
			hitRate, hitsAdded, missesAdded)
	}
}

// 测试缓存在并发环境下的性能
func BenchmarkConcurrentCache(b *testing.B) {
	// 预热缓存
	globalCache.Clear()
	data := createEnhancedTestStruct()
	_, _ = MarshalByGroups(data, "basic")

	// 使用多个goroutine并发访问缓存
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := MarshalByGroups(data, "basic")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 测试不同缓存大小对性能的影响
func BenchmarkCacheSizes(b *testing.B) {
	originalSize := DefaultMaxCacheSize
	defer SetMaxCacheSize(originalSize) // 恢复初始设置

	// 测试多种缓存大小
	cacheSizes := []int{10, 100, 1000, 10000}

	for _, size := range cacheSizes {
		b.Run("Size_"+string(rune('0'+size/1000))+"K", func(b *testing.B) {
			// 设置缓存大小
			globalCache.Clear()
			SetMaxCacheSize(size)

			// 创建足够多的不同类型耗尽缓存
			types := make([]interface{}, 0, size*2)
			for i := 0; i < size*2; i++ {
				// 使用匿名结构体创建唯一类型
				types = append(types, struct {
					ID    int    `json:"id" groups:"test"`
					Value string `json:"value" groups:"test"`
					Index int    `json:"index" groups:"test"`
				}{
					ID:    i,
					Value: "test",
					Index: i % 100,
				})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				typeIndex := i % len(types)
				_, err := MarshalByGroups(types[typeIndex], "test")
				if err != nil {
					b.Fatal(err)
				}
			}

			// 报告缓存状态
			stats := GetCacheStats()
			b.Logf("Cache stats: Size=%d/%d, Hit Ratio=%.2f%%",
				stats.CurrentSize, stats.MaxSize, stats.HitRatio*100)
		})
	}
}

// 测试批量淘汰策略
func BenchmarkBatchEviction(b *testing.B) {
	originalSize := DefaultMaxCacheSize
	defer SetMaxCacheSize(originalSize)

	// 设置较小的缓存大小以便触发淘汰
	SetMaxCacheSize(100)
	globalCache.Clear()

	// 创建200个不同类型
	types := make([]interface{}, 0, 200)
	for i := 0; i < 200; i++ {
		types = append(types, struct {
			ID    int    `json:"id" groups:"test"`
			Value string `json:"value" groups:"test"`
		}{
			ID:    i,
			Value: "test",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 循环使用所有类型，确保淘汰发生
		typeIndex := i % len(types)
		_, err := MarshalByGroups(types[typeIndex], "test")
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
	stats := GetCacheStats()
	b.Logf("Final cache stats: Size=%d/%d, Hits=%d, Misses=%d, Ratio=%.2f%%",
		stats.CurrentSize, stats.MaxSize, stats.Hits, stats.Misses, stats.HitRatio*100)
}

// 测试缓存的真实世界场景: API服务器序列化相似对象
func BenchmarkSimulatedAPIServer(b *testing.B) {
	// 模拟API服务器处理不同但相似的用户对象
	userTemplates := []BenchUser{
		createTestUser(), // 用户1
		createTestUser(), // 用户2 (将会修改)
		createTestUser(), // 用户3 (将会修改)
	}

	// 修改模板以创建不同用户
	userTemplates[1].ID = 2
	userTemplates[1].Name = "Second User"
	userTemplates[1].Email = "second@example.com"

	userTemplates[2].ID = 3
	userTemplates[2].Name = "Third User"
	userTemplates[2].Email = "third@example.com"

	// 预热缓存
	globalCache.Clear()
	for _, user := range userTemplates {
		_, _ = MarshalByGroups(user, "public")
	}

	// 模拟并发API请求
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			user := userTemplates[i%len(userTemplates)]
			i++
			_, err := MarshalByGroups(user, "public")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 测试缓存禁用条件下的内存分配
func BenchmarkMemoryAllocations(b *testing.B) {
	data := createEnhancedTestStruct()

	// 子测试1: 启用缓存
	b.Run("WithCache", func(b *testing.B) {
		globalCache.Clear()
		// 预热
		_, _ = MarshalByGroups(data, "basic")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := MarshalByGroups(data, "basic")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// 子测试2: 禁用缓存
	b.Run("WithoutCache", func(b *testing.B) {
		originalSize := DefaultMaxCacheSize
		SetMaxCacheSize(0) // 禁用缓存

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := MarshalByGroups(data, "basic")
			if err != nil {
				b.Fatal(err)
			}
		}

		b.StopTimer()
		SetMaxCacheSize(originalSize) // 恢复缓存大小
	})
}
