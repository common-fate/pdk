package boilerplate

import "embed"

//go:embed all:templates/**
var TemplateFiles embed.FS
