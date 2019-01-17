package main

import (
	"fmt"

	"github.com/thanethomson/golang-networking/lib/experiments"
)

func main() {
	if err := experiments.RunUnixDomainSocketTimeoutExperiment(); err != nil {
		fmt.Println(err.Error())
	}
}
