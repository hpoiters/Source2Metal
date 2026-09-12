package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode"
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

func TestDiscoverFilesCurrentFolderVersusSubfolders(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "Controlemap")
	deep := filepath.Join(root, "Schijfset_A", "DTZ")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{
		filepath.Join(parent, "NeverUp.rtbw"):                  "parent",
		filepath.Join(root, "Root.rtbw"):                       "root",
		filepath.Join(root, "ignore.txt"):                      "ignore",
		filepath.Join(root, "Schijfset_A", "Nested.rtbz"):      "nested",
		filepath.Join(root, "Schijfset_A", "DTZ", "Deep.RTBW"): "deep",
	} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// A link inside the tree must never be allowed to lead the checker back to
	// a file above the Controlemap. Systems without symlink permission simply
	// skip this extra fixture.
	_ = os.Symlink(filepath.Join(parent, "NeverUp.rtbw"), filepath.Join(root, "Schijfset_A", "LinkedUp.rtbw"))

	flat, _, err := discoverFilesInScope(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) != 1 || flat[0].name != "Root.rtbw" {
		t.Fatalf("flat inventory=%v", flat)
	}

	recursive, total, err := discoverFilesInScope(root, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join("Root.rtbw"),
		filepath.Join("Schijfset_A", "DTZ", "Deep.RTBW"),
		filepath.Join("Schijfset_A", "Nested.rtbz"),
	}
	if len(recursive) != len(want) {
		t.Fatalf("recursive inventory=%v", recursive)
	}
	for i := range want {
		if recursive[i].name != want[i] {
			t.Fatalf("recursive[%d]=%q want %q", i, recursive[i].name, want[i])
		}
		if strings.Contains(recursive[i].name, "NeverUp") || strings.HasPrefix(recursive[i].name, "..") {
			t.Fatalf("parent path leaked into inventory: %q", recursive[i].name)
		}
	}
	if total != int64(len("root")+len("nested")+len("deep")) {
		t.Fatalf("recursive total=%d", total)
	}
}

func TestRecursiveDriveRootScanSkipsWindowsVolumeMetadata(t *testing.T) {
	root := t.TempDir()
	legitimate := filepath.Join(root, "Backup WDL DTZ", "WDL 1241")
	legitimateNamedMetadata := filepath.Join(root, "Archive", "$RECYCLE.BIN")
	recycleBin := filepath.Join(root, "$RECYCLE.BIN", "S-1-5-21-test")
	systemVolume := filepath.Join(root, "system volume information", "tracking")
	for _, dir := range []string{legitimate, legitimateNamedMetadata, recycleBin, systemVolume} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for path, data := range map[string]string{
		filepath.Join(legitimate, "KQvKR.rtbw"):               "wanted-wdl",
		filepath.Join(legitimate, "KQvKR.rtbz"):               "wanted-dtz",
		filepath.Join(legitimateNamedMetadata, "Nested.rtbw"): "wanted-nested",
		filepath.Join(recycleBin, "Deleted.rtbw"):             "must-not-scan",
		filepath.Join(systemVolume, "Protected.rtbz"):         "must-not-scan",
	} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}

	files, total, err := discoverFilesInScope(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("recursive inventory=%v; Windows volume metadata was not pruned", files)
	}
	for _, f := range files {
		first := strings.Split(filepath.ToSlash(f.name), "/")[0]
		if strings.EqualFold(first, "$RECYCLE.BIN") || strings.EqualFold(first, "System Volume Information") {
			t.Fatalf("protected Windows path leaked into inventory: %q", f.name)
		}
	}
	if total != int64(len("wanted-wdl")+len("wanted-dtz")+len("wanted-nested")) {
		t.Fatalf("recursive total=%d", total)
	}
}

func TestScanScopeChoiceAndNoParentRule(t *testing.T) {
	recursive, ok := chooseScanScope(bufio.NewReader(strings.NewReader("\n")), "nl")
	if !ok || recursive {
		t.Fatalf("default scope recursive=%v ok=%v", recursive, ok)
	}
	recursive, ok = chooseScanScope(bufio.NewReader(strings.NewReader("2\n")), "nl")
	if !ok || !recursive {
		t.Fatalf("recursive scope recursive=%v ok=%v", recursive, ok)
	}
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		if scanScopeTexts(lang).boundary == "" {
			t.Fatalf("%s: missing no-parent boundary text", lang)
		}
	}
}

