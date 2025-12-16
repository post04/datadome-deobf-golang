package main

import (
	"fmt"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/post04/datadome-deobf-golang/deobf"
)

func main() {
	f, err := os.ReadFile("./source/tags.js")
	if err != nil {
		panic(err)
	}
	start := time.Now()
	deobfCode := deobf.Deobfuscate(string(f), deobf.ScriptTypeTAGS)
	fmt.Println("Time taken:", time.Since(start))

	os.WriteFile("./output/tags.js", []byte(deobfCode), 0666)
}
