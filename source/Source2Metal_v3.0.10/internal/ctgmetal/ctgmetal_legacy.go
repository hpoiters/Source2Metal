// Ctg2Metal v2.1.0
// Native, read-only ChessBase/Fritz CTG/CTO/CTB -> weighted Metal.pgn builder.
//
// CTG binary layout, hash table and move-code mapping are based on the GPLv3+
// DroidFish CtgBook.java implementation by Peter Osterlund and public reverse
// engineering of the CTG format. This derived program is distributed under
// GNU GPL v3 or later. See LICENSE_GPL-3.0.txt.
package ctgmetal

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	version       = "2.1.0"
	pageSize      = 4096
	defaultMaxPly = 80
	maxAllowedPly = 120

	// Standard Metal selection: intentionally conservative.
	minAnchorGames    = 32
	minDecisive       = 4
	z80               = 0.8416212335729143 // correct one-sided 80% normal quantile
	bestWindow        = 0.035              // decisive posterior probability window
	maxAnchorsPerNode = 2
	maxWeight         = 12

	// v2.1: optional fallback for a small and/or very broad CTG.  It is never
	// entered silently: the interactive user must explicitly choose it after
	// the standard selection produced zero Metal records.  Evidence is allowed
	// to be thinner, but output influence is deliberately capped.
	smallMinAnchorGames    = 8
	smallMinDecisive       = 2
	smallWilsonFloor       = 0.40
	smallBestWindow        = 0.025
	smallMaxAnchorsPerNode = 1
	smallMaxWeight         = 3

	// v1.4.5 lineage: raised from the old 5M diagnostic ceiling.
	visitedLimit  = 50_000_000
	progressWidth = 16
)

type SelectionPolicy struct {
	Name        string
	MinGames    int64
	MinDecisive int64
	WilsonFloor float64
	BestWindow  float64
	MaxAnchors  int
	MaxWeight   int
}

var strictPolicy = SelectionPolicy{
	Name: "Standaard (streng)", MinGames: minAnchorGames, MinDecisive: minDecisive,
	WilsonFloor: 0.50, BestWindow: bestWindow, MaxAnchors: maxAnchorsPerNode, MaxWeight: maxWeight,
}

var smallBroadPolicy = SelectionPolicy{
	Name: "Kleine/Brede-boekmodus (voorzichtig)", MinGames: smallMinAnchorGames, MinDecisive: smallMinDecisive,
	WilsonFloor: smallWilsonFloor, BestWindow: smallBestWindow, MaxAnchors: smallMaxAnchorsPerNode, MaxWeight: smallMaxWeight,
}

const learningInstructions = `Source2Metal / Ctg2Metal - INLEREN IN FRITZ
================================================

Gebruik bij "Leren van database" / "Bijleren uit database":

    Overwinningen        AAN
    Verliespartijen      AAN
    Wit                  UIT
    Zwart                UIT
    Speler               UIT
    Spelernaam           LEEG
    Partijen             ALLE partijen

De PGN gebruikt de spelernaam "Metal", maar normaal is geen spelersfilter nodig.
`

var (
	errBookSkipped = errors.New("boek door gebruiker overgeslagen")
	errRunAborted  = errors.New("run door gebruiker afgebroken")
)

const (
	empty  = 0
	pawn   = 1
	knight = 2
	bishop = 3
	rook   = 4
	queen  = 5
	king   = 6
	white  = 1
	black  = -1
	wk     = 1
	wq     = 2
	bk     = 4
	bq     = 8
)

type Config struct {
	Input     string
	InputSet  bool
	All       bool
	Book      string
	MaxPly    int
	MaxPlySet bool
	Debug     bool
	NoPause   bool
	SelfTest  bool
}

type BookSet struct{ Base, CTG, CTO, CTB string }

type DepthStat struct {
	Nodes, RawMoves, LegalMoves, ChildHits, ChildMisses int64
	DecodeInvalid, DecodeIllegal, Anchors, Records      int64
}

// SelectionDiagnostics counts the FIRST reason why a statistical move
// candidate did not satisfy a policy.  This makes a zero-record result
// understandable instead of presenting it as a generic failure.
type SelectionDiagnostics struct {
	WithStats, NoWDL                                     int64
	BelowMinGames, BelowMinDecisive, NotPositive         int64
	BelowWilson, BaseQualified, OutsideBestWindow        int64
	BeforeAnchorCap, AnchorCapDiscarded, SelectedAnchors int64
	SmallBroadBasePotential                              int64
}

type Report struct {
	BookName                                                                      string
	MaxPly                                                                        int
	CTGBytes, CTOBytes, CTBBytes, PGNBytes                                        int64
	CTGSha, CTOSha, CTBSha, PGNSha                                                string
	Elapsed                                                                       time.Duration
	CTGPages, ScanPositions, ScanMoveEntries                                      int64
	InvalidPages, InvalidRecords, PageCountMismatches                             int64
	PositionsTraversed, LookupAttempts, LookupHits, LookupMisses                  int64
	RawBookMoves, LegalBookMoves, DecodeInvalid, DecodeIllegal                    int64
	TranspositionSkipped, NeutralRecommendations, PawnlessSkipped                 int64
	VisitedCapacityStops, LossOnlyPruned, LossOnlyFollowed                        int64
	StatsChildren, ZeroStatsChildren, PositiveEdgeCandidates, ConfidentCandidates int64
	ChildLookupAttempts, ChildLookupHits, ChildLookupMisses                       int64
	HashIndexProbes, UpperBoundaryProbes                                          int64
	Depth                                                                         [maxAllowedPly + 1]DepthStat
	AnchorNodes, NoAnchorNodes, AnchorSignals, WeightCopies                       int64
	RecordsEmitted, LegalMovesWritten, PGNRecordsVerified                         int64
	MaxPlyReached                                                                 int
	RootLookup                                                                    bool
	RootRawMoves, RootLegalMoves                                                  int64
	Method                                                                        string
	Policy                                                                        SelectionPolicy
	SelectionMode                                                                 string
	SelectionDiag                                                                 SelectionDiagnostics
	StrictAttemptZero                                                             bool
	StrictPositions, StrictStatsChildren, StrictRecords                           int64
	StrictDiag                                                                    SelectionDiagnostics
}

func main() {
	cfg, err := parseArgs()
	if err != nil {
		fatal(cfg, err)
		return
	}
	if cfg.SelfTest {
		if err := selfTest(); err != nil {
			fatal(cfg, err)
			return
		}
		fmt.Println("Ctg2Metal v" + version + " selftest: OK")
		return
	}
	if err := run(cfg); err != nil {
		fatal(cfg, err)
		return
	}
	if !cfg.NoPause && len(os.Args) == 1 {
		pause()
	}
}

func fatal(cfg Config, err error) {
	clearProgress()
	fmt.Fprintln(os.Stderr, "\nFOUT:", err)
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
	if !cfg.NoPause && len(os.Args) == 1 {
		pause()
	}
	os.Exit(2)
}

func pause() {
	fmt.Print("\nDruk op Enter om af te sluiten...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func parseArgs() (Config, error) {
	var c Config
	flag.StringVar(&c.Input, "input", "", "invoermap (standaard: map van deze exe)")
	flag.BoolVar(&c.All, "all", false, "verwerk alle complete CTG-sets")
	flag.StringVar(&c.Book, "book", "", "verwerk alleen deze boeknaam")
	mp := flag.Int("max-ply", defaultMaxPly, "maximale diepte 1..120 ply")
	flag.BoolVar(&c.Debug, "debug", false, "extra foutinformatie")
	flag.BoolVar(&c.NoPause, "no-pause", false, "niet wachten op Enter")
	flag.BoolVar(&c.SelfTest, "selftest", false, "voer interne tests uit")
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "max-ply":
			c.MaxPlySet = true
		case "input":
			c.InputSet = true
		}
	})
	c.MaxPly = *mp
	if c.MaxPly < 1 || c.MaxPly > maxAllowedPly {
		return c, fmt.Errorf("--max-ply moet 1..%d zijn", maxAllowedPly)
	}
	if !c.InputSet {
		exe, err := os.Executable()
		if err != nil {
			return c, err
		}
		c.Input = filepath.Dir(exe)
	}
	if c.All && c.Book != "" {
		return c, errors.New("gebruik --all of --book, niet beide")
	}
	return c, nil
}

func run(cfg Config) error {
	input, err := filepath.Abs(cfg.Input)
	if err != nil {
		return err
	}
	st, err := os.Stat(input)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("invoermap bestaat niet: %s", input)
	}

	// v2.0.0: clean up only the obsolete EMPTY Dutch output directory.
	// A non-empty directory is never touched.
	legacyNote := cleanupEmptyLegacyOutput(input)

	sets, err := findSets(input)
	if err != nil {
		return err
	}
	if len(sets) == 0 {
		return fmt.Errorf("geen complete .ctg + .cto + .ctb set gevonden in %s", input)
	}

	if cfg.Book != "" {
		var f []BookSet
		for _, s := range sets {
			if strings.EqualFold(s.Base, cfg.Book) {
				f = append(f, s)
			}
		}
		if len(f) == 0 {
			return fmt.Errorf("boekset niet gevonden: %s", cfg.Book)
		}
		sets = f
	} else if !cfg.All && len(sets) > 1 {
		chosen, all, err := chooseBooks(sets)
		if err != nil {
			return err
		}
		if !all {
			sets = []BookSet{chosen}
		}
	}

	if !cfg.MaxPlySet {
		n, err := chooseMaxPly(defaultMaxPly)
		if err != nil {
			return err
		}
		cfg.MaxPly = n
	}

	stamp := time.Now().Format("2006-01-02_15-04-05")
	runRoot := filepath.Join(input, "Ctg2Metal_output", stamp)
	if err := os.MkdirAll(runRoot, 0755); err != nil {
		return err
	}
	if err := writeLearningInstructions(runRoot); err != nil {
		return fmt.Errorf("instructiebestand schrijven: %w", err)
	}

	fmt.Printf("\nCtg2Metal v%s native\n", version)
	fmt.Println("Invoer :", input)
	fmt.Println("Uitvoer:", runRoot)
	fmt.Printf("Max ply: %d (%.1f zetten)\n", cfg.MaxPly, float64(cfg.MaxPly)/2)
	fmt.Println("Boeken :", len(sets))
	if legacyNote != "" {
		fmt.Println(legacyNote)
	}

	ok, fail, skipped := 0, 0, 0
	aborted := false
	for _, set := range sets {
		out := filepath.Join(runRoot, safeName(set.Base))
		if err := os.MkdirAll(out, 0755); err != nil {
			return err
		}
		if err := writeLearningInstructions(out); err != nil {
			return fmt.Errorf("instructiebestand schrijven voor %s: %w", set.Base, err)
		}
		fmt.Printf("\n=== %s ===\n", set.Base)
		err := processBook(set, out, cfg)
		switch {
		case err == nil:
			ok++
		case errors.Is(err, errBookSkipped):
			skipped++
			fmt.Printf("OVERGESLAGEN: %s - geen Metal.pgn gemaakt.\n", set.Base)
		case errors.Is(err, errRunAborted):
			aborted = true
			fmt.Println("AFGEBROKEN op verzoek van de gebruiker.")
		default:
			fail++
			_ = os.WriteFile(filepath.Join(out, "Ctg2Metal_FOUT.txt"), []byte(fmt.Sprintf(
				"Ctg2Metal v%s - FOUTRAPPORT\n\nBoek: %s\nMax ply: %d\n\nFout: %v\n",
				version, set.Base, cfg.MaxPly, err)), 0644)
			fmt.Printf("MISLUKT: %s : %v\n", set.Base, err)
		}
		if aborted {
			break
		}
	}
	fmt.Printf("\nKlaar. Geslaagd: %d | overgeslagen: %d | mislukt: %d\n", ok, skipped, fail)
	fmt.Println("Resultaten:", runRoot)
	if ok > 0 {
		fmt.Println()
		fmt.Println("INLEREN IN FRITZ:")
		fmt.Println("  Zet Overwinningen en Verliespartijen AAN; Wit, Zwart en Speler UIT.")
		fmt.Println("  Laat het spelernaamveld LEEG en leer ALLE partijen in.")
		fmt.Println("  Alleen als Fritz een spelerfilter verplicht: gebruik exact de naam: \"Metal\" (hoofdlettergevoelig).")
		fmt.Println("  Zie INSTRUCTIE_INLEREN.txt in de uitvoermap.")
	} else if skipped > 0 || aborted {
		fmt.Println("Geen Metal.pgn gemaakt in deze run. Zie het boekrapport voor de selectie-uitleg.")
	}
	if aborted {
		return nil
	}
	if fail > 0 {
		return fmt.Errorf("%d boek(en) mislukt; zie Ctg2Metal_FOUT.txt", fail)
	}
	return nil
}

func writeLearningInstructions(dir string) error {
	return os.WriteFile(filepath.Join(dir, "INSTRUCTIE_INLEREN.txt"), []byte(learningInstructions), 0644)
}

func cleanupEmptyLegacyOutput(input string) string {
	p := filepath.Join(input, "Ctg2Metal_Uitvoer")
	ents, err := os.ReadDir(p)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		return ""
	}
	if len(ents) != 0 {
		return "Let op: Ctg2Metal_Uitvoer bestaat en is niet leeg; daarom niet aangeraakt."
	}
	if err := os.Remove(p); err == nil {
		return "Oude lege map Ctg2Metal_Uitvoer verwijderd."
	}
	return ""
}

