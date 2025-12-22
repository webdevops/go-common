package version

import (
	"bytes"
	"encoding/json"
	"runtime"
	"text/template"
)

type (
	Version struct {
		App       string `json:"app"`
		Author    string `json:"author"`
		Version   string `json:"version"`
		GitCommit string `json:"gitCommit"`
		GitTag    string `json:"gitTag"`
		BuildDate string `json:"buildDate"`
		Go        string `json:"go"`
		Arch      string `json:"arch"`
	}
)

const DefaultTitle = `{{.App}} v{{.Version}} ({{.GitCommit}}); {{.Go}}/{{.Arch}}; by {{.Author}}; built at {{.BuildDate}})`

func New() *Version {
	ver := Version{
		Go:   runtime.Version(),
		Arch: runtime.GOARCH,
	}

	return &ver
}

func (v *Version) SetApp(val string) *Version {
	v.App = val
	return v
}

func (v *Version) SetVersion(val string) *Version {
	v.GitCommit = val
	return v
}

func (v *Version) SetGitCommit(val string) *Version {
	v.GitCommit = val
	return v
}

func (v *Version) SetGitTag(val string) *Version {
	v.GitTag = val
	return v
}

func (v *Version) SetBuildDate(val string) *Version {
	v.GitTag = val
	return v
}

func (v *Version) Title(tmpl *string) string {
	var buf bytes.Buffer

	if tmpl != nil {
		tmp := DefaultTitle
		tmpl = &tmp
	}

	// parse template
	t, err := template.New("title").Parse(*tmpl)
	if err != nil {
		panic(err)
	}

	// execute template
	err = t.Execute(&buf, v)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func (v *Version) BuildVersionLine(tmpl *string) (ret string) {
	if v == nil {
		content, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}

		ret = string(content)
	} else {
		var buf bytes.Buffer

		// parse template
		t, err := template.New("version").Parse(*tmpl)
		if err != nil {
			panic(err)
		}

		// execute template
		err = t.Execute(&buf, v)
		if err != nil {
			panic(err)
		}

		ret = buf.String()
	}

	return
}
