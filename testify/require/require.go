package require

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func NotNil(t *testing.T, value any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotNil(t, value, msgAndArgs...) {
		t.FailNow()
	}
}
