package jsongroup

// GroupMode 定义分组模式，决定字段是否被序列化的逻辑
type GroupMode int

const (
	// GroupModeOr 默认模式：字段标签包含任意一个指定分组即序列化
	GroupModeOr GroupMode = iota
	// GroupModeAnd 字段标签必须包含所有指定分组才序列化
	GroupModeAnd
)

// 默认设置常量
const (
	// DefaultMaxDepth 默认的最大递归深度限制
	DefaultMaxDepth = 32
	// DefaultMaxCacheSize 默认的字段缓存条目上限
	DefaultMaxCacheSize = 1000
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
	// MaxDepth 最大递归深度限制，防止栈溢出，默认为32
	// 设置为0表示不限制深度（不推荐）
	MaxDepth int
	// DisableCircularCheck 是否禁用循环引用检测，默认为false
	// 禁用可能提高性能，但遇到循环引用时会导致栈溢出
	DisableCircularCheck bool
	// MaxCacheSize 字段缓存的最大条目数，默认为1000
	// 设置为0表示不限制缓存大小（不推荐用于生产环境）
	MaxCacheSize int
}

// New 返回默认选项配置
func New() *Options {
	return &Options{
		GroupMode:             GroupModeOr,
		TopLevelKey:           "",
		TagKey:                "groups",
		UseInterfaceForNested: false,
		NullIfEmpty:           false,
		IgnoreNilPointers:     true,
		MaxDepth:              DefaultMaxDepth,
		DisableCircularCheck:  false,
		MaxCacheSize:          DefaultMaxCacheSize,
	}
}

// WithTopLevelKey 设置顶层包装键名
func (o *Options) WithTopLevelKey(key string) *Options {
	o.TopLevelKey = key
	return o
}

// WithGroupMode 设置分组模式
func (o *Options) WithGroupMode(mode GroupMode) *Options {
	o.GroupMode = mode
	return o
}

// WithTagKey 设置标签键名
func (o *Options) WithTagKey(key string) *Options {
	o.TagKey = key
	return o
}

// WithNullIfEmpty 设置是否对空值输出null
func (o *Options) WithNullIfEmpty(enable bool) *Options {
	o.NullIfEmpty = enable
	// 当启用NullIfEmpty时，自动禁用IgnoreNilPointers
	if enable {
		o.IgnoreNilPointers = false
	}
	return o
}

// WithIgnoreNilPointers 设置是否忽略nil指针字段
func (o *Options) WithIgnoreNilPointers(enable bool) *Options {
	o.IgnoreNilPointers = enable
	return o
}

// WithUseInterfaceForNested 设置是否对嵌套结构使用any
func (o *Options) WithUseInterfaceForNested(enable bool) *Options {
	o.UseInterfaceForNested = enable
	return o
}

// WithMaxDepth 设置最大递归深度限制
// depth应为正数，设置为0表示不限制（不推荐）
func (o *Options) WithMaxDepth(depth int) *Options {
	o.MaxDepth = depth
	return o
}

// WithDisableCircularCheck 设置是否禁用循环引用检测
func (o *Options) WithDisableCircularCheck(disable bool) *Options {
	o.DisableCircularCheck = disable
	return o
}

// WithMaxCacheSize 设置字段缓存的最大条目数
// size应为正数，设置为0表示不限制（不推荐）
func (o *Options) WithMaxCacheSize(size int) *Options {
	o.MaxCacheSize = size
	return o
}
