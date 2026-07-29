package sema

import (
	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
)

type Import struct {
	Declaration *ast.RequireDecl
	Path        string
	Prefix      string
	Target      *Module
}

type Module struct {
	Source  source.File
	Syntax  *ast.File
	Imports map[string]*Import
}

func (module *Module) Import(prefix string) (*Import, bool) {
	if module == nil {
		return nil, false
	}

	resolved, ok := module.Imports[prefix]
	return resolved, ok
}

type Project struct {
	Root    string
	Modules []*Module
}

func (project *Project) Module(relPath string) (*Module, bool) {
	if project == nil {
		return nil, false
	}

	for _, module := range project.Modules {
		if module.Source.RelPath == relPath {
			return module, true
		}
	}

	return nil, false
}

type Result struct {
	Project     *Project
	Diagnostics []diagnostic.Diagnostic
}
