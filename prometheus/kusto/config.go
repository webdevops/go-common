package kusto

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	MetricFieldTypeExpand  = "expand"
	MetricFieldTypeIgnore  = "ignore"
	MetricFieldTypeId      = "id"
	MetricFieldTypeValue   = "value"
	MetricFieldTypeDefault = "string"
	MetricFieldTypeBool    = "bool"
	MetricFieldTypeBoolean = "boolean"

	MetricFieldFilterToLower    = "tolower"
	MetricFieldFilterToUpper    = "toupper"
	MetricFieldFilterToTitle    = "totitle"
	MetricFieldFilterToRegexp   = "regexp"
	MetricFieldFilterToUnixtime = "tounixtime"
)

type (
	Config struct {
		Queries []Query `json:"queries"`
	}

	Query struct {
		*QueryMetric
		QueryMode     string    `json:"queryMode"`
		Workspaces    *[]string `json:"workspaces"`
		Metric        string    `json:"metric"`
		Module        string    `json:"module"`
		Query         string    `json:"query"`
		Timespan      *string   `json:"timespan"`
		Subscriptions *[]string `json:"subscriptions"`
	}

	QueryMetric struct {
		Value        *float64          `json:"value"`
		Fields       []MetricField     `json:"fields"`
		Labels       map[string]string `json:"labels"`
		DefaultField MetricField       `json:"defaultField"`
		Publish      *bool             `json:"publish"`
	}

	MetricField struct {
		Name    string              `json:"name"`
		Metric  string              `json:"metric"`
		Source  string              `json:"source"`
		Target  string              `json:"target"`
		Type    string              `json:"type"`
		Labels  map[string]string   `json:"labels"`
		Filters []MetricFieldFilter `json:"filters"`
		Expand  *QueryMetric        `json:"expand"`
	}

	MetricFieldFilter struct {
		Type         string `json:"type"`
		RegExp       string `json:"regexp"`
		Replacement  string `json:"replacement"`
		parsedRegexp *regexp.Regexp
	}

	MetricFieldFilterParser struct {
		Type        string `json:"type"`
		RegExp      string `json:"regexp"`
		Replacement string `json:"replacement"`
	}
)

func (c *Config) Validate() error {
	if len(c.Queries) == 0 {
		return errors.New("no queries found")
	}

	for _, queryConfig := range c.Queries {
		if err := queryConfig.Validate(); err != nil {
			return fmt.Errorf("query \"%v\": %w", queryConfig.QueryMetric, err)
		}
	}

	return nil
}

