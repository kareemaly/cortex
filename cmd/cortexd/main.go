package main

import (
	"fmt"

	"github.com/kareemaly/cortex1/pkg/version"
)

func main() {
	fmt.Println(version.String("cortexd"))
}
