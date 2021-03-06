package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func IsNeeded(Name string, IsFile bool, extractFile string) bool {
	jsonFile, err := os.Open(extractFile) // Open the installer that defines what to extract
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println(err)
	}

	type Inner struct {
		Key2 []string `json:"files"`
		Key3 []string `json:"directories"`
	}
	type Outmost struct {
		Key Inner `json:"extract"`
	}
	var cont Outmost
	json.Unmarshal([]byte(byteValue), &cont)

	parts := strings.Split(Name, "/")
	if IsFile {
		for i, elm := range cont.Key.Key2 {
			match, err := filepath.Match(elm, strings.TrimPrefix(Name, parts[0]))
			if err != nil {
				panic(err)
			}
			if match {
				return true
			} else {
				if i == len(cont.Key.Key2)-1 {
					return IsNeeded(Name, false, extractFile)
				}
			}
		}
	} else {
		for i, elm := range cont.Key.Key3 {
			if strings.Contains(strings.TrimPrefix(Name, parts[0]), elm) {
				return true
			} else {
				if i == len(cont.Key.Key3)-1 {
					return false
				}
			}
		}
	}
	return false
}

func Unzip(source, destination string, extractFile string) error {
	archive, err := zip.OpenReader(source) // Open the downloaded update
	if err != nil {
		return err
	}
	defer archive.Close()

Loop:
	for _, file := range archive.Reader.File {
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()

		if extractFile != "false" {
			IsNeededRes := IsNeeded(file.Name, true, extractFile)

			if !IsNeededRes {
				continue Loop
			}
		}

		parts := strings.Split(file.Name, "/")
		fileName := strings.TrimPrefix(file.Name, parts[0]+"/")
		path := filepath.Join(destination, fileName)
		log.Println(path)
		// Remove file if it already exists; no problem if it doesn't; other cases can error out below
		_ = os.Remove(path)
		// Create a directory at path, including parents
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
		// If file is _supposed_ to be a directory, we're done
		if file.FileInfo().IsDir() {
			continue
		}
		// otherwise, remove that directory (_not_ including parents)
		err = os.Remove(path)
		if err != nil {
			return err
		}
		// and create the actual file.  This ensures that the parent directories exist!
		// An archive may have a single file with a nested path, rather than a file for each parent dir
		writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer writer.Close()
		// log.Printf(file.Name)
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	destination := flag.String("dest", "./", "Path where to extract the update")
	updateFile := flag.String("zip", "./temp/update.zip", "Path to the zip file containing the update")
	extractFile := flag.String("extractfile", "false", "Path to the JSON file containing what to extract, false to extract everything")

	runNPM := flag.Bool("npm", false, "Run `npm install` after extracting")

	flag.Parse()

	fmt.Println("unzip:", *updateFile)

	log.Printf("Starting unzipping...")
	err := Unzip(*updateFile, *destination, *extractFile)
	if err != nil {
		log.Fatal(err)
	}

	if *runNPM {
		log.Printf("npm installing...")
		cmd := exec.Command("npm", "install")
		cmd.Dir = "./"

		out, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s", out)
	}
}
