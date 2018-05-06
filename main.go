package main

import (
	"bufio"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var replacer *strings.Replacer
var tf_file_ext = "*.tf"
var var_prefix = "var."
var dst_file = "./variables.tf"
var varTemplate = template.Must(template.New("var_file").Parse(`{{range .}}
variable "{{ . }}" {
	description  = ""
}
 {{end}}
`))

type TerraformVars struct {
	Variables []string
}

func init() {
	replacer = strings.NewReplacer(":", ".",
		"]", "",
		"}", "",
		"{", "",
		"\"", "",
		")", "",
		"(", "",
		"[", "",
		",", "",
		"var.", "",
		" ", "",
	)
}

func checkError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func containsElement(slice []string, value string) bool {
	if len(slice) == 0 {
		return false
	}
	for _, s := range slice {
		if value == s {
			return true
		}
	}
	return false
}

func getAllFiles(ext string) ([]string, error) {
	dir, err := os.Getwd()
	checkError(err)
	var files []string
	log.Infof("Finding files in %q directory", dir)
	files, err = filepath.Glob(tf_file_ext)
	checkError(err)

	if len(files) == 0 {
		log.Infof("No files with .tf extensions found in %q", dir)
		os.Exit(0)
	}
	return files, nil
}

func (t *TerraformVars) matchVarPref(row, var_prefix string) {
	if strings.Contains(row, var_prefix) {
		pattern := regexp.MustCompile(`var.([a-z?_]+)`)
		match := pattern.FindAllStringSubmatch(row, 1)
		if len(match) != 0 {
			res := replacer.Replace(match[0][0])
			if !containsElement(t.Variables, res) {
				t.Variables = append(t.Variables, res)
			}
		}
	}
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

func main() {
	if fileExists(dst_file) {
		log.Warnf("File %q already exists, please remove it or it will be overridden", dst_file)
	}

	tf_files, err := getAllFiles(tf_file_ext)
	checkError(err)
	var wg sync.WaitGroup
	messages := make(chan string)
	wg.Add(len(tf_files))
	t := &TerraformVars{}

	for _, file := range tf_files {
		go func(file string) {
			defer wg.Done()
			fileHandle, _ := os.Open(file)
			defer fileHandle.Close()
			fileScanner := bufio.NewScanner(fileHandle)
			for fileScanner.Scan() {
				messages <- fileScanner.Text()
			}
		}(file)
	}
	go func() {
		for text := range messages {
			t.matchVarPref(text, var_prefix)
		}
	}()
	wg.Wait()
	f, err := os.Create(dst_file)
	checkError(err)

	err = varTemplate.Execute(f, t.Variables)
	checkError(err)

}
