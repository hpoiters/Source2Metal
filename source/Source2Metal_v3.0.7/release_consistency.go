package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

// releaseConsistencyCheck is a permanent release guard. It deliberately checks
// the distributable artifacts and the embedded source package together, so a
// later maintainer cannot silently update an EXE, menu, document or embedded ZIP
// while leaving the rest of the release behind.
func releaseConsistencyCheck() error {
	if version != "3.0.7" {
		return fmt.Errorf("release consistency: unexpected Source2Metal version %q", version)
	}
	if syzygyCheckVersion != "2.0.7" {
		return fmt.Errorf("release consistency: unexpected SyzygyCheck version %q", syzygyCheckVersion)
	}

	programRequired := []string{
		"SyzygyCheck_v" + syzygyCheckVersion + ".exe",
		"README_EN.txt", "README_DE.txt", "README_NL.txt", "README_FR.txt",
		"README_ES.txt", "README_ZH.txt", "README_RU.txt",
		"SHA256_SyzygyCheck_v" + syzygyCheckVersion + ".txt",
	}
	for _, name := range programRequired {
		if _, err := zipRead(embeddedSyzygyPackageZip, name); err != nil {
			return fmt.Errorf("release consistency: Syzygy program package: %w", err)
		}
	}

	sourceRequired := []string{
		"main.go", "main_test.go", "console_windows.go", "console_other.go", "go.mod",
		"tbcheck.exe", "README_SOURCE_NL.txt", "README_SOURCE_EN.txt",
		"SHA256_SyzygyCheck_v" + syzygyCheckVersion + "_SOURCE.txt",
	}
	for _, name := range sourceRequired {
		if _, err := zipRead(embeddedSyzygySourceZip, name); err != nil {
			return fmt.Errorf("release consistency: Syzygy source package: %w", err)
		}
	}
	syMain, err := zipRead(embeddedSyzygySourceZip, "main.go")
	if err != nil {
		return err
	}
	for _, marker := range []string{
		`const version = "` + syzygyCheckVersion + `"`,
		"visibleConsoleWidth()", "renderProgress(", "w != lastWidth", "clearConsoleScreen()",
	} {
		if !bytes.Contains(syMain, []byte(marker)) {
			return fmt.Errorf("release consistency: Syzygy source missing renderer/version marker %q", marker)
		}
	}
	syReadme, err := zipRead(embeddedSyzygySourceZip, "README_SOURCE_NL.txt")
	if err != nil {
		return err
	}
	for _, marker := range []string{"ÉÉN adaptieve fysieke consoleregel", "resize/reflow", "regressieregel"} {
		if !bytes.Contains(syReadme, []byte(marker)) {
			return fmt.Errorf("release consistency: Syzygy source README missing regression marker %q", marker)
		}
	}

	s2mRequired := []string{
		"types.go", "main.go", "report.go", "embedded_syzygy_package.go", "embedded_syzygy_source.go",
		"release_consistency.go", "release_consistency_test.go",
		"syzygycheck_package.zip", "syzygycheck_source.zip",
		"assets/SyzygyCheck_package.zip", "assets/SyzygyCheck_SOURCE.zip",
		"DEVELOPMENT_HISTORY/RELEASE_NOTES_V3.0.7.txt",
		"DEVELOPMENT_INFO/RELEASE_CONSISTENCY_RULES_NL.txt",
		"INFO_EN/Source2Metal_Info_EN.txt", "INFO_DE/Source2Metal_Info_DE.txt",
		"INFO_NL/Source2Metal_Info_NL.txt", "INFO_FR/Source2Metal_Info_FR.txt",
		"INFO_ES/Source2Metal_Info_ES.txt", "INFO_ZH/Source2Metal_Info_ZH.txt",
		"INFO_RU/Source2Metal_Info_RU.txt",
	}
	for _, name := range s2mRequired {
		if _, err := zipRead(embeddedSourceZip, name); err != nil {
			return fmt.Errorf("release consistency: Source2Metal source package: %w", err)
		}
	}
	typesSrc, _ := zipRead(embeddedSourceZip, "types.go")
	typesText := string(typesSrc)
	if !strings.Contains(typesText, `version`) || !strings.Contains(typesText, `"3.0.7"`) ||
		!strings.Contains(typesText, `syzygyCheckVersion`) || !strings.Contains(typesText, `"2.0.7"`) {
		return fmt.Errorf("release consistency: embedded Source2Metal source has stale version constants")
	}
	mainSrc, _ := zipRead(embeddedSourceZip, "main.go")
	for _, marker := range []string{"extract-syzygy-source", "SyzygyCheck source-code package", "Choice [0-4]"} {
		if !bytes.Contains(mainSrc, []byte(marker)) {
			return fmt.Errorf("release consistency: utility/source route marker missing: %q", marker)
		}
	}
	reportSrc, _ := zipRead(embeddedSourceZip, "report.go")
	if bytes.Contains(reportSrc, []byte(`start Source2Metal en kies "Broncodepakket uitpakken"`)) {
		return fmt.Errorf("release consistency: obsolete source-extraction menu text is still present")
	}
	if !bytes.Contains(reportSrc, []byte("Hulpprogramma's > Source2Metal-broncodepakket uitpakken")) {
		return fmt.Errorf("release consistency: current Source2Metal source-extraction route missing")
	}

	for _, pair := range []struct {
		name string
		want []byte
	}{
		{"syzygycheck_package.zip", embeddedSyzygyPackageZip},
		{"assets/SyzygyCheck_package.zip", embeddedSyzygyPackageZip},
		{"syzygycheck_source.zip", embeddedSyzygySourceZip},
		{"assets/SyzygyCheck_SOURCE.zip", embeddedSyzygySourceZip},
	} {
		got, err := zipRead(embeddedSourceZip, pair.name)
		if err != nil {
			return err
		}
		if sha256.Sum256(got) != sha256.Sum256(pair.want) {
			return fmt.Errorf("release consistency: %s differs from runtime embedded package", pair.name)
		}
	}

	for _, name := range []string{
		"INFO_EN/Source2Metal_Info_EN.txt", "INFO_DE/Source2Metal_Info_DE.txt",
		"INFO_NL/Source2Metal_Info_NL.txt", "INFO_FR/Source2Metal_Info_FR.txt",
		"INFO_ES/Source2Metal_Info_ES.txt", "INFO_ZH/Source2Metal_Info_ZH.txt",
		"INFO_RU/Source2Metal_Info_RU.txt",
	} {
		b, _ := zipRead(embeddedSourceZip, name)
		if !bytes.Contains(b, []byte("v3.0.7")) || !bytes.Contains(b, []byte("SyzygyCheck")) {
			return fmt.Errorf("release consistency: stale or incomplete INFO file %s", name)
		}
	}

	rel, _ := zipRead(embeddedSourceZip, "DEVELOPMENT_HISTORY/RELEASE_NOTES_V3.0.7.txt")
	for _, marker := range []string{"SyzygyCheck v2.0.7", "permanente automatische release-consistentiecontrole", "SyzygyCheck_v2.0.7_SOURCE.zip"} {
		if !strings.Contains(string(rel), marker) {
			return fmt.Errorf("release consistency: release notes missing %q", marker)
		}
	}
	return nil
}

func zipRead(data []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	return nil, fmt.Errorf("missing %s", name)
}
