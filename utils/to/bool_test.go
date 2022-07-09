package to

import (
	"testing"
)

func TestBool(t *testing.T) {
	var boolPtrVal bool
	var boolVal bool

	boolVal = Bool(nil)
	helperExpectBool(t, &boolVal, false)

	boolPtrVal = true
	boolVal = Bool(&boolPtrVal)
	helperExpectBool(t, &boolVal, true)

	boolPtrVal = false
	boolVal = Bool(&boolPtrVal)
	helperExpectBool(t, &boolVal, false)
}

func TestBoolPtr(t *testing.T) {
	boolPtr := BoolPtr(true)
	helperExpectBool(t, boolPtr, true)

	boolPtr = BoolPtr(false)
	helperExpectBool(t, boolPtr, false)
}

func helperExpectBool(t *testing.T, val *bool, expectVal bool) {
	t.Helper()

	if val == nil {
		t.Fatalf(`Expected boolVal to be not nil, got %v`, val)
		return
	}

	if *val != expectVal {
		t.Fatalf(`Expected boolVal to be %v, got %v`, val, expectVal)
	}
}
