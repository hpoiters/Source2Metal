package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var tagRE = regexp.MustCompile(`^\[([A-Za-z0-9_]+)\s+"(.*)"\]\s*$`)
var movePrefixRE = regexp.MustCompile(`^\d+\.(?:\.\.)?(.*)$`)
var nagRE = regexp.MustCompile(`^\$\d+$`)

func parsePGNFile(path string, onGame func(PGNGame) error, progress func(done, total int64)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, _ := f.Stat()
	total := st.Size()
	r := bufio.NewReaderSize(f, 1<<20)
	var tags map[string]string
	var order []string
	var moves strings.Builder
	var done int64
	var lastProgress int64
	movetextStarted := false

	flush := func() error {
		if tags == nil && strings.TrimSpace(moves.String()) == "" {
			return nil
		}
		g := PGNGame{Tags: tags, TagOrder: order, MoveText: moves.String(), Source: path}
		if g.Tags == nil {
			g.Tags = map[string]string{}
		}
		if err := onGame(g); err != nil {
			return err
		}
		tags = nil
		order = nil
		moves.Reset()
		movetextStarted = false
		return nil
	}

	for {
		line, e := r.ReadString('\n')
		done += int64(len(line))
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			if movetextStarted {
				if err := flush(); err != nil {
					return err
				}
			}
			m := tagRE.FindStringSubmatch(trim)
			if len(m) == 3 {
				if tags == nil {
					tags = make(map[string]string)
				}
				if _, ok := tags[m[1]]; !ok {
					order = append(order, m[1])
				}
				tags[m[1]] = unescapeTag(m[2])
			}
		} else if trim != "" {
			movetextStarted = true
			moves.WriteString(line)
		}
		if progress != nil && done-lastProgress >= 1<<20 {
			progress(done, total)
			lastProgress = done
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
	}
	if err := flush(); err != nil {
		return err
	}
	if progress != nil {
		progress(total, total)
	}
	return nil
}

func unescapeTag(s string) string {
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

func stripMainline(text string) (tokens []string, result string, variations, comments int64) {
	var b strings.Builder
	depth := 0
	brace := 0
	semi := false
	for _, r := range text {
		if semi {
			if r == '\n' || r == '\r' {
				semi = false
				if depth == 0 && brace == 0 {
					b.WriteRune(' ')
				}
			}
			continue
		}
		if brace > 0 {
			if r == '}' {
				brace--
				if brace == 0 {
					comments++
				}
			}
			continue
		}
		if r == '{' {
			brace = 1
			continue
		}
		if r == ';' {
			semi = true
			comments++
			continue
		}
		if r == '(' {
			depth++
			if depth == 1 {
				variations++
			}
			continue
		}
		if r == ')' {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth > 0 {
			continue
		}
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}

	for _, raw := range strings.Fields(b.String()) {
		t := strings.TrimSpace(raw)
		if t == "" || nagRE.MatchString(t) {
			continue
		}
		if t == "1-0" || t == "0-1" || t == "1/2-1/2" || t == "*" {
			result = t
			break
		}
		if m := movePrefixRE.FindStringSubmatch(t); len(m) == 2 {
			if m[1] == "" {
				continue
			}
			t = m[1]
		}
		if t == "..." || strings.HasSuffix(t, ".") {
			continue
		}
		for len(t) > 0 && (t[len(t)-1] == '!' || t[len(t)-1] == '?') {
			t = t[:len(t)-1]
		}
		if t == "" || t == "e.p." {
			continue
		}
		tokens = append(tokens, t)
	}
	return
}

func parseElo(tags map[string]string, key string) (int, bool) {
	s := strings.TrimSpace(tags[key])
	if s == "" || s == "?" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func writeMovetext(w io.Writer, san []string, result string) error {
	bw := bufio.NewWriterSize(w, 1<<20)
	for i, m := range san {
		if i%2 == 0 {
			if _, err := fmt.Fprintf(bw, "%d. ", i/2+1); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(bw, m, " "); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(bw, result)
	if err != nil {
		return err
	}
	return bw.Flush()
}
