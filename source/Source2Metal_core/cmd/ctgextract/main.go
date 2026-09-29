// Optional diagnostic runner; normal users use the integrated Source2Metal EXE.
package main

import (
	"flag"
	"fmt"
	"os"
	"source2metal/internal/ctgraw"
)

func main() {
	base := flag.String("book", "", "book path without extension")
	out := flag.String("out", "raw.pgn", "output")
	depth := flag.Int("depth", 100, "0=unlimited")
	workers := flag.Int("workers", 0, "0=half of CPUs")
	flag.Parse()
	r, e := ctgraw.BuildGraph(*base+".ctg", *base+".cto", *base+".ctb", *out, *depth, *workers, func(s string, final bool) { fmt.Println(s) })
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", r)
}
