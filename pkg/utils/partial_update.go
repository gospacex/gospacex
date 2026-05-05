package utils

import (
	"reflect"

	"github.com/jinzhu/copier"
)

// PartialUpdate 将 src 中非零值字段合并到 dst
// 对于结构体指针 dst 和 src，只更新 src 中非零值的字段，零值字段保持 dst 原值不变
// 支持 src 和 dst 字段数不一致的情况（处理连表查询时 dst 字段数少于 src）
// 使用字段名匹配而非索引匹配，兼容 proto 生成的结构体字段顺序不同的情况
func PartialUpdate(dst, src interface{}) error {
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(src).Elem()
	srcType := srcVal.Type()

	// 临时创建目标对象的副本，用于保存原始值
	tmpVal := reflect.New(dstVal.Type()).Elem()
	tmpVal.Set(dstVal)

	// 先进行全量复制（copier 会自动处理字段名匹配，忽略 dst 中不存在的字段）
	if err := copier.Copy(dst, src); err != nil {
		return err
	}

	// 构建 src 的字段名到字段索引的映射（用于按名字查找）
	srcFieldIndex := make(map[string]int)
	for i := 0; i < srcVal.NumField(); i++ {
		srcFieldIndex[srcType.Field(i).Name] = i
	}

	// 遍历 dst 的所有字段，按名字在 src 中查找对应字段
	for i := 0; i < dstVal.NumField(); i++ {
		dstField := dstVal.Field(i)
		dstFieldInfo := dstVal.Type().Field(i)

		// 在 src 中查找同名字段
		srcIdx, ok := srcFieldIndex[dstFieldInfo.Name]
		if !ok {
			// src 没有对应字段，恢复 dst 字段为原始值
			tmpField := tmpVal.Field(i)
			if dstField.CanSet() {
				dstField.Set(tmpField)
			}
			continue
		}

		srcField := srcVal.Field(srcIdx)

		// 如果 src 字段是零值，将 dst 字段恢复为原始值
		if IsZeroValue(srcField) {
			tmpField := tmpVal.Field(i)
			if dstField.CanSet() {
				dstField.Set(tmpField)
			}
		}
	}

	return nil
}

// IsZeroValue 判断反射值是否为零值
func IsZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	default:
		return false
	}
}