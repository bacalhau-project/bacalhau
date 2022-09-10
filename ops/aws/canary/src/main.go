package main

import (
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
)

func main() {
	fmt.Println("Hello, world!")
	jobSelection := computenode.Anywhere
	if jobSelection == computenode.Anywhere {
		fmt.Println("Hello, from Anywhere!")
	}
}
