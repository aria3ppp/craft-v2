package main

import (
	"encoding/json"
	"errors"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type Module struct {
	Path  string `json:"Path"`
	GoMod string `json:"GoMod"`
}

func (m Module) Validate() (Module, error) {
	if m.Path == "" {
		return Module{}, errors.New("empty 'Path' field")
	}
	if m.GoMod == "" {
		return Module{}, errors.New("empty 'GoMod' field")
	}

	return m, nil
}

func modInfo() (m Module, err error) {
	var jsonBytes []byte

	jsonBytes, err = exec.Command("go", "list", "-m", "-json").Output()
	if err != nil {
		return Module{}, err
	}

	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return Module{}, err
	}

	return m.Validate()
}

func relativePathFromRoot(
	pwd string,
	mod Module,
) (relpath string, pkgImportPath string, err error) {
	gomodDir := filepath.Dir(mod.GoMod)

	relpath, err = filepath.Rel(gomodDir, pwd)
	if err != nil {
		return "", "", err
	}

	relpathSlash := strings.ReplaceAll(relpath, "\\", "/")
	pkgImportPath = path.Join(mod.Path, relpathSlash)

	return relpath, pkgImportPath, nil
}
