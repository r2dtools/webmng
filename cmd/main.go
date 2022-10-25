package main

import (
	"github.com/r2dtools/webmng/cmd/mng"
)

func main() {
	if err := mng.RootCmd.Execute(); err != nil {
		panic(err)
	}
}
