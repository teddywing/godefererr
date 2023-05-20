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
