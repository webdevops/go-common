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
		Queries []ConfigQuery `json:"queries"`
	}

	ConfigQuery struct {
		*ConfigQueryMetric
		QueryMode     string    `json:"queryMode"`
		Workspaces    *[]string `json:"workspaces"`
		Metric        string    `json:"metric"`
		Module        string    `json:"module"`
		Query         string    `json:"query"`
		Timespan      *string   `json:"timespan"`
		Subscriptions *[]string `json:"subscriptions"`
	}

	ConfigQueryMetric struct {
		Value        *float64                 `json:"value"`
		Fields       []ConfigQueryMetricField `json:"fields"`
		Labels       map[string]string        `json:"labels"`
		DefaultField ConfigQueryMetricField   `json:"defaultField"`
		Publish      *bool                    `json:"publish"`
	}

	ConfigQueryMetricField struct {
		Name    string                         `json:"name"`
		Metric  string                         `json:"metric"`
		Source  string                         `json:"source"`
		Target  string                         `json:"target"`
		Type    string                         `json:"type"`
		Labels  map[string]string              `json:"labels"`
		Filters []ConfigQueryMetricFieldFilter `json:"filters"`
		Expand  *ConfigQueryMetric             `json:"expand"`
	}

	ConfigQueryMetricFieldFilter struct {
		Type         string `json:"type"`
		RegExp       string `json:"regexp"`
		Replacement  string `json:"replacement"`
		parsedRegexp *regexp.Regexp
	}

	ConfigQueryMetricFieldFilterParser struct {
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
			return fmt.Errorf("query \"%v\": %w", queryConfig.Metric, err)
		}
	}

	return nil
}

func (c *ConfigQuery) Validate() error {
	if err := c.ConfigQueryMetric.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *ConfigQueryMetric) Validate() error {
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

func (c *ConfigQueryMetric) IsPublished() bool {
	if c.Publish != nil {
		return *c.Publish
	}

	return true
}

func (c *ConfigQueryMetricField) Validate() error {
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

func (c *ConfigQueryMetricField) GetType() (ret string) {
	ret = strings.ToLower(c.Type)

	if ret == "" {
		ret = MetricFieldTypeDefault
	}

	return
}

func (c *ConfigQueryMetricFieldFilter) Validate() error {
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

func (m *ConfigQueryMetric) IsExpand(field string) bool {
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

func (m *ConfigQueryMetric) GetFieldConfigMap() (list map[string][]ConfigQueryMetricField) {
	list = map[string][]ConfigQueryMetricField{}

	for _, field := range m.Fields {
		if _, ok := list[field.Name]; !ok {
			list[field.Name] = []ConfigQueryMetricField{}
		}
		list[field.GetSourceField()] = append(list[field.GetSourceField()], field)
	}

	return
}

func (f *ConfigQueryMetricField) GetSourceField() (ret string) {
	ret = f.Source
	if ret == "" {
		ret = f.Name
	}
	return
}

func (f *ConfigQueryMetricField) IsExpand() bool {
	return f.Type == MetricFieldTypeExpand || f.Expand != nil
}

func (f *ConfigQueryMetricField) IsSourceField() bool {
	return f.Source != ""
}

func (f *ConfigQueryMetricField) IsTypeIgnore() bool {
	return f.GetType() == MetricFieldTypeIgnore
}

func (f *ConfigQueryMetricField) IsTypeId() bool {
	return f.GetType() == MetricFieldTypeId
}

func (f *ConfigQueryMetricField) IsTypeValue() bool {
	return f.GetType() == MetricFieldTypeValue
}

func (f *ConfigQueryMetricField) GetTargetFieldName(sourceName string) (ret string) {
	ret = sourceName
	if f.Target != "" {
		ret = f.Target
	} else if f.Name != "" {
		ret = f.Name
	}
	return
}

func (f *ConfigQueryMetricField) TransformString(value string) (ret string) {
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

func (f *ConfigQueryMetricField) TransformFloat64(value float64) (ret string) {
	ret = fmt.Sprintf("%v", value)
	ret = f.TransformString(ret)
	return
}

func (f *ConfigQueryMetricField) TransformBool(value bool) (ret string) {
	if value {
		ret = "true"
	} else {
		ret = "false"
	}
	ret = f.TransformString(ret)
	return
}

func (f *ConfigQueryMetricFieldFilter) UnmarshalJSON(c []byte) error {
	var multi ConfigQueryMetricFieldFilterParser
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
