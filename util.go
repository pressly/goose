package main

import (
	"os"
	"text/template"
)

// common routines

func writeTemplateToFile(path string, t *template.Template, data interface{}) (string, error) {
	f, e := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		return "", e
	}
	defer f.Close()

	e = t.Execute(f, data)
	if e != nil {
		return "", e
	}

	return f.Name(), nil
}
