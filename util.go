package main

import (
	"io"
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

func copyFile(dst, src string) (int64, error) {
	sf, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sf.Close()

	df, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer df.Close()

	return io.Copy(df, sf)
}
