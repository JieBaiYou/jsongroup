package jsongroup

// GroupMode 定义分组模式，决定字段是否被序列化的逻辑
type GroupMode int

const (
	// GroupModeOr 默认模式：字段标签包含任意一个指定分组即序列化
	GroupModeOr GroupMode = iota
	// GroupModeAnd 字段标签必须包含所有指定分组才序列化
	GroupModeAnd
)

// Options 定义序列化的选项配置
type Options struct {
	// GroupMode 分组模式：Or 或 And 逻辑
	GroupMode GroupMode
	// TopLevelKey 顶层包装的键名，为空则不包装
	TopLevelKey string
	// TagKey 结构体标签键名，默认为 "groups"
	TagKey string
	// UseInterfaceForNested 是否在递归序列化时使用 any 而非具体类型
	UseInterfaceForNested bool
	// NullIfEmpty 当指针为nil或字段为空值时输出null，而不是跳过该字段
	// 注意：此选项会覆盖omitempty的行为
	NullIfEmpty bool
	// IgnoreNilPointers 忽略所有nil指针字段，不输出（优先级高于NullIfEmpty）
	IgnoreNilPointers bool
}

// DefaultOptions 返回默认选项配置
func DefaultOptions() Options {
	return Options{
		GroupMode:             GroupModeOr,
		TopLevelKey:           "",
		TagKey:                "groups",
		UseInterfaceForNested: false,
		NullIfEmpty:           false,
		IgnoreNilPointers:     true,
	}
}

// WithTopLevelKey 设置顶层包装键名
func (o Options) WithTopLevelKey(key string) Options {
	o.TopLevelKey = key
	return o
}

// WithGroupMode 设置分组模式
func (o Options) WithGroupMode(mode GroupMode) Options {
	o.GroupMode = mode
	return o
}

// WithTagKey 设置标签键名
func (o Options) WithTagKey(key string) Options {
	o.TagKey = key
	return o
}

// WithNullIfEmpty 设置是否对空值输出null
func (o Options) WithNullIfEmpty(enable bool) Options {
	o.NullIfEmpty = enable
	// 当启用NullIfEmpty时，自动禁用IgnoreNilPointers
	if enable {
		o.IgnoreNilPointers = false
	}
	return o
}

// WithIgnoreNilPointers 设置是否忽略nil指针字段
func (o Options) WithIgnoreNilPointers(enable bool) Options {
	o.IgnoreNilPointers = enable
	return o
}

// WithUseInterfaceForNested 设置是否对嵌套结构使用any
func (o Options) WithUseInterfaceForNested(enable bool) Options {
	o.UseInterfaceForNested = enable
	return o
}
