package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseTBCheckOutput(t *testing.T) {
	ok, bad := parseTBCheckOutput("KQvKR.rtbw: OK!\nKBNPvKPP.rtbw: FAIL!\n")
	if len(ok) != 1 || ok[0] != "KQvKR.rtbw" {
		t.Fatalf("ok=%v", ok)
	}
	if len(bad) != 1 || bad[0] != "KBNPvKPP.rtbw" {
		t.Fatalf("bad=%v", bad)
	}
}
func TestMakeBatches(t *testing.T) {
	f := make([]item, 65)
	b := makeBatches(f, 16)
	if len(b) != 5 || len(b[0]) != 16 || len(b[1]) != 16 || len(b[2]) != 16 || len(b[3]) != 16 || len(b[4]) != 1 {
		t.Fatalf("batch sizes are wrong: %v", []int{len(b[0]), len(b[1]), len(b[2]), len(b[3]), len(b[4])})
	}
}
func TestProgressBarUsesEighthCells(t *testing.T) {
	// 1/8 of one 24-cell bar is about 0.5208%. At 0.53% the first
	// physical partial-cell should therefore be visible without inventing
	// progress below the display's actual cell resolution.
	s := progressBarFloat(0.53, 24)
	if s == "[░░░░░░░░░░░░░░░░░░░░░░░░]" {
		t.Fatal("expected first physical 1/8-cell to be visible")
	}
}

func TestMakeBatchesBySize(t *testing.T) {
	f := []item{{"a", 6}, {"b", 6}, {"c", 6}, {"d", 1}}
	b := makeBatchesBySize(f, 8, 10)
	if len(b) != 3 || len(b[0]) != 1 || len(b[1]) != 1 || len(b[2]) != 2 {
		t.Fatalf("unexpected size batches: %#v", b)
	}
}

func TestProgressLineNeverWrapsOrFeeds(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 136, 160, 200, 240} {
		line := formatProgressLine(width, 1, 3022, 1024*1024*1024, 0, 16*1024*1024*1024*1024, time.Now().Add(-time.Minute), 1, 1000, 8, 0, true)
		if strings.ContainsAny(line, "\r\n") {
			t.Fatalf("width %d: live line contains CR/LF: %q", width, line)
		}
		if runeLen(line) > width-1 {
			t.Fatalf("width %d: live line too wide: %d > %d: %q", width, runeLen(line), width-1, line)
		}
	}
}

func TestAutomaticSelfTestUsesOnlyOnboardFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}
	dir := t.TempDir()
	// A real tablebase-looking file is present only as a sentinel. The self-test
	// must leave it untouched and must not need it as source material.
	name := "KQvKR.rtbw"
	original := bytes.Repeat([]byte{0x5a}, 4096)
	if err := os.WriteFile(filepath.Join(dir, name), original, 0600); err != nil {
		t.Fatal(err)
	}
	// Simulate a stale self-test remnant from a previous crash.
	if err := os.WriteFile(filepath.Join(dir, intentionalTestFile), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}

	helper := filepath.Join(dir, "fake_tbcheck.sh")
	script := "#!/bin/sh\nprintf '%s: FAIL!\\n' \"$1\"\n"
	if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	r := runAutomaticIntentionalTest(helper, dir)
	if !r.ExpectedFailDetected {
		t.Fatalf("expected intentional FAIL, got %+v output=%q", r, r.Output)
	}
	after, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("real tablebase sentinel was modified")
	}
	if _, err := os.Stat(filepath.Join(dir, intentionalTestFile)); !os.IsNotExist(err) {
		t.Fatalf("temporary onboard self-test file was not removed: %v", err)
	}
}

func TestOnboardFixtureHasChecksumShape(t *testing.T) {
	b := onboardIntentionalFailFixture()
	if len(b) != 80 {
		t.Fatalf("fixture size=%d, want 80", len(b))
	}
	if len(b)&0x3f != 0x10 {
		t.Fatalf("fixture does not have Ronald tbcheck checksum-bearing size shape: %d", len(b))
	}
}

