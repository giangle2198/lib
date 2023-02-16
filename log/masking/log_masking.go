package log

import (
	"encoding/json"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Object(key string, value interface{}) zapcore.Field {
	marshalByte, err := json.Marshal(value)
	if err != nil {
		zap.S().Panic(err)
		return zapcore.Field{Key: key, Type: zapcore.ObjectMarshalerType, Interface: value}
	}

	masked := GetJSONMaskLogging().MaskJSON(string(marshalByte))
	return zapcore.Field{Key: key, Type: zapcore.StringType, Interface: masked, String: masked}
}

func InitEncoderForJSON(maskFields map[string]string) {
	InitJSONMaskLogging(maskFields)
}
