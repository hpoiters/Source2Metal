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
	out := filepath.Join(dir, syzygyCheckPackageFilename)

	if data, err := os.ReadFile(out); err == nil {
		h := sha256.Sum256(data)
		if fmt.Sprintf("%x", h[:]) == embeddedSyzygyPackageSHA256() {
			return out, nil
		}
		return "", fmt.Errorf(L(
			"the SyzygyCheck release bundle already exists here but differs from the embedded package; rename or remove it first",
			"das SyzygyCheck-Release-Paket ist hier bereits vorhanden, unterscheidet sich aber vom eingebetteten Paket; zuerst umbenennen oder entfernen",
			"de SyzygyCheck-releasebundle bestaat hier al maar wijkt af van het ingebedde pakket; hernoem of verwijder hem eerst",
			"le paquet de publication SyzygyCheck existe déjà ici mais diffère du paquet intégré ; renommez-le ou supprimez-le d'abord",
			"el paquete de publicación SyzygyCheck ya existe aquí pero difiere del paquete integrado; cámbiele el nombre o elimínelo primero",
			"此处已存在 SyzygyCheck 发布包，但它与内嵌包不同；请先重命名或删除它",
			"Пакет выпуска SyzygyCheck уже существует здесь, но отличается от встроенного пакета; сначала переименуйте или удалите его",
		))
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := atomicWriteFile(out, embeddedSyzygyPackageZip, 0644); err != nil {
		return "", err
	}
	return out, nil
}
