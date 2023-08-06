package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	startPath := "/Users/frrist/Workspace/src/github.com/bacalhau-project/bacalhau" // replace with your root path

	functionList := []string{
		"Condition", "Conditionf", "Contains", "Containsf", "DirExists", "DirExistsf",
		"ElementsMatch", "ElementsMatchf", "Empty", "Emptyf", "Equal", "EqualError",
		"EqualErrorf", "EqualExportedValues", "EqualExportedValuesf", "EqualValues",
		"EqualValuesf", "Equalf", "Error", "ErrorAs", "ErrorAsf", "ErrorContains",
		"ErrorContainsf", "ErrorIs", "ErrorIsf", "Errorf", "Eventually",
		"EventuallyWithT", "EventuallyWithTf", "Eventuallyf", "Exactly", "Exactlyf",
		"Fail", "FailNow", "FailNowf", "Failf", "False", "Falsef", "FileExists",
		"FileExistsf", "Greater", "GreaterOrEqual", "GreaterOrEqualf", "Greaterf",
		"HTTPBodyContains", "HTTPBodyContainsf", "HTTPBodyNotContains",
		"HTTPBodyNotContainsf", "HTTPError", "HTTPErrorf", "HTTPRedirect",
		"HTTPRedirectf", "HTTPStatusCode", "HTTPStatusCodef", "HTTPSuccess",
		"HTTPSuccessf", "Implements", "Implementsf", "InDelta", "InDeltaMapValues",
		"InDeltaMapValuesf", "InDeltaSlice", "InDeltaSlicef", "InDeltaf", "InEpsilon",
		"InEpsilonSlice", "InEpsilonSlicef", "InEpsilonf", "IsDecreasing",
		"IsDecreasingf", "IsIncreasing", "IsIncreasingf", "IsNonDecreasing",
		"IsNonDecreasingf", "IsNonIncreasing", "IsNonIncreasingf", "IsType",
		"IsTypef", "JSONEq", "JSONEqf", "Len", "Lenf", "Less", "LessOrEqual",
		"LessOrEqualf", "Lessf", "Negative", "Negativef", "Never", "Neverf", "Nil",
		"Nilf", "NoDirExists", "NoDirExistsf", "NoError", "NoErrorf", "NoFileExists",
		"NoFileExistsf", "NotContains", "NotContainsf", "NotEmpty", "NotEmptyf",
		"NotEqual", "NotEqualValues", "NotEqualValuesf", "NotEqualf", "NotErrorIs",
		"NotErrorIsf", "NotNil", "NotNilf", "NotPanics", "NotPanicsf", "NotRegexp",
		"NotRegexpf", "NotSame", "NotSamef", "NotSubset", "NotSubsetf", "NotZero",
		"NotZerof", "Panics", "PanicsWithError", "PanicsWithErrorf", "PanicsWithValue",
		"PanicsWithValuef", "Panicsf", "Positive", "Positivef", "Regexp", "Regexpf",
		"Same", "Samef", "Subset", "Subsetf", "True", "Truef", "WithinDuration",
		"WithinDurationf", "WithinRange", "WithinRangef", "YAMLEq", "YAMLEqf", "Zero",
		"Zerof",
	}

	err := filepath.Walk(startPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Ext(path) == ".go" && strings.HasSuffix(path, "_test.go") {
				processGoFile(path, functionList)
			}
			return nil
		})

	if err != nil {
		log.Println(err)
	}

	fmt.Println("All test files corrected successfully.")
}

func processGoFile(filename string, functionList []string) {
	code, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return
	}

	correctedCode := transformRequireCalls(string(code), functionList)

	err = ioutil.WriteFile(filename, []byte(correctedCode), 0644)
	if err != nil {
		log.Println(err)
	}
}

func transformRequireCalls(code string, functionList []string) string {
	for _, funcName := range functionList {
		re := regexp.MustCompile(`require\.` + funcName + `\s*\(\s*([^)]+)\.T\(\)\s*,`)
		code = re.ReplaceAllString(code, `${1}.Require().`+funcName+`(`)
	}
	return code
}