func chooseBooks(sets []BookSet) (BookSet, bool, error) {
	fmt.Println("Meerdere complete CTG-sets gevonden:")
	fmt.Println("  0 = ALLE boeken")
	for i, s := range sets {
		fmt.Printf("  %d = %s\n", i+1, s.Base)
	}
	fmt.Printf("Keuze [0-%d] (Enter = alle): ", len(sets))
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return BookSet{}, true, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > len(sets) {
		return BookSet{}, false, fmt.Errorf("ongeldige boekkeuze: %q", s)
	}
	return sets[n-1], false, nil
}

func chooseMaxPly(def int) (int, error) {
	fmt.Printf("Maximale diepte in ply [1-%d] (Enter = %d): ", maxAllowedPly, def)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > maxAllowedPly {
		return 0, fmt.Errorf("maximale diepte moet 1..%d ply zijn", maxAllowedPly)
	}
	return n, nil
}

func findSets(dir string) ([]BookSet, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type trio struct{ ctg, cto, ctb string }
	m := map[string]*trio{}
	names := map[string]string{}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".ctg" && ext != ".cto" && ext != ".ctb" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		key := strings.ToLower(base)
		if m[key] == nil {
			m[key] = &trio{}
			names[key] = base
		}
		p := filepath.Join(dir, e.Name())
		switch ext {
		case ".ctg":
			m[key].ctg = p
		case ".cto":
			m[key].cto = p
		case ".ctb":
			m[key].ctb = p
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []BookSet
	for _, k := range keys {
		t := m[k]
		if t.ctg != "" && t.cto != "" && t.ctb != "" {
			out = append(out, BookSet{names[k], t.ctg, t.cto, t.ctb})
		}
	}
	return out, nil
}

func parseReportEstimate(path, wantSha string, wantMaxPly int) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var gotSha string
	gotMax := -1
	var positions int64
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "Max diepte:"):
			var n int
			if _, err := fmt.Sscanf(line, "Max diepte: %d ply", &n); err == nil {
				gotMax = n
			}
		case strings.HasPrefix(line, "CTG ") && strings.Contains(line, "SHA-256"):
			parts := strings.Fields(line)
			for i := 0; i+1 < len(parts); i++ {
				if parts[i] == "SHA-256" {
					gotSha = parts[i+1]
					break
				}
			}
		case strings.HasPrefix(line, "Bereikte posities"):
			if i := strings.Index(line, ":"); i >= 0 {
				v := strings.TrimSpace(line[i+1:])
				if n, err := strconv.ParseInt(v, 10, 64); err == nil {
					positions = n
				}
			}
		}
	}
	return positions, positions > 0 && gotMax == wantMaxPly && strings.EqualFold(gotSha, wantSha)
}

func previousTraversalEstimate(out, bookName, ctgSha string, maxPly int) (int64, bool) {
	// out = ...\\Ctg2Metal_output\\timestamp\\book. Search older timestamp folders.
	historyRoot := filepath.Dir(filepath.Dir(out))
	pattern := filepath.Join(historyRoot, "*", safeName(bookName), "Ctg2Metal_report.txt")
	matches, _ := filepath.Glob(pattern)
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	currentRun := filepath.Dir(out)
	for _, p := range matches {
		if filepath.Dir(filepath.Dir(p)) == currentRun {
			continue
		}
		if n, ok := parseReportEstimate(p, ctgSha, maxPly); ok {
			return n, true
		}
	}
	return 0, false
}

func phase3InitialEstimate(out string, set BookSet, r *Report, cfg Config) (int64, string) {
	if n, ok := previousTraversalEstimate(out, set.Base, r.CTGSha, cfg.MaxPly); ok {
		return n, "vorige run"
	}
	// First-run fallback. A CTG scan knows position records and move entries,
	// but not the exact reachable/transposition-aware traversal size. Their sum
	// is used only for an explicitly approximate ETA, never as a percentage.
	n := r.ScanPositions + r.ScanMoveEntries
	if n < r.ScanPositions {
		n = r.ScanPositions
	}
	return n, "preflight"
}

func runTraversalAttempt(set BookSet, out, pgnPath string, book *CtgBook, root Board, rootPD *PositionData, r *Report, cfg Config, policy SelectionPolicy, runStart time.Time) (SelectionDiagnostics, error) {
	f, err := os.Create(pgnPath)
	if err != nil {
		return SelectionDiagnostics{}, err
	}
	bw := bufio.NewWriterSize(f, 1<<20)
	diag := SelectionDiagnostics{}
	phase3Estimate, phase3EstimateSrc := phase3InitialEstimate(out, set, r, cfg)
	b := Builder{
		Book:              book,
		Out:               bw,
		R:                 r,
		Cfg:               cfg,
		Policy:            policy,
		Diag:              &diag,
		Visited:           make(map[uint64]struct{}, 1<<20),
		RunStart:          runStart,
		PhaseStart:        time.Now(),
		Phase3Estimate:    phase3Estimate,
		Phase3EstimateSrc: phase3EstimateSrc,
	}
	err = b.traverse(root, rootPD, 0, nil, nil)
	flushErr := bw.Flush()
	closeErr := f.Close()
	if err != nil {
		return diag, err
	}
	if flushErr != nil {
		return diag, flushErr
	}
	if closeErr != nil {
		return diag, closeErr
	}

	// Phase 3 has no trustworthy denominator. Only now, when traversal really
	// returned, is 100% truthful.
	b.showTraversalDone()
	r.Policy = policy
	r.SelectionMode = policy.Name
	r.SelectionDiag = diag
	fmt.Printf("    Bereikt: %s posities | child-hit/miss %s/%s | legaal %s | records %s\n",
		fmtInt(r.PositionsTraversed), fmtInt(r.ChildLookupHits), fmtInt(r.ChildLookupMisses),
		fmtInt(r.LegalBookMoves), fmtInt(r.RecordsEmitted))
	return diag, nil
}

func explainZeroMetal(bookName string, r *Report, d SelectionDiagnostics) {
	fmt.Println()
	fmt.Println("LET OP - dit CTG-boek is NIET kapot en is correct gelezen.")
	fmt.Printf("De standaard Metal-selectie vond in %s geen voldoende zekere leerankers.\n", bookName)
	fmt.Printf("Ctg2Metal onderzocht %s statistische child-posities en maakte daarom bewust geen lege of twijfelachtige Metal.pgn.\n", fmtInt(r.StatsChildren))
	fmt.Println()
	fmt.Println("Dit kan vooral voorkomen bij een klein en/of zeer breed boek: de beschikbare")
	fmt.Println("partijstatistiek is dan over veel varianten verdeeld. Ctg2Metal hanteert daarom")
	fmt.Println("GEEN hard minimumaantal boekzetten; de hoeveelheid bewijs per afzonderlijke zet")
	fmt.Println("is belangrijker dan de totale grootte van de CTG.")
	fmt.Println()
	fmt.Println("Diagnose standaardselectie (eerste afwijsreden per kandidaat):")
	fmt.Printf("  Geen W/D/L-statistiek       : %s\n", fmtInt(d.NoWDL))
	fmt.Printf("  Minder dan %d partijen       : %s\n", strictPolicy.MinGames, fmtInt(d.BelowMinGames))
	fmt.Printf("  Minder dan %d beslissend W+L : %s\n", strictPolicy.MinDecisive, fmtInt(d.BelowMinDecisive))
	fmt.Printf("  Winst niet groter dan verlies: %s\n", fmtInt(d.NotPositive))
	fmt.Printf("  Wilson80-ondergrens < %.0f%%   : %s\n", strictPolicy.WilsonFloor*100, fmtInt(d.BelowWilson))
	fmt.Printf("  Basisvoorwaarden gehaald    : %s\n", fmtInt(d.BaseQualified))
	fmt.Printf("  Potentieel voorzichtige modus: %s\n", fmtInt(d.SmallBroadBasePotential))
}

func chooseZeroMetalAction(canTrySmall bool) (byte, error) {
	fmt.Println()
	if canTrySmall {
		fmt.Println("K = voorzichtig opnieuw proberen in Kleine/Brede-boekmodus")
		fmt.Println("    (lagere bewijslast, maximaal 1 anker per positie en maximaal gewicht 3)")
	}
	fmt.Println("V = dit boek overslaan; bij batchverwerking doorgaan met het volgende boek")
	fmt.Println("Enter = de hele run afbreken")
	for {
		fmt.Print("Keuze: ")
		s, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" {
			return 'A', nil
		}
		if s == "V" {
			return 'V', nil
		}
		if s == "K" && canTrySmall {
			return 'K', nil
		}
		if canTrySmall {
			fmt.Println("Gebruik K, V of Enter.")
		} else {
			fmt.Println("Gebruik V of Enter; ook de voorzichtige modus heeft hier onvoldoende bewijs.")
		}
	}
}

func processBook(set BookSet, out string, cfg Config) error {
	start := time.Now()
	r := &Report{BookName: set.Base, MaxPly: cfg.MaxPly, Method: "Wilson80 decisive W/L + draw-neutral total score; strict first, Source2Metal conservative small/broad fallback only when diagnostics show potential", Policy: strictPolicy, SelectionMode: strictPolicy.Name}
	var err error
	if r.CTGBytes, err = fileSize(set.CTG); err != nil {
		return err
	}
	if r.CTOBytes, err = fileSize(set.CTO); err != nil {
		return err
	}
	if r.CTBBytes, err = fileSize(set.CTB); err != nil {
		return err
	}

	fmt.Println("1/5 Bestanden controleren + SHA-256...")
	if r.CTGSha, err = sha256Progress(set.CTG, "1/5 CTG", 0.00, 0.33); err != nil {
		return err
	}
	if r.CTOSha, err = sha256Progress(set.CTO, "1/5 CTO", 0.33, 0.66); err != nil {
		return err
	}
	if r.CTBSha, err = sha256Progress(set.CTB, "1/5 CTB", 0.66, 1.00); err != nil {
		return err
	}
	clearProgress()
	fmt.Println("    OK - bronbestanden alleen-lezen geopend.")

	fmt.Println("2/5 Preflight CTG-scan (streaming, read-only)...")
	if err := preflightScan(set.CTG, r); err != nil {
		return err
	}
	clearProgress()
	fmt.Printf("    OK - %s posities | %s boekzetten | %s pagina's\n", fmtInt(r.ScanPositions), fmtInt(r.ScanMoveEntries), fmtInt(r.CTGPages))
	if r.InvalidPages != 0 || r.InvalidRecords != 0 {
		return fmt.Errorf("preflight vond ongeldige CTG-pagina/records; uit veiligheid geen Metal.pgn gemaakt")
	}

	fmt.Println("3/5 CTG-index openen, posities volgen en Metal-ankers selecteren...")
	pgnPath := filepath.Join(out, "Metal.pgn")
	book, err := openCtgBook(set, r)
	if err != nil {
		return err
	}
	defer book.Close()

	root := startBoard()
	r.LookupAttempts++
	rootPD, err := book.lookup(root, false)
	if err != nil {
		return err
	}
	if rootPD == nil {
		r.LookupMisses++
		r.Elapsed = time.Since(start)
		_ = writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r)
		return errors.New("beginstelling niet gevonden via CTG/CTO/CTB-index; parser/index-diagnose vereist")
	}
	r.RootLookup = true
	r.LookupHits++
	r.RootRawMoves = int64(len(rootPD.rawMoves()))
	legal := root.legalMoves()
	for _, rm := range rootPD.rawMoves() {
		if d := rootPD.decodeRaw(rm.Code); d != nil && findExact(legal, d) != nil {
			r.RootLegalMoves++
		}
	}
	if r.RootLegalMoves == 0 {
		r.Elapsed = time.Since(start)
		_ = writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r)
		return fmt.Errorf("beginstelling gevonden maar geen CTG-zet decodeert exact legaal (ruw: %d)", r.RootRawMoves)
	}

	// Snapshot all source/preflight/root information.  If the user explicitly
	// chooses the small/broad fallback, traversal counters start clean rather
	// than being doubled by the second pass.
	preTraversal := *r
	strictDiag, err := runTraversalAttempt(set, out, pgnPath, book, root, rootPD, r, cfg, strictPolicy, start)
	if err != nil {
		return err
	}

	if r.RecordsEmitted == 0 {
		r.StrictAttemptZero = true
		r.StrictPositions = r.PositionsTraversed
		r.StrictStatsChildren = r.StatsChildren
		r.StrictRecords = r.RecordsEmitted
		r.StrictDiag = strictDiag
		r.Elapsed = time.Since(start)
		_ = os.Remove(pgnPath)
		// A zero-record strict pass is always explained in the console. A
		// cautious fallback is allowed only when the strict diagnostics show
		// genuine statistical potential; it is never a hidden threshold change.
		explainZeroMetal(set.Base, r, strictDiag)

		if r.StatsChildren == 0 || r.StatsChildren == r.ZeroStatsChildren {
			fmt.Println("Er is bovendien geen bruikbare W/D/L-statistiek aanwezig; een voorzichtige modus zou hier niets betrouwbaars kunnen toevoegen.")
		}
		canTrySmall := strictDiag.SmallBroadBasePotential > 0 && r.StatsChildren > r.ZeroStatsChildren
		if integratedAutoSkip {
			if !canTrySmall {
				fmt.Println("Source2Metal: voorzichtige fallback NIET gestart - diagnostiek toont onvoldoende statistisch potentieel.")
				_ = writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r)
				return errBookSkipped
			}
			fmt.Println("Source2Metal: strikte selectie gaf 0 records; de expliciete voorzichtige Kleine/Brede-fallback wordt nu geprobeerd.")
		} else {
			action, err := chooseZeroMetalAction(canTrySmall)
			if err != nil {
				return err
			}
			if action == 'A' {
				_ = writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r)
				return errRunAborted
			}
			if action == 'V' {
				_ = writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r)
				return errBookSkipped
			}
		}

		// The conservative fallback was explicitly selected by the standalone user
		// or explicitly announced by Source2Metal integrated mode. Restore the clean
		// pre-traversal report, but preserve the diagnosis from the strict pass.
		nr := preTraversal
		nr.StrictAttemptZero = true
		nr.StrictPositions = r.StrictPositions
		nr.StrictStatsChildren = r.StrictStatsChildren
		nr.StrictRecords = 0
		nr.StrictDiag = strictDiag
		nr.Policy = smallBroadPolicy
		nr.SelectionMode = smallBroadPolicy.Name
		r = &nr
		book.R = r

		fmt.Println()
		fmt.Println("3/5 Kleine/Brede-boekmodus gestart (expliciete voorzichtige fallback)...")
		fmt.Printf("    Voorzichtig: min %d partijen | min %d beslissend | Wilson80 >= %.0f%% | max %d anker/positie | max gewicht %d\n",
			smallBroadPolicy.MinGames, smallBroadPolicy.MinDecisive, smallBroadPolicy.WilsonFloor*100,
			smallBroadPolicy.MaxAnchors, smallBroadPolicy.MaxWeight)
		_, err = runTraversalAttempt(set, out, pgnPath, book, root, rootPD, r, cfg, smallBroadPolicy, start)
		if err != nil {
			return err
		}
		if r.RecordsEmitted == 0 {
			_ = os.Remove(pgnPath)
			r.Elapsed = time.Since(start)
			_ = writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r)
			fmt.Println("Ook de voorzichtige Kleine/Brede-boekmodus vond geen verantwoord Metal-signaal.")
			fmt.Println("Het boek wordt daarom veilig overgeslagen; de originele CTG is niet gewijzigd.")
			return errBookSkipped
		}
		fmt.Printf("    Kleine/Brede-boekmodus geslaagd: %s Metal-records met begrensde invloed.\n", fmtInt(r.RecordsEmitted))
	}

	fmt.Println("4/5 Metal.pgn structureel verifieren...")
	if err := verifyPgnProgress(pgnPath, r); err != nil {
		return err
	}
	clearProgress()
	if r.PGNRecordsVerified != r.RecordsEmitted {
		return fmt.Errorf("PGN-verificatie: verwacht %d records, gevonden %d", r.RecordsEmitted, r.PGNRecordsVerified)
	}
	if r.PGNBytes, err = fileSize(pgnPath); err != nil {
		return err
	}
	if r.PGNSha, err = sha256Progress(pgnPath, "4/5 SHA-256", 0, 1); err != nil {
		return err
	}
	clearProgress()
	fmt.Printf("    OK - %s records geverifieerd.\n", fmtInt(r.PGNRecordsVerified))

	fmt.Println("5/5 Rapport schrijven...")
	r.Elapsed = time.Since(start)
	if err := writeReport(filepath.Join(out, "Ctg2Metal_report.txt"), r); err != nil {
		return err
	}
	fmt.Printf("Metal.pgn: %s records | %s bytes | duur %s\n", fmtInt(r.RecordsEmitted), fmtInt(r.PGNBytes), fmtDuration(r.Elapsed))
	if r.Policy.Name == smallBroadPolicy.Name {
		fmt.Println("Let op: deze Metal.pgn is gemaakt in de voorzichtige Kleine/Brede-boekmodus; zie het rapport.")
	}
	return nil
}

