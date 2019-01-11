/*
MIT License

Copyright (c) 2019 Janis Viksne

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

/*
go install github.com\jviksne\jvgores

jvgores -src="C:\Users\someuser\go\src\github.com\jviksne\jvgores\sample\data" -dst="C:\Users\someuser\go\src\github.com\jviksne\jvgores\sample\output.go" -str="*.txt,*.htm?,*.js,*.css" -def=byte -sep=/ -pkg=res
*/

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

type FileVars struct {
	Cmd           string // Command used to generate the file
	PackageName   string
	ByteFiles     string
	StrFiles      string
	GetResBytesFn string
	GetResStrFn   string
}

var (
	fileVars       FileVars
	getResStrFn    string
	getResBytesFn  string
	walkDirMode    bool
	src            string
	dst            string
	pathPrefix     string
	bytePatterns   []string
	strPatterns    []string
	defFormat      string
	byteValues     []string
	strValues      []string
	pathSep        string
	isFirstByteVal bool = true
	isFirstStrVal  bool = true
	silent         bool = false
)

const (
	fileTemplText = `// Code generated by ` + "`%%.Cmd%%`" + `; DO NOT EDIT.
	
package %%.PackageName%%

var byteFiles = map[string][]byte{%%.ByteFiles%%}

var strFiles = map[string]string{%%.StrFiles%%}%%.GetResBytesFn%%%%.GetResStrFn%%
`
	getResBytesFnTemplText = `

// %%.%% returns embedded file contents as a byte slice.
// If the file is not found, nil is returned.
func %%.%%(name string) []byte {
	return byteFiles[name]
}`

	getResStrFnTemplText = `
	
// %%.%% returns embedded file contents as a string.
// In case no such file exists, an empty string is returned
// with the second argument returned as false.
func %%.%%(name string) (string, bool) {
	s, ok := strFiles[name]
	return s, ok
}`
)

func main() {

	var (
		bytePatternList string
		strPatternList  string
	)

	fileVars.Cmd = strings.Join(os.Args, " ")

	flag.StringVar(&src, "src", "", "source path of some directory or a single file, mandatory")

	flag.StringVar(&dst, "dst", "", "path to output file, if omitted the contents will be output to console")

	flag.StringVar(&fileVars.PackageName, "pkg", "main", "package name to be used in the output file, defaults to main")

	flag.StringVar(&getResStrFn, "getresstrfn", "GetResStr", "name of the function for retrieving the string contents of a file, defaults to GetResStr")

	flag.StringVar(&getResBytesFn, "getresbytesfn", "GetResBytes", "name of the function for retrieving the contents of a file as a byte slice, defaults to GetResBytes")

	flag.StringVar(&pathPrefix, "prefix", "", "some prefix to add to the path that is passed to GetResStr() and GetResBytes() functions for identifying the files")

	flag.StringVar(&bytePatternList, "byte", "", "a comma separated list of shell file name patterns to identify files that should be accessible as byte slices")

	flag.StringVar(&strPatternList, "str", "", "a comma separated list of shell file name patterns to identify files that should be available as strings")

	flag.StringVar(&defFormat, "def", "", `default format in case neither "-byte" nor "-str" patterns are matched: byte => byte (default), str => string, both => put in both, skip => skip the file`)

	flag.StringVar(&pathSep, "sep", "", `the path separator to be used within the go program`)

	flag.BoolVar(&silent, "silent", false, `do not print status information upon successful generation of the file`)

	flag.Parse()

	if src == "" {
		log.Fatal("Please specify the source directory or file by passing -src=\"some_path\" argument!")
	}

	if dst == "" { // If no output file is specified, force silent mode to print only the contents of the file
		silent = true
	}

	switch defFormat {
	case "byte":
	case "str":
	case "skip":
	case "":
		defFormat = "byte"
	default:
		log.Fatalf("Bad value specified for argument def: \"%s\".\nAllowed values are \"byte\", \"str\", \"both\", \"skip\".", defFormat)
	}

	if fileVars.PackageName == "" {
		fileVars.PackageName = "main"
	}

	if getResStrFn == "" {
		getResStrFn = "GetResStr"
	} else if getResStrFn == "nil" {
		getResStrFn = ""
	}

	if getResBytesFn == "" {
		getResBytesFn = "GetResBytes"
	} else if getResBytesFn == "nil" {
		getResBytesFn = ""
	}

	if bytePatternList != "" {
		bytePatterns = strings.Split(bytePatternList, ",")
	}

	if strPatternList != "" {
		strPatterns = strings.Split(strPatternList, ",")
	}

	if pathSep != "" && pathSep == string(filepath.Separator) {
		pathSep = "" // do not replace the path separator if it already matches the current OS path separator
	}

	byteValues = make([]string, 0, 50)

	strValues = make([]string, 0, 50)

	fileT, err := template.New("file").Delims("%%", "%%").Parse(fileTemplText)
	if err != nil {
		log.Fatalf("Error parsing file template: %s", err.Error())
	}

	byteFnT, err := template.New("getresbytes").Delims("%%", "%%").Parse(getResBytesFnTemplText)
	if err != nil {
		log.Fatalf("Error parsing GetResBytes() function template: %s", err.Error())
	}

	strFnT, err := template.New("getresstr").Delims("%%", "%%").Parse(getResStrFnTemplText)
	if err != nil {
		log.Fatalf("Error parsing GetResStr() template: %s", err.Error())
	}

	info, err := os.Stat(src)
	if err != nil {
		log.Fatalf("Error reading source directory or file \"%s\": %s", src, err.Error())
	}

	if info.IsDir() {

		walkDirMode = true

		// append "/" to the source directory to strip it off when
		// creating relativePath in walkFn
		if len(src) > 0 && src[len(src)-1] != os.PathSeparator {
			src = src + string(os.PathSeparator)
		}

		err = filepath.Walk(src, walkFn)
		checkErr(err)

	} else {
		walkDirMode = true
		err = walkFn(src, info, nil)
		checkErr(err)
	}

	fileVars.ByteFiles = strings.Join(byteValues, "")
	fileVars.StrFiles = strings.Join(strValues, "")

	if getResBytesFn != "" {
		var tpl bytes.Buffer
		err = byteFnT.Execute(&tpl, getResBytesFn)
		if err != nil {
			log.Fatalf("Error executing GetResBytes() function template: ", err.Error())
		}
		fileVars.GetResBytesFn = tpl.String()
	}

	if getResStrFn != "" {
		var tpl bytes.Buffer
		err = strFnT.Execute(&tpl, getResStrFn)
		if err != nil {
			log.Fatalf("Error executing GetResStr() function template: ", err.Error())
		}
		fileVars.GetResStrFn = tpl.String()
	}

	// If no destination file is specified, print it to console
	if dst == "" {
		err = fileT.Execute(os.Stdout, fileVars)
	} else {

		f, err := os.Create(dst)
		if err != nil {
			log.Fatalf("Error opening destination file \"%s\" for writing: %s", dst, err.Error())
		}

		defer f.Close()

		err = fileT.Execute(f, fileVars)

		f.Sync()

	}

	if !silent {
		fmt.Println("File successfully created: " + dst)
	}

}