func TestProgressLineV6NeverWrapsOrFeeds(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 136, 160, 200, 240} {
		line := formatProgressLineV6(width, 12, 3022, 1024*1024*1024, 512*1024*1024, 16*1024*1024*1024*1024, time.Now().Add(-time.Minute), 0, 1, 1000, 8, 0, true)
		if strings.ContainsAny(line, "\r\n") {
			t.Fatalf("width %d: live line contains CR/LF: %q", width, line)
		}
		if runeLen(line) > width-1 {
			t.Fatalf("width %d: live line too wide: %d > %d: %q", width, runeLen(line), width-1, line)
		}
	}
}

func TestPagefileGateEnterContinuesAndAAborts(t *testing.T) {
	if !pagefileSafetyGate(bufio.NewReader(strings.NewReader("\n")), "nl") {
		t.Fatal("Enter should continue")
	}
	if pagefileSafetyGate(bufio.NewReader(strings.NewReader("a\n")), "nl") {
		t.Fatal("A should abort")
	}
}

func TestPagefileWarningPresentInAllLanguages(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		if len(pagefileWarningLines(lang)) < 3 {
			t.Fatalf("%s: missing pagefile warning", lang)
		}
		if !strings.Contains(strings.ToLower(pagefileChoicePrompt(lang)), "a") {
			t.Fatalf("%s: abort choice not visible", lang)
		}
	}
}

func TestSelfTestReassuresOwnFilesUntouched(t *testing.T) {
	lines := intentionalTestScreenLines("nl", intentionalTestResult{Present: true, ExpectedFailDetected: true})
	joined := strings.Join(lines, "\n")
	want := "Deze TEST-FAIL veranderde niets aan uw eigen bestanden."
	if !strings.Contains(joined, want) {
		t.Fatalf("missing reassurance line %q in %q", want, joined)
	}
}

func TestReportSavedDesktopMessageDutch(t *testing.T) {
	desktop, err := desktopPath()
	if err != nil {
		t.Skipf("desktop path unavailable: %v", err)
	}
	p := filepath.Join(desktop, "SyzygyCheck_Result_20990101_0000.txt")
	lines := reportSavedScreenLines("nl", p)
	if len(lines) != 2 {
		t.Fatalf("lines=%v", lines)
	}
	if lines[0] != "Resultaatrapport opgeslagen op uw Windows-bureaublad (Desktop)." {
		t.Fatalf("unexpected location message: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "Volledig pad: ") || !strings.Contains(lines[1], p) {
		t.Fatalf("unexpected path message: %q", lines[1])
	}
}

func TestStaticWrapNeverExceedsWidth(t *testing.T) {
	text := "Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man. AL-interface, voortgangsweergave, zelftest, checkpointing en rapportage zijn daaromheen gebouwd."
	for _, width := range []int{20, 40, 60, 80, 120} {
		for _, line := range wrapStaticText(text, width) {
			if runeLen(line) > width {
				t.Fatalf("width %d: %d: %q", width, runeLen(line), line)
			}
		}
	}
}

func TestReportDutchLocalizedAndCompact(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "Desktop"), 0755); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 10, 13, 35, 0, 0, time.Local)
	end := start.Add(2 * time.Minute)
	files := []item{{name: "A.rtbw", size: 100}, {name: "B.rtbw", size: 100}, {name: "Bad.rtbw", size: 100}}
	scan := realScanResult{
		Started: start, Ended: end, CompletedFiles: 3, CompletedBytes: 300, AverageReadBps: 98.1e6,
		Failures: []string{"Bad.rtbw"}, Errors: map[string]string{}, Workers: 10,
		EngineMessages: []string{"tbcheck origin: temporary verified helper", "A.rtbw: OK!\nB.rtbw: OK!\nBad.rtbw: FAIL!"},
	}
	testResult := intentionalTestResult{Present: true, ExpectedFailDetected: true}
	path, err := writeReportV6("nl", "/egtb", files, 300, scan, testResult)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"Taal:", "Duur van deze run: 02:00    Gemiddelde leessnelheid: 98.1 MB/s", "Bestanden gevonden: 3", "Werkers (threads): 10", "RESULTAAT (ECHTE SYZYGY-BESTANDEN): FOUT", "CHECKSUMFOUTEN:", "- Bad.rtbw", "Deze TEST-FAIL veranderde niets aan uw eigen bestanden."} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in report:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"Language:", "RESULT (REAL SYZYGY FILES)", ": OK!", "ENGINE / CHECKPOINT MESSAGES"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("unexpected %q in compact localized report:\n%s", unwanted, text)
		}
	}
}

