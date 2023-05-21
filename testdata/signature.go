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

func returnsOtherVariable() (err error) {
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
