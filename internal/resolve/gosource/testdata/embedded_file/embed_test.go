package main

import (
	_ "embed"
	"fmt"
	"testing"
)

//go:embed hello_test.txt
var embeddedTestStr string

func TestEmbed(t *testing.T) {
	fmt.Println(embeddedTestStr)
}
