package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// This name is deliberately NOT .rtbw/.rtbz. It can therefore never be
// discovered as a real tablebase, even if Windows is powered off during the
// few milliseconds in which the fixture exists.
const intentionalTestFile = "SyzygyCheck_SELFTEST_FAIL.tmp"

type intentionalTestResult struct {
	Present              bool
	File                 string
	ExpectedFailDetected bool
	UnexpectedOK         bool
	Output               string
	Err                  error
}

// onboardIntentionalFailFixture is an 80-byte checksum-shaped fixture:
// 64 bytes payload followed by 16 deliberately incorrect checksum bytes.
// the unchanged tbcheck by Ronald de Man therefore exercises its normal embedded
// checksum verification path and must report FAIL!. It is not a chess
// tablebase and is never included in the user's .rtbw/.rtbz inventory.
func onboardIntentionalFailFixture() []byte {
	payload := make([]byte, 64)
	copy(payload, []byte("SyzygyCheck AL diagnostic fixture - intentional checksum mismatch"))
	sum := bytes.Repeat([]byte{0xA5}, 16)
	return append(payload, sum...)
}

// cleanupStaleIntentionalTest silently removes only our exact reserved file.
// This is intentionally called before the UI starts, so a remnant from a
// power loss/crash can never contaminate the next self-test.
func cleanupStaleIntentionalTest(dir string) {
	_ = os.RemoveAll(filepath.Join(dir, intentionalTestFile))
}

func cleanupStaleIntentionalTestAtStartup() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cleanupStaleIntentionalTest(filepath.Dir(exe))
}

// runAutomaticIntentionalTest never reads or modifies a real tablebase. It
// creates the tiny onboard FAIL fixture next to SyzygyCheck, checks only that
// fixture with the unchanged tbcheck by Ronald de Man, and removes it before return.
func runAutomaticIntentionalTest(helper, appDir string) intentionalTestResult {
	cleanupStaleIntentionalTest(appDir)
	testPath := filepath.Join(appDir, intentionalTestFile)
	if err := os.WriteFile(testPath, onboardIntentionalFailFixture(), 0600); err != nil {
		return intentionalTestResult{Present: true, File: intentionalTestFile, Err: err}
	}
	defer cleanupStaleIntentionalTest(appDir)

	cmd := exec.Command(helper, intentionalTestFile)
	cmd.Dir = appDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	runErr := cmd.Run()
	text := out.String()
	checked, failed := parseTBCheckOutput(text)
	r := intentionalTestResult{Present: true, File: intentionalTestFile, Output: text}
	for _, n := range failed {
		if strings.EqualFold(filepath.Base(n), intentionalTestFile) {
			r.ExpectedFailDetected = true
		}
	}
	for _, n := range checked {
		if strings.EqualFold(filepath.Base(n), intentionalTestFile) {
			r.UnexpectedOK = true
		}
	}
	if runErr != nil {
		r.Err = runErr
	} else if !r.ExpectedFailDetected && !r.UnexpectedOK {
		r.Err = fmt.Errorf("tbcheck returned no OK!/FAIL! result for the onboard intentional FAIL fixture")
	}
	return r
}

