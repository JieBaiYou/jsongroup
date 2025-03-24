package jsongroup

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ErrType 错误类型枚举
type ErrType int

const (
	// ErrTypeUnknown 未知错误
	ErrTypeUnknown ErrType = iota
	// ErrTypeMaxDepthExceeded 超过最大递归深度限制
	ErrTypeMaxDepthExceeded
	// ErrTypeCircularReference 检测到循环引用
	ErrTypeCircularReference
	// ErrTypeUnsupportedType 不支持的类型
	ErrTypeUnsupportedType
	// ErrTypeReflection 反射操作错误
	ErrTypeReflection
	// ErrTypeCacheOverflow 缓存溢出错误
	ErrTypeCacheOverflow
)

// Error 自定义错误结构，提供详细的错误上下文
type Error struct {
	// Type 错误类型
	Type ErrType
	// Message 错误描述
	Message string
	// Path 错误发生的路径（字段路径）
	Path string
	// Value 相关的值（可能为nil）
	Value any
	// Cause 原始错误（可能为nil）
	Cause error
}

// Error 实现error接口
func (e *Error) Error() string {
	msg := e.Message
	if e.Path != "" {
		msg = fmt.Sprintf("%s at path '%s'", msg, e.Path)
	}
	if e.Cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

// Unwrap 实现errors.Unwrap接口，便于错误链处理
func (e *Error) Unwrap() error {
	return e.Cause
}

// MaxDepthError 创建超出最大递归深度的错误
func MaxDepthError(path string, value reflect.Value, maxDepth int) *Error {
	var val any
	if value.IsValid() {
		val = value.Interface()
	}
	return &Error{
		Type:    ErrTypeMaxDepthExceeded,
		Message: fmt.Sprintf("已超过最大递归深度限制(%d)", maxDepth),
		Path:    path,
		Value:   val,
	}
}

// CircularReferenceError 创建循环引用错误
func CircularReferenceError(path string, value reflect.Value) *Error {
	var val any
	if value.IsValid() {
		val = value.Interface()
	}
	return &Error{
		Type:    ErrTypeCircularReference,
		Message: "检测到循环引用",
		Path:    path,
		Value:   val,
	}
}

// UnsupportedTypeError 创建不支持类型的错误
func UnsupportedTypeError(path string, typeName any) *Error {
	var typeStr string

	switch v := typeName.(type) {
	case reflect.Value:
		if v.IsValid() {
			typeStr = v.Type().String()
		} else {
			typeStr = "无效值"
		}
	case string:
		typeStr = v
	default:
		typeStr = fmt.Sprintf("%T", typeName)
	}

	return &Error{
		Type:    ErrTypeUnsupportedType,
		Message: fmt.Sprintf("不支持的类型: %s", typeStr),
		Path:    path,
		Value:   typeName,
	}
}

// ReflectionError 创建反射操作错误
func ReflectionError(path string, err error) *Error {
	return &Error{
		Type:    ErrTypeReflection,
		Message: "反射操作错误",
		Path:    path,
		Cause:   err,
	}
}

// CacheOverflowError 创建缓存溢出错误
func CacheOverflowError(cacheType string, maxSize int) *Error {
	return &Error{
		Type:    ErrTypeCacheOverflow,
		Message: fmt.Sprintf("%s缓存已达到最大条目数限制(%d)", cacheType, maxSize),
	}
}

// RecoverFromPanic 捕获并处理panic，转换为标准error
func RecoverFromPanic(path string) func() error {
	return func() (err error) {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = &Error{
					Type:    ErrTypeReflection,
					Message: "反射操作导致panic",
					Path:    path,
					Cause:   v,
				}
			default:
				err = &Error{
					Type:    ErrTypeUnknown,
					Message: fmt.Sprintf("未知panic: %v", r),
					Path:    path,
				}
			}
		}
		return
	}
}

// WrapJSONError 将标准JSON错误包装为我们的自定义错误类型
func WrapJSONError(err error, path string) error {
	if err == nil {
		return nil
	}

	// 检查是否已经是我们的错误类型
	var ourErr *Error
	if errors.As(err, &ourErr) {
		return err
	}

	// 转换标准JSON库错误为自定义错误
	switch e := err.(type) {
	case *json.UnsupportedTypeError:
		return UnsupportedTypeError(path, e.Type.String())
	case *json.UnsupportedValueError:
		// 检查是否是循环引用错误
		if strings.Contains(e.Error(), "encountered a cycle") {
			return CircularReferenceError(path, e.Value)
		}
		return UnsupportedTypeError(path, fmt.Sprintf("%T", e.Value))
	case *json.MarshalerError:
		return ReflectionError(path, e.Err)
	case *json.SyntaxError, *json.InvalidUnmarshalError:
		return ReflectionError(path, e)
	default:
		return &Error{
			Type:    ErrTypeUnknown,
			Message: e.Error(),
			Path:    path,
			Cause:   e,
		}
	}
}
