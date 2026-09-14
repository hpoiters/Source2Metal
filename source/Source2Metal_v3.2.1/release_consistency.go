package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

// releaseConsistencyCheck guards the current distributable utility. Historical
// sources and source archives deliberately do not belong inside the EXE.
func releaseConsistencyCheck() error {
	if version != "3.2.1" {
		return fmt.Errorf("release consistency: unexpected Source2Metal version %q", version)
	}
	if syzygyCheckVersion != "2.2.0" || syzygyCheckDisplay != "2.2.0" {
		return fmt.Errorf("release consistency: unexpected SyzygyCheck release %q", syzygyCheckVersion)
	}
	if embeddedSyzygyPackageSHA256() != "d80f8fd70f81b4f06cbd89547a7e346aa08688853b2ce2df25b30e93ccd6cb2c" {
		return fmt.Errorf("release consistency: embedded SyzygyCheck utility differs from the published v2.2.0 release")
	}

	docs := "Docs_AL/"
	program := "Program_AL/"
	required := []string{
		syzygyCheckExeFilename,
		"1_READ_FIRST_AL.txt",
		docs + "LICENSE",
		docs + "README_AL.txt",
		docs + "README_EN.txt",
		docs + "README_NL.txt",
		docs + "RELEASE_NOTES_v2.2.0.txt",
		docs + "PAGEFILE_GUIDE_DE.txt",
		docs + "PAGEFILE_GUIDE_EN.txt",
		docs + "PAGEFILE_GUIDE_ES.txt",
		docs + "PAGEFILE_GUIDE_FR.txt",
		docs + "PAGEFILE_GUIDE_NL.txt",
		docs + "PAGEFILE_GUIDE_RU.txt",
		docs + "PAGEFILE_GUIDE_ZH.txt",
		docs + "RONALD_DE_MAN_NOTICE.txt",
		program + "SYZYGY_HERSTEL_NL.txt",
		program + "SYZYGY_RECOVERY_EN.txt",
		program + "SYZYGY_RECOVERY_RU.txt",
		program + "SYZYGY_RECOVERY_ZH.txt",
		program + "SYZYGY_RECUPERACION_ES.txt",
		program + "SYZYGY_RECUPERATION_FR.txt",
		program + "SYZYGY_WIEDERHERSTELLUNG_DE.txt",
	}
	allowed := make(map[string]struct{}, len(required))
	for _, name := range required {
		allowed[name] = struct{}{}
		if _, err := zipRead(embeddedSyzygyPackageZip, name); err != nil {
			return fmt.Errorf("release consistency: SyzygyCheck utility: %w", err)
		}
	}

	names := zipNames(embeddedSyzygyPackageZip)
	if len(names) != len(required) {
		return fmt.Errorf("release consistency: SyzygyCheck utility contains %d files; expected exactly %d", len(names), len(required))
	}
	for _, name := range names {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("release consistency: unexpected utility file: %s", name)
		}
		lower := strings.ToLower(name)
		for _, forbidden := range []string{"source.zip", "repo_files", "makemem", "validation_", ".go"} {
			if strings.Contains(lower, forbidden) {
				return fmt.Errorf("release consistency: obsolete utility ballast present: %s", name)
			}
		}
	}

	syExe, err := zipRead(embeddedSyzygyPackageZip, syzygyCheckExeFilename)
	if err != nil {
		return err
	}
	const syExeHash = "6e8d8429a55fbdb51e449a85829aa07d84b614fbe1894e1c7049428b4ba61b58"
	if fmt.Sprintf("%x", sha256.Sum256(syExe)) != syExeHash {
		return fmt.Errorf("release consistency: SyzygyCheck executable differs from the published v2.2.0 executable")
	}

	readme, _ := zipRead(embeddedSyzygyPackageZip, docs+"README_AL.txt")
	readmeText := strings.Join(strings.Fields(string(readme)), " ")
	for _, marker := range []string{
		"SyzygyCheck v2.2.0", syzygyCheckExeFilename, "Docs_AL", "Program_AL",
	} {
		if !strings.Contains(readmeText, marker) {
			return fmt.Errorf("release consistency: SyzygyCheck HTML guide missing %q", marker)
		}
	}

	notice, _ := zipRead(embeddedSyzygyPackageZip, docs+"RONALD_DE_MAN_NOTICE.txt")
	if !bytes.Contains(notice, []byte("Ronald de Man")) {
		return fmt.Errorf("release consistency: Ronald de Man credit missing")
	}
	return nil
}

func zipNames(data []byte) []string {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			names = append(names, f.Name)
		}
	}
	return names
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