func fileSize(p string) (int64, error) {
	st, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

// ---- Compact live progress --------------------------------------------------

var liveLineLen int

func liveLine(s string) {
	if integratedLive != nil {
		integratedLive(s, false)
		return
	}
	// Standalone Ctg2Metal keeps its historical compact renderer.
	pad := 0
	if liveLineLen > len(s) {
		pad = liveLineLen - len(s)
	}
	fmt.Print("\r", s)
	if pad > 0 {
		fmt.Print(strings.Repeat(" ", pad))
		fmt.Print(strings.Repeat("\b", pad))
	}
	liveLineLen = len(s)
}

func clearProgress() {
	if integratedLive != nil {
		integratedLive("", false)
		return
	}
	if liveLineLen == 0 {
		return
	}
	fmt.Print("\r", strings.Repeat(" ", liveLineLen), "\r")
	liveLineLen = 0
}

func finishProgress(s string) {
	if integratedLive != nil {
		integratedLive(s, true)
		return
	}
	clearProgress()
	fmt.Println(s)
}

func showProgress(label string, frac float64, done, total int64, start time.Time, unit string) {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * progressWidth)
	if filled > progressWidth {
		filled = progressWidth
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", progressWidth-filled)
	elapsed := time.Since(start)
	rate := 0.0
	if elapsed.Seconds() > 0 {
		rate = float64(done) / elapsed.Seconds()
	}
	eta := "--:--"
	if rate > 0 && total > done {
		eta = fmtDuration(time.Duration(float64(total-done) / rate * float64(time.Second)))
	}

	// Compact units: raw exact counts remain in the report; the live view is human-friendly.
	line := fmt.Sprintf("  %-11s [%s] %5.1f%% | %s/%s %s | %s/s | ETA %s",
		label, bar, frac*100, compactCount(done), compactCount(total), unit, compactRate(rate, unit), eta)
	liveLine(line)
}

func fmtInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	start := 0
	if strings.HasPrefix(s, "-") {
		start = 1
	}
	for i := len(s) - 3; i > start; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}

func compactCount(n int64) string {
	a := math.Abs(float64(n))
	sign := ""
	if n < 0 {
		sign = "-"
	}
	switch {
	case a >= 1e9:
		return fmt.Sprintf("%s%.2fG", sign, a/1e9)
	case a >= 1e6:
		return fmt.Sprintf("%s%.2fM", sign, a/1e6)
	case a >= 1e3:
		return fmt.Sprintf("%s%.1fk", sign, a/1e3)
	default:
		return strconv.FormatInt(n, 10)
	}
}

func fmtRate(rate float64) string {
	if rate < 0 {
		rate = 0
	}
	return fmtInt(int64(math.Round(rate)))
}

func compactRate(rate float64, unit string) string {
	if rate < 0 {
		rate = 0
	}
	suffix := ""
	if unit == "bytes" {
		suffix = "B"
	}
	switch {
	case rate >= 1e9:
		return fmt.Sprintf("%.1fG%s", rate/1e9, suffix)
	case rate >= 1e6:
		return fmt.Sprintf("%.1fM%s", rate/1e6, suffix)
	case rate >= 1e3:
		return fmt.Sprintf("%.1fk%s", rate/1e3, suffix)
	default:
		return fmt.Sprintf("%.0f%s", rate, suffix)
	}
}

func fmtDuration(d time.Duration) string {
	if d < 0 {
		return "--:--"
	}
	s := int64(d.Seconds() + 0.5)
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}

func sha256Progress(path, label string, p0, p1 float64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, _ := f.Stat()
	total := st.Size()
	h := sha256.New()
	buf := make([]byte, 1<<20)
	var done int64
	start := time.Now()
	last := time.Time{}
	for {
		n, e := f.Read(buf)
		if n > 0 {
			_, _ = h.Write(buf[:n])
			done += int64(n)
			if time.Since(last) > 250*time.Millisecond {
				frac := 1.0
				if total > 0 {
					frac = float64(done) / float64(total)
				}
				showProgress(label, p0+(p1-p0)*frac, done, total, start, "bytes")
				last = time.Now()
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", e
		}
	}
	showProgress(label, p1, total, total, start, "bytes")
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func preflightScan(path string, r *Report) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.Size() < pageSize || st.Size()%pageSize != 0 {
		return fmt.Errorf("CTG-grootte is geen veelvoud van 4096 bytes: %d", st.Size())
	}
	pages := st.Size() / pageSize
	r.CTGPages = pages - 1
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, pageSize)
	if _, err = f.Seek(pageSize, io.SeekStart); err != nil {
		return err
	}
	start := time.Now()
	last := time.Time{}
	for pg := int64(1); pg < pages; pg++ {
		if _, err = io.ReadFull(f, buf); err != nil {
			return err
		}
		nPos := be16(buf, 0)
		nBytes := be16(buf, 2)
		if nBytes < 4 || nBytes > pageSize {
			r.InvalidPages++
		} else {
			off, count := 4, 0
			for off < nBytes && count < nPos {
				if off+2 > nBytes {
					r.InvalidRecords++
					break
				}
				posLen := int(buf[off] & 0x1f)
				if posLen < 1 || off+posLen >= nBytes {
					r.InvalidRecords++
					break
				}
				moveBytes := int(buf[off+posLen])
				if moveBytes < 1 || moveBytes&1 == 0 {
					r.InvalidRecords++
					break
				}
				recLen := posLen + moveBytes + 33
				if off+recLen > nBytes {
					r.InvalidRecords++
					break
				}
				r.ScanPositions++
				r.ScanMoveEntries += int64((moveBytes - 1) / 2)
				off += recLen
				count++
			}
			if count != nPos {
				r.PageCountMismatches++
			}
		}
		if time.Since(last) > 200*time.Millisecond {
			showProgress("2/5 Preflight", float64(pg)/float64(pages-1), pg, pages-1, start, "pages")
			last = time.Now()
		}
	}
	showProgress("2/5 Preflight", 1, pages-1, pages-1, start, "pages")
	return nil
}

func verifyPgnProgress(path string, r *Report) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, _ := f.Stat()
	total := st.Size()
	rd := bufio.NewReaderSize(f, 1<<20)
	var events, movelines, done int64
	start := time.Now()
	last := time.Time{}
	for {
		line, e := rd.ReadString('\n')
		done += int64(len(line))
		t := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(t, "[Event \"Ctg2Metal") {
			events++
		}
		if t != "" && !strings.HasPrefix(t, "[") && (strings.HasSuffix(t, " 1-0") || strings.HasSuffix(t, " 0-1") || t == "1-0" || t == "0-1") {
			movelines++
		}
		if time.Since(last) > 250*time.Millisecond {
			frac := 1.0
			if total > 0 {
				frac = float64(done) / float64(total)
			}
			showProgress("4/5 Verify", frac, done, total, start, "bytes")
			last = time.Now()
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
	}
	if events < movelines {
		r.PGNRecordsVerified = events
	} else {
		r.PGNRecordsVerified = movelines
	}
	showProgress("4/5 Verify", 1, total, total, start, "bytes")
	return nil
}

