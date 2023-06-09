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

import "errors"

func shouldDeclareErrInSignature() error { // want "return signature should use named error parameter err"
	// Should use variable declared in signature. We don't need to report this
	// as if the variable is declared in the signature, a redeclaration causes
	// a compile error.
	var err error

	err = nil
	if err != nil {
		return err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return nil // want "does not return 'err'"
}

func doesDeclareErrInSignature() (err error) {
	err = nil
	if err != nil {
		return err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return nil // want "does not return 'err'"
}

func returnedErrorMustMatchDeferErrorName() (err error) {
	err = nil
	if err != nil {
		return err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	err2 := errors.New("returned error")

	return err2 // want "does not return 'err'"
}

func deferUsesUnconventionalErrName() error { // want "return signature should use named error parameter anErr"
	var anErr error

	anErr = nil
	if anErr != nil {
		return anErr
	}

	defer func() {
		anErr = errors.New("defer error")
	}()

	return anErr
}


func multipleReturnValuesString() (string, error) { // want "return signature should use named error parameter err"
	var err error = nil
	if err != nil {
		return "", err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return "ret", err
}

type AStruct struct {}

func multipleReturnValuesStructBool() (*AStruct, bool, error) { // want "return signature should use named error parameter err"
	var err error = nil
	if err != nil {
		return nil, false, err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return &AStruct{}, true, err
}

func good() (err error) {
	err = nil
	if err != nil {
		return err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return err
}

func noErrorInReturn() string {
	return "test"
}