func TestReportFilenameUsesMinutesAndCollisionSuffix(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 9, 10, 13, 37, 33, 0, time.Local)
	p1, err := writeReportData(dir, when, []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	p2, err := writeReportData(dir, when, []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p1) != "SyzygyCheck_Result_20260910_1337.txt" {
		t.Fatalf("first name=%q", filepath.Base(p1))
	}
	if filepath.Base(p2) != "SyzygyCheck_Result_20260910_1337_2.txt" {
		t.Fatalf("second name=%q", filepath.Base(p2))
	}
}

func TestLanguageEscapeLabelAlwaysHasRecognisableRoute(t *testing.T) {
	cases := map[string][]string{
		"en": {"Change language", "更改语言", "Изменить язык"},
		"zh": {"更改语言", "Change language", "Изменить язык"},
		"ru": {"Изменить язык", "Change language", "更改语言"},
		"nl": {"Taal wijzigen", "Change language", "更改语言", "Изменить язык"},
		"de": {"Sprache ändern", "Change language", "更改语言", "Изменить язык"},
		"fr": {"Changer de langue", "Change language", "更改语言", "Изменить язык"},
		"es": {"Cambiar idioma", "Change language", "更改语言", "Изменить язык"},
	}
	for lang, wants := range cases {
		got := languageEscapeLabel(lang, texts[lang].change)
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Fatalf("lang %s: %q missing %q", lang, got, want)
			}
		}
		if runeLen("  2 = "+got) > 79 {
			t.Fatalf("lang %s escape menu line too long for an 80-column console: %d cells: %q", lang, runeLen("  2 = "+got), got)
		}
	}
}

func TestProgressBlockV11NeverWrapsOrFeeds(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		for _, width := range []int{20, 40, 60, 80, 100, 136, 160, 200, 240} {
			lines := formatProgressBlockV11(lang, width, 843, 1511, 4*1024*1024*1024*1024, 3*1024*1024*1024*1024, 16*1024*1024*1024*1024, time.Now().Add(-2*time.Hour), 0, 2, true)
			for i, line := range lines {
				if strings.ContainsAny(line, "\r\n") {
					t.Fatalf("%s width %d line %d contains CR/LF: %q", lang, width, i+1, line)
				}
				if runeLen(line) > width-1 {
					t.Fatalf("%s width %d line %d too wide: %d > %d: %q", lang, width, i+1, runeLen(line), width-1, line)
				}
			}
		}
	}
}