func writeReport(path string, r *Report) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Ctg2Metal v%s - rapport\n\n", version)
	fmt.Fprintf(&sb, "Boek: %s\n", r.BookName)
	fmt.Fprintf(&sb, "Max diepte: %d ply (%.1f zetten)\n", r.MaxPly, float64(r.MaxPly)/2)
	fmt.Fprintf(&sb, "Methode: directe CTG/CTO/CTB probing; bronbestanden alleen-lezen; streaming uitvoer.\n")
	fmt.Fprintf(&sb, "Selectie: remises neutraal; W/L-betrouwbaarheid via one-sided 80%% Wilson-ondergrens.\n")
	fmt.Fprintf(&sb, "Selectiemodus: %s\n", r.SelectionMode)
	if r.Policy.Name != "" {
		fmt.Fprintf(&sb, "Drempels: min %d partijen; min %d beslissend; Wilson80 >= %.0f%%; best-window %.1f pp; max %d anker(s)/positie; max gewicht %d.\n",
			r.Policy.MinGames, r.Policy.MinDecisive, r.Policy.WilsonFloor*100, r.Policy.BestWindow*100, r.Policy.MaxAnchors, r.Policy.MaxWeight)
	}
	fmt.Fprintf(&sb, "Pionloze posities: fail-closed overgeslagen (complexere CTG-symmetrie).\n")
	fmt.Fprintf(&sb, "Recommendations/NAG: NIET gebruikt als sterktesignaal.\n")
	fmt.Fprintf(&sb, "Visited limiet: %d posities.\n\n", visitedLimit)

	fmt.Fprintf(&sb, "BRONBESTANDEN\n")
	fmt.Fprintf(&sb, "CTG %d bytes  SHA-256 %s\n", r.CTGBytes, r.CTGSha)
	fmt.Fprintf(&sb, "CTO %d bytes  SHA-256 %s\n", r.CTOBytes, r.CTOSha)
	fmt.Fprintf(&sb, "CTB %d bytes  SHA-256 %s\n\n", r.CTBBytes, r.CTBSha)

	fmt.Fprintf(&sb, "PREFLIGHT-SCAN\n")
	fmt.Fprintf(&sb, "CTG data pages       : %d\n", r.CTGPages)
	fmt.Fprintf(&sb, "Positierecords       : %d\n", r.ScanPositions)
	fmt.Fprintf(&sb, "Move entries         : %d\n", r.ScanMoveEntries)
	fmt.Fprintf(&sb, "Ongeldige pages      : %d\n", r.InvalidPages)
	fmt.Fprintf(&sb, "Ongeldige records    : %d\n", r.InvalidRecords)
	fmt.Fprintf(&sb, "Page-count mismatch  : %d\n\n", r.PageCountMismatches)

	fmt.Fprintf(&sb, "ROOT-DIAGNOSE\n")
	fmt.Fprintf(&sb, "Beginstelling lookup : %v\n", r.RootLookup)
	fmt.Fprintf(&sb, "Ruwe beginzetten     : %d\n", r.RootRawMoves)
	fmt.Fprintf(&sb, "Exact legale beginz. : %d\n\n", r.RootLegalMoves)

	fmt.Fprintf(&sb, "SELECTIE / VALIDATIE\n")
	fmt.Fprintf(&sb, "Bereikte posities    : %d\n", r.PositionsTraversed)
	fmt.Fprintf(&sb, "CTG lookups          : %d\n", r.LookupAttempts)
	fmt.Fprintf(&sb, "Lookup hits          : %d\n", r.LookupHits)
	fmt.Fprintf(&sb, "Lookup misses        : %d\n", r.LookupMisses)
	fmt.Fprintf(&sb, "Child lookup hits    : %d\n", r.ChildLookupHits)
	fmt.Fprintf(&sb, "Child lookup misses  : %d\n", r.ChildLookupMisses)
	fmt.Fprintf(&sb, "CTO hash probes      : %d\n", r.HashIndexProbes)
	fmt.Fprintf(&sb, "CTO boundary probes  : %d\n", r.UpperBoundaryProbes)
	fmt.Fprintf(&sb, "Ruwe boekzetten      : %d\n", r.RawBookMoves)
	fmt.Fprintf(&sb, "Exact legaal         : %d\n", r.LegalBookMoves)
	fmt.Fprintf(&sb, "Decode onbekend      : %d\n", r.DecodeInvalid)
	fmt.Fprintf(&sb, "Decode niet-legaal   : %d\n", r.DecodeIllegal)
	fmt.Fprintf(&sb, "Stat child-posities  : %d\n", r.StatsChildren)
	fmt.Fprintf(&sb, "Zonder W/D/L         : %d\n", r.ZeroStatsChildren)
	fmt.Fprintf(&sb, "Positieve W>L cand.  : %d\n", r.PositiveEdgeCandidates)
	fmt.Fprintf(&sb, "Basisgekwalificeerd   : %d\n", r.ConfidentCandidates)
	fmt.Fprintf(&sb, "Transposities skip   : %d\n", r.TranspositionSkipped)
	fmt.Fprintf(&sb, "Neutrale recom       : %d\n", r.NeutralRecommendations)
	fmt.Fprintf(&sb, "Pawnless skip        : %d\n", r.PawnlessSkipped)
	fmt.Fprintf(&sb, "Loss-only gevolgd    : %d\n", r.LossOnlyFollowed)
	fmt.Fprintf(&sb, "Visited-cap stop     : %d\n", r.VisitedCapacityStops)
	fmt.Fprintf(&sb, "Max ply bereikt      : %d\n\n", r.MaxPlyReached)

	fmt.Fprintf(&sb, "Metal-ankers\n")
	p := r.Policy
	if p.Name == "" {
		p = strictPolicy
	}
	fmt.Fprintf(&sb, "Modus                : %s\n", p.Name)
	fmt.Fprintf(&sb, "Min totaal evidence  : %d games\n", p.MinGames)
	fmt.Fprintf(&sb, "Min beslissend W+L   : %d games\n", p.MinDecisive)
	fmt.Fprintf(&sb, "Ondergrens           : one-sided 80%% Wilson(W/(W+L)) >= %.0f%%\n", p.WilsonFloor*100)
	fmt.Fprintf(&sb, "Relatief venster     : posterior decisive quality binnen %.1f procentpunt van beste GEKWALIFICEERDE kandidaat\n", p.BestWindow*100)
	fmt.Fprintf(&sb, "Max ankers/positie   : %d\n", p.MaxAnchors)
	fmt.Fprintf(&sb, "Max duplicaatgewicht : %d\n", p.MaxWeight)
	fmt.Fprintf(&sb, "Ankerposities        : %d\n", r.AnchorNodes)
	fmt.Fprintf(&sb, "Zonder anker         : %d\n", r.NoAnchorNodes)
	fmt.Fprintf(&sb, "Ankersignalen        : %d\n", r.AnchorSignals)
	fmt.Fprintf(&sb, "Gewichtkopieen       : %d\n\n", r.WeightCopies)

	fmt.Fprintf(&sb, "SELECTIEDIAGNOSE HUIDIGE MODUS (eerste afwijsreden per kandidaat)\n")
	fmt.Fprintf(&sb, "Met statistiek       : %d\n", r.SelectionDiag.WithStats)
	fmt.Fprintf(&sb, "Geen W/D/L           : %d\n", r.SelectionDiag.NoWDL)
	fmt.Fprintf(&sb, "Onder min partijen   : %d\n", r.SelectionDiag.BelowMinGames)
	fmt.Fprintf(&sb, "Onder min beslissend : %d\n", r.SelectionDiag.BelowMinDecisive)
	fmt.Fprintf(&sb, "Winst <= verlies     : %d\n", r.SelectionDiag.NotPositive)
	fmt.Fprintf(&sb, "Onder Wilson-drempel : %d\n", r.SelectionDiag.BelowWilson)
	fmt.Fprintf(&sb, "Basis gekwalificeerd : %d\n", r.SelectionDiag.BaseQualified)
	fmt.Fprintf(&sb, "Buiten best-window   : %d\n", r.SelectionDiag.OutsideBestWindow)
	fmt.Fprintf(&sb, "Voor anker-cap       : %d\n", r.SelectionDiag.BeforeAnchorCap)
	fmt.Fprintf(&sb, "Door anker-cap weg   : %d\n", r.SelectionDiag.AnchorCapDiscarded)
	fmt.Fprintf(&sb, "Geselecteerde ankers : %d\n\n", r.SelectionDiag.SelectedAnchors)

	if r.StrictAttemptZero {
		fmt.Fprintf(&sb, "EERSTE STANDAARDPOGING LEVERDE 0 RECORDS\n")
		fmt.Fprintf(&sb, "Bereikte posities    : %d\n", r.StrictPositions)
		fmt.Fprintf(&sb, "Stat child-posities  : %d\n", r.StrictStatsChildren)
		fmt.Fprintf(&sb, "Geen W/D/L           : %d\n", r.StrictDiag.NoWDL)
		fmt.Fprintf(&sb, "Onder 32 partijen    : %d\n", r.StrictDiag.BelowMinGames)
		fmt.Fprintf(&sb, "Onder 4 beslissend   : %d\n", r.StrictDiag.BelowMinDecisive)
		fmt.Fprintf(&sb, "Winst <= verlies     : %d\n", r.StrictDiag.NotPositive)
		fmt.Fprintf(&sb, "Wilson80 < 50%%       : %d\n", r.StrictDiag.BelowWilson)
		fmt.Fprintf(&sb, "Basis gekwalificeerd : %d\n", r.StrictDiag.BaseQualified)
		fmt.Fprintf(&sb, "Potentieel klein/breed: %d\n\n", r.StrictDiag.SmallBroadBasePotential)
	}

	fmt.Fprintf(&sb, "UITVOER / CONTROLE\n")
	fmt.Fprintf(&sb, "PGN spelernaam       : Metal\n")
	fmt.Fprintf(&sb, "Leerfilter Fritz     : Overwinningen + Verliespartijen AAN; Wit/Zwart/Speler UIT; naamveld leeg; alle partijen\n")
	fmt.Fprintf(&sb, "PGN records          : %d\n", r.RecordsEmitted)
	fmt.Fprintf(&sb, "PGN records verified : %d\n", r.PGNRecordsVerified)
	fmt.Fprintf(&sb, "Legale zetten written: %d\n", r.LegalMovesWritten)
	fmt.Fprintf(&sb, "Metal.pgn bytes      : %d\n", r.PGNBytes)
	fmt.Fprintf(&sb, "Metal.pgn SHA-256    : %s\n", r.PGNSha)
	fmt.Fprintf(&sb, "Duur                  : %s\n", fmtDuration(r.Elapsed))

	fmt.Fprintf(&sb, "\nDIEPTEVERDELING (ply: nodes raw legal childhit childmiss anchors records)\n")
	for ply := 0; ply <= r.MaxPly && ply <= maxAllowedPly; ply++ {
		d := r.Depth[ply]
		if d.Nodes == 0 && d.RawMoves == 0 && d.Records == 0 {
			continue
		}
		fmt.Fprintf(&sb, "%3d: %d %d %d %d %d %d %d\n", ply, d.Nodes, d.RawMoves, d.LegalMoves, d.ChildHits, d.ChildMisses, d.Anchors, d.Records)
	}

	fmt.Fprintf(&sb, "\nOPMERKING\n")
	fmt.Fprintf(&sb, "Dit programma schrijft de originele CTG/CTO/CTB-bestanden nooit.\n")
	fmt.Fprintf(&sb, "Fase 3 toont bewust geen voortgangspercentage: de vooraf gescande CTG-records zijn geen betrouwbare noemer voor de bereikbare transpositieboom.\n")
	fmt.Fprintf(&sb, "Fase 3 toont wel een ETA met ~: die is een raming en wordt waar mogelijk gekalibreerd op een eerdere run van hetzelfde CTG en dezelfde plydiepte.\n")
	fmt.Fprintf(&sb, "Vanaf v2.1 mag alleen een kandidaat die de basisdrempels zelf haalt de lokale best-score bepalen; dunne toevalsstatistiek kan daardoor geen geldige ankers meer blokkeren.\n")
	if r.Policy.Name == smallBroadPolicy.Name {
		fmt.Fprintf(&sb, "Deze uitvoer is na expliciete gebruikerskeuze gemaakt in Kleine/Brede-boekmodus. De lagere bewijslast wordt gecompenseerd door maximaal 1 anker per positie en maximaal gewicht 3.\n")
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// ---- CTG reader ------------------------------------------------------------

type CtgBook struct {
	ctg, cto, ctb *os.File
	lower, upper  int32
	R             *Report
	cache         map[int32][]byte
	cacheOrder    []int32
}

func openCtgBook(set BookSet, r *Report) (*CtgBook, error) {
	g, e := os.Open(set.CTG)
	if e != nil {
		return nil, e
	}
	o, e := os.Open(set.CTO)
	if e != nil {
		_ = g.Close()
		return nil, e
	}
	b, e := os.Open(set.CTB)
	if e != nil {
		_ = g.Close()
		_ = o.Close()
		return nil, e
	}
	hdr := make([]byte, 12)
	if _, e = b.ReadAt(hdr, 0); e != nil {
		_ = g.Close()
		_ = o.Close()
		_ = b.Close()
		return nil, e
	}
	lower := int32(binary.BigEndian.Uint32(hdr[4:8]))
	upper := int32(binary.BigEndian.Uint32(hdr[8:12]))
	if lower < 0 || upper < lower {
		_ = g.Close()
		_ = o.Close()
		_ = b.Close()
		return nil, fmt.Errorf("ongeldige CTB hashgrenzen: %d..%d", lower, upper)
	}
	st, _ := o.Stat()
	if st == nil || st.Size() < 20 {
		_ = g.Close()
		_ = o.Close()
		_ = b.Close()
		return nil, fmt.Errorf("CTO te klein of ongeldig")
	}
	return &CtgBook{g, o, b, lower, upper, r, make(map[int32][]byte), nil}, nil
}

func (b *CtgBook) Close() {
	if b.ctg != nil {
		_ = b.ctg.Close()
	}
	if b.cto != nil {
		_ = b.cto.Close()
	}
	if b.ctb != nil {
		_ = b.ctb.Close()
	}
}

func ctgHashIndices(enc []byte, lower, upper int32) []int32 {
	h := ctgHash(enc)
	ret := make([]int32, 0, 32)
	// Match DroidFish: the first index that reaches or overshoots upper is
	// still probed. CTB upper is a stop threshold, not a hard CTO index cap.
	for n := int32(0); n < 0x7fffffff; n = 2*n + 1 {
		idx := (h & n) + n
		if idx < lower {
			continue
		}
		ret = append(ret, idx)
		if idx >= upper {
			break
		}
	}
	return ret
}

func (b *CtgBook) lookup(orig Board, count bool) (*PositionData, error) {
	if count {
		b.R.LookupAttempts++
	}
	c := canonical(orig)
	if c == nil {
		return nil, nil
	}
	enc := encodePosition(c.Board)
	for _, idx := range ctgHashIndices(enc, b.lower, b.upper) {
		b.R.HashIndexProbes++
		if idx >= b.upper {
			b.R.UpperBoundaryProbes++
		}
		off := int64(16) + 4*int64(idx)
		pbuf := make([]byte, 4)
		if _, e := b.cto.ReadAt(pbuf, off); e != nil {
			return nil, fmt.Errorf("CTO index %d buiten/ongeldig (offset %d): %w", idx, off, e)
		}
		page := int32(binary.BigEndian.Uint32(pbuf))
		if page >= 0 {
			buf, e := b.page(page)
			if e != nil {
				return nil, e
			}
			if pd := findInPage(buf, enc, c); pd != nil {
				return pd, nil
			}
		}
	}
	return nil, nil
}

func (b *CtgBook) page(page int32) ([]byte, error) {
	if x := b.cache[page]; x != nil {
		return x, nil
	}
	off := int64(page+1) * pageSize
	st, _ := b.ctg.Stat()
	if st == nil || off < 0 || off+pageSize > st.Size() {
		return nil, nil
	}
	buf := make([]byte, pageSize)
	if _, e := b.ctg.ReadAt(buf, off); e != nil {
		return nil, e
	}
	if len(b.cache) >= 128 {
		old := b.cacheOrder[0]
		delete(b.cache, old)
		b.cacheOrder = b.cacheOrder[1:]
	}
	b.cache[page] = buf
	b.cacheOrder = append(b.cacheOrder, page)
	return buf, nil
}

func findInPage(buf, enc []byte, c *Canon) *PositionData {
	if buf == nil || len(buf) < 4 {
		return nil
	}
	nPos := be16(buf, 0)
	nBytes := be16(buf, 2)
	if nBytes < 4 || nBytes > len(buf) {
		return nil
	}
	off := 4
	for i := 0; i < nPos && off < nBytes; i++ {
		if off >= len(buf) {
			return nil
		}
		posLen := int(buf[off] & 0x1f)
		if posLen < 1 || off+posLen >= nBytes {
			return nil
		}
		moveBytes := int(buf[off+posLen])
		recLen := posLen + moveBytes + 33
		if moveBytes < 1 || off+recLen > nBytes {
			return nil
		}
		match := len(enc) == posLen && off+len(enc) <= nBytes
		if match {
			for j := range enc {
				if buf[off+j] != enc[j] {
					match = false
					break
				}
			}
		}
		if match {
			x := append([]byte(nil), buf[off:off+recLen]...)
			return newPositionData(x, c)
		}
		off += recLen
	}
	return nil
}

type Canon struct {
	Board                 Board
	MirrorColor, MirrorLR bool
}

func canonical(src Board) *Canon {
	if !src.hasPawns() {
		return nil
	}
	b := src.copy()
	mc, ml := false, false
	if b.Side == black {
		b.mirrorColor()
		mc = true
	}
	if b.Castle == 0 {
		k := b.findKing(white)
		if k >= 0 && k&7 < 4 {
			b.mirrorLR()
			ml = true
		}
	}
	b.EP = b.fixedEP()
	return &Canon{b, mc, ml}
}

type bitWriter struct {
	a      []byte
	length int
}

func (w *bitWriter) bit(v bool) {
	bi := w.length >> 3
	if bi >= len(w.a) {
		w.a = append(w.a, 0)
	}
	sh := 7 - (w.length & 7)
	if v {
		w.a[bi] |= byte(1 << sh)
	}
	w.length++
}

func (w *bitWriter) bits(v, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bit(v&(1<<i) != 0)
	}
}

