package to

import (
	"testing"
)

func TestString(t *testing.T) {
	var stringPtrVal string
	var stringVal string

	stringVal = String(nil)
	helperExpectString(t, &stringVal, "")

	stringPtrVal = ""
	stringVal = String(&stringPtrVal)
	helperExpectString(t, &stringVal, "")

	stringPtrVal = "foobar"
	stringVal = String(&stringPtrVal)
	helperExpectString(t, &stringVal, "foobar")
}

func TestStringPtr(t *testing.T) {
	stringPtr := StringPtr("")
	helperExpectString(t, stringPtr, "")

	stringPtr = StringPtr("foobar")
	helperExpectString(t, stringPtr, "foobar")
}

func helperExpectString(t *testing.T, val *string, expectVal string) {
	t.Helper()

	if val == nil {
		t.Fatalf(`Expected stringVal to be not nil, got %v`, val)
		return
	}

	if *val != expectVal {
		t.Fatalf(`Expected stringVal to be %v, got %v`, val, expectVal)
	}
}
