// Copyright (c) 2023  Teddy Wing
//
// This file is part of Godefererr.
//
// Godefererr is free software: you can redistribute it and/or
// modify it under the terms of the GNU General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// Godefererr is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Godefererr. If not, see <https://www.gnu.org/licenses/>.

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