func (w *bitWriter) pad() int {
	b := w.length & 7
	if b == 0 {
		return 0
	}
	return 8 - b
}

func (w *bitWriter) bytes() []byte {
	return append([]byte(nil), w.a[:(w.length+7)/8]...)
}

func encodePosition(b Board) []byte {
	w := &bitWriter{}
	w.bits(0, 8)
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			p := b.Sq[y*8+x]
			switch p {
			case empty:
				w.bits(0x00, 1)
			case king:
				w.bits(0x20, 6)
			case queen:
				w.bits(0x22, 6)
			case rook:
				w.bits(0x16, 5)
			case bishop:
				w.bits(0x14, 5)
			case knight:
				w.bits(0x12, 5)
			case pawn:
				w.bits(0x06, 3)
			case -king:
				w.bits(0x21, 6)
			case -queen:
				w.bits(0x23, 6)
			case -rook:
				w.bits(0x17, 5)
			case -bishop:
				w.bits(0x15, 5)
			case -knight:
				w.bits(0x13, 5)
			case -pawn:
				w.bits(0x07, 3)
			default:
				panic("onbekend stuk")
			}
		}
	}
	ep, cs := b.EP >= 0, b.Castle != 0
	if !ep && !cs {
		w.bit(false)
	}
	special := 0
	if ep {
		special += 3
	}
	if cs {
		special += 4
	}
	for w.pad() != special {
		w.bit(false)
	}
	if ep {
		w.bits(b.EP&7, 3)
	}
	if cs {
		w.bit(b.Castle&bk != 0)
		w.bit(b.Castle&bq != 0)
		w.bit(b.Castle&wk != 0)
		w.bit(b.Castle&wq != 0)
	}
	if w.length&7 != 0 {
		panic("CTG bit packing fout")
	}
	out := w.bytes()
	h := len(out)
	if ep {
		h |= 0x20
	}
	if cs {
		h |= 0x40
	}
	out[0] = byte(h)
	return out
}

var hashTbl = [64]int32{
	0x3100d2bf, 0x3118e3de, 0x34ab1372, 0x2807a847, 0x1633f566, 0x2143b359, 0x26d56488, 0x3b9e6f59,
	0x37755656, 0x3089ca7b, 0x18e92d85, 0x0cd0e9d8, 0x1a9e3b54, 0x3eaa902f, 0x0d9bfaae, 0x2f32b45b,
	0x31ed6102, 0x3d3c8398, 0x146660e3, 0x0f8d4b76, 0x02c77a5f, 0x146c8799, 0x1c47f51f, 0x249f8f36,
	0x24772043, 0x1fbc1e4d, 0x1e86b3fa, 0x37df36a6, 0x16ed30e4, 0x02c3148e, 0x216e5929, 0x0636b34e,
	0x317f9f56, 0x15f09d70, 0x131026fb, 0x38c784b1, 0x29ac3305, 0x2b485dc5, 0x3c049ddc, 0x35a9fbcd,
	0x31d5373b, 0x2b246799, 0x0a2923d3, 0x08a96e9d, 0x30031a9f, 0x08f525b5, 0x33611c06, 0x2409db98,
	0x0ca4feb2, 0x1000b71e, 0x30566e32, 0x39447d31, 0x194e3752, 0x08233a95, 0x0f38fe36, 0x29c7cd57,
	0x0f7b3a39, 0x328e8a16, 0x1e7d1388, 0x0fba78f5, 0x274c7e7c, 0x1e8be65c, 0x2fa0b0bb, 0x1eb6c371,
}

func ctgHash(a []byte) int32 {
	var h, tmp int32
	for _, bb := range a {
		ch := int32(int8(bb))
		tmp += ((0x0f - (ch & 0x0f)) << 2) + 1
		h += hashTbl[tmp&63]
		tmp += ((0xf0 - (ch & 0xf0)) >> 2) + 1
		h += hashTbl[tmp&63]
	}
	return h
}

func be16(b []byte, o int) int   { return int(binary.BigEndian.Uint16(b[o : o+2])) }
func be24(b []byte, o int) int64 { return int64(b[o])<<16 | int64(b[o+1])<<8 | int64(b[o+2]) }

type RawMove struct{ Code, Flags int }

type PositionData struct {
	Buf                                []byte
	PosLen, MoveBytes                  int
	Canon                              *Canon
	Total, WhiteWins, BlackWins, Draws int64
	Recommendation                     int
}

func newPositionData(b []byte, c *Canon) *PositionData {
	pl := int(b[0] & 0x1f)
	mb := int(b[pl])
	s := pl + mb
	return &PositionData{b, pl, mb, c, be24(b, s), be24(b, s+3), be24(b, s+6), be24(b, s+9), int(b[s+30])}
}

func (p *PositionData) rawMoves() []RawMove {
	n := (p.MoveBytes - 1) / 2
	a := make([]RawMove, 0, n)
	for i := 0; i < n; i++ {
		a = append(a, RawMove{int(p.Buf[p.PosLen+1+2*i]), int(p.Buf[p.PosLen+2+2*i])})
	}
	return a
}

func (p *PositionData) decodeRaw(code int) *Move {
	mi := moveInfo[code]
	if mi == nil {
		return nil
	}
	b := p.Canon.Board
	from := findNth(b, mi.Piece, mi.PieceNo)
	if from < 0 {
		return nil
	}
	x := ((from & 7) + mi.Dx) & 7
	y := ((from >> 3) + mi.Dy) & 7
	to := y*8 + x
	promo := 0
	if abs(b.Sq[from]) == pawn && y == 7 {
		promo = queen
	}
	m := Move{from, to, promo}
	if p.Canon.MirrorLR {
		m = mirrorMoveLR(m)
	}
	if p.Canon.MirrorColor {
		m = mirrorMoveColor(m)
	}
	return &m
}

func findNth(b Board, piece, n int) int {
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			s := y*8 + x
			if b.Sq[s] == piece {
				if n == 0 {
					return s
				}
				n--
			}
		}
	}
	return -1
}

// ---- Selection / traversal -------------------------------------------------

type Stats struct {
	Wins, Draws, Losses, N, Decisive      int64
	Score, DecisivePosterior, WilsonLower float64
}

func statsFromChild(p *PositionData) Stats {
	w, l, d := p.BlackWins, p.WhiteWins, p.Draws
	sum := w + l + d
	n := p.Total
	if n < sum {
		n = sum
	}
	score := 0.0
	if sum > 0 {
		score = (float64(w) + 0.5*float64(d)) / float64(sum)
	}
	dec := w + l
	post := 0.5
	lo := 0.0
	if dec > 0 {
		phat := float64(w) / float64(dec)
		z := z80
		den := 1 + z*z/float64(dec)
		center := phat + z*z/(2*float64(dec))
		adj := z * math.Sqrt(phat*(1-phat)/float64(dec)+z*z/(4*float64(dec)*float64(dec)))
		lo = (center - adj) / den
		post = (float64(w) + 1) / (float64(dec) + 2)
	}
	return Stats{w, d, l, n, dec, score, post, lo}
}

func qualifiesBase(s Stats, p SelectionPolicy) bool {
	return s.N >= p.MinGames && s.Decisive >= p.MinDecisive && s.Wins > s.Losses && s.WilsonLower >= p.WilsonFloor
}

func (d *SelectionDiagnostics) observe(s Stats, p SelectionPolicy) bool {
	d.WithStats++
	if s.Wins+s.Draws+s.Losses == 0 {
		d.NoWDL++
		return false
	}
	if s.N < p.MinGames {
		d.BelowMinGames++
		return false
	}
	if s.Decisive < p.MinDecisive {
		d.BelowMinDecisive++
		return false
	}
	if s.Wins <= s.Losses {
		d.NotPositive++
		return false
	}
	if s.WilsonLower < p.WilsonFloor {
		d.BelowWilson++
		return false
	}
	d.BaseQualified++
	return true
}

func (s Stats) weight(p SelectionPolicy) int {
	// The small/broad fallback uses the same shape, but its MaxWeight is capped
	// at 3.  This intentionally limits the influence of thinner evidence.
	confidenceBase := p.WilsonFloor
	confidenceSpan := 0.20
	confidence := clamp((s.WilsonLower-confidenceBase)/confidenceSpan, 0, 1)
	evidence := clamp(math.Log1p(float64(s.N))/math.Log1p(10000), 0, 1)
	purity := clamp((s.DecisivePosterior-0.5)/0.5, 0, 1)
	v := 1 + int(math.Round(float64(p.MaxWeight-1)*(0.45*confidence+0.35*evidence+0.20*purity)))
	if v < 1 {
		v = 1
	}
	if v > p.MaxWeight {
		v = p.MaxWeight
	}
	return v
}

func clamp(x, a, b float64) float64 {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
}

type Candidate struct {
	Move                              Move
	Flags                             int
	Child                             Board
	ChildPD                           *PositionData
	Stats                             Stats
	HasStats, LossOnly, BaseQualified bool
}

type Builder struct {
	Book              *CtgBook
	Out               *bufio.Writer
	R                 *Report
	Cfg               Config
	Policy            SelectionPolicy
	Diag              *SelectionDiagnostics
	Visited           map[uint64]struct{}
	RunStart          time.Time // start of the complete book processing
	PhaseStart        time.Time // start of phase 3, used only for phase speed
	Phase3Estimate    int64     // estimated total traversed positions; never shown as an exact percentage
	Phase3EstimateSrc string    // "vorige run" or "preflight"; report/debug only
	lastProgress      time.Time
}

