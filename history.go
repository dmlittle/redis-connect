package main

import (
	"fmt"
	"os"
	"strings"
)

func loadHistory() {
	if f, err := os.Open(historyPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
}

func appendHistory(cmds []string) {
	cloneCmds := make([]string, len(cmds))
	copy(cloneCmds, cmds)

	line.AppendHistory(strings.Join(cloneCmds, " "))
}

func saveHistory() {
	if f, err := os.Create(historyPath); err != nil {
		fmt.Printf("Error writing history file: %s", err.Error())
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
