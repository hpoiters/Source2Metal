package main

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// The embedded ZIP preserves the exact SyzygyCheck source used for this release.
// Source2Metal only writes this ZIP; it never compiles or executes it.
//
//go:embed syzygycheck_source.zip
var embeddedSyzygySourceZip []byte

func embeddedSyzygySourceSHA256() string {
	h := sha256.Sum256(embeddedSyzygySourceZip)
	return fmt.Sprintf("%x", h[:])
}

func extractSyzygySourcePackage() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	out := filepath.Join(dir, syzygyCheckSourceFilename)
	if data, err := os.ReadFile(out); err == nil {
		h := sha256.Sum256(data)
		if fmt.Sprintf("%x", h[:]) == embeddedSyzygySourceSHA256() {
			return out, nil
		}
		return "", fmt.Errorf(L(
			"the SyzygyCheck source ZIP already exists here but differs from the embedded source package; rename or remove it first",
			"die SyzygyCheck-Quellcode-ZIP existiert hier bereits, unterscheidet sich aber vom eingebetteten Quellcodepaket; zuerst umbenennen oder entfernen",
			"de SyzygyCheck-broncode-ZIP bestaat hier al maar wijkt af van het ingebedde broncodepakket; hernoem of verwijder hem eerst",
			"le ZIP source SyzygyCheck existe déjà ici mais diffère du paquet source intégré ; renommez-le ou supprimez-le d'abord",
			"el ZIP fuente SyzygyCheck ya existe aquí pero difiere del paquete fuente integrado; cámbiele el nombre o elimínelo primero",
			"此处已存在 SyzygyCheck 源代码 ZIP，但它与内嵌源代码包不同；请先重命名或删除它",
			"ZIP исходного кода SyzygyCheck уже существует здесь, но отличается от встроенного пакета; сначала переименуйте или удалите его",
		))
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := atomicWriteFile(out, embeddedSyzygySourceZip, 0644); err != nil {
		return "", err
	}
	return out, nil
}
