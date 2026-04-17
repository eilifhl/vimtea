package assert

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func Equal(t *testing.T, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		return true
	}
	t.Errorf("%s: expected %v, got %v", message(msgAndArgs...), expected, actual)
	return false
}

func NotEqual(t *testing.T, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		return true
	}
	t.Errorf("%s: values should not be equal (%v)", message(msgAndArgs...), actual)
	return false
}

func True(t *testing.T, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if value {
		return true
	}
	t.Errorf("%s: expected true", message(msgAndArgs...))
	return false
}

func False(t *testing.T, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if !value {
		return true
	}
	t.Errorf("%s: expected false", message(msgAndArgs...))
	return false
}

func Nil(t *testing.T, value any, msgAndArgs ...any) bool {
	t.Helper()
	if isNil(value) {
		return true
	}
	t.Errorf("%s: expected nil, got %v", message(msgAndArgs...), value)
	return false
}

func NotNil(t *testing.T, value any, msgAndArgs ...any) bool {
	t.Helper()
	if !isNil(value) {
		return true
	}
	t.Errorf("%s: expected non-nil", message(msgAndArgs...))
	return false
}

func Empty(t *testing.T, value any, msgAndArgs ...any) bool {
	t.Helper()
	if lengthOf(value) == 0 {
		return true
	}
	t.Errorf("%s: expected empty, got %v", message(msgAndArgs...), value)
	return false
}

func NotEmpty(t *testing.T, value any, msgAndArgs ...any) bool {
	t.Helper()
	if lengthOf(value) != 0 {
		return true
	}
	t.Errorf("%s: expected non-empty", message(msgAndArgs...))
	return false
}

func Len(t *testing.T, value any, expected int, msgAndArgs ...any) bool {
	t.Helper()
	actual := lengthOf(value)
	if actual == expected {
		return true
	}
	t.Errorf("%s: expected length %d, got %d", message(msgAndArgs...), expected, actual)
	return false
}

func Contains(t *testing.T, s, contains any, msgAndArgs ...any) bool {
	t.Helper()
	if strings.Contains(fmt.Sprint(s), fmt.Sprint(contains)) {
		return true
	}
	t.Errorf("%s: expected %v to contain %v", message(msgAndArgs...), s, contains)
	return false
}

func NotContains(t *testing.T, s, contains any, msgAndArgs ...any) bool {
	t.Helper()
	if !strings.Contains(fmt.Sprint(s), fmt.Sprint(contains)) {
		return true
	}
	t.Errorf("%s: expected %v not to contain %v", message(msgAndArgs...), s, contains)
	return false
}

func Greater(t *testing.T, a, b any, msgAndArgs ...any) bool {
	t.Helper()
	if toFloat(a) > toFloat(b) {
		return true
	}
	t.Errorf("%s: expected %v to be greater than %v", message(msgAndArgs...), a, b)
	return false
}

func GreaterOrEqual(t *testing.T, a, b any, msgAndArgs ...any) bool {
	t.Helper()
	if toFloat(a) >= toFloat(b) {
		return true
	}
	t.Errorf("%s: expected %v to be >= %v", message(msgAndArgs...), a, b)
	return false
}

func Less(t *testing.T, a, b any, msgAndArgs ...any) bool {
	t.Helper()
	if toFloat(a) < toFloat(b) {
		return true
	}
	t.Errorf("%s: expected %v to be less than %v", message(msgAndArgs...), a, b)
	return false
}

func message(args ...any) string {
	if len(args) == 0 {
		return "assertion failed"
	}
	return fmt.Sprint(args...)
}

func lengthOf(value any) int {
	if value == nil {
		return 0
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len()
	default:
		return 0
	}
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func toFloat(value any) float64 {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		return 0
	}
}
