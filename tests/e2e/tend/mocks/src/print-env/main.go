package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	env := os.Environ()
	sort.Strings(env)
	for _, e := range env {
		// Only print vars we care about for test stability
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "XDG_") {
			fmt.Println(e)
		}
	}
}
