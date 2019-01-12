# jvgores

jvgores embeds files into a go file and provides functions for accessing their contents.

It adds all files recursively under a path specified.
The output file provides functions for accessing the contents as byte slices or strings.

## Installation

`go install github.com\jviksne\jvgores`

## Usage

From command line:

`jvgores -src="C:\Users\someuser\go\src\github.com\jviksne\jvgores\sample\data" -dst="C:\Users\someuser\go\src\github.com\jviksne\jvgores\sample\output.go" -str="*.txt,*.htm?,*.js,*.css" -def=byte -sep=/ -pkg=res`

Go:generate:

`//go:generate jvgores -src="test/data" -dst="test/output.go" -str="*.txt,*.htm?,*.js,*.css" -def=byte -sep=/ -pkg=res`

The flags are:
```
-src="/source/files"
    source path of some directory or a single file, mandatory

-dst="/path/to/some/file.go"
    path to output file, if omitted the contents will be output to console

-pkg="main"
    package name of the output file, defaults to main

-getresstrfn="GetResStr"
    name of the function for retrieving the string contents of a file, defaults to GetResStr, pass "nil" to ommit the function

-mustresstrfn="MustResStr"
    name of the helper function for retrieving the string contents of a file that panics upon the path not found, defaults to MustResStr, pass "nil" to ommit the function

-getresbytesfn="GetResBytes"
    name of the function for retrieving the contents of a file as a byte slice, defaults to GetResBytes, pass "nil" to ommit the function

-mustresbytesfn="MustResBytes"
    name of the helper function for retrieving the contents of a file as a byte slice that panics upon the path not found, defaults to MustResBytes, pass "nil" to ommit the function

-prefix="/some/prefix/"
    some prefix to add to the path that is passed to GetResStr() and GetResBytes() functions for identifying the files

-byte="*.png,*.jp?g"
    a comma separated list of shell file name patterns to identify files that should be accessible as byte slices

-str="*.htm?,*.css,*.js,*.txt"
    a comma separated list of shell file name patterns to identify files that should be available as strings

-def=byte|str|both|skip
    default format in case neither "-byte" nor "-str" patterns are matched: byte => byte (default), str => string, both => put in both, skip => skip the file

-sep="/"
    the path separator to be used within the go program

-silent
    do not print status information upon successful generation of the file

-help
	prints help
```

## Sample output file

```go
// Code generated by `jvgores -src=C:\Users\someuser\go\src\github.com\jviksne\jvgores\sample\data -dst=C:\Users\someuser\go\src\github.com\jviksne\jvgores\sample\output.go -str=*.txt,*.htm?,*.js,*.css -def=byte -sep=/ -pkg=res`; DO NOT EDIT.

package res

var byteFiles = map[string][]byte{
	"zero.png": []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 31, 21, 196, 137, 0, 0, 0, 6, 98, 75, 71, 68, 0, 255, 0, 255, 0, 255, 160, 189, 167, 147, 0, 0, 0, 9, 112, 72, 89, 115, 0, 0, 46, 35, 0, 0, 46, 35, 1, 120, 165, 63, 118, 0, 0, 0, 11, 73, 68, 65, 84, 8, 215, 99, 96, 0, 2, 0, 0, 5, 0, 1, 226, 38, 5, 155, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}}

var strFiles = map[string]string{
	"some-text-file.txt":                  "Some text content.",
	"subdir/some-text-file-in-subdir.txt": "First line.\r\nSecond line.\r\nVarious 'quotes' \"on\" `this` line.\r\n"}

// GetResBytes returns embedded file contents as a byte slice.
// If the file is not found, nil is returned.
func GetResBytes(name string) []byte {
	return byteFiles[name]
}

// GetResStr returns embedded file contents as a string.
// In case no such file exists, an empty string is returned
// with the second argument returned as false.
func GetResStr(name string) (string, bool) {
	s, ok := strFiles[name]
	return s, ok
}
```
