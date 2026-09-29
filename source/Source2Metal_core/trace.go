package main

import "strings"

func getSourceTrace(c *Counters, kind SourceKind, base, sourcePath string) *SourceTrace {
	for i := range c.SourceTraces {
		t := &c.SourceTraces[i]
		if t.Kind == kind && strings.EqualFold(t.Base, base) && strings.EqualFold(t.SourcePath, sourcePath) {
			return t
		}
	}
	c.SourceTraces = append(c.SourceTraces, SourceTrace{Kind: kind, Base: base, SourcePath: sourcePath})
	return &c.SourceTraces[len(c.SourceTraces)-1]
}