func intentionalTestScreenLines(lang string, r intentionalTestResult) []string {
	if !r.Present {
		return nil
	}
	if r.ExpectedFailDetected {
		var lines []string
		switch lang {
		case "nl":
			lines = []string{"TEST-FAIL GEDETECTEERD — DIT IS OPZETTELIJK.", "De onboard zelftestfile is correct als fout herkend en alweer verwijderd."}
		case "de":
			lines = []string{"TEST-FEHLER ERKANNT — DIES IST ABSICHTLICH.", "Die interne Selbsttestdatei wurde korrekt erkannt und bereits entfernt."}
		case "fr":
			lines = []string{"ÉCHEC DE TEST DÉTECTÉ — C'EST VOLONTAIRE.", "Le fichier d'auto-test interne a été correctement reconnu puis supprimé."}
		case "es":
			lines = []string{"FALLO DE PRUEBA DETECTADO — ES INTENCIONAL.", "El archivo interno de autoprueba fue detectado correctamente y ya se eliminó."}
		case "zh":
			lines = []string{"已检测到测试失败 — 这是故意造成的。", "内置自检文件已被正确识别并已删除。"}
		case "ru":
			lines = []string{"ОБНАРУЖЕН ТЕСТОВЫЙ СБОЙ — ОН СОЗДАН НАМЕРЕННО.", "Встроенный файл самотеста правильно распознан и уже удалён."}
		default:
			lines = []string{"TEST-FAIL DETECTED — THIS IS INTENTIONAL.", "The onboard self-test file was correctly recognized and has already been removed."}
		}
		return append(lines, selfTestFilesUntouchedLine(lang))
	}
	switch lang {
	case "nl":
		return []string{"ZELFTEST MISLUKT — de opzettelijke onboard TEST-FAIL werd NIET correct gedetecteerd.", "De echte Syzygy-controle wordt daarom NIET gestart; een betrouwbaar resultaat kan niet worden gegarandeerd."}
	case "de":
		return []string{"SELBSTTEST FEHLGESCHLAGEN — der absichtliche interne TEST-FEHLER wurde NICHT korrekt erkannt.", "Die echte Syzygy-Prüfung wird deshalb NICHT gestartet."}
	case "fr":
		return []string{"AUTO-TEST ÉCHOUÉ — l'ÉCHEC DE TEST interne volontaire n'a PAS été correctement détecté.", "La vérification réelle de Syzygy ne sera donc PAS lancée."}
	case "es":
		return []string{"AUTOPRUEBA FALLIDA — el FALLO DE PRUEBA interno intencional NO se detectó correctamente.", "Por ello NO se iniciará la comprobación real de Syzygy."}
	case "zh":
		return []string{"自检失败 — 内置的故意测试失败未被正确检测。", "因此不会启动真实的 Syzygy 检查。"}
	case "ru":
		return []string{"САМОТЕСТ НЕ ПРОЙДЕН — встроенный намеренный ТЕСТОВЫЙ СБОЙ НЕ был правильно обнаружен.", "Поэтому настоящая проверка Syzygy НЕ будет запущена."}
	default:
		return []string{"SELF-TEST FAILED — the onboard intentional TEST-FAIL was NOT detected correctly.", "The real Syzygy check will therefore NOT start; a reliable result cannot be guaranteed."}
	}
}

func selfTestHeading(lang string) string {
	switch lang {
	case "nl":
		return "Diagnostische zelftest: onboard TEST-FAIL tijdelijk plaatsen en controleren..."
	case "de":
		return "Diagnostischer Selbsttest: internen TEST-FEHLER temporär anlegen und prüfen..."
	case "fr":
		return "Auto-test de diagnostic : création temporaire de l'ÉCHEC DE TEST interne..."
	case "es":
		return "Autoprueba de diagnóstico: creando temporalmente el FALLO DE PRUEBA interno..."
	case "zh":
		return "诊断自检：正在临时创建并检查内置测试失败文件..."
	case "ru":
		return "Диагностический самотест: временное создание встроенного тестового сбоя..."
	default:
		return "Diagnostic self-test: temporarily creating and checking the onboard intentional FAIL fixture..."
	}
}

func continueAfterSelfTestPrompt(lang string) string {
	switch lang {
	case "nl":
		return "\nEnter = ECHTE Syzygy-controle starten   Esc = terug: "
	case "de":
		return "\nEnter = ECHTE Syzygy-Prüfung starten   Esc = zurück: "
	case "fr":
		return "\nEntrée = lancer la VRAIE vérification Syzygy   Échap = retour : "
	case "es":
		return "\nEnter = iniciar la comprobación REAL de Syzygy   Esc = volver: "
	case "zh":
		return "\nEnter = 开始真实的 Syzygy 检查   Esc = 返回："
	case "ru":
		return "\nEnter = начать НАСТОЯЩУЮ проверку Syzygy   Esc = назад: "
	default:
		return "\nEnter = start the REAL Syzygy check   Esc = back: "
	}
}