func (c *Query) Validate() error {
	if err := c.QueryMetric.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *QueryMetric) Validate() error {
	// validate default field
	c.DefaultField.Name = "default"
	if err := c.DefaultField.Validate(); err != nil {
		return err
	}

	// validate fields
	for _, field := range c.Fields {
		if err := field.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *QueryMetric) IsPublished() bool {
	if c.Publish != nil {
		return *c.Publish
	}

	return true
}

func (c *MetricField) Validate() error {
	if c.Name == "" {
		return errors.New("no field name set")
	}

	switch c.GetType() {
	case MetricFieldTypeDefault:
	case MetricFieldTypeBool:
	case MetricFieldTypeBoolean:
	case MetricFieldTypeExpand:
	case MetricFieldTypeId:
	case MetricFieldTypeValue:
	case MetricFieldTypeIgnore:
	default:
		return fmt.Errorf("field \"%s\": unsupported type \"%s\"", c.Name, c.GetType())
	}

	for _, filter := range c.Filters {
		if err := filter.Validate(); err != nil {
			return fmt.Errorf("field \"%v\": %w", c.Name, err)
		}
	}

	return nil
}

func (c *MetricField) GetType() (ret string) {
	ret = strings.ToLower(c.Type)

	if ret == "" {
		ret = MetricFieldTypeDefault
	}

	return
}

func (c *MetricFieldFilter) Validate() error {
	if c.Type == "" {
		return errors.New("no type name set")
	}

	switch strings.ToLower(c.Type) {
	case MetricFieldFilterToLower:
	case MetricFieldFilterToUpper:
	case MetricFieldFilterToTitle:
	case MetricFieldFilterToRegexp:
		if c.RegExp == "" {
			return errors.New("no regexp for filter set")
		}
		c.parsedRegexp = regexp.MustCompile(c.RegExp)
	default:
		return fmt.Errorf("filter \"%v\" not supported", c.Type)
	}

	return nil
}

func (m *QueryMetric) IsExpand(field string) bool {
	for _, fieldConfig := range m.Fields {
		if fieldConfig.Name == field {
			if fieldConfig.IsExpand() {
				return true
			}
			break
		}
	}

	return false
}

func (m *QueryMetric) GetFieldConfigMap() (list map[string][]MetricField) {
	list = map[string][]MetricField{}

	for _, field := range m.Fields {
		if _, ok := list[field.Name]; !ok {
			list[field.Name] = []MetricField{}
		}
		list[field.GetSourceField()] = append(list[field.GetSourceField()], field)
	}

	return
}

func (f *MetricField) GetSourceField() (ret string) {
	ret = f.Source
	if ret == "" {
		ret = f.Name
	}
	return
}

func (f *MetricField) IsExpand() bool {
	return f.Type == MetricFieldTypeExpand || f.Expand != nil
}

func (f *MetricField) IsSourceField() bool {
	return f.Source != ""
}

func (f *MetricField) IsTypeIgnore() bool {
	return f.GetType() == MetricFieldTypeIgnore
}

func (f *MetricField) IsTypeId() bool {
	return f.GetType() == MetricFieldTypeId
}

func (f *MetricField) IsTypeValue() bool {
	return f.GetType() == MetricFieldTypeValue
}

func (f *MetricField) GetTargetFieldName(sourceName string) (ret string) {
	ret = sourceName
	if f.Target != "" {
		ret = f.Target
	} else if f.Name != "" {
		ret = f.Name
	}
	return
}

func (f *MetricField) TransformString(value string) (ret string) {
	ret = value

	switch f.Type {
	case MetricFieldTypeBoolean:
		fallthrough
	case MetricFieldTypeBool:
		switch strings.ToLower(ret) {
		case "1":
			fallthrough
		case "true":
			fallthrough
		case "yes":
			ret = "true"
		default:
			ret = "false"
		}
	}
	for _, filter := range f.Filters {
		switch strings.ToLower(filter.Type) {
		case MetricFieldFilterToLower:
			ret = strings.ToLower(ret)
		case MetricFieldFilterToUpper:
			ret = strings.ToUpper(ret)
		case MetricFieldFilterToTitle:
			ret = strings.ToTitle(ret)
		case MetricFieldFilterToUnixtime:
			ret = convertStringToUnixtime(ret)
		case MetricFieldFilterToRegexp:
			if filter.parsedRegexp == nil {
				filter.parsedRegexp = regexp.MustCompile(filter.RegExp)
			}
			ret = filter.parsedRegexp.ReplaceAllString(ret, filter.Replacement)
		}
	}
	return
}

func (f *MetricField) TransformFloat64(value float64) (ret string) {
	ret = fmt.Sprintf("%v", value)
	ret = f.TransformString(ret)
	return
}

func (f *MetricField) TransformBool(value bool) (ret string) {
	if value {
		ret = "true"
	} else {
		ret = "false"
	}
	ret = f.TransformString(ret)
	return
}

func (f *MetricFieldFilter) UnmarshalJSON(c []byte) error {
	var multi MetricFieldFilterParser
	err := json.Unmarshal(c, &multi)
	if err != nil {
		var single string
		err := json.Unmarshal(c, &single)
		if err != nil {
			return err
		}
		f.Type = single
	} else {
		f.Type = multi.Type
		f.RegExp = multi.RegExp
		f.Replacement = multi.Replacement
	}
	return nil
}

func NewConfig(path string) (config Config) {
	var filecontent []byte

	config = Config{}

	/*  #nosec G304 */
	if data, err := os.ReadFile(path); err == nil {
		filecontent = data
	} else {
		panic(err)
	}

	if err := yaml.Unmarshal(filecontent, &config); err != nil {
		panic(err)
	}

	return
}
