package template

import "embed"

//go:embed html/index.html
var IndexHtml embed.FS

//go:embed all:html/*
var Html embed.FS

//go:embed test.txt
var TestTxt embed.FS
