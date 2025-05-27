package utils

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type SupportedEnvTypes interface {
	string | int64 | bool
}

func GetEnvWithDefault[T SupportedEnvTypes](key string, defaultValue T) T {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}

	var result any
	var err error

	switch any(defaultValue).(type) {
	case string:
		result = value
	case int64:
		var parsed int64
		parsed, err = strconv.ParseInt(value, 10, 64)
		result = parsed
	case bool:
		var parsed bool
		parsed, err = strconv.ParseBool(strings.ToLower(value))
		result = parsed
	default:
		slog.Warn("unsupported environment variable type, using default value", "env", key, "default", defaultValue)
		return defaultValue
	}

	if err != nil {
		slog.Warn("error parsing environment variable, using default value", "env", key, "default", defaultValue, "error", err)
		return defaultValue
	}

	return result.(T)
}
