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
	"errors"
	"log"
)

func genDeclErrNotReported() error {
	defer func() {
		var err error
		err = errors.New("newly-declared error")
		if err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}

func assignmentDeclErrNotReported() error {
	defer func() {
		err := errors.New("newly-declared error")
		if err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}
