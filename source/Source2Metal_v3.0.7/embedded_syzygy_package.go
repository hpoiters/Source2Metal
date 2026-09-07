package main

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// The embedded ZIP is only a distributable package. Source2Metal never executes
// SyzygyCheck and never scans or validates Syzygy tablebase files.
//
//go:embed syzygycheck_package.zip
var embeddedSyzygyPackageZip []byte

func embeddedSyzygyPackageSHA256() string {
	h := sha256.Sum256(embeddedSyzygyPackageZip)
	return fmt.Sprintf("%x", h[:])
}

func extractSyzygyPackage() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	out := filepath.Join(dir, "SyzygyCheck_package.zip")

	if data, err := os.ReadFile(out); err == nil {
		h := sha256.Sum256(data)
		if fmt.Sprintf("%x", h[:]) == embeddedSyzygyPackageSHA256() {
			return out, nil
		}
		return "", fmt.Errorf(L(
			"SyzygyCheck_package.zip already exists here but differs from the embedded package; rename or remove it first",
			"SyzygyCheck_package.zip ist hier bereits vorhanden, unterscheidet sich aber vom eingebetteten Paket; zuerst umbenennen oder entfernen",
			"SyzygyCheck_package.zip bestaat hier al maar wijkt af van het ingebedde pakket; hernoem of verwijder het eerst",
			"SyzygyCheck_package.zip existe déjà ici mais diffère du paquet intégré ; renommez-le ou supprimez-le d'abord",
			"SyzygyCheck_package.zip ya existe aquí pero difiere del paquete integrado; cámbiele el nombre o elimínelo primero",
			"此处已存在 SyzygyCheck_package.zip，但它与内嵌包不同；请先重命名或删除它",
			"SyzygyCheck_package.zip уже существует здесь, но отличается от встроенного пакета; сначала переименуйте или удалите его",
		))
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := atomicWriteFile(out, embeddedSyzygyPackageZip, 0644); err != nil {
		return "", err
	}
	return out, nil
}