func TestEveryScanScopeChoiceSequenceInAllLanguages(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		t.Run(lang, func(t *testing.T) {
			text := scanScopeTexts(lang)
			for field, value := range map[string]string{
				"heading": text.heading, "current": text.current, "recursive": text.recursive,
				"boundary": text.boundary, "prompt": text.prompt,
			} {
				if strings.TrimSpace(value) == "" {
					t.Fatalf("empty %s text", field)
				}
			}
			for input, wantRecursive := range map[string]bool{"\n": false, "1\n": false, "2\n": true, "9\n2\n": true} {
				recursive, ok := chooseScanScope(bufio.NewReader(strings.NewReader(input)), lang)
				if !ok || recursive != wantRecursive {
					t.Fatalf("input %q: recursive=%v ok=%v", input, recursive, ok)
				}
			}
			if recursive, ok := chooseScanScope(bufio.NewReader(strings.NewReader("esc\n")), lang); ok || recursive {
				t.Fatalf("Esc did not return: recursive=%v ok=%v", recursive, ok)
			}
		})
	}
}

func TestRecursiveInventoryRunsHelperWithRelativeSubfolderPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}
	root := t.TempDir()
	sub := filepath.Join(root, "Schijfset B", "WDL")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(sub, "Nested.rtbw")
	if err := os.WriteFile(filePath, []byte("tablebase fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(root, "fake_tbcheck.sh")
	script := "#!/bin/sh\nshift\nshift\nfor f in \"$@\"; do printf '%s: OK!\\n' \"$f\"; done\n"
	if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	files, total, err := discoverFilesInScope(root, true)
	if err != nil {
		t.Fatal(err)
	}
	result := runRealScan(bufio.NewReader(strings.NewReader("")), "en", helper, "test helper", root, files, total, 2, true)
	if result.FatalErr != nil || len(result.Errors) != 0 || len(result.Failures) != 0 || result.CompletedFiles != 1 {
		t.Fatalf("recursive helper result=%+v", result)
	}
	if !result.Recursive {
		t.Fatal("recursive scan scope was not retained in result")
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
	p := filepath.Join(desktop, "CheckResult", "SyzygyCheck_Result_20990101_0000.txt")
	lines := reportSavedScreenLines("nl", p)
	if len(lines) != 2 {
		t.Fatalf("lines=%v", lines)
	}
	if lines[0] != "Resultaatrapport opgeslagen in de map CheckResult op uw Windows-bureaublad (Desktop)." {
		t.Fatalf("unexpected location message: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "Volledig pad: ") || !strings.Contains(lines[1], p) {
		t.Fatalf("unexpected path message: %q", lines[1])
	}
}

func TestLocalizedReportFolderNames(t *testing.T) {
	want := map[string]string{
		"en": "CheckResult",
		"de": "CheckResult",
		"nl": "CheckResult",
		"fr": "CheckResult",
		"es": "CheckResult",
		"zh": "检查结果",
		"ru": "РезультатПроверки",
	}
	for lang, name := range want {
		if got := reportFolderName(lang); got != name {
			t.Fatalf("%s: got %q want %q", lang, got, name)
		}
		if !strings.Contains(reportDestinationTexts(lang).subfolder, name) {
			t.Fatalf("%s: destination explanation does not name %q", lang, name)
		}
	}
}

func TestReportDestinationDesktopAndCheckedFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("LOCALAPPDATA", t.TempDir())
	originalDesktop := desktopPathFn
	desktopPathFn = func() (string, error) { return filepath.Join(home, "Desktop"), nil }
	t.Cleanup(func() { desktopPathFn = originalDesktop })

	desktopChoice, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("\n")), "nl", "/egtb")
	if !ok || desktopChoice != filepath.Join(home, "Desktop", "CheckResult") {
		t.Fatalf("desktop choice=%q ok=%v", desktopChoice, ok)
	}
	checkedChoice, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("2\n")), "en", "/egtb")
	if !ok || checkedChoice != filepath.Join("/egtb", "CheckResult") {
		t.Fatalf("checked-folder choice=%q ok=%v", checkedChoice, ok)
	}
}

