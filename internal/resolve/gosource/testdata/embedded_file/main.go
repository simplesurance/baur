package main

import (
	"embed"
	_ "embed"
	"fmt"
	"io/fs"
)

//go:embed hello.txt
var embeddedStr string

//go:embed data/*
var dataFiles embed.FS

func main() {
	fmt.Println(embeddedStr)

	matches, err := fs.Glob(dataFiles, "*/*")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v\n", matches)
}
