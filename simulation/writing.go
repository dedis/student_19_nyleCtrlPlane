package main

import (
	"bufio"
	"os"
	"path/filepath"

	"go.dedis.ch/onet/v3/log"
)

func rmFile(fileStr string) {
	// delete file
	var err = os.Remove(fileStr)
	if err != nil {
		return
	}
	log.Lvl3("==> done deleting file", fileStr)
}

func writeToFile(str, fileStr string) {
	os.MkdirAll(filepath.Dir(fileStr), 0777)
	file, _ := os.OpenFile(fileStr, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	w := bufio.NewWriter(file)
	w.WriteString(str + "\n")
	w.Flush()
	file.Close()
}
