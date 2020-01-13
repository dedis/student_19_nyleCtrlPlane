package membershipchainservice

import (
	"bufio"
	"fmt"
	"os"
	"reflect"

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
	file, _ := os.OpenFile(fileStr, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)
	w.WriteString(str)
	w.WriteString("\n")
	w.Flush()
	file.Close()
}

func getMemoryUsage(m map[string]map[string]float64) string {
	size := reflect.TypeOf(m).Size()
	for _, x := range m {
		size += reflect.TypeOf(x).Size()
		for _, y := range x {
			size += reflect.TypeOf(y).Size()
		}
	}
	return fmt.Sprintf("%d", size)
}
