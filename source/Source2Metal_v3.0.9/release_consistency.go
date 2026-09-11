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
	if version != "3.0.9" {
		return fmt.Errorf("release consistency: unexpected Source2Metal version %q", version)
	}
	if syzygyCheckVersion != "2.0.9" || syzygyCheckVariant != "V9F" || syzygyCheckDisplay != "2.0.9 V9F" {
		return fmt.Errorf("release consistency: unexpected SyzygyCheck release %q %q", syzygyCheckVersion, syzygyCheckVariant)
	}
	if embeddedSyzygyPackageSHA256() != "e3fe90ae9c8aebd6f0179b7dfc34a2858ff920aff28495306fe757aa09f9c671" {
		return fmt.Errorf("release consistency: embedded Syzygy V9F GitHub release bundle differs from the published artifact")
	}
	if embeddedSyzygySourceSHA256() != "b9a2e4d294e40fc086b35eb4b511be484244613d2d75e9b14a6f1804d62719b4" {
		return fmt.Errorf("release consistency: embedded Syzygy V9F source differs from the published artifact")
	}

	// The program package is the exact, published V9F GitHub release bundle.
	programRequired := []string{
		syzygyCheckExeFilename,
		syzygyCheckSourceFilename,
		"SyzygyCheck_v2.0.9_V9F_GITHUB_REPO_FILES.zip",
		"SHA256_SyzygyCheck_v2.0.9_V9F_FINAL.txt",
		"GITHUB_RELEASE_NOTES_SyzygyCheck_v2.0.9_V9F_FINAL.txt",
		"VALIDATION_SyzygyCheck_Result_20260911_1807.txt",
		"DOC_CONSISTENCY_NOTE_V9F_FINAL.txt",
	}
	for _, name := range programRequired {
		if _, err := zipRead(embeddedSyzygyPackageZip, name); err != nil {
			return fmt.Errorf("release consistency: Syzygy program package: %w", err)
		}
	}
	syExe, err := zipRead(embeddedSyzygyPackageZip, syzygyCheckExeFilename)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(syExe)) != "b384f37f5c67df5ac1e02ab55190e85a8034fd530d2be6a6d0c2fce5ea391c35" {
		return fmt.Errorf("release consistency: embedded Syzygy V9F executable differs from the published executable")
	}
	bundledSource, err := zipRead(embeddedSyzygyPackageZip, syzygyCheckSourceFilename)
	if err != nil {
		return err
	}
	if sha256.Sum256(bundledSource) != sha256.Sum256(embeddedSyzygySourceZip) {
		return fmt.Errorf("release consistency: Syzygy source in published bundle differs from separately embedded source")
	}
	checksumText, err := zipRead(embeddedSyzygyPackageZip, "SHA256_SyzygyCheck_v2.0.9_V9F_FINAL.txt")
	if err != nil {
		return err
	}
	for _, marker := range []string{
		"b384f37f5c67df5ac1e02ab55190e85a8034fd530d2be6a6d0c2fce5ea391c35  " + syzygyCheckExeFilename,
		"b9a2e4d294e40fc086b35eb4b511be484244613d2d75e9b14a6f1804d62719b4  " + syzygyCheckSourceFilename,
	} {
		if !bytes.Contains(checksumText, []byte(marker)) {
			return fmt.Errorf("release consistency: published Syzygy checksum list missing %q", marker)
		}
	}

	sourceRequired := []string{
		"main.go", "main_test.go", "console_windows.go", "console_other.go", "go.mod",
		"threads.go", "checkpoint.go", "progress_v11.go", "report_v6.go",
		"README.md", "README_NL.txt", "README_EN.txt",
		"RELEASE_NOTES_v2.0.9.txt", "BUILD_v2.0.9.txt",
		"RONALD_DE_MAN_NOTICE.txt", "RONALD_DE_MAN_SOURCE_REFERENCE.txt",
		"TEXT_CONSISTENCY_CHECK_v2.0.9_V9F.txt", "STATIC_RELEASE_CHECK_v2.0.9_V9F.txt",
	}
	for _, name := range sourceRequired {
		if _, err := zipRead(embeddedSyzygySourceZip, syzygyCheckSourceRoot+name); err != nil {
			return fmt.Errorf("release consistency: Syzygy source package: %w", err)
		}
	}
	syMain, err := zipRead(embeddedSyzygySourceZip, syzygyCheckSourceRoot+"main.go")
	if err != nil {
		return err
	}
	for _, marker := range []string{
		`version               = "` + syzygyCheckDisplay + `"`,
		"chooseWorkerThreads(", "runAutomaticIntentionalTest(", "runRealScan(",
	} {
		if !bytes.Contains(syMain, []byte(marker)) {
			return fmt.Errorf("release consistency: Syzygy source missing release marker %q", marker)
		}
	}
	syReadme, err := zipRead(embeddedSyzygySourceZip, syzygyCheckSourceRoot+"README_NL.txt")
	if err != nil {
		return err
	}
	for _, marker := range []string{
		"SyzygyCheck v2.0.9 V9F", "Werkers (threads)",
		"Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man.",
		"Windows Terminal", "checkpoint",
	} {
		if !bytes.Contains(syReadme, []byte(marker)) {
			return fmt.Errorf("release consistency: Syzygy source README missing V9F marker %q", marker)
		}
	}
	staticCheck, err := zipRead(embeddedSyzygySourceZip, syzygyCheckSourceRoot+"STATIC_RELEASE_CHECK_v2.0.9_V9F.txt")
	if err != nil {
		return err
	}
	for _, marker := range []string{"Resultaat: 48/48 geslaagd; 0 mislukt.", "[OK] Dutch canonical credit", "[OK] no Ronald de Mans", "[OK] no Ronald de Man's"} {
		if !bytes.Contains(staticCheck, []byte(marker)) {
			return fmt.Errorf("release consistency: Syzygy V9F static check missing %q", marker)
		}
	}

	s2mRequired := []string{
		"types.go", "main.go", "report.go", "embedded_syzygy_package.go", "embedded_syzygy_source.go",
		"release_consistency.go", "release_consistency_test.go",
		"syzygycheck_package.zip", "syzygycheck_source.zip",
		"assets/SyzygyCheck_package.zip", "assets/SyzygyCheck_SOURCE.zip",
		"DEVELOPMENT_HISTORY/RELEASE_NOTES_V3.0.9.txt",
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
	if !strings.Contains(typesText, `version`) || !strings.Contains(typesText, `"3.0.9"`) ||
		!strings.Contains(typesText, `syzygyCheckVersion`) || !strings.Contains(typesText, `"2.0.9"`) ||
		!strings.Contains(typesText, `syzygyCheckVariant`) || !strings.Contains(typesText, `"V9F"`) {
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
		if !bytes.Contains(b, []byte("v3.0.9")) || !bytes.Contains(b, []byte("SyzygyCheck v2.0.9 V9F")) {
			return fmt.Errorf("release consistency: stale or incomplete INFO file %s", name)
		}
	}

	rel, _ := zipRead(embeddedSourceZip, "DEVELOPMENT_HISTORY/RELEASE_NOTES_V3.0.9.txt")
	for _, marker := range []string{"SyzygyCheck v2.0.9 V9F", syzygyCheckPackageFilename, syzygyCheckSourceFilename, "canonieke Nederlandse credit"} {
		if !strings.Contains(string(rel), marker) {
			return fmt.Errorf("release consistency: release notes missing %q", marker)
		}
	}
	notices, _ := zipRead(embeddedSourceZip, "THIRD_PARTY_NOTICES.txt")
	for _, marker := range []string{
		"Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man.",
		"https://github.com/syzygy1/tb",
		"Tablebasebestanden, bestandsnamen en resultaten worden door dit mechanisme niet geüpload.",
	} {
		if !bytes.Contains(notices, []byte(marker)) {
			return fmt.Errorf("release consistency: third-party notices missing %q", marker)
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