func TestReportDestinationFreeChoiceAddsCollectionFolder(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	original := browseForFolderFn
	browseForFolderFn = func(title string) (string, error) { return "/chosen", nil }
	t.Cleanup(func() { browseForFolderFn = original })

	chosen, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("3\n")), "fr", "/egtb")
	if !ok || chosen != filepath.Join("/chosen", "CheckResult") {
		t.Fatalf("free choice=%q ok=%v", chosen, ok)
	}
}

func TestReportDestinationRemembersControlFolderAsNextDefault(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	first, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("2\n")), "nl", "/eerste")
	if !ok || first != filepath.Join("/eerste", "CheckResult") {
		t.Fatalf("first control-folder choice=%q ok=%v", first, ok)
	}
	second, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("\n")), "nl", "/tweede")
	if !ok || second != filepath.Join("/tweede", "CheckResult") {
		t.Fatalf("remembered control-folder default=%q ok=%v", second, ok)
	}
}

func TestReportDestinationRemembersExactCustomPathAsNextDefault(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	custom := t.TempDir()
	original := browseForFolderFn
	browseCalls := 0
	browseForFolderFn = func(title string) (string, error) {
		browseCalls++
		return custom, nil
	}
	t.Cleanup(func() { browseForFolderFn = original })

	first, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("3\n")), "en", "/control")
	if !ok || first != filepath.Join(custom, "CheckResult") {
		t.Fatalf("first custom choice=%q ok=%v", first, ok)
	}
	second, ok := chooseReportFolder(bufio.NewReader(strings.NewReader("\n")), "en", "/control")
	if !ok || second != filepath.Join(custom, "CheckResult") {
		t.Fatalf("remembered custom default=%q ok=%v", second, ok)
	}
	if browseCalls != 1 {
		t.Fatalf("remembered custom default reopened Explorer: calls=%d", browseCalls)
	}
}

func TestMissingRememberedCustomPathFallsBackToDesktopDefault(t *testing.T) {
	home := t.TempDir()
	originalDesktop := desktopPathFn
	desktopPathFn = func() (string, error) { return filepath.Join(home, "Desktop"), nil }
	t.Cleanup(func() { desktopPathFn = originalDesktop })
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		t.Run(lang, func(t *testing.T) {
			t.Setenv("LOCALAPPDATA", t.TempDir())
			missing := filepath.Join(home, "missing-"+lang)
			saveReportDestinationPreference(reportDestinationPreference{Mode: "custom", CustomPath: missing})
			var got string
			var ok bool
			output := captureTestStdout(t, func() {
				got, ok = chooseReportFolder(bufio.NewReader(strings.NewReader("9\n\n")), lang, "/control")
			})
			if !ok || got != filepath.Join(home, "Desktop", reportFolderName(lang)) {
				t.Fatalf("missing custom fallback=%q ok=%v", got, ok)
			}
			text := reportDestinationTexts(lang)
			wantWarning := fmt.Sprintf(text.previousUnavailable, missing)
			compact := func(s string) string {
				return strings.Map(func(r rune) rune {
					if unicode.IsSpace(r) {
						return -1
					}
					return r
				}, s)
			}
			if strings.Count(compact(output), compact(wantWarning)) != 1 {
				t.Fatalf("missing custom warning not shown exactly once:\n%s", output)
			}
			if !strings.Contains(output, "\n\n"+text.heading+"\n") {
				t.Fatalf("warning is not a separate paragraph before the menu:\n%s", output)
			}
			if !strings.Contains(output, fmt.Sprintf(text.prompt, 1)) {
				t.Fatalf("Desktop was not shown as the new default:\n%s", output)
			}
			pref, unavailable := loadReportDestinationPreference()
			if pref.Mode != "desktop" || unavailable != "" {
				t.Fatalf("saved fallback preference=%+v unavailable=%q", pref, unavailable)
			}
		})
	}
}

