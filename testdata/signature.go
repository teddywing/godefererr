package main

import "errors"

func declareErrInSignature() error { // want "return signature should be '(err error)'"
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
