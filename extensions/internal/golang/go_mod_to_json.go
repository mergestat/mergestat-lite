package golang

import (
	"encoding/json"

	"go.riyazali.net/sqlite"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

type GoModToJSON struct{}

type Version struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type VersionInterval struct {
	Low  string `json:"low"`
	High string `json:"high"`
}

type Require struct {
	Mod      Version `json:"mod"`
	Indirect bool    `json:"indirect,omitempty"`
}

type Exclude struct {
	Mod Version
}

type Replace struct {
	Old Version `json:"old"`
	New Version `json:"new"`
}

type Retract struct {
	VersionInterval
	Rationale string `json:"rationale"`
}

type GoModFile struct {
	Version Version    `json:"version"`
	Go      string     `json:"go"`
	Require []*Require `json:"require"`
	Exclude []*Exclude `json:"exclude"`
	Replace []*Replace `json:"replace"`
	Retract []*Retract `json:"retract"`
}

func goModVersionToVersion(v module.Version) Version {
	return Version{
		Path:    v.Path,
		Version: v.Version,
	}
}

func (f *GoModToJSON) Args() int           { return 1 }
func (f *GoModToJSON) Deterministic() bool { return true }
func (f *GoModToJSON) Apply(context *sqlite.Context, value ...sqlite.Value) {
	parsed, err := modfile.Parse("go.mod", value[0].Blob(), nil)
	if err != nil {
		context.ResultError(err)
		return
	}
	file := &GoModFile{
		Version: goModVersionToVersion(parsed.Module.Mod),
		Go:      parsed.Go.Version,
		Require: make([]*Require, len(parsed.Require)),
		Exclude: make([]*Exclude, len(parsed.Exclude)),
		Replace: make([]*Replace, len(parsed.Replace)),
		Retract: make([]*Retract, len(parsed.Retract)),
	}

	for i, r := range parsed.Require {
		file.Require[i] = &Require{
			Mod:      goModVersionToVersion(r.Mod),
			Indirect: r.Indirect,
		}
	}

	for i, e := range parsed.Exclude {
		file.Exclude[i] = &Exclude{
			Mod: goModVersionToVersion(e.Mod),
		}
	}

	for i, r := range parsed.Replace {
		file.Replace[i] = &Replace{
			Old: goModVersionToVersion(r.Old),
			New: goModVersionToVersion(r.New),
		}
	}

	for i, r := range parsed.Retract {
		file.Retract[i] = &Retract{
			VersionInterval: VersionInterval{
				Low: r.Low, High: r.High,
			},
			Rationale: r.Rationale,
		}
	}

	str, err := json.Marshal(file)
	if err != nil {
		context.ResultError(err)
		return
	}

	context.ResultText(string(str))
}
