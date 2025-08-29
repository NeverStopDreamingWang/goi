package utils

import (
	"reflect"

	"github.com/NeverStopDreamingWang/goi/jwt"
)

type Payloads struct {
	jwt.Payloads
	User_id  int64  `json:"user_id"`
	Username string `json:"username"`
}

func Update(instance interface{}, validated_data interface{}) {
	instanceValue := reflect.ValueOf(instance)
	validatedDataValue := reflect.ValueOf(validated_data)

	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}
	if validatedDataValue.Kind() == reflect.Ptr {
		validatedDataValue = validatedDataValue.Elem()
	}
	instanceType := instanceValue.Type()

	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceType.Field(i)
		instanceField := instanceValue.Field(i)
		validatedField := validatedDataValue.FieldByName(field.Name)

		if validatedField.Kind() == reflect.Ptr && validatedField.IsNil() {
			continue
		}
		if !instanceField.CanSet() {
			continue
		}
		instanceField.Set(validatedField)
	}
}
