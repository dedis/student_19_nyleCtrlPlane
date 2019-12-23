package membershipchainservice

import (
	"bufio"
	"os"
)

func writeToFile(str, fileStr string) {
	file, _ := os.OpenFile(fileStr, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)
	w.WriteString(str)
	w.WriteString("\n")
	w.Flush()
	file.Close()
}
