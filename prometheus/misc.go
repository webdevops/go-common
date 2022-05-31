package prometheus

import (
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	stringTitleCaser = cases.Title(language.English, cases.NoLower)
)

func timeToFloat64(v time.Time) float64 {
	return float64(v.Unix())
}

func StringToTitle(val string) string {
	return stringTitleCaser.String(val)
}
