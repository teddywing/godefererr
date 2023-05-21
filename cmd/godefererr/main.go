package main

import (
	"os"

	"golang.org/x/tools/go/analysis/singlechecker"
	"gopkg.teddywing.com/defererr"
)

var version = "0.0.1"

func main() {
	if len(os.Args) > 1 &&
		(os.Args[1] == "-V" ||
			os.Args[1] == "--version") {

		println(version)
		os.Exit(0)
	}

	singlechecker.Main(defererr.Analyzer)
}