func walkFn(path string, info os.FileInfo, err error) error {

	var (
		isByte       bool = false
		isStr        bool = false
		relativePath string
	)

	// Skip the root dir without printing any information
	if info.IsDir() && path == src {
		return nil
	}

	if walkDirMode {
		if strings.HasPrefix(path, src) {
			relativePath = pathPrefix + path[len(src):]
		} else {
			relativePath = pathPrefix + path
		}
	} else {
		relativePath = pathPrefix + info.Name()
	}

	if pathSep != "" {
		relativePath = strings.Replace(relativePath, string(filepath.Separator), pathSep, -1)
	}

	if !silent {
		fmt.Printf("Processing \"%s\": ", relativePath)
	}

	// If some error occurred print a warning and proceed
	if err != nil {
		if silent {
			if dst != "" { // If not printing to console, then output a warning
				fmt.Println(fmt.Sprintf("File or directory \"%s\" skipped due to an error: %s", relativePath, err.Error()))
			}
		} else {
			fmt.Println(fmt.Sprintf("skipped due to an error: %s", err.Error()))
		}
		return nil
	}

	// Skip directories
	if info.IsDir() {
		if !silent {
			fmt.Println("directory")
		}
		return nil
	}

	if bytePatterns != nil {

		for _, pattern := range bytePatterns {
			matched, err := filepath.Match(pattern, info.Name())

			if err != nil {
				log.Fatalf("Error parsing byte file pattern \"%s\": %s", pattern, err.Error())
			}

			if matched {
				isByte = true
				break
			}
		}

	}

	// It can be both made available for byte and string format
	if strPatterns != nil {

		for _, pattern := range strPatterns {
			matched, err := filepath.Match(pattern, info.Name())

			if err != nil {
				log.Fatalf("Error parsing string file pattern \"%s\": %s", pattern, err.Error())
			}

			if matched {
				isStr = true
				break
			}
		}

	}

	if !isByte && !isStr {
		switch defFormat {

		case "byte":
			isByte = true
			break

		case "str":
			isStr = true
			break

		case "both":
			isByte = true
			isStr = true

		}
	}

	if !silent {
		if !isByte && !isStr {
			fmt.Println("skipped (not matching neither byte nor string patterns)")
		} else if isByte && isStr {
			fmt.Println("added to bytes and strings")
		} else if isByte {
			fmt.Println("added to bytes")
		} else if isStr {
			fmt.Println("added to strings")
		}
	}

	data, err := ioutil.ReadFile(path)

	// If some error occurred print a warning and proceed
	if err != nil {
		if silent {
			if dst != "" { // If not printing to console, then output a warning
				fmt.Println(fmt.Sprintf("File \"%s\" skipped due to an error: %s", relativePath, err.Error()))
			}
		} else {
			fmt.Println(fmt.Sprintf("skipped due to an error: %s", err.Error()))
		}
		return nil
	}

	if isByte {

		if isFirstByteVal {
			isFirstByteVal = false
			byteValues = append(byteValues, "\n")
		} else {
			byteValues = append(byteValues, ",\n")
		}

		byteValues = append(byteValues,
			"\t"+strconv.Quote(relativePath)+": []byte{")

		for i, v := range data {
			if i > 0 {
				byteValues = append(byteValues, fmt.Sprintf(", %d", v))
			} else {
				byteValues = append(byteValues, fmt.Sprintf("%d", v))
			}
		}

		byteValues = append(byteValues, "}")
	}

	if isStr {
		if isFirstStrVal {
			isFirstStrVal = false
			strValues = append(strValues, "\n")
		} else {
			strValues = append(strValues, ",\n")
		}
		strValues = append(strValues, "\t"+strconv.Quote(relativePath)+": "+strconv.Quote(string(data)))
	}

	return nil
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
