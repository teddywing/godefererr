package defererr_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
	"gopkg.teddywing.com/defererr"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, defererr.Analyzer, ".")
}
