package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func shouldSkipDir(name string) bool {
	n := strings.ToLower(name)
	return n == ".git" || n == "tijdelijk" || n == "temp" ||
		strings.HasSuffix(n, "_output") || n == "!source2metal_output"
}

func scanSources(root string) (Inventory, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, err
	}
	inv := Inventory{Root: root}
	type ctgCandidate struct{ dir, base, ctg string }
	var ctgs []ctgCandidate

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != root && shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		st, err := d.Info()
		if err != nil {
			return err
		}
		switch ext {
		case ".bin":
			inv.Sources = append(inv.Sources, Source{Kind: KindBIN, Path: path, Base: strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())), Bytes: st.Size(), Complete: true})
		case ".pgn":
			low := strings.ToLower(d.Name())
			if strings.HasPrefix(low, "source2metal_raw") || strings.HasPrefix(low, "source2metal_metal") {
				return nil
			}
			inv.Sources = append(inv.Sources, Source{Kind: KindPGN, Path: path, Base: strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())), Bytes: st.Size(), Complete: true})
		case ".ctg":
			ctgs = append(ctgs, ctgCandidate{filepath.Dir(path), strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())), path})
		case ".cbh":
			base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			dir := filepath.Dir(path)
			aux := []string{filepath.Join(dir, base+".cbg"), filepath.Join(dir, base+".cbp"), filepath.Join(dir, base+".cbt")}
			complete := true
			for _, a := range aux {
				if _, e := os.Stat(a); e != nil {
					complete = false
				}
			}
			inv.Sources = append(inv.Sources, Source{Kind: KindCBH, Path: path, Base: base, Bytes: st.Size(), Complete: complete, Aux: aux})
		case ".2cbh":
			base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			dir := filepath.Dir(path)
			aux := []string{filepath.Join(dir, base+".2cbg"), filepath.Join(dir, base+".2lid")}
			complete := true
			for _, a := range aux {
				if _, e := os.Stat(a); e != nil {
					complete = false
				}
			}
			inv.Sources = append(inv.Sources, Source{Kind: Kind2CBH, Path: path, Base: base, Bytes: st.Size(), Complete: complete, Aux: aux})
		}
		return nil
	})
	if err != nil {
		return Inventory{}, err
	}

	for _, c := range ctgs {
		cto := filepath.Join(c.dir, c.base+".cto")
		ctb := filepath.Join(c.dir, c.base+".ctb")
		_, e1 := os.Stat(cto)
		_, e2 := os.Stat(ctb)
		complete := e1 == nil && e2 == nil
		st, _ := os.Stat(c.ctg)
		s := Source{Kind: KindCTG, Path: c.ctg, Base: c.base, Complete: complete, Aux: []string{cto, ctb}}
		if st != nil {
			s.Bytes = st.Size()
		}
		inv.Sources = append(inv.Sources, s)
	}

	sort.Slice(inv.Sources, func(i, j int) bool {
		if inv.Sources[i].Kind != inv.Sources[j].Kind {
			return inv.Sources[i].Kind < inv.Sources[j].Kind
		}
		return strings.ToLower(inv.Sources[i].Path) < strings.ToLower(inv.Sources[j].Path)
	})
	return inv, nil
}

func printInventory(inv Inventory) {
	fmt.Println("\n" + U("BRONINVENTARIS"))
	fmt.Println("Root:", inv.Root)
	if len(inv.Sources) == 0 {
		fmt.Println("  " + U("Geen ondersteunde bronbestanden gevonden."))
		return
	}
	for i, s := range inv.Sources {
		status := ""
		if !s.Complete {
			switch s.Kind {
			case KindCTG:
				status = " [ONVOLLEDIGE CTG-SET]"
			case KindCBH:
				status = " [ONVOLLEDIGE CBH-SET]"
			case Kind2CBH:
				status = " [ONVOLLEDIGE 2CBH-SET]"
			}
		}
		fmt.Printf("  %3d. %-4s %s%s\n", i+1, s.Kind, s.Path, status)
	}
}
