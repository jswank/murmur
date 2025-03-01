package murmur

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Target is a struct that represents a target.json file
// target = {
//   _type:: 'target',
//   name: ''   // repo name, i.e. ets-cloudops-infrastrcuture
//   repo: '',  // full repo name, i.e. cfacorp/ets-cloudops-infrastructure
//   path: '',  // top level destination to write outputs
//   branch: 'master',  // git branch name
//   types: [],  // types of outputs ("datasources", "connections", etc)
// };

type Target struct {
	Dir      string   `json:"-"`
	Filename string   `json:"-"`
	Prefix   string   `json:"-"`
	App      string   `json:"app"`
	Branch   string   `json:"branch"`
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Repo     string   `json:"repo"`
	Types    []string `json:"types"`
}

// NewTargetsFromFile creates a new Target struct from a JSON file
func NewTargetsFromFile(filename string) ([]Target, error) {
	var targets []Target
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &targets)
	if err != nil {
		return targets, err
	}

	dir := filepath.Dir(filename)
	fname := filepath.Base(filename)
	prefix := fname[:len(fname)-len("-targets.json")]

	for i := range targets {
		targets[i].Dir = dir
		targets[i].Filename = fname
		targets[i].Prefix = prefix
	}
	return targets, nil
}

func (t Target) CloneDir() string {
	if t.Repo == "." {
		return "."
	} else {
		return fmt.Sprintf("%s:%s", t.Name, t.Branch)
	}
}
