package main

import (
	"gofile-cli/cmd"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	cmd.Execute()
}
