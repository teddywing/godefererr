package main

import "errors"

func shouldDeclareErrInSignature() error { // want "return signature should be '\\(err error\\)'"
	var err error // Should use variable declared in signature

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

func deferUsesUnconventionalErrName() error { // want "return signature should be '\\(anErr error\\)'"
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

func multipleReturnValuesString() (string, error) { // want "return signature should be '\\(string1 string, err error\\)'"
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

func multipleReturnValuesStruct() (*AStruct, error) { // want "return signature should be '\\(aStruct1 *AStruct, err error\\)'"
	var err error = nil
	if err != nil {
		return nil, err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return &AStruct{}, err
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
