package membershipchainservice

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
)

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
	for mm := range m {
		size += reflect.TypeOf(mm).Size() * 8
	}
	return fmt.Sprintf("%d", size)
}