func (b *Builder) traverse(board Board, pd *PositionData, ply int, path []Move, sanPath []string) error {
	if ply >= b.Cfg.MaxPly {
		if ply > b.R.MaxPlyReached {
			b.R.MaxPlyReached = ply
		}
		return nil
	}
	if ply > b.R.MaxPlyReached {
		b.R.MaxPlyReached = ply
	}
	k := board.key()
	if _, ok := b.Visited[k]; ok {
		b.R.TranspositionSkipped++
		return nil
	}
	if len(b.Visited) >= visitedLimit {
		b.R.VisitedCapacityStops++
		return nil
	}
	b.Visited[k] = struct{}{}
	b.R.PositionsTraversed++
	if ply <= maxAllowedPly {
		b.R.Depth[ply].Nodes++
	}
	if time.Since(b.lastProgress) > 250*time.Millisecond {
		b.showTraversal()
		b.lastProgress = time.Now()
	}
	if !board.hasPawns() {
		b.R.PawnlessSkipped++
		return nil
	}

	// A child position already found by the parent is passed directly, so it is
	// not looked up a second time on entry to recursion.
	if pd == nil {
		b.R.LookupAttempts++
		var err error
		pd, err = b.Book.lookup(board, false)
		if err != nil {
			return err
		}
		if pd == nil {
			b.R.LookupMisses++
			return nil
		}
		b.R.LookupHits++
	}

	if pd.Recommendation == 0 {
		b.R.NeutralRecommendations++
	}
	legal := board.legalMoves()
	cs := make([]Candidate, 0, len(pd.rawMoves()))
	for _, raw := range pd.rawMoves() {
		b.R.RawBookMoves++
		if ply <= maxAllowedPly {
			b.R.Depth[ply].RawMoves++
		}
		d := pd.decodeRaw(raw.Code)
		if d == nil {
			b.R.DecodeInvalid++
			if ply <= maxAllowedPly {
				b.R.Depth[ply].DecodeInvalid++
			}
			continue
		}
		ex := findExact(legal, d)
		if ex == nil {
			b.R.DecodeIllegal++
			if ply <= maxAllowedPly {
				b.R.Depth[ply].DecodeIllegal++
			}
			continue
		}
		b.R.LegalBookMoves++
		if ply <= maxAllowedPly {
			b.R.Depth[ply].LegalMoves++
		}
		child := board.after(*ex)
		var childPD *PositionData
		if child.hasPawns() {
			b.R.LookupAttempts++
			b.R.ChildLookupAttempts++
			var err error
			childPD, err = b.Book.lookup(child, false)
			if err != nil {
				return err
			}
			if childPD != nil {
				b.R.LookupHits++
				b.R.ChildLookupHits++
				if ply <= maxAllowedPly {
					b.R.Depth[ply].ChildHits++
				}
			} else {
				b.R.LookupMisses++
				b.R.ChildLookupMisses++
				if ply <= maxAllowedPly {
					b.R.Depth[ply].ChildMisses++
				}
			}
		}
		c := Candidate{Move: *ex, Flags: raw.Flags, Child: child, ChildPD: childPD}
		if childPD != nil {
			b.R.StatsChildren++
			c.Stats = statsFromChild(childPD)
			c.HasStats = true
			if c.Stats.Wins+c.Stats.Draws+c.Stats.Losses == 0 {
				b.R.ZeroStatsChildren++
			}
			if c.Stats.Wins > c.Stats.Losses {
				b.R.PositiveEdgeCandidates++
			}
			if c.Stats.Wins == 0 && c.Stats.Draws == 0 && c.Stats.Losses > 0 {
				c.LossOnly = true
			}

			// v2.1 fix: only candidates that already satisfy the base evidence
			// policy may define the local "best" reference.  In older versions a
			// tiny 1-0 sample could set an unreachable best score and accidentally
			// suppress an otherwise valid, well-supported anchor.
			if b.Diag != nil {
				c.BaseQualified = b.Diag.observe(c.Stats, b.Policy)
				if qualifiesBase(c.Stats, smallBroadPolicy) {
					b.Diag.SmallBroadBasePotential++
				}
			} else {
				c.BaseQualified = qualifiesBase(c.Stats, b.Policy)
			}
			if c.BaseQualified {
				b.R.ConfidentCandidates++
			}
		}
		cs = append(cs, c)
	}

	if len(cs) == 0 {
		b.R.NoAnchorNodes++
		return nil
	}

	best := -1.0
	for _, c := range cs {
		if c.BaseQualified && c.Stats.DecisivePosterior > best {
			best = c.Stats.DecisivePosterior
		}
	}
	anchors := make([]Candidate, 0, b.Policy.MaxAnchors)
	if best >= 0 {
		for _, c := range cs {
			if !c.BaseQualified {
				continue
			}
			if c.Stats.DecisivePosterior >= best-b.Policy.BestWindow {
				anchors = append(anchors, c)
			} else if b.Diag != nil {
				b.Diag.OutsideBestWindow++
			}
		}
	}
	if b.Diag != nil {
		b.Diag.BeforeAnchorCap += int64(len(anchors))
	}
	sort.Slice(anchors, func(i, j int) bool {
		if anchors[i].Stats.WilsonLower == anchors[j].Stats.WilsonLower {
			return anchors[i].Stats.N > anchors[j].Stats.N
		}
		return anchors[i].Stats.WilsonLower > anchors[j].Stats.WilsonLower
	})
	if len(anchors) > b.Policy.MaxAnchors {
		if b.Diag != nil {
			b.Diag.AnchorCapDiscarded += int64(len(anchors) - b.Policy.MaxAnchors)
		}
		anchors = anchors[:b.Policy.MaxAnchors]
	}
	if b.Diag != nil {
		b.Diag.SelectedAnchors += int64(len(anchors))
	}
	if len(anchors) == 0 {
		b.R.NoAnchorNodes++
	} else {
		b.R.AnchorNodes++
		if ply <= maxAllowedPly {
			b.R.Depth[ply].Anchors += int64(len(anchors))
		}
	}
	for _, a := range anchors {
		san := board.san(a.Move, legal)
		np := append(path, a.Move)
		ns := append(sanPath, san)
		w := a.Stats.weight(b.Policy)
		before := b.R.RecordsEmitted
		if err := b.emit(np, ns, board.Side, w, a.Stats, ply+1); err != nil {
			return err
		}
		if ply <= maxAllowedPly {
			b.R.Depth[ply].Records += b.R.RecordsEmitted - before
		}
	}

	// Exploration is structural only. Strength statistics never prune the CTG
	// tree: a locally bad edge can still lead to a large valid subtree.
	sort.Slice(cs, func(i, j int) bool {
		ni, nj := int64(-1), int64(-1)
		qi, qj := -1.0, -1.0
		if cs[i].HasStats {
			ni, qi = cs[i].Stats.N, cs[i].Stats.DecisivePosterior
		}
		if cs[j].HasStats {
			nj, qj = cs[j].Stats.N, cs[j].Stats.DecisivePosterior
		}
		if ni == nj {
			return qi > qj
		}
		return ni > nj
	})
	for _, c := range cs {
		if c.ChildPD == nil {
			continue
		}
		if c.LossOnly {
			b.R.LossOnlyFollowed++
		}
		san := board.san(c.Move, legal)
		if err := b.traverse(c.Child, c.ChildPD, ply+1, append(path, c.Move), append(sanPath, san)); err != nil {
			return err
		}
	}
	return nil
}

func pulseBar(elapsed time.Duration) string {
	w := progressWidth
	seg := 4
	maxStart := w - seg
	if maxStart <= 0 {
		return strings.Repeat(">", w)
	}
	step := int(elapsed/(250*time.Millisecond)) % (2 * maxStart)
	if step > maxStart {
		step = 2*maxStart - step
	}
	return strings.Repeat("-", step) + strings.Repeat(">", seg) + strings.Repeat("-", w-step-seg)
}

func (b *Builder) showTraversal() {
	phaseElapsed := time.Since(b.PhaseStart)
	done := b.R.PositionsTraversed
	rate := 0.0
	if phaseElapsed.Seconds() > 0 {
		rate = float64(done) / phaseElapsed.Seconds()
	}

	eta := "berekenen..."
	if phaseElapsed >= 5*time.Second && rate > 0 && b.Phase3Estimate > 0 {
		remaining := b.Phase3Estimate - done
		if remaining > 0 {
			eta = "~" + fmtDuration(time.Duration(float64(remaining)/rate*float64(time.Second)))
		} else {
			// The denominator is deliberately only an estimate. If it has been
			// exceeded, never pretend the phase is already at zero remaining.
			eta = "~bijna klaar"
		}
	}
	line := fmt.Sprintf("  3/5 Analyse [%s] bezig | pos %s | %s/s | ply %d | rec %s | ETA %s",
		pulseBar(phaseElapsed), compactCount(done), compactRate(rate, "pos"), b.R.MaxPlyReached,
		compactCount(b.R.RecordsEmitted), eta)
	liveLine(line)
}

func (b *Builder) showTraversalDone() {
	finishProgress(fmt.Sprintf("  3/5 Analyse [%s] 100%% | pos %s | rec %s | klaar",
		strings.Repeat("#", progressWidth), compactCount(b.R.PositionsTraversed), compactCount(b.R.RecordsEmitted)))
}

func (b *Builder) emit(path []Move, san []string, mover int, weight int, s Stats, anchorPly int) error {
	result := "0-1"
	if mover == white {
		result = "1-0"
	}
	moves := moveText(san, result)
	date := time.Now().Format("2006.01.02")
	for k := 0; k < weight; k++ {
		_, err := fmt.Fprintf(b.Out,
			"[Event \"Ctg2Metal v%s\"]\n[Site \"Local CTG\"]\n[Date \"%s\"]\n[Round \"-\"]\n[White \"Metal\"]\n[Black \"Metal\"]\n[Result \"%s\"]\n[SourceBook \"%s\"]\n[AnchorPly \"%d\"]\n[Evidence \"N=%d W=%d D=%d L=%d Decisive=%d Wilson80=%.4f\"]\n[Weight \"%d\"]\n\n%s\n\n",
			version, date, result, escapeTag(b.R.BookName), anchorPly,
			s.N, s.Wins, s.Draws, s.Losses, s.Decisive, s.WilsonLower, weight, moves)
		if err != nil {
			return err
		}
		b.R.RecordsEmitted++
		b.R.LegalMovesWritten += int64(len(san))
	}
	b.R.AnchorSignals++
	b.R.WeightCopies += int64(weight)
	return nil
}

func moveText(san []string, result string) string {
	var sb strings.Builder
	for i, x := range san {
		if i&1 == 0 {
			fmt.Fprintf(&sb, "%d. ", i/2+1)
		}
		sb.WriteString(x)
		sb.WriteByte(' ')
	}
	sb.WriteString(result)
	return strings.TrimSpace(sb.String())
}

func escapeTag(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "\"", "\\\"")
}

// ---- Chess board -----------------------------------------------------------

type Move struct{ From, To, Promo int }

func (m Move) uci() string {
	s := sqName(m.From) + sqName(m.To)
	if m.Promo != 0 {
		s += strings.ToLower(pieceLetter(m.Promo))
	}
	return s
}

type Board struct {
	Sq               [64]int
	Side, Castle, EP int
}

func startBoard() Board {
	var b Board
	b.Side = white
	b.Castle = wk | wq | bk | bq
	b.EP = -1
	back := []int{rook, knight, bishop, queen, king, bishop, knight, rook}
	for x := 0; x < 8; x++ {
		b.Sq[x] = back[x]
		b.Sq[8+x] = pawn
		b.Sq[48+x] = -pawn
		b.Sq[56+x] = -back[x]
	}
	return b
}

func (b Board) copy() Board { return b }

func (b Board) hasPawns() bool {
	for _, p := range b.Sq {
		if abs(p) == pawn {
			return true
		}
	}
	return false
}

func (b Board) findKing(color int) int {
	k := color * king
	for i, p := range b.Sq {
		if p == k {
			return i
		}
	}
	return -1
}

func (b *Board) mirrorColor() {
	var n [64]int
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			n[(7-y)*8+x] = -b.Sq[y*8+x]
		}
	}
	b.Sq = n
	b.Side = -b.Side
	c := 0
	if b.Castle&wk != 0 {
		c |= bk
	}
	if b.Castle&wq != 0 {
		c |= bq
	}
	if b.Castle&bk != 0 {
		c |= wk
	}
	if b.Castle&bq != 0 {
		c |= wq
	}
	b.Castle = c
	if b.EP >= 0 {
		b.EP = (7-(b.EP>>3))*8 + (b.EP & 7)
	}
}

func (b *Board) mirrorLR() {
	var n [64]int
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			n[y*8+(7-x)] = b.Sq[y*8+x]
		}
	}
	b.Sq = n
	if b.EP >= 0 {
		b.EP = (b.EP & 56) + (7 - (b.EP & 7))
	}
}

func (b Board) fixedEP() int {
	if b.EP < 0 {
		return -1
	}
	for _, m := range b.legalMoves() {
		if m.To == b.EP && abs(b.Sq[m.From]) == pawn {
			return b.EP
		}
	}
	return -1
}

func (b Board) legalMoves() []Move {
	ps := b.pseudoMoves()
	out := make([]Move, 0, len(ps))
	us := b.Side
	for _, m := range ps {
		c := b.afterUnchecked(m)
		if !c.inCheck(us) {
			out = append(out, m)
		}
	}
	return out
}

func (b Board) pseudoMoves() []Move {
	a := make([]Move, 0, 64)
	for s, pc := range b.Sq {
		if pc == 0 || sign(pc) != b.Side {
			continue
		}
		t := abs(pc)
		x, y := s&7, s>>3
		if t == pawn {
			dy := 1
			if b.Side == black {
				dy = -1
			}
			ny := y + dy
			if ny >= 0 && ny < 8 {
				to := ny*8 + x
				if b.Sq[to] == 0 {
					b.addPawn(&a, s, to, ny)
					home := 1
					if b.Side == black {
						home = 6
					}
					if y == home {
						to2 := (y+2*dy)*8 + x
						if b.Sq[to2] == 0 {
							a = append(a, Move{s, to2, 0})
						}
					}
				}
				for _, dx := range []int{-1, 1} {
					nx := x + dx
					if nx < 0 || nx > 7 {
						continue
					}
					cap := ny*8 + nx
					if (b.Sq[cap] != 0 && sign(b.Sq[cap]) == -b.Side) || cap == b.EP {
						b.addPawn(&a, s, cap, ny)
					}
				}
			}
		} else if t == knight {
			for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
				b.addStep(&a, s, x+d[0], y+d[1])
			}
		} else if t == bishop || t == rook || t == queen {
			if t == bishop || t == queen {
				b.ray(&a, s, 1, 1)
				b.ray(&a, s, 1, -1)
				b.ray(&a, s, -1, 1)
				b.ray(&a, s, -1, -1)
			}
			if t == rook || t == queen {
				b.ray(&a, s, 1, 0)
				b.ray(&a, s, -1, 0)
				b.ray(&a, s, 0, 1)
				b.ray(&a, s, 0, -1)
			}
		} else if t == king {
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx != 0 || dy != 0 {
						b.addStep(&a, s, x+dx, y+dy)
					}
				}
			}
			b.addCastles(&a, s)
		}
	}
	return a
}

func (b Board) addPawn(a *[]Move, from, to, rank int) {
	if rank == 0 || rank == 7 {
		for _, p := range []int{queen, rook, bishop, knight} {
			*a = append(*a, Move{from, to, p})
		}
	} else {
		*a = append(*a, Move{from, to, 0})
	}
}

func (b Board) addStep(a *[]Move, from, x, y int) {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return
	}
	to := y*8 + x
	if b.Sq[to] == 0 || sign(b.Sq[to]) == -b.Side {
		*a = append(*a, Move{from, to, 0})
	}
}