func TestProgressBlockV11DutchLayout(t *testing.T) {
	lines := formatProgressBlockV11("nl", 100, 24, 57, 4*1024*1024*1024, 4*1024*1024*1024, 10*1024*1024*1024, time.Now().Add(-time.Hour), 0, 1, false)
	if !strings.Contains(lines[0], "Bezig met bestand: 24/57") {
		t.Fatalf("unexpected file line: %q", lines[0])
	}
	if !strings.Contains(lines[0], "Leessnelheid:") || !strings.Contains(lines[0], "MB/s") {
		t.Fatalf("read-speed missing from file line: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "/ [") {
		t.Fatalf("unexpected bar line: %q", lines[1])
	}
	if !strings.Contains(lines[2], "Resttijd:") {
		t.Fatalf("remaining-time label missing: %q", lines[2])
	}
	if strings.Contains(lines[2], "ETA") {
		t.Fatalf("ETA jargon should not appear: %q", lines[2])
	}
	// V9F: the two right-hand labels must start in exactly the same column.
	speedCol := strings.Index(lines[0], "Leessnelheid:")
	restCol := strings.Index(lines[2], "Resttijd:")
	if speedCol < 0 || restCol < 0 || speedCol != restCol {
		t.Fatalf("right columns are not aligned: speed=%d rest=%d\n%q\n%q", speedCol, restCol, lines[0], lines[2])
	}
}

func TestAverageReadRateMatchesDisplayBasis(t *testing.T) {
	got := averageReadRate(1_200_000_000, 200_000_000, 10*time.Second)
	if got != 100_000_000 {
		t.Fatalf("got %.0f want 100000000", got)
	}
	if v := formatReadSpeedValue(got); v != "100.0 MB/s" {
		t.Fatalf("unexpected formatted speed: %q", v)
	}
}

func TestEstimateCurrentBatchIndex(t *testing.T) {
	b := []item{{name: "a", size: 100}, {name: "b", size: 200}, {name: "c", size: 300}}
	cases := []struct {
		bytes int64
		want  int
	}{{0, 0}, {99, 0}, {100, 1}, {299, 1}, {300, 2}, {599, 2}, {600, 2}}
	for _, c := range cases {
		if got := estimateCurrentBatchIndex(b, c.bytes); got != c.want {
			t.Fatalf("bytes %d: got %d want %d", c.bytes, got, c.want)
		}
	}
}

func TestRemainingV11UsesHoursWithoutETA(t *testing.T) {
	got := formatRemainingV11("nl", 78*time.Minute, false)
	if got != "Resttijd: 1:18 h" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "ETA") {
		t.Fatal("ETA must not be shown")
	}
}

func TestRecoveryGuideFilenameAllLanguages(t *testing.T) {
	want := map[string]string{
		"en": "SYZYGY_RECOVERY_EN.txt",
		"de": "SYZYGY_WIEDERHERSTELLUNG_DE.txt",
		"nl": "SYZYGY_HERSTEL_NL.txt",
		"fr": "SYZYGY_RECUPERATION_FR.txt",
		"es": "SYZYGY_RECUPERACION_ES.txt",
		"zh": "SYZYGY_RECOVERY_ZH.txt",
		"ru": "SYZYGY_RECOVERY_RU.txt",
	}
	for lang, filename := range want {
		if got := recoveryGuideFilename(lang); got != filename {
			t.Fatalf("%s: got %q want %q", lang, got, filename)
		}
	}
}

func TestReportWithFailureRefersToRecoveryGuide(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "Desktop"), 0755); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 10, 21, 17, 0, 0, time.Local)
	scan := realScanResult{
		Started: start, Ended: start.Add(time.Minute), CompletedFiles: 1, CompletedBytes: 64,
		Failures: []string{"Bad.rtbz"}, Errors: map[string]string{},
	}
	path, err := writeReportV6("nl", "/egtb", []item{{name: "Bad.rtbz", size: 64}}, 64, scan, intentionalTestResult{Present: true, ExpectedFailDetected: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "SYZYGY_HERSTEL_NL.txt") {
		t.Fatalf("missing recovery guide reference:\n%s", text)
	}
	if !strings.Contains(text, "HERSTEL / VERVANGENDE SYZYGY-BESTANDEN") {
		t.Fatalf("missing recovery heading:\n%s", text)
	}
}

func TestCleanReportDoesNotAddRecoverySection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "Desktop"), 0755); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 10, 21, 17, 0, 0, time.Local)
	scan := realScanResult{
		Started: start, Ended: start.Add(time.Minute), CompletedFiles: 1, CompletedBytes: 64,
		Failures: nil, Errors: map[string]string{},
	}
	path, err := writeReportV6("nl", "/egtb", []item{{name: "Good.rtbz", size: 64}}, 64, scan, intentionalTestResult{Present: true, ExpectedFailDetected: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "SYZYGY_HERSTEL_NL.txt") {
		t.Fatalf("clean report should stay compact and not show recovery guide:\n%s", string(b))
	}
}

func TestDrawRunHeaderKeepsActiveMapVisible(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	drawRunHeader("nl", `E:\\!!TB7\\DTZ`, 1511, 10)
	_ = w.Close()
	os.Stdout = old
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{
		"Syzygy checksumcontrole",
		`Map: E:\\!!TB7\\DTZ`,
		"Gevonden: 1511 bestand(en) (.rtbw/.rtbz)",
		"Werkers (threads): 10",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("live header missing %q:\n%s", want, text)
		}
	}
}

