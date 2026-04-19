package main

import (
	"fmt"

	"github.com/LyleLiu666/simplykb/examples/internal/exampleenv"
)

func main() {
	fmt.Println(exampleenv.DefaultDatabaseURL())
}