func (b Board) ray(a *[]Move, from, dx, dy int) {
	x, y := (from&7)+dx, (from>>3)+dy
	for x >= 0 && x < 8 && y >= 0 && y < 8 {
		to := y*8 + x
		if b.Sq[to] == 0 {
			*a = append(*a, Move{from, to, 0})
		} else {
			if sign(b.Sq[to]) == -b.Side {
				*a = append(*a, Move{from, to, 0})
			}
			break
		}
		x += dx
		y += dy
	}
}

func (b Board) addCastles(a *[]Move, ks int) {
	if b.Side == white && ks == 4 {
		if b.Castle&wk != 0 && b.Sq[5] == 0 && b.Sq[6] == 0 && b.Sq[7] == rook && !b.attacked(4, black) && !b.attacked(5, black) && !b.attacked(6, black) {
			*a = append(*a, Move{4, 6, 0})
		}
		if b.Castle&wq != 0 && b.Sq[3] == 0 && b.Sq[2] == 0 && b.Sq[1] == 0 && b.Sq[0] == rook && !b.attacked(4, black) && !b.attacked(3, black) && !b.attacked(2, black) {
			*a = append(*a, Move{4, 2, 0})
		}
	}
	if b.Side == black && ks == 60 {
		if b.Castle&bk != 0 && b.Sq[61] == 0 && b.Sq[62] == 0 && b.Sq[63] == -rook && !b.attacked(60, white) && !b.attacked(61, white) && !b.attacked(62, white) {
			*a = append(*a, Move{60, 62, 0})
		}
		if b.Castle&bq != 0 && b.Sq[59] == 0 && b.Sq[58] == 0 && b.Sq[57] == 0 && b.Sq[56] == -rook && !b.attacked(60, white) && !b.attacked(59, white) && !b.attacked(58, white) {
			*a = append(*a, Move{60, 58, 0})
		}
	}
}

func (b Board) attacked(s, by int) bool {
	x, y := s&7, s>>3
	pdy := -1
	if by == black {
		pdy = 1
	}
	py := y + pdy
	if py >= 0 && py < 8 {
		for _, dx := range []int{-1, 1} {
			px := x + dx
			if px >= 0 && px < 8 && b.Sq[py*8+px] == by*pawn {
				return true
			}
		}
	}
	for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
		nx, ny := x+d[0], y+d[1]
		if nx >= 0 && nx < 8 && ny >= 0 && ny < 8 && b.Sq[ny*8+nx] == by*knight {
			return true
		}
	}
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for i, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		for nx >= 0 && nx < 8 && ny >= 0 && ny < 8 {
			pc := b.Sq[ny*8+nx]
			if pc != 0 {
				if sign(pc) == by {
					t := abs(pc)
					if t == queen || (i < 4 && t == rook) || (i >= 4 && t == bishop) {
						return true
					}
				}
				break
			}
			nx += d[0]
			ny += d[1]
		}
	}
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < 8 && ny >= 0 && ny < 8 && b.Sq[ny*8+nx] == by*king {
				return true
			}
		}
	}
	return false
}

func (b Board) inCheck(color int) bool {
	k := b.findKing(color)
	return k >= 0 && b.attacked(k, -color)
}

func (b Board) after(m Move) Board { return b.afterUnchecked(m) }

func (b Board) afterUnchecked(m Move) Board {
	c := b
	pc := c.Sq[m.From]
	us := sign(pc)
	capt := c.Sq[m.To]
	if abs(pc) == pawn && m.To == c.EP && capt == 0 {
		capSq := m.To - 8
		if us == black {
			capSq = m.To + 8
		}
		c.Sq[capSq] = 0
	}
	if m.Promo != 0 {
		c.Sq[m.To] = us * m.Promo
	} else {
		c.Sq[m.To] = pc
	}
	c.Sq[m.From] = 0
	if abs(pc) == king && abs((m.To&7)-(m.From&7)) == 2 {
		if m.To > m.From {
			rf := (m.From & 56) + 7
			rt := m.From + 1
			c.Sq[rt] = c.Sq[rf]
			c.Sq[rf] = 0
		} else {
			rf := m.From & 56
			rt := m.From - 1
			c.Sq[rt] = c.Sq[rf]
			c.Sq[rf] = 0
		}
	}
	if pc == king {
		c.Castle &^= wk | wq
	}
	if pc == -king {
		c.Castle &^= bk | bq
	}
	if m.From == 0 || m.To == 0 {
		c.Castle &^= wq
	}
	if m.From == 7 || m.To == 7 {
		c.Castle &^= wk
	}
	if m.From == 56 || m.To == 56 {
		c.Castle &^= bq
	}
	if m.From == 63 || m.To == 63 {
		c.Castle &^= bk
	}
	c.EP = -1
	if abs(pc) == pawn && abs((m.To>>3)-(m.From>>3)) == 2 {
		c.EP = (m.From + m.To) / 2
	}
	c.Side = -c.Side
	return c
}

func (b Board) san(m Move, legal []Move) string {
	pc := b.Sq[m.From]
	t := abs(pc)
	if t == king && abs((m.To&7)-(m.From&7)) == 2 {
		z := "O-O-O"
		if m.To > m.From {
			z = "O-O"
		}
		c := b.after(m)
		if c.inCheck(c.Side) {
			if len(c.legalMoves()) == 0 {
				z += "#"
			} else {
				z += "+"
			}
		}
		return z
	}
	epCap := t == pawn && m.To == b.EP && b.Sq[m.To] == 0
	cap := b.Sq[m.To] != 0 || epCap
	var sb strings.Builder
	if t != pawn {
		sb.WriteString(pieceLetter(t))
		sameFile, sameRank, other := false, false, false
		for _, o := range legal {
			if o.To == m.To && o.From != m.From && abs(b.Sq[o.From]) == t {
				other = true
				if o.From&7 == m.From&7 {
					sameFile = true
				}
				if o.From>>3 == m.From>>3 {
					sameRank = true
				}
			}
		}
		if other {
			if !sameFile {
				sb.WriteByte(byte('a' + (m.From & 7)))
			} else if !sameRank {
				sb.WriteByte(byte('1' + (m.From >> 3)))
			} else {
				sb.WriteString(sqName(m.From))
			}
		}
	} else if cap {
		sb.WriteByte(byte('a' + (m.From & 7)))
	}
	if cap {
		sb.WriteByte('x')
	}
	sb.WriteString(sqName(m.To))
	if m.Promo != 0 {
		sb.WriteByte('=')
		sb.WriteString(pieceLetter(m.Promo))
	}
	c := b.after(m)
	if c.inCheck(c.Side) {
		if len(c.legalMoves()) == 0 {
			sb.WriteByte('#')
		} else {
			sb.WriteByte('+')
		}
	}
	return sb.String()
}

func (b Board) key() uint64 {
	h := uint64(1469598103934665603)
	for _, p := range b.Sq {
		h ^= uint64(p + 7)
		h *= 1099511628211
	}
	h ^= uint64(b.Side + 2)
	h *= 1099511628211
	h ^= uint64(b.Castle)
	h *= 1099511628211
	h ^= uint64(b.EP + 1)
	h *= 1099511628211
	return h
}

func findExact(legal []Move, d *Move) *Move {
	for i := range legal {
		m := legal[i]
		if m.From == d.From && m.To == d.To && m.Promo == d.Promo {
			return &legal[i]
		}
	}
	return nil
}

func mirrorMoveLR(m Move) Move {
	return Move{(m.From & 56) + (7 - (m.From & 7)), (m.To & 56) + (7 - (m.To & 7)), m.Promo}
}

func mirrorMoveColor(m Move) Move {
	return Move{((7 - (m.From >> 3)) << 3) + (m.From & 7), ((7 - (m.To >> 3)) << 3) + (m.To & 7), m.Promo}
}

func pieceLetter(p int) string {
	switch p {
	case knight:
		return "N"
	case bishop:
		return "B"
	case rook:
		return "R"
	case queen:
		return "Q"
	case king:
		return "K"
	}
	return ""
}

func sqName(s int) string { return string([]byte{byte('a' + (s & 7)), byte('1' + (s >> 3))}) }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// ---- CTG move code map (DroidFish CtgBook.java) ----------------------------

type MoveInfo struct{ Piece, PieceNo, Dx, Dy int }

var moveInfo [256]*MoveInfo

func mi(c, p, n, x, y int) { moveInfo[c] = &MoveInfo{p, n, x, y} }

func init() {
	mi(0x00, pawn, 4, +1, +1)
	mi(0x01, knight, 1, -2, -1)
	mi(0x03, queen, 1, +2, 0)
	mi(0x04, pawn, 1, 0, +1)
	mi(0x05, queen, 0, 0, +1)
	mi(0x06, pawn, 3, -1, +1)
	mi(0x08, queen, 1, +4, 0)
	mi(0x09, bishop, 1, +6, +6)
	mi(0x0a, king, 0, 0, -1)
	mi(0x0c, pawn, 0, -1, +1)
	mi(0x0d, bishop, 0, +3, +3)
	mi(0x0e, rook, 1, +3, 0)
	mi(0x0f, knight, 0, -2, -1)
	mi(0x12, bishop, 0, +7, +7)
	mi(0x13, king, 0, 0, +1)
	mi(0x14, pawn, 7, +1, +1)
	mi(0x15, bishop, 0, +5, +5)
	mi(0x18, pawn, 6, 0, +1)
	mi(0x1a, queen, 1, 0, +6)
	mi(0x1b, bishop, 0, -1, +1)
	mi(0x1d, bishop, 1, +7, +7)
	mi(0x21, rook, 1, +7, 0)
	mi(0x22, bishop, 1, -2, +2)
	mi(0x23, queen, 1, +6, +6)
	mi(0x24, pawn, 7, -1, +1)
	mi(0x26, bishop, 0, -7, +7)
	mi(0x27, pawn, 2, -1, +1)
	mi(0x28, queen, 0, +5, +5)
	mi(0x29, queen, 0, +6, 0)
	mi(0x2a, knight, 1, +1, -2)
	mi(0x2d, pawn, 5, +1, +1)
	mi(0x2e, bishop, 0, +1, +1)
	mi(0x2f, queen, 0, +1, 0)
	mi(0x30, knight, 1, -1, -2)
	mi(0x31, queen, 0, +3, 0)
	mi(0x32, bishop, 1, +5, +5)
	mi(0x34, knight, 0, +1, +2)
	mi(0x36, knight, 0, +2, +1)
	mi(0x37, queen, 0, 0, +4)
	mi(0x38, queen, 1, -4, +4)
	mi(0x39, queen, 0, +5, 0)
	mi(0x3a, bishop, 0, +6, +6)
	mi(0x3b, queen, 1, -5, +5)
	mi(0x3c, bishop, 0, -5, +5)
	mi(0x41, queen, 1, +5, +5)
	mi(0x42, queen, 0, -7, +7)
	mi(0x44, king, 0, +1, -1)
	mi(0x45, queen, 0, +3, +3)
	mi(0x4a, pawn, 7, 0, +2)
	mi(0x4b, queen, 0, -5, +5)
	mi(0x4c, knight, 1, +1, +2)
	mi(0x4d, queen, 1, 0, +1)
	mi(0x50, rook, 0, 0, +6)
	mi(0x52, rook, 0, +6, 0)
	mi(0x54, bishop, 1, -1, +1)
	mi(0x55, pawn, 2, 0, +1)
	mi(0x5c, pawn, 6, +1, +1)
	mi(0x5f, pawn, 4, 0, +2)
	mi(0x61, queen, 0, +6, +6)
	mi(0x62, pawn, 1, 0, +2)
	mi(0x63, queen, 1, -7, +7)
	mi(0x66, bishop, 0, -3, +3)
	mi(0x67, king, 0, +1, +1)
	mi(0x69, rook, 1, 0, +7)
	mi(0x6a, bishop, 0, +4, +4)
	mi(0x6b, king, 0, +2, 0)
	mi(0x6e, rook, 0, +5, 0)
	mi(0x6f, queen, 1, +7, +7)
	mi(0x72, bishop, 1, -7, +7)
	mi(0x74, queen, 0, +2, 0)
	mi(0x79, bishop, 1, -6, +6)
	mi(0x7a, rook, 0, 0, +3)
	mi(0x7b, rook, 1, 0, +6)
	mi(0x7c, pawn, 2, +1, +1)
	mi(0x7d, rook, 1, 0, +1)
	mi(0x7e, queen, 0, -3, +3)
	mi(0x7f, rook, 0, +1, 0)
	mi(0x80, queen, 0, -6, +6)
	mi(0x81, rook, 0, 0, +1)
	mi(0x82, pawn, 5, -1, +1)
	mi(0x85, knight, 0, -1, +2)
	mi(0x86, rook, 0, +7, 0)
	mi(0x87, rook, 0, 0, +5)
	mi(0x8a, knight, 0, +1, -2)
	mi(0x8b, pawn, 0, +1, +1)
	mi(0x8c, king, 0, -1, -1)
	mi(0x8e, queen, 1, -2, +2)
	mi(0x8f, queen, 0, +7, 0)
	mi(0x92, queen, 1, +1, +1)
	mi(0x94, queen, 0, 0, +3)
	mi(0x96, pawn, 1, +1, +1)
	mi(0x97, king, 0, -1, 0)
	mi(0x98, rook, 0, +3, 0)
	mi(0x99, rook, 0, 0, +4)
	mi(0x9a, queen, 0, 0, +6)
	mi(0x9b, pawn, 2, 0, +2)
	mi(0x9d, queen, 0, 0, +2)
	mi(0x9f, bishop, 1, -4, +4)
	mi(0xa0, queen, 1, 0, +3)
	mi(0xa2, queen, 0, +2, +2)
	mi(0xa3, pawn, 7, 0, +1)
	mi(0xa5, rook, 1, 0, +5)
	mi(0xa9, rook, 1, +2, 0)
	mi(0xab, queen, 1, -6, +6)
	mi(0xad, rook, 1, +4, 0)
	mi(0xae, queen, 1, +3, +3)
	mi(0xb0, queen, 1, 0, +4)
	mi(0xb1, pawn, 5, 0, +2)
	mi(0xb2, bishop, 0, -6, +6)
	mi(0xb5, rook, 1, +5, 0)
	mi(0xb7, queen, 0, 0, +5)
	mi(0xb9, bishop, 1, +3, +3)
	mi(0xbb, pawn, 4, 0, +1)
	mi(0xbc, queen, 1, +5, 0)
	mi(0xbd, queen, 1, 0, +2)
	mi(0xbe, king, 0, +1, 0)
	mi(0xc1, bishop, 0, +2, +2)
	mi(0xc2, bishop, 1, +2, +2)
	mi(0xc3, bishop, 0, -2, +2)
	mi(0xc4, rook, 1, +1, 0)
	mi(0xc5, rook, 1, 0, +4)
	mi(0xc6, queen, 1, 0, +5)
	mi(0xc7, pawn, 6, -1, +1)
	mi(0xc8, pawn, 6, 0, +2)
	mi(0xc9, queen, 1, 0, +7)
	mi(0xca, bishop, 1, -3, +3)
	mi(0xcb, pawn, 5, 0, +1)
	mi(0xcc, bishop, 1, -5, +5)
	mi(0xcd, rook, 0, +2, 0)
	mi(0xcf, pawn, 3, 0, +1)
	mi(0xd1, pawn, 1, -1, +1)
	mi(0xd2, knight, 1, +2, +1)
	mi(0xd3, knight, 1, -2, +1)
	mi(0xd7, queen, 0, -1, +1)
	mi(0xd8, rook, 1, +6, 0)
	mi(0xd9, queen, 0, -2, +2)
	mi(0xda, knight, 0, -1, -2)
	mi(0xdb, pawn, 0, 0, +2)
	mi(0xde, pawn, 4, -1, +1)
	mi(0xdf, king, 0, -1, +1)
	mi(0xe0, knight, 1, +2, -1)
	mi(0xe1, rook, 0, 0, +7)
	mi(0xe3, rook, 1, 0, +3)
	mi(0xe5, queen, 0, +4, 0)
	mi(0xe6, pawn, 3, 0, +2)
	mi(0xe7, queen, 0, +4, +4)
	mi(0xe8, rook, 0, 0, +2)
	mi(0xe9, knight, 0, +2, -1)
	mi(0xeb, pawn, 3, +1, +1)
	mi(0xec, pawn, 0, 0, +1)
	mi(0xed, queen, 0, +7, +7)
	mi(0xee, queen, 1, -1, +1)
	mi(0xef, rook, 0, +4, 0)
	mi(0xf0, queen, 1, +7, 0)
	mi(0xf1, queen, 0, +1, +1)
	mi(0xf3, knight, 1, -1, +2)
	mi(0xf4, rook, 1, 0, +2)
	mi(0xf5, bishop, 1, +1, +1)
	mi(0xf6, king, 0, -2, 0)
	mi(0xf7, knight, 0, -2, +1)
	mi(0xf8, queen, 1, +1, 0)
	mi(0xf9, queen, 1, 0, +6)
	mi(0xfa, queen, 1, +3, 0)
	mi(0xfb, queen, 1, +2, +2)
	mi(0xfd, queen, 0, 0, +7)
	mi(0xfe, queen, 1, -3, +3)
}

