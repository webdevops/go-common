package to

import (
	"testing"
)

func TestNumberInt(t *testing.T) {
	var numericPtrVal int
	var numericVal int

	var emptyPtr *int
	numericVal = Number(emptyPtr)
	helperExpectNumber(t, numericVal, 0)

	numericPtrVal = int(123)
	numericVal = Number(&numericPtrVal)
	helperExpectNumber(t, numericVal, 123)

	numericPtrVal = int(0)
	numericVal = Number(&numericPtrVal)
	helperExpectNumber(t, numericVal, 0)
}

func TestNumberIntPtr(t *testing.T) {
	var numericVal int
	var numericPtrVal *int

	numericVal = 0
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 0)

	numericVal = 1
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 1)

	numericVal = 123
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 123)
}

func TestNumberInt64(t *testing.T) {
	var numericPtrVal int64
	var numericVal int64

	var emptyPtr *int64
	numericVal = Number(emptyPtr)
	helperExpectNumber(t, numericVal, 0)

	numericPtrVal = int64(123)
	numericVal = Number(&numericPtrVal)
	helperExpectNumber(t, numericVal, 123)

	numericPtrVal = int64(0)
	numericVal = Number(&numericPtrVal)
	helperExpectNumber(t, numericVal, 0)
}

func TestNumberInt64Ptr(t *testing.T) {
	var numericVal int64
	var numericPtrVal *int64

	numericVal = 0
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 0)

	numericVal = 1
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 1)

	numericVal = 123
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 123)
}

func TestNumberFloat64(t *testing.T) {
	var numericPtrVal float64
	var numericVal float64

	var emptyPtr *float64
	numericVal = Number(emptyPtr)
	helperExpectNumber(t, numericVal, 0)

	numericPtrVal = float64(123)
	numericVal = Number(&numericPtrVal)
	helperExpectNumber(t, numericVal, 123)

	numericPtrVal = float64(0)
	numericVal = Number(&numericPtrVal)
	helperExpectNumber(t, numericVal, 0)
}

func TestNumberFloat64Ptr(t *testing.T) {
	var numericVal float64
	var numericPtrVal *float64

	numericVal = 0
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 0)

	numericVal = 1
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 1)

	numericVal = 123
	numericPtrVal = NumberPtr(numericVal)
	helperExpectNumber(t, *numericPtrVal, 123)
}

func helperExpectNumber[N NumberInterface](t *testing.T, val N, expectVal N) {
	t.Helper()

	if val != expectVal {
		t.Fatalf(`Expected numericVal to be %v, got %v`, val, expectVal)
	}
}
