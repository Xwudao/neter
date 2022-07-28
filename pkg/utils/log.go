package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func CheckErrWithStatus(err error) {
	if err != nil {
		Error(err)
		os.Exit(0)
	}
}
func LoadFiles(dir string, filter func(filename string) bool) (filenames []string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		filename := filepath.Join(dir, file.Name())
		if file.IsDir() {
			filenames = append(filenames, LoadFiles(filename, filter)...)
		} else {
			if filter(filename) {
				filenames = append(filenames, filename)
			}
		}
	}
	return
}

func InStrArr(arr []string, aim string) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i] == aim {
			return true
		}
	}
	return false
}
func CheckExist(p string) bool {
	_, err := os.Stat(p)
	if err != nil {
		return false
	}
	return true
}

func CurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

func Error(err error) {
	if err != nil {
		log.SetPrefix("[ERROR]")
		log.Println(err.Error())
	}
}
func Info(s string) {
	log.SetPrefix("")
	log.Println(s)
}