func safeName(s string) string {
	r := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return r.Replace(s)
}

// ---- Selftest: move generator + synthetic CTG lookup + weighted PGN path ----

func selfTest() error {
	b := startBoard()
	if len(b.legalMoves()) != 20 {
		return fmt.Errorf("selftest movegen beginstelling: %d != 20", len(b.legalMoves()))
	}
	e4 := Move{12, 28, 0}
	if findExact(b.legalMoves(), &e4) == nil {
		return errors.New("selftest e2e4 niet legaal")
	}

	dir, err := os.MkdirTemp("", "ctg2metal-selftest-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	set := BookSet{"synthetic", filepath.Join(dir, "synthetic.ctg"), filepath.Join(dir, "synthetic.cto"), filepath.Join(dir, "synthetic.ctb")}

	rootEnc := encodePosition(b)
	child := b.after(e4)
	cc := canonical(child)
	if cc == nil {
		return errors.New("selftest child canonicalisatie gaf nil")
	}
	childEnc := encodePosition(cc.Board)

	rec := func(enc []byte, moveCode *byte, total, ww, bw, dr int64) []byte {
		mb := 1
		if moveCode != nil {
			mb = 3
		}
		x := make([]byte, len(enc)+mb+33)
		copy(x, enc)
		x[len(enc)] = byte(mb)
		if moveCode != nil {
			x[len(enc)+1] = *moveCode
			x[len(enc)+2] = 0
		}
		s := len(enc) + mb
		put24(x, s, total)
		put24(x, s+3, ww)
		put24(x, s+6, bw)
		put24(x, s+9, dr)
		return x
	}

	code := byte(0x5f) // e2-e4 in the DroidFish CTG move table.
	r1 := rec(rootEnc, &code, 100, 30, 40, 30)
	// Child canonical side-to-move is white; raw BLACK wins are mover wins.
	r2 := rec(childEnc, nil, 100, 10, 60, 30)
	page := make([]byte, pageSize)
	binary.BigEndian.PutUint16(page[0:2], 2)
	nbytes := 4 + len(r1) + len(r2)
	binary.BigEndian.PutUint16(page[2:4], uint16(nbytes))
	copy(page[4:], r1)
	copy(page[4+len(r1):], r2)
	ctg := make([]byte, pageSize*2)
	copy(ctg[pageSize:], page)
	if err = os.WriteFile(set.CTG, ctg, 0644); err != nil {
		return err
	}

	// Both encoded positions must hash to index 0 for this tiny synthetic CTO.
	// Force that by writing page 0 at the only CTO slot and CTB threshold 0;
	// lookup probes index 0 first for all hashes.
	cto := make([]byte, 20)
	binary.BigEndian.PutUint32(cto[16:20], 0)
	if err = os.WriteFile(set.CTO, cto, 0644); err != nil {
		return err
	}
	ctb := make([]byte, 12)
	binary.BigEndian.PutUint32(ctb[4:8], 0)
	binary.BigEndian.PutUint32(ctb[8:12], 0)
	if err = os.WriteFile(set.CTB, ctb, 0644); err != nil {
		return err
	}

	rp := &Report{}
	bkf, err := openCtgBook(set, rp)
	if err != nil {
		return err
	}
	defer bkf.Close()
	pd, err := bkf.lookup(b, false)
	if err != nil || pd == nil {
		return fmt.Errorf("selftest root lookup: %v", err)
	}
	rms := pd.rawMoves()
	if len(rms) != 1 {
		return fmt.Errorf("selftest raw moves %d", len(rms))
	}
	dm := pd.decodeRaw(rms[0].Code)
	if dm == nil || dm.uci() != "e2e4" {
		return fmt.Errorf("selftest decode: %v", dm)
	}
	cpd, err := bkf.lookup(child, false)
	if err != nil || cpd == nil {
		return fmt.Errorf("selftest child lookup: %v", err)
	}
	s := statsFromChild(cpd)
	if !qualifiesBase(s, strictPolicy) {
		return fmt.Errorf("selftest anchor-base rejected: W=%d D=%d L=%d lower=%.3f", s.Wins, s.Draws, s.Losses, s.WilsonLower)
	}

	// v2.1 regression: an impressive but under-evidenced candidate may NOT set
	// the local best-window reference and thereby suppress a valid candidate.
	thin := Stats{Wins: 10, Losses: 0, N: 10, Decisive: 10, DecisivePosterior: 11.0 / 12.0, WilsonLower: 0.90}
	robust := Stats{Wins: 40, Losses: 20, Draws: 40, N: 100, Decisive: 60, DecisivePosterior: 41.0 / 62.0, WilsonLower: 0.55}
	if qualifiesBase(thin, strictPolicy) {
		return errors.New("selftest dunne kandidaat kwalificeert onverwacht voor standaardmodus")
	}
	if !qualifiesBase(robust, strictPolicy) {
		return errors.New("selftest robuuste kandidaat kwalificeert niet")
	}
	bestQualified := -1.0
	for _, x := range []Stats{thin, robust} {
		if qualifiesBase(x, strictPolicy) && x.DecisivePosterior > bestQualified {
			bestQualified = x.DecisivePosterior
		}
	}
	if math.Abs(bestQualified-robust.DecisivePosterior) > 1e-12 {
		return errors.New("selftest best-window wordt nog door dunne kandidaat bepaald")
	}

	// v2.1 regression: a thin but still directional sample can be eligible only
	// in the explicitly chosen small/broad mode, whose influence stays capped.
	small := Stats{Wins: 2, Losses: 0, Draws: 6, N: 8, Decisive: 2, DecisivePosterior: 0.75, WilsonLower: 0.70}
	if qualifiesBase(small, strictPolicy) {
		return errors.New("selftest kleine kandidaat kwalificeert ten onrechte voor standaardmodus")
	}
	if !qualifiesBase(small, smallBroadPolicy) {
		return errors.New("selftest kleine kandidaat kwalificeert niet voor Kleine/Brede-boekmodus")
	}
	if w := small.weight(smallBroadPolicy); w < 1 || w > smallBroadPolicy.MaxWeight {
		return fmt.Errorf("selftest Kleine/Brede-gewicht buiten cap: %d", w)
	}

	pgnPath := filepath.Join(dir, "Metal.pgn")
	pf, err := os.Create(pgnPath)
	if err != nil {
		return err
	}
	pw := bufio.NewWriter(pf)
	testReport := &Report{BookName: "synthetic", MaxPly: 4, ScanPositions: 2, Policy: strictPolicy, SelectionMode: strictPolicy.Name}
	now := time.Now()
	testDiag := SelectionDiagnostics{}
	builder := Builder{Book: bkf, Out: pw, R: testReport, Cfg: Config{MaxPly: 4}, Policy: strictPolicy, Diag: &testDiag, Visited: make(map[uint64]struct{}), RunStart: now, PhaseStart: now, Phase3Estimate: 100, Phase3EstimateSrc: "selftest", lastProgress: now}
	if err := builder.traverse(b, pd, 0, nil, nil); err != nil {
		_ = pf.Close()
		return fmt.Errorf("selftest traversal: %w", err)
	}
	if err := pw.Flush(); err != nil {
		_ = pf.Close()
		return err
	}
	if err := pf.Close(); err != nil {
		return err
	}
	if testReport.RecordsEmitted == 0 {
		return errors.New("selftest traversal maakte 0 records")
	}
	data, err := os.ReadFile(pgnPath)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), "1. e4") {
		return errors.New("selftest PGN mist 1. e4")
	}
	if !strings.Contains(string(data), `[White "Metal"]`) || !strings.Contains(string(data), `[Black "Metal"]`) {
		return errors.New("selftest PGN spelernaam is niet exact Metal")
	}
	if strings.Contains(string(data), `[White "METAL"]`) || strings.Contains(string(data), `[Black "METAL"]`) {
		return errors.New("selftest PGN bevat nog de oude spelernaam METAL")
	}

	// v2.1 regression: learning instruction explicitly disables the player filter.
	if !strings.Contains(learningInstructions, "Overwinningen        AAN") || !strings.Contains(learningInstructions, "Verliespartijen      AAN") || !strings.Contains(learningInstructions, "Speler               UIT") || !strings.Contains(learningInstructions, "Spelernaam           LEEG") {
		return errors.New("selftest leerinstructie mist gecorrigeerde Fritz-instellingen")
	}
	if err := writeLearningInstructions(dir); err != nil {
		return fmt.Errorf("selftest leerinstructie schrijven: %w", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "INSTRUCTIE_INLEREN.txt")); err != nil || !strings.Contains(string(data), "Speler") {
		return errors.New("selftest leerinstructiebestand ontbreekt of is ongeldig")
	}

	// v2.1 regression: legacy folder cleanup may ONLY remove an empty folder.
	emptyLegacy := filepath.Join(dir, "Ctg2Metal_Uitvoer")
	if err := os.Mkdir(emptyLegacy, 0755); err != nil {
		return err
	}
	_ = cleanupEmptyLegacyOutput(dir)
	if _, err := os.Stat(emptyLegacy); !os.IsNotExist(err) {
		return errors.New("selftest legacy lege map niet verwijderd")
	}
	if err := os.Mkdir(emptyLegacy, 0755); err != nil {
		return err
	}
	sentinel := filepath.Join(emptyLegacy, "BEWAREN.txt")
	if err := os.WriteFile(sentinel, []byte("niet verwijderen"), 0644); err != nil {
		return err
	}
	_ = cleanupEmptyLegacyOutput(dir)
	if _, err := os.Stat(sentinel); err != nil {
		return errors.New("selftest niet-lege legacy map werd aangeraakt")
	}
	return nil
}

func put24(b []byte, o int, v int64) {
	b[o] = byte(v >> 16)
	b[o+1] = byte(v >> 8)
	b[o+2] = byte(v)
}
