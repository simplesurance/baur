//go:build maintest

package main

import (
	"fmt"
	"testing"

	"github.com/simplesurance/baur-test/generator"
)

func TestPrint(t *testing.T) {
	fmt.Println(generator.UUID())
}
