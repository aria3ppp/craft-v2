package craft

type Context struct {
	MacroPackageImports  map[string]string
	GoFile               string
	RelativePath         string
	CurrentPkgImportPath string
	PWD                  string
}

func (c *Context) PackageImport(pkg string) string {
	return c.MacroPackageImports[pkg]
}