func TestRonaldDeManCreditCanonicalDutch(t *testing.T) {
	want := "Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man. AL-interface, voortgangsweergave, zelftest, checkpointing en rapportage zijn daaromheen gebouwd."
	if got := texts["nl"].credit; got != want {
		t.Fatalf("Dutch credit mismatch:\nwant: %q\n got: %q", want, got)
	}
	rt := reportTexts("nl")
	if rt.basedOn != "Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man." {
		t.Fatalf("Dutch report credit mismatch: %q", rt.basedOn)
	}
	for _, bad := range []string{"Ronald de Man" + "s", "Ronald de Man" + "'s"} {
		if strings.Contains(texts["nl"].credit, bad) || strings.Contains(rt.basedOn, bad) {
			t.Fatalf("ambiguous credit variant present: %q", bad)
		}
	}
}

func TestThreadChoiceEnterUsesHalf(t *testing.T) {
	got, ok := chooseWorkerThreads(bufio.NewReader(strings.NewReader("\n")), "nl", 20)
	if !ok || got != 10 {
		t.Fatalf("got %d ok=%v want 10,true", got, ok)
	}
}

func TestThreadChoiceAllAndCustom(t *testing.T) {
	if got, ok := chooseWorkerThreads(bufio.NewReader(strings.NewReader("1\n")), "nl", 20); !ok || got != 20 {
		t.Fatalf("all: got %d ok=%v want 20,true", got, ok)
	}
	if got, ok := chooseWorkerThreads(bufio.NewReader(strings.NewReader("3\n7\n")), "nl", 20); !ok || got != 7 {
		t.Fatalf("custom: got %d ok=%v want 7,true", got, ok)
	}
}

func TestThreadChoiceEscapeGoesBack(t *testing.T) {
	if got, ok := chooseWorkerThreads(bufio.NewReader(strings.NewReader("esc\n")), "nl", 20); ok || got != 0 {
		t.Fatalf("escape: got %d ok=%v want 0,false", got, ok)
	}
}

func TestEscapeBackPromptsPresentWhereBackIsUseful(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		// V9F intentionally keeps the main-menu prompt quiet; Esc still works
		// there as an unadvertised language shortcut. Subchoices advertise back.
		if !strings.Contains(threadChoicePrompt(lang), "Esc") && !strings.Contains(threadChoicePrompt(lang), "Échap") {
			t.Fatalf("thread choice for %s lacks escape route: %q", lang, threadChoicePrompt(lang))
		}
		if !strings.Contains(texts[lang].choiceLang, "Esc") && !strings.Contains(texts[lang].choiceLang, "Échap") {
			t.Fatalf("language choice for %s lacks escape route: %q", lang, texts[lang].choiceLang)
		}
	}
}

func TestThreadChoiceSingleCPUStillUsesOne(t *testing.T) {
	if got := halfThreads(1); got != 1 {
		t.Fatalf("got %d want 1", got)
	}
}

func TestTBCheckArgsPassesSelectedWorkers(t *testing.T) {
	got := tbcheckArgs(10, "A.rtbw", "B.rtbw")
	want := []string{"--threads", "10", "A.rtbw", "B.rtbw"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestMainMenuPromptDoesNotAdvertiseEscLanguage(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		prompt := texts[lang].choiceMain
		if strings.Contains(strings.ToLower(prompt), "esc") || strings.Contains(prompt, "Échap") {
			t.Fatalf("lang %s main-menu prompt still advertises Esc language shortcut: %q", lang, prompt)
		}
	}
}

func TestRemainingV9FLocalHourUnits(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"en", "Remaining: 1:18 h"},
		{"de", "Restzeit: 1:18 h"},
		{"nl", "Resttijd: 1:18 h"},
		{"fr", "Temps restant: 1:18 h"},
		{"es", "Tiempo restante: 1:18 h"},
		{"ru", "Осталось: 1:18 ч (h)"},
		{"zh", "剩余时间: 1:18 小时 (h)"},
	}
	for _, c := range cases {
		if got := formatRemainingV11(c.lang, 78*time.Minute, false); got != c.want {
			t.Fatalf("lang %s: got %q want %q", c.lang, got, c.want)
		}
	}
}
