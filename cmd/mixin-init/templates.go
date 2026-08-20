package main

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed templates/main.go.tmpl templates/ci.yml.tmpl templates/LICENSE
var templateFS embed.FS

// templateData fills in the placeholders in main.go.tmpl and ci.yml.tmpl.
type templateData struct {
	Name   string
	Author string
}

func renderTemplate(name string, data templateData) ([]byte, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/"+name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readLicense() ([]byte, error) {
	return templateFS.ReadFile("templates/LICENSE")
}
