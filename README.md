godefererr
==========

[![GoDoc](https://godocs.io/gopkg.teddywing.com/defererr?status.svg)][Documentation]

An analyser that reports incorrectly returned errors from defer closures.

Errors can be returned from a `defer` closure by assigning the error to a
captured variable declared in the function signature. This analyser looks for
defer closures that assign captured error variables and checks that they are
correctly declared and returned.


## Example
Given the following program:

``` go
package main

import (
	"errors"
	"log"
)

func main() {
	err := returnErrorFromDefer()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("test")
}

func returnErrorFromDefer() error { // return signature should be '(err error)'
	var err error = nil
	if err != nil {
		return err
	}

	defer func() {
		err = errors.New("defer error")
	}()

	return nil // return should use 'err'
}
```

the analyser produces the following results:

	$ godefererr ./...
	package_doc_example.go:17:29: return signature should use named error parameter err
	package_doc_example.go:27:9: does not return 'err'


## Install

	$ go install gopkg.teddywing.com/defererr/cmd/godefererr@latest


## License
Copyright © 2023 Teddy Wing. Licensed under the GNU GPLv3+ (see the included
COPYING file).


[Documentation]: https://godocs.io/gopkg.teddywing.com/defererr
