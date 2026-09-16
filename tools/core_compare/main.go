package main

import (
	"bytes"
	"debug/gosym"
	"debug/pe"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"validation_tools/x86asm"
)

type Fn struct {
	Name string
	Body []byte
	File string
	Line int
}

func load(path string) map[string]Fn {
	f, e := pe.Open(path)
	if e != nil {
		panic(e)
	}
	tx := f.Section(".text")
	text, _ := tx.Data()
	base := f.OptionalHeader.(*pe.OptionalHeader64).ImageBase
	result := map[string]Fn{}
	for _, s := range f.Sections {
		b, _ := s.Data()
		at := bytes.Index(b, []byte{0xf1, 0xff, 0xff, 0xff, 0, 0, 1, 8})
		if at < 0 {
			continue
		}
		tab, e := gosym.NewTable(nil, gosym.NewLineTable(b[at:], base+uint64(tx.VirtualAddress)))
		if e != nil {
			panic(e)
		}
		for _, fn := range tab.Funcs {
			if !strings.HasPrefix(fn.Name, "main.") && !strings.HasPrefix(fn.Name, "source2metal/") {
				continue
			}
			a := fn.Entry - base - uint64(tx.VirtualAddress)
			end := fn.End - base - uint64(tx.VirtualAddress)
			code := append([]byte(nil), text[a:end]...)
			for i := 0; i < len(code); {
				in, e := x86asm.Decode(code[i:], 64)
				if e != nil {
					panic(e)
				}
				if in.PCRel > 0 {
					for j := 0; j < in.PCRel; j++ {
						code[i+in.PCRelOff+j] = 0
					}
				}
				i += in.Len
			}
			file, line, _ := tab.PCToLine(fn.Entry)
			result[fn.Name] = Fn{fn.Name, code, file, line}
		}
	}
	return result
}
func main() {
	a, b := load(os.Args[1]), load(os.Args[2])
	same := []string{}
	changed := []string{}
	missing := []string{}
	for n, f := range a {
		v, ok := b[n]
		if !ok {
			missing = append(missing, n)
		} else if bytes.Equal(f.Body, v.Body) {
			same = append(same, n)
		} else {
			changed = append(changed, n)
		}
	}
	out := map[string]any{"same": same, "changed": changed, "missing": missing}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}
