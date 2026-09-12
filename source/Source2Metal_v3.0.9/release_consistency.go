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
	if version != "3.0.9" {
		return fmt.Errorf("release consistency: unexpected Source2Metal version %q", version)
	}
	if syzygyCheckVersion != "2.0.9" || syzygyCheckVariant != "V9F" || syzygyCheckDisplay != "2.0.9 V9F" {
		return fmt.Errorf("release consistency: unexpected SyzygyCheck release %q %q", syzygyCheckVersion, syzygyCheckVariant)
	}
	if embeddedSyzygyPackageSHA256() != "93d0146c5caac3427637b685ac0970ba077809061fccdea0e120351a9329cae1" {
		return fmt.Errorf("release consistency: embedded lean SyzygyCheck utility differs from the approved package")
	}

	docs := releaseDocsDir + "/"
	required := []string{
		syzygyCheckExeFilename,
		releaseReadmeFilename,
		docs + "LICENSE_GPL-3.0.txt",
		docs + "RELEASE_NOTES_v2.0.9.txt",
		docs + "PAGEFILE_GUIDE_EN.txt",
		docs + "PAGEFILE_GUIDE_RU.txt",
		docs + "PAGEFILE_GUIDE_ZH.txt",
		docs + "SYZYGY_RECOVERY_EN.txt",
		docs + "SYZYGY_RECOVERY_RU.txt",
		docs + "SYZYGY_RECOVERY_ZH.txt",
		docs + "RONALD_DE_MAN_NOTICE.txt",
		docs + "SHA256_SyzygyCheck_v2.0.9_V9F.txt",
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
	const syExeHash = "b384f37f5c67df5ac1e02ab55190e85a8034fd530d2be6a6d0c2fce5ea391c35"
	if fmt.Sprintf("%x", sha256.Sum256(syExe)) != syExeHash {
		return fmt.Errorf("release consistency: SyzygyCheck executable differs from the validated V9F executable")
	}

	checksumText, _ := zipRead(embeddedSyzygyPackageZip, docs+"SHA256_SyzygyCheck_v2.0.9_V9F.txt")
	if !bytes.Contains(checksumText, []byte(syExeHash+"  "+syzygyCheckExeFilename)) {
		return fmt.Errorf("release consistency: SyzygyCheck checksum file does not match the executable")
	}

	readme, _ := zipRead(embeddedSyzygyPackageZip, releaseReadmeFilename)
	readmeText := strings.Join(strings.Fields(string(readme)), " ")
	for _, marker := range []string{
		"<!doctype html>", `id="languages"`, `href="#en"`, `href="#ru"`, `href="#zh"`,
		`id="en" lang="en"`, `id="ru" lang="ru"`, `id="zh" lang="zh"`,
		"SyzygyCheck v2.0.9 V9F", syzygyCheckExeFilename, "Ronald de Man",
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