func TestEveryReportDestinationChoiceSequenceInAllLanguages(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		t.Run(lang, func(t *testing.T) {
			t.Setenv("LOCALAPPDATA", t.TempDir())
			desktop := t.TempDir()
			custom := t.TempDir()
			controlA := filepath.Join(t.TempDir(), "ControlA")
			controlB := filepath.Join(t.TempDir(), "ControlB")
			folderName := reportFolderName(lang)
			text := reportDestinationTexts(lang)
			for field, value := range map[string]string{
				"heading": text.heading, "subfolder": text.subfolder, "desktop": text.desktop,
				"control": text.control, "choose": text.choose, "defaultWord": text.defaultWord,
				"prompt": text.prompt, "previous": text.previous, "browseError": text.browseError,
				"previousUnavailable": text.previousUnavailable,
			} {
				if strings.TrimSpace(value) == "" {
					t.Fatalf("empty %s text", field)
				}
			}
			if strings.Count(text.previousUnavailable, "%s") != 1 {
				t.Fatalf("unavailable-folder warning must contain one path placeholder: %q", text.previousUnavailable)
			}

			originalDesktop := desktopPathFn
			originalBrowse := browseForFolderFn
			desktopPathFn = func() (string, error) { return desktop, nil }
			browseCalls := 0
			browseForFolderFn = func(title string) (string, error) {
				browseCalls++
				return custom, nil
			}
			t.Cleanup(func() {
				desktopPathFn = originalDesktop
				browseForFolderFn = originalBrowse
			})

			assertChoice := func(input, control, want string) string {
				t.Helper()
				var got string
				var ok bool
				output := captureTestStdout(t, func() {
					got, ok = chooseReportFolder(bufio.NewReader(strings.NewReader(input)), lang, control)
				})
				if !ok || got != want {
					t.Fatalf("input %q: got=%q want=%q ok=%v", input, got, want, ok)
				}
				return output
			}

			// First use: Desktop is the sole visible default.
			output := assertChoice("\n", controlA, filepath.Join(desktop, folderName))
			if strings.Count(output, "["+text.defaultWord+"]") != 1 || !strings.Contains(output, fmt.Sprintf(text.prompt, 1)) {
				t.Fatalf("first-use default text is inconsistent:\n%s", output)
			}

			// Controlemap is remembered as a type, so a moved EXE uses its current folder.
			assertChoice("2\n", controlA, filepath.Join(controlA, folderName))
			output = assertChoice("\n", controlB, filepath.Join(controlB, folderName))
			if strings.Count(output, "["+text.defaultWord+"]") != 1 || !strings.Contains(output, fmt.Sprintf(text.prompt, 2)) {
				t.Fatalf("control-folder default text is inconsistent:\n%s", output)
			}

			// Invalid input stays in the menu and can be followed by a valid choice.
			assertChoice("9\n2\n", controlA, filepath.Join(controlA, folderName))

			// Explicit free choice opens Explorer once; Enter next time reuses the exact path.
			assertChoice("3\n", controlA, filepath.Join(custom, folderName))
			output = assertChoice("\n", controlB, filepath.Join(custom, folderName))
			if browseCalls != 1 {
				t.Fatalf("remembered custom default reopened Explorer: calls=%d", browseCalls)
			}
			if !strings.Contains(output, custom) || strings.Count(output, "["+text.defaultWord+"]") != 1 || !strings.Contains(output, fmt.Sprintf(text.prompt, 3)) {
				t.Fatalf("custom default text is inconsistent:\n%s", output)
			}
			customReplacement := t.TempDir()
			browseForFolderFn = func(title string) (string, error) {
				browseCalls++
				return customReplacement, nil
			}
			assertChoice("3\n", controlA, filepath.Join(customReplacement, folderName))
			assertChoice("\n", controlB, filepath.Join(customReplacement, folderName))
			if browseCalls != 2 {
				t.Fatalf("explicit custom replacement did not open Explorer exactly once: calls=%d", browseCalls)
			}

			// Either fixed destination can be selected again and becomes the new default.
			assertChoice("1\n", controlA, filepath.Join(desktop, folderName))
			assertChoice("\n", controlB, filepath.Join(desktop, folderName))
			assertChoice("2\n", controlA, filepath.Join(controlA, folderName))

			// Cancelling Explorer returns to the same menu; the following choice wins.
			browseForFolderFn = func(title string) (string, error) {
				browseCalls++
				return "", errFolderSelectionCancelled
			}
			assertChoice("3\n1\n", controlA, filepath.Join(desktop, folderName))
		})
	}
}

