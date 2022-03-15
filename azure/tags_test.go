package azure

import (
	"encoding/json"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/prometheus/client_golang/prometheus"
)

type (
	TagTest struct {
		Expect       prometheus.Labels
		ResourceTags interface{}
		Tags         []string
	}
)

func Test_Tags(t *testing.T) {

	tests := []TagTest{
		{
			Expect:       prometheus.Labels{"tag_owner": "foobar"},
			ResourceTags: map[string]string{"foo": "bar", "owner": "foobar"},
			Tags:         []string{"owner"},
		},
		{
			Expect:       prometheus.Labels{"tag_owner": "foobar"},
			ResourceTags: map[string]*string{"foo": to.StringPtr("bar"), "owner": to.StringPtr("foobar")},
			Tags:         []string{"owner"},
		},

		{
			Expect:       prometheus.Labels{"tag_foo": "bar", "tag_owner": "foobar"},
			ResourceTags: map[string]string{"foo": "bar", "owner": "foobar"},
			Tags:         []string{"owner", "foo"},
		},
		{
			Expect:       prometheus.Labels{"tag_foo": "bar", "tag_owner": "foobar"},
			ResourceTags: map[string]*string{"foo": to.StringPtr("bar"), "owner": to.StringPtr("foobar")},
			Tags:         []string{"owner", "foo"},
		},

		{
			Expect:       prometheus.Labels{"tag_bar": "", "tag_owner": "foobar"},
			ResourceTags: map[string]string{"foo": "bar", "owner": "foobar"},
			Tags:         []string{"owner", "bar"},
		},
		{
			Expect:       prometheus.Labels{"tag_bar": "", "tag_owner": "foobar"},
			ResourceTags: map[string]*string{"foo": to.StringPtr("bar"), "owner": to.StringPtr("foobar")},
			Tags:         []string{"owner", "bar"},
		},

		{
			Expect:       prometheus.Labels{"tag_bar": "", "tag_foo": "bar", "tag_owner": "foobar"},
			ResourceTags: map[string]string{"foo": "bar", "owner": "foobar"},
			Tags:         []string{"owner", "bar", "foo"},
		},
		{
			Expect:       prometheus.Labels{"tag_bar": "", "tag_foo": "bar", "tag_owner": "foobar"},
			ResourceTags: map[string]*string{"foo": to.StringPtr("bar"), "owner": to.StringPtr("foobar")},
			Tags:         []string{"owner", "bar", "foo"},
		},
	}

	for _, testcase := range tests {
		assumePrometheusLabels(t, testcase.Expect, AddResourceTagsToPrometheusLabels(prometheus.Labels{}, testcase.ResourceTags, testcase.Tags))
	}
}

func Test_TagsCase(t *testing.T) {

	tests := []TagTest{
		{
			Expect:       prometheus.Labels{"tag_owner": "foobar"},
			ResourceTags: map[string]string{"OWNER": "foobar"},
			Tags:         []string{"owner"},
		},

		{
			Expect:       prometheus.Labels{"tag_owner": "foobar"},
			ResourceTags: map[string]string{"OWNER": "foobar"},
			Tags:         []string{"OwNeR"},
		},
	}

	for _, testcase := range tests {
		assumePrometheusLabels(t, testcase.Expect, AddResourceTagsToPrometheusLabels(prometheus.Labels{}, testcase.ResourceTags, testcase.Tags))
	}
}

func Test_TagsSettings(t *testing.T) {

	tests := []TagTest{
		{
			Expect:       prometheus.Labels{"tag_owner": "FOOBAR"},
			ResourceTags: map[string]string{"OWNER": "foobar"},
			Tags:         []string{"owner?toUpper"},
		},

		{
			Expect:       prometheus.Labels{"tag_owner": "foobar"},
			ResourceTags: map[string]string{"OWNER": "FooBar"},
			Tags:         []string{"owner?toLower"},
		},
	}

	for _, testcase := range tests {
		assumePrometheusLabels(t, testcase.Expect, AddResourceTagsToPrometheusLabels(prometheus.Labels{}, testcase.ResourceTags, testcase.Tags))
	}
}

func assumePrometheusLabels(t *testing.T, expect prometheus.Labels, got prometheus.Labels) {
	t.Helper()

	expectJson, err := json.Marshal(expect)
	if err != nil {
		t.Errorf("unable to marshal: %v", err)
	}

	gotJson, err := json.Marshal(got)
	if err != nil {
		t.Errorf("unable to marshal: %v", err)
	}

	expectJsonString := string(expectJson)
	gotJsonString := string(gotJson)

	if expectJsonString != gotJsonString {
		t.Errorf("expected:%v \n got: %v", expectJsonString, gotJsonString)
	}
}
