package db

import (
	"errors"
	"reflect"

	"github.com/NeverStopDreamingWang/goi/v3/internal/i18n"
)

// ModelStructValue 验证模型并取得结构体值
//
// 参数:
//   - model: any 结构体或非 nil 的结构体指针，不支持多级指针
//
// 返回:
//   - reflect.Value: 结构体值
//   - error: 模型为 nil 或类型不符合要求时的错误
func ModelStructValue(model any) (reflect.Value, error) {
	if model == nil {
		modelIsNilErrorMsg := i18n.T("db.model_is_nil", map[string]any{"name": "model"})
		return reflect.Value{}, errors.New(modelIsNilErrorMsg)
	}

	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		if modelValue.IsNil() {
			modelIsNilErrorMsg := i18n.T("db.model_is_nil", map[string]any{"name": "model"})
			return reflect.Value{}, errors.New(modelIsNilErrorMsg)
		}
		modelValue = modelValue.Elem()
	}
	if modelValue.Kind() != reflect.Struct {
		modelIsNotStructErrorMsg := i18n.T("db.model_is_not_struct", map[string]any{"name": "model"})
		return reflect.Value{}, errors.New(modelIsNotStructErrorMsg)
	}
	return modelValue, nil
}

// ValidateOperationModel 检查操作模型与已配置模型的结构体类型是否一致
//
// 参数:
//   - configuredModel: any 已配置的模型，支持结构体或结构体指针
//   - operationModel: any 本次操作的模型，支持结构体或结构体指针
//
// 返回:
//   - reflect.Value: 操作模型的结构体值
//   - error: 模型无效或结构体类型不一致时的错误
func ValidateOperationModel(configuredModel any, operationModel any) (reflect.Value, error) {
	operationValue, err := ModelStructValue(operationModel)
	if err != nil {
		return reflect.Value{}, err
	}

	configuredValue, err := ModelStructValue(configuredModel)
	if err != nil {
		return reflect.Value{}, err
	}
	if operationValue.Type() != configuredValue.Type() {
		modelTypeIsNotMatchErrorMsg := i18n.T("db.model_type_is_not_match", map[string]any{
			"want_type": configuredValue.Type().String(),
			"got_type":  operationValue.Type().String(),
		})
		return reflect.Value{}, errors.New(modelTypeIsNotMatchErrorMsg)
	}
	return operationValue, nil
}