func TestRememberedCustomDestinationSurvivesLanguageChange(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	custom := t.TempDir()
	original := browseForFolderFn
	browseCalls := 0
	browseForFolderFn = func(title string) (string, error) {
		browseCalls++
		return custom, nil
	}
	t.Cleanup(func() { browseForFolderFn = original })

	if got, _ := chooseReportFolder(bufio.NewReader(strings.NewReader("3\n")), "en", "/control"); got != filepath.Join(custom, "CheckResult") {
		t.Fatalf("English custom destination=%q", got)
	}
	if got, _ := chooseReportFolder(bufio.NewReader(strings.NewReader("\n")), "ru", "/control"); got != filepath.Join(custom, "РезультатПроверки") {
		t.Fatalf("Russian remembered destination=%q", got)
	}
	if got, _ := chooseReportFolder(bufio.NewReader(strings.NewReader("\n")), "zh", "/control"); got != filepath.Join(custom, "检查结果") {
		t.Fatalf("Chinese remembered destination=%q", got)
	}
	if browseCalls != 1 {
		t.Fatalf("language change reopened Explorer: calls=%d", browseCalls)
	}
}

func TestNewChoiceMenusStayInsideResizedConsole(t *testing.T) {
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		t.Run(lang, func(t *testing.T) {
			t.Setenv("LOCALAPPDATA", t.TempDir())
			longCustom := filepath.Join(t.TempDir(), strings.Repeat("LongPathPart", 12))
			if err := os.MkdirAll(longCustom, 0755); err != nil {
				t.Fatal(err)
			}
			saveReportDestinationPreference(reportDestinationPreference{Mode: "custom", CustomPath: longCustom})

			for _, width := range []int{20, 30, 40, 60, 80, 100} {
				t.Setenv("COLUMNS", fmt.Sprintf("%d", width))
				reportOutput := captureTestStdout(t, func() {
					_, _ = chooseReportFolder(bufio.NewReader(strings.NewReader("\n")), lang, "/control")
				})
				scopeOutput := captureTestStdout(t, func() {
					_, _ = chooseScanScope(bufio.NewReader(strings.NewReader("\n")), lang)
				})
				for menu, output := range map[string]string{"report": reportOutput, "scope": scopeOutput} {
					for lineNumber, line := range strings.Split(output, "\n") {
						if strings.ContainsRune(line, '\r') {
							t.Fatalf("%s width %d line %d contains CR: %q", menu, width, lineNumber+1, line)
						}
						if runeLen(line) > width-1 {
							t.Fatalf("%s width %d line %d is %d cells: %q", menu, width, lineNumber+1, runeLen(line), line)
						}
					}
				}
			}
		})
	}
}

func TestConsoleCellWidthHandlesCJKAndCombiningText(t *testing.T) {
	if got := runeLen("A中🙂"); got != 5 {
		t.Fatalf("display width=%d want 5", got)
	}
	if got := runeLen("e\u0301"); got != 1 {
		t.Fatalf("combining display width=%d want 1", got)
	}
	left, right := splitAtConsoleCells("中AB", 3)
	if left != "中A" || right != "B" {
		t.Fatalf("split=%q + %q", left, right)
	}
}

func captureTestStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	b, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestReportWriteFallbackUsesLocalizedCollectionFolder(t *testing.T) {
	checked := t.TempDir()
	start := time.Date(2026, 9, 12, 12, 0, 0, 0, time.Local)
	scan := realScanResult{
		Started: start, Ended: start.Add(time.Second), Errors: map[string]string{},
	}
	path, err := writeReportV6("ru", checked, filepath.Join("/proc", "unwritable"), nil, 0, scan, intentionalTestResult{})
	if err != nil {
		t.Fatal(err)
	}
	wantParent := filepath.Join(checked, "РезультатПроверки")
	if filepath.Dir(path) != wantParent {
		t.Fatalf("fallback parent=%q want %q", filepath.Dir(path), wantParent)
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
	path, err := writeReportV6("nl", "/egtb", filepath.Join(home, "Desktop", "CheckResult"), files, 300, scan, testResult)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"Taal:", "Duur van deze run: 02:00    Gemiddelde leessnelheid: 98.1 MB/s", "Bestanden gevonden: 3", "Werkers (threads): 10", "RESULTAAT (ECHTE SYZYGY-BESTANDEN): FOUT", "CHECKSUMFOUTEN:", "- " + filepath.Join("/egtb", "Bad.rtbw"), "Deze TEST-FAIL veranderde niets aan uw eigen bestanden."} {
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

func TestReportUsesFullPathForEveryFailureAndError(t *testing.T) {
	reportDir := t.TempDir()
	controlDir := filepath.Join(reportDir, "Controlemap")
	start := time.Date(2026, 9, 12, 14, 0, 0, 0, time.Local)
	scan := realScanResult{
		Started: start, Ended: start.Add(time.Second), CompletedFiles: 2, Recursive: true,
		Failures: []string{filepath.Join("Schijf_A", "Bad.rtbw")},
		Errors:   map[string]string{filepath.Join("Schijf_B", "Unreadable.rtbz"): "read error"},
	}
	path, err := writeReportV6("nl", controlDir, reportDir, nil, 0, scan, intentionalTestResult{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{
		fullResultPath(controlDir, filepath.Join("Schijf_A", "Bad.rtbw")),
		fullResultPath(controlDir, filepath.Join("Schijf_B", "Unreadable.rtbz")),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing full problem path %q in report:\n%s", want, text)
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
		wantPath := "Program_AL\\" + filename
		if got := recoveryGuideReleasePath(lang); got != wantPath {
			t.Fatalf("%s release path: got %q want %q", lang, got, wantPath)
		}
	}
}

func TestFinishedProgressBlockLeavesOneBlankLineBeforeNextSection(t *testing.T) {
	output := captureTestStdout(t, func() {
		renderer := &progressBlockRenderer{}
		renderer.render([3]string{"file line", "bar line", "percentage line"})
		renderer.finish()
		renderer.finish() // finishing twice must not add another separator
		fmt.Println()
		fmt.Print("next section")
	})
	want := "file line\nbar line\npercentage line\n\nnext section"
	if output != want {
		t.Fatalf("unexpected section separation:\n got %q\nwant %q", output, want)
	}
}

func TestReleaseDocumentationAndWorkflowConsistency(t *testing.T) {
	requiredText := map[string][]string{
		"1_READ_FIRST_AL.txt": {
			"Docs_AL\\README_AL.txt",
			"Program_AL",
			"https://github.com/hpoiters/SyzygyCheck",
			"SyzygyCheck_v2.2.0_SOURCE.zip",
			"ENGLISH", "DEUTSCH", "NEDERLANDS", "FRANÇAIS", "ESPAÑOL", "中文", "РУССКИЙ",
		},
		"README_AL.txt": {
			"Program_AL\\SYZYGY_RECOVERY_EN.txt",
			"Program_AL\\SYZYGY_WIEDERHERSTELLUNG_DE.txt",
			"Program_AL\\SYZYGY_HERSTEL_NL.txt",
			"Program_AL\\SYZYGY_RECUPERATION_FR.txt",
			"Program_AL\\SYZYGY_RECUPERACION_ES.txt",
			"Program_AL\\SYZYGY_RECOVERY_ZH.txt",
			"Program_AL\\SYZYGY_RECOVERY_RU.txt",
			"a separate warning", "separate Warnung", "afzonderlijke", "avertissement",
			"aviso separado", "单独显示警告", "отдельное предупреждение",
			"SyzygyCheck_v2.2.0_SOURCE.zip", "De source", "не включён",
		},
		"README_EN.txt":                        {"!SyzygyCheck_v2.2.0.exe", "1_READ_FIRST_AL.txt", "Docs_AL\\", "Program_AL\\", "Program_AL\\SYZYGY_RECOVERY_EN.txt", "unavailable previous path", "Desktop is now the default", "$RECYCLE.BIN", "System Volume Information", "SyzygyCheck_v2.2.0_SOURCE.zip"},
		"README_NL.txt":                        {"!SyzygyCheck_v2.2.0.exe", "1_READ_FIRST_AL.txt", "Docs_AL\\", "Program_AL\\", "Program_AL\\SYZYGY_HERSTEL_NL.txt", "onbereikbare vorige pad", "Desktop nu de standaard", "$RECYCLE.BIN", "System Volume Information", "SyzygyCheck_v2.2.0_SOURCE.zip"},
		"README.md":                            {"1_READ_FIRST_AL.txt", "Docs_AL/", "Program_AL/", "automatically generated tag archives", "$RECYCLE.BIN", "System Volume Information", "SyzygyCheck_v2.2.0_SOURCE.zip"},
		"RELEASE_POLICY.md":                    {"complete ZIP", "1_READ_FIRST_AL.txt", "Docs_AL/", "Program_AL/", "after implementation and packaging", "before GitHub publication", "_SOURCE.zip"},
		"RELEASE_CONSISTENCY_CHECK_v2.2.0.txt": {"PASSED LOCALLY", "Exactly one blank line", "unavailable remembered custom result path", "Program_AL", "No .go file", "without creating a new release", "$RECYCLE.BIN", "SyzygyCheck_v2.2.0_SOURCE.zip"},
		"RELEASE_NOTES_v2.2.0.txt":             {"exactly one blank line", "seven-language warning", "1_READ_FIRST_AL.txt", "Docs_AL\\", "Program_AL\\", "$RECYCLE.BIN", "System Volume Information", "SyzygyCheck_v2.2.0_SOURCE.zip"},
		"BUILD_v2.2.0.txt":                     {"go test ./...", "go vet ./...", "seven-language warning", "1_READ_FIRST_AL.txt", "Docs_AL\\", "Program_AL\\", "$RECYCLE.BIN", "System Volume Information", "SyzygyCheck_v2.2.0_SOURCE.zip"},
	}
	for name, fragments := range requiredText {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(b)
		for _, fragment := range fragments {
			if !strings.Contains(text, fragment) {
				t.Errorf("%s is missing required text %q", name, fragment)
			}
		}
		for _, forbidden := range []string{"Ronald de Mans", "Ronald de Man's"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s contains forbidden credit form %q", name, forbidden)
			}
		}
		if strings.Contains(text, "!SyzygyCheck.exe") {
			t.Errorf("%s contains the generic executable name instead of the packaged filename", name)
		}
	}

	workflowBytes, err := os.ReadFile(filepath.Join(".github", "workflows", "release-v2.2.0.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(workflowBytes)
	for _, required := range []string{
		"gh release upload v2.2.0",
		"SyzygyCheck_v2.2.0_RELEASE.$env:GITHUB_RUN_ID.new.zip",
		"dist/SyzygyCheck_v2.2.0_SOURCE.zip",
		"SyzygyCheck_v2.2.0_SOURCE.$env:GITHUB_RUN_ID.new.zip",
		"Temporary release ZIP upload is missing or its remote digest differs",
		"Temporary source ZIP upload is missing or its remote digest differs",
		"Final source ZIP name or digest verification failed",
		"gh api --method PATCH",
		"git tag -f v2.2.0",
		"Docs_AL/README_AL.txt",
		"Program_AL/SYZYGY_HERSTEL_NL.txt",
		"Developer source leaked into the user release ZIP.",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("release workflow is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"gh release create v2.2.0",
		"--clobber",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("release workflow still contains forbidden publication step %q", forbidden)
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
	path, err := writeReportV6("nl", "/egtb", filepath.Join(home, "Desktop", "CheckResult"), []item{{name: "Bad.rtbz", size: 64}}, 64, scan, intentionalTestResult{Present: true, ExpectedFailDetected: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "Program_AL\\SYZYGY_HERSTEL_NL.txt") {
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
	path, err := writeReportV6("nl", "/egtb", filepath.Join(home, "Desktop", "CheckResult"), []item{{name: "Good.rtbz", size: 64}}, 64, scan, intentionalTestResult{Present: true, ExpectedFailDetected: true})
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
	drawRunHeader("nl", `E:\\!!TB7\\DTZ`, 1511, 10, false)
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
		`Controlemap: E:\\!!TB7\\DTZ`,
		"Checkbereik: Alleen deze Controlemap [standaard]",
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
