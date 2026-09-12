package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	version               = "2.2.0"
	maxBatchFiles         = 8
	targetBatchBytes      = int64(16 * 1024 * 1024 * 1024) // 16 GiB: hard progress checkpoints stay close to one visible bar step on ~16 TB sets
	expectedTBCheckSHA256 = "9cb2f0ba2343f343bab29f9b9cd6aad6e4a2612702a93e06aa84a417f8eabbcc"
)

type textSet struct {
	all, language, mainMenu, check, change, exit, choiceMain string
	chooseDefault, remember, changeTitle, choiceLang         string
	found, onlyErrors, progress, checkTitle, done, mismatch  string
	engineStart, noFiles, pressEnter, helperMissing          string
	reportSaved, reportFailed, credit                        string
}

var texts = map[string]textSet{
	"en": {
		"All-Language", "Language", "MAIN MENU", "Check Syzygy files [default]", "Change language", "Exit", "Choice [0-2] (Enter = 1): ",
		"Choose your default language", "Your choice will be remembered for the next start.", "CHANGE LANGUAGE", "Choice [1-7] (Enter = %d; Esc = back): ",
		"Found: %d file(s) (.rtbw/.rtbz)", "Any errors found during the check are shown below.", "Progress:", "Syzygy checksum check", "Done: %d file(s) checked; %d error(s) found.", "ERROR: checksum mismatch - %s",
		"ERROR: the checksum engine could not be started: %v", "No .rtbw or .rtbz files were found in this folder.", "Press Enter to exit...", "ERROR: tbcheck.exe was not found next to SyzygyCheck. This v2.2.0 build uses the original tbcheck verification engine by Ronald de Man.",
		"Result report saved to: %s", "WARNING: result report could not be written: %v", "Based on the original Syzygy tablebase verification code by Ronald de Man. The AL interface, progress display, self-test, checkpointing and reporting are built around it.",
	},
	"de": {
		"Alle Sprachen", "Sprache", "HAUPTMENÜ", "Syzygy-Dateien prüfen [Standard]", "Sprache ändern", "Beenden", "Auswahl [0-2] (Enter = 1): ",
		"Standardsprache wählen", "Die Auswahl wird für den nächsten Start gespeichert.", "SPRACHE ÄNDERN", "Auswahl [1-7] (Enter = %d; Esc = zurück): ",
		"Gefunden: %d Datei(en) (.rtbw/.rtbz)", "Eventuelle Fehler werden während der Prüfung unten angezeigt.", "Fortschritt:", "Syzygy Prüfsummenprüfung", "Fertig: %d Datei(en) geprüft; %d Fehler gefunden.", "FEHLER: Prüfsumme stimmt nicht überein - %s",
		"FEHLER: Prüfsummen-Engine konnte nicht gestartet werden: %v", "Keine .rtbw- oder .rtbz-Dateien in diesem Ordner gefunden.", "Zum Beenden Enter drücken...", "FEHLER: tbcheck.exe wurde neben SyzygyCheck nicht gefunden. Diese Version v2.2.0 verwendet die ursprüngliche tbcheck-Prüfengine von Ronald de Man.",
		"Ergebnisbericht gespeichert unter: %s", "WARNUNG: Ergebnisbericht konnte nicht geschrieben werden: %v", "Basierend auf dem ursprünglichen Syzygy-Tablebase-Prüfcode von Ronald de Man. AL-Oberfläche, Fortschrittsanzeige, Selbsttest, Checkpointing und Bericht sind darum herum aufgebaut.",
	},
	"nl": {
		"Alle talen", "Taal", "HOOFDMENU", "Syzygy-bestanden controleren [standaard]", "Taal wijzigen", "Afsluiten", "Keuze [0-2] (Enter = 1): ",
		"Kies uw standaardtaal", "Uw keuze wordt voor de volgende start onthouden.", "TAAL WIJZIGEN", "Keuze [1-7] (Enter = %d; Esc = terug): ",
		"Gevonden: %d bestand(en) (.rtbw/.rtbz)", "Eventuele fouten worden tijdens de controle hieronder getoond.", "Voortgang:", "Syzygy checksumcontrole", "Klaar: %d bestand(en) gecontroleerd; %d fout(en) gevonden.", "FOUT: checksum komt niet overeen - %s",
		"FOUT: de checksum-engine kon niet worden gestart: %v", "Geen .rtbw- of .rtbz-bestanden in deze Controlemap gevonden.", "Druk op Enter om af te sluiten...", "FOUT: tbcheck.exe staat niet naast SyzygyCheck. Deze v2.2.0-versie gebruikt de oorspronkelijke tbcheck-verificatie-engine van Ronald de Man.",
		"Resultaatrapport opgeslagen op: %s", "WAARSCHUWING: resultaatrapport kon niet worden geschreven: %v", "Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man. AL-interface, voortgangsweergave, zelftest, checkpointing en rapportage zijn daaromheen gebouwd.",
	},
	"fr": {
		"Toutes langues", "Langue", "MENU PRINCIPAL", "Vérifier les fichiers Syzygy [défaut]", "Changer de langue", "Quitter", "Choix [0-2] (Entrée = 1) : ",
		"Choisissez votre langue par défaut", "Votre choix sera mémorisé au prochain démarrage.", "CHANGER DE LANGUE", "Choix [1-7] (Entrée = %d ; Échap = retour) : ",
		"Trouvé : %d fichier(s) (.rtbw/.rtbz)", "Les erreurs éventuelles sont affichées ci-dessous pendant le contrôle.", "Progression :", "Contrôle de somme Syzygy", "Terminé : %d fichier(s) vérifié(s) ; %d erreur(s).", "ERREUR : somme de contrôle incorrecte - %s",
		"ERREUR : le moteur de contrôle n'a pas pu démarrer : %v", "Aucun fichier .rtbw ou .rtbz trouvé dans ce dossier.", "Appuyez sur Entrée pour quitter...", "ERREUR : tbcheck.exe est introuvable à côté de SyzygyCheck. Cette version v2.2.0 utilise le moteur de vérification tbcheck original de Ronald de Man.",
		"Rapport enregistré dans : %s", "AVERTISSEMENT : le rapport n'a pas pu être écrit : %v", "Basé sur le code de vérification original des tablebases Syzygy de Ronald de Man. L’interface AL, la progression, l’auto-test, les points de reprise et le rapport sont construits autour de ce noyau.",
	},
	"es": {
		"Todos los idiomas", "Idioma", "MENÚ PRINCIPAL", "Comprobar archivos Syzygy [predeterminado]", "Cambiar idioma", "Salir", "Opción [0-2] (Enter = 1): ",
		"Elija su idioma predeterminado", "Su elección se recordará en el próximo inicio.", "CAMBIAR IDIOMA", "Opción [1-7] (Enter = %d; Esc = volver): ",
		"Encontrados: %d archivo(s) (.rtbw/.rtbz)", "Los posibles errores se muestran abajo durante la comprobación.", "Progreso:", "Comprobación de suma Syzygy", "Completado: %d archivo(s) comprobado(s); %d error(es).", "ERROR: la suma no coincide - %s",
		"ERROR: no se pudo iniciar el motor de comprobación: %v", "No se encontraron archivos .rtbw o .rtbz en esta carpeta.", "Pulse Enter para salir...", "ERROR: no se encontró tbcheck.exe junto a SyzygyCheck. Esta versión v2.2.0 usa el motor de verificación tbcheck original de Ronald de Man.",
		"Informe guardado en: %s", "ADVERTENCIA: no se pudo escribir el informe: %v", "Basado en el código original de verificación de tablebases Syzygy de Ronald de Man. La interfaz AL, el progreso, la autoprueba, los puntos de reanudación y el informe están construidos alrededor de ese núcleo.",
	},
	"zh": {
		"全语言", "语言", "主菜单", "检查 Syzygy 文件 [默认]", "更改语言", "退出", "选择 [0-2]（回车 = 1）：",
		"选择默认语言", "您的选择将在下次启动时保留。", "更改语言", "选择 [1-7]（回车 = %d；Esc = 返回）：",
		"找到：%d 个文件（.rtbw/.rtbz）", "检查过程中发现的错误会显示在下方。", "进度：", "Syzygy 校验和检查", "完成：已检查 %d 个文件；发现 %d 个错误。", "错误：校验和不匹配 - %s",
		"错误：无法启动校验和引擎：%v", "此文件夹中未找到 .rtbw 或 .rtbz 文件。", "按 Enter 退出...", "错误：SyzygyCheck 旁边未找到 tbcheck.exe。本 v2.2.0 版本使用 Ronald de Man 原始的 tbcheck 验证引擎。",
		"结果报告已保存到：%s", "警告：无法写入结果报告：%v", "基于 Ronald de Man 原始的 Syzygy 表库验证代码。AL 多语言界面、进度显示、自检、检查点和报告功能构建在该验证核心外围。",
	},
	"ru": {
		"Все языки", "Язык", "ГЛАВНОЕ МЕНЮ", "Проверить файлы Syzygy [по умолчанию]", "Изменить язык", "Выход", "Выбор [0-2] (Enter = 1): ",
		"Выберите язык по умолчанию", "Ваш выбор будет сохранён для следующего запуска.", "ИЗМЕНИТЬ ЯЗЫК", "Выбор [1-7] (Enter = %d; Esc = назад): ",
		"Найдено: %d файл(ов) (.rtbw/.rtbz)", "Ошибки, найденные во время проверки, отображаются ниже.", "Прогресс:", "Проверка контрольных сумм Syzygy", "Готово: проверено %d файл(ов); ошибок: %d.", "ОШИБКА: контрольная сумма не совпадает - %s",
		"ОШИБКА: не удалось запустить механизм проверки: %v", "В этой папке не найдены файлы .rtbw или .rtbz.", "Нажмите Enter для выхода...", "ОШИБКА: tbcheck.exe не найден рядом с SyzygyCheck. Эта версия v2.2.0 использует оригинальный движок проверки tbcheck Ronald de Man.",
		"Отчёт сохранён: %s", "ПРЕДУПРЕЖДЕНИЕ: отчёт не удалось записать: %v", "Основано на оригинальном коде проверки таблиц Syzygy Ronald de Man. Интерфейс AL, индикация прогресса, самотест, checkpoint и отчётность построены вокруг этого проверочного ядра.",
	},
}

var langOrder = []struct{ code, display string }{
	{"en", "English (English)"},
	{"de", "Deutsch (German)"},
	{"nl", "Nederlands (Dutch)"},
	{"fr", "Français (French)"},
	{"es", "Español (Spanish)"},
	{"zh", "中文 (Chinese)"},
	{"ru", "Русский (Russian)"},
}

type item struct {
	name string
	size int64
}

type batchResult struct {
	checked []string
	failed  []string
	output  string
	err     error
}

func main() {
	cleanupStaleIntentionalTestAtStartup()
	initConsole()
	r := bufio.NewReader(os.Stdin)
	lang, ok := loadLanguage()
	if !ok {
		lang = chooseLanguage(r, "en", true)
		saveLanguage(lang)
		clearConsoleScreen()
	}
	for {
		drawMain(lang)
		in, escaped := readMenuInput(r)
		if escaped {
			lang = chooseLanguage(r, lang, false)
			saveLanguage(lang)
			clearConsoleScreen()
			continue
		}
		if in == "" || in == "1" {
			if runCheck(r, lang) {
				return
			}
			clearConsoleScreen()
			continue
		}
		switch in {
		case "0":
			return
		case "2":
			lang = chooseLanguage(r, lang, false)
			saveLanguage(lang)
			clearConsoleScreen()
		default:
			clearConsoleScreen()
		}
	}
}

func drawMain(lang string) {
	t := texts[lang]
	fmt.Printf("SyzygyCheck v%s - %s\n", version, t.all)
	fmt.Printf("%s: %s\n", t.language, displayFor(lang))
	printWrappedStatic(t.credit)
	fmt.Println()
	fmt.Println(t.mainMenu)
	fmt.Printf("  1 = %s\n", t.check)
	fmt.Printf("  2 = %s\n", languageEscapeLabel(lang, t.change))
	fmt.Printf("  0 = %s\n", t.exit)
	fmt.Print(t.choiceMain)
}

func chooseLanguage(r *bufio.Reader, current string, first bool) string {
	t := texts[current]
	if first {
		fmt.Println(t.chooseDefault)
		fmt.Println(t.remember)
	} else {
		fmt.Printf("\n%s\n", t.changeTitle)
	}
	for i, x := range langOrder {
		fmt.Printf("  %d = %s\n", i+1, x.display)
	}
	def := langIndex(current) + 1
	fmt.Printf(t.choiceLang, def)
	in, escaped := readMenuInput(r)
	if escaped || in == "" {
		return current
	}
	n, err := strconv.Atoi(in)
	if err == nil && n >= 1 && n <= len(langOrder) {
		return langOrder[n-1].code
	}
	return current
}

func runCheck(r *bufio.Reader, lang string) bool {
	t := texts[lang]
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("\n"+t.engineStart+"\n", err)
		waitEnter(r, t.pressEnter)
		return true
	}
	dir := filepath.Dir(exe)

	// Before doing any diagnostic or tablebase work, give the user a short
	// virtual-memory/pagefile safety gate. Huge Syzygy collections can run for
	// many hours; this is the right moment to stop and adjust Windows first.
	if !pagefileSafetyGate(r, lang) {
		return false
	}

	// Detect the logical processors Windows/Go makes available to this process.
	// The user can keep the balanced half-thread default, use everything, or
	// choose a precise value. This affects the tbcheck by Ronald de Man via
	// --threads; the AL wrapper does not reimplement checksum work.
	detectedThreads := runtime.NumCPU()
	workers, ok := chooseWorkerThreads(r, lang, detectedThreads)
	if !ok {
		return false
	}

	// The self-test is deliberately independent of the user's tablebases.
	// It runs BEFORE inventory, so no real .rtbw/.rtbz is touched for testing.
	fmt.Printf("\n%s\n", selfTestHeading(lang))
	helperHandle, err := acquireTBCheck(dir)
	if err != nil {
		fmt.Printf("\n"+t.engineStart+"\n", err)
		waitEnter(r, t.pressEnter)
		return true
	}
	defer helperHandle.Close()

	testResult := runAutomaticIntentionalTest(helperHandle.Path, dir)
	fmt.Println()
	for _, line := range intentionalTestScreenLines(lang, testResult) {
		fmt.Println(line)
	}
	if !testResult.ExpectedFailDetected {
		reportFolder, _ := chooseReportFolder(r, lang, dir)
		reportPath, reportErr := writeSelfTestFailureReportV6(lang, dir, reportFolder, helperHandle.Origin, testResult)
		if reportErr == nil {
			fmt.Println()
			for _, line := range reportSavedScreenLines(lang, reportPath) {
				fmt.Println(line)
			}
		} else {
			fmt.Printf("\n"+t.reportFailed+"\n", reportErr)
		}
		fmt.Println()
		waitEnter(r, t.pressEnter)
		return true
	}

	// Keep the self-test result on screen until the user explicitly continues.
	if !waitContinueOrBack(r, continueAfterSelfTestPrompt(lang)) {
		return false
	}
	recursive, ok := chooseScanScope(r, lang)
	if !ok {
		return false
	}

	// Only now inventory the real database: same directory as the EXE, no
	// parent folders. Subfolders are included only when explicitly selected.
	files, totalBytes, err := discoverFilesInScope(dir, recursive)
	if err != nil {
		fmt.Printf("\n"+t.engineStart+"\n", err)
		waitEnter(r, t.pressEnter)
		return true
	}
	if len(files) == 0 {
		fmt.Printf("\n%s\n", noFilesInScope(lang, recursive))
		waitEnter(r, t.pressEnter)
		return true
	}

	scan := runRealScan(r, lang, helperHandle.Path, helperHandle.Origin, dir, files, totalBytes, workers, recursive)
	if scan.Cancelled {
		return false
	}
	if scan.Ended.IsZero() {
		scan.Ended = time.Now()
	}
	reportFolder, _ := chooseReportFolder(r, lang, dir)
	reportPath, reportErr := writeReportV6(lang, dir, reportFolder, files, totalBytes, scan, testResult)

	clearConsoleScreen()
	fmt.Printf("SyzygyCheck v%s - %s\n", version, t.all)
	fmt.Printf("%s: %s\n", t.language, displayFor(lang))
	printWrappedStatic(t.credit)
	fmt.Println()
	fmt.Println(t.checkTitle)
	fmt.Printf("%s: %s\n", mapLabel(lang), dir)
	fmt.Printf("%s: %s\n", scanScopeLabel(lang), scanScopeDescription(lang, scan.Recursive))
	fmt.Printf(t.found+"\n\n", len(files))
	fmt.Printf(t.done+"\n", scan.CompletedFiles, len(scan.Failures)+len(scan.Errors))
	if scan.FatalErr != nil {
		fmt.Printf("\n"+t.engineStart+"\n", scan.FatalErr)
	}
	if reportErr == nil {
		fmt.Println()
		for _, line := range reportSavedScreenLines(lang, reportPath) {
			fmt.Println(line)
		}
	} else {
		fmt.Printf("\n"+t.reportFailed+"\n", reportErr)
	}
	fmt.Println()
	waitEnter(r, t.pressEnter)
	return true
}

func discoverFiles(dir string) ([]item, int64, error) {
	return discoverFilesInScope(dir, false)
}

func discoverFilesInScope(dir string, recursive bool) ([]item, int64, error) {
	var files []item
	var total int64
	add := func(name string, e fs.DirEntry) error {
		if e.Type()&os.ModeSymlink != 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".rtbw" && ext != ".rtbz" {
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, item{name: name, size: info.Size()})
		total += info.Size()
		return nil
	}

	if recursive {
		err := filepath.WalkDir(dir, func(path string, e fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			// A scan started from a Windows drive root must never descend into
			// operating-system metadata. $RECYCLE.BIN may contain deleted .rtbw
			// or .rtbz files which are intentionally inaccessible through their
			// recycle-bin storage path; passing one to tbcheck would abort an
			// otherwise valid whole-drive scan. System Volume Information is the
			// other standard protected directory at a volume root.
			if e.IsDir() && isWindowsVolumeMetadataDir(dir, path) {
				return fs.SkipDir
			}
			if e.IsDir() {
				return nil
			}
			// WalkDir does not follow symbolic directory links. Keeping only a
			// validated relative descendant path also makes the no-parent rule
			// explicit, even on unusual Windows folder layouts.
			rel, err := filepath.Rel(dir, path)
			if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				if err != nil {
					return err
				}
				return fmt.Errorf("refusing path outside the control folder: %s", path)
			}
			return add(rel, e)
		})
		if err != nil {
			return nil, 0, err
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, 0, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if err := add(e.Name(), e); err != nil {
				return nil, 0, err
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].name) < strings.ToLower(files[j].name) })
	return files, total, nil
}

func isWindowsVolumeMetadataDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) || filepath.Dir(rel) != "." {
		return false
	}
	base := filepath.Base(rel)
	return strings.EqualFold(base, "$RECYCLE.BIN") || strings.EqualFold(base, "System Volume Information")
}

func makeBatches(files []item, maxFiles int) [][]item {
	return makeBatchesBySize(files, maxFiles, 0)
}

// makeBatchesBySize keeps the multi-file tbcheck behaviour from Ronald de Man while
// also limiting the amount of data between hard, checksum-confirmed progress
// checkpoints. A single tablebase larger than targetBytes is never split: it
// remains one intact tbcheck argument and live activity is shown separately.
func makeBatchesBySize(files []item, maxFiles int, targetBytes int64) [][]item {
	if maxFiles < 1 {
		maxFiles = 1
	}
	var out [][]item
	for len(files) > 0 {
		n := 0
		var bytes int64
		for n < len(files) && n < maxFiles {
			next := files[n].size
			if n > 0 && targetBytes > 0 && bytes+next > targetBytes {
				break
			}
			bytes += next
			n++
			if targetBytes > 0 && bytes >= targetBytes {
				break
			}
		}
		if n == 0 { // defensive only; guarantees forward progress
			n = 1
		}
		b := append([]item(nil), files[:n]...)
		out = append(out, b)
		files = files[n:]
	}
	return out
}

func sumItemBytes(items []item) int64 {
	var n int64
	for _, f := range items {
		n += f.size
	}
	return n
}

func parseTBCheckOutput(out string) (checked, failed []string) {
	s := bufio.NewScanner(strings.NewReader(out))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasSuffix(line, ": OK!") {
			checked = append(checked, strings.TrimSpace(strings.TrimSuffix(line, ": OK!")))
		} else if strings.HasSuffix(line, ": FAIL!") {
			failed = append(failed, strings.TrimSpace(strings.TrimSuffix(line, ": FAIL!")))
		}
	}
	return checked, failed
}

func drawRunHeader(lang, dir string, n, workers int, recursive bool) {
	t := texts[lang]
	fmt.Printf("SyzygyCheck v%s - %s\n", version, t.all)
	fmt.Printf("%s: %s\n", t.language, displayFor(lang))
	printWrappedStatic(t.credit)
	fmt.Println()
	// Keep the active location visible throughout long checks. This restores
	// the useful V1-NL behaviour: when a scan runs for many hours, the user
	// can immediately see which tablebase directory must be left untouched.
	fmt.Println(t.checkTitle)
	fmt.Printf("%s: %s\n", mapLabel(lang), dir)
	fmt.Printf("%s: %s\n", scanScopeLabel(lang), scanScopeDescription(lang, recursive))
	fmt.Printf(t.found+"\n", n)
	fmt.Printf("%s: %d\n", workersSummaryLabel(lang), workers)
	fmt.Println(t.onlyErrors)
	fmt.Printf("\n%s\n\n", t.progress)
}

func renderProgress(width, completedFiles, totalFiles int, displayBytes, exactBytes, totalBytes int64, started time.Time, batchIdx, batchCount, batchSize, spin int, liveFromIO bool) {
	line := formatProgressLine(width, completedFiles, totalFiles, displayBytes, exactBytes, totalBytes, started, batchIdx, batchCount, batchSize, spin, liveFromIO)
	clearCurrentLine()
	fmt.Print(line)
}

func formatProgressLine(width, completedFiles, totalFiles int, displayBytes, exactBytes, totalBytes int64, started time.Time, batchIdx, batchCount, batchSize, spin int, liveFromIO bool) string {
	if width < 20 {
		width = 20
	}
	if exactBytes < 0 {
		exactBytes = 0
	}
	if displayBytes < exactBytes {
		displayBytes = exactBytes
	}
	if totalBytes > 0 && displayBytes > totalBytes {
		displayBytes = totalBytes
	}

	pctData := 0.0
	if totalBytes > 0 {
		pctData = float64(displayBytes) * 100 / float64(totalBytes)
	}
	elapsed := time.Since(started)

	rateBasis := exactBytes
	approxRate := false
	if liveFromIO && displayBytes > exactBytes {
		rateBasis = displayBytes
		approxRate = true
	}
	rate := 0.0
	if rateBasis > 0 && elapsed > 0 {
		rate = float64(rateBasis) / elapsed.Seconds()
	}
	eta := "ETA ..."
	if rate > 0 {
		remain := totalBytes - rateBasis
		if remain < 0 {
			remain = 0
		}
		prefix := "ETA "
		if approxRate {
			prefix = "~ETA "
		}
		eta = prefix + fmtDuration(time.Duration(float64(remain)/rate)*time.Second)
	}

	spinners := []string{"|", "/", "-", "\\"}
	activity := spinners[spin%len(spinners)]
	approx := ""
	if liveFromIO && displayBytes > exactBytes {
		approx = "~"
	}

	// Never write into the final visible console cell. Some Windows console
	// hosts auto-wrap there; leaving one spare cell keeps the live status line
	// resize/reflow safe and prevents accidental line feeds while it is active.
	limit := width - 1
	if limit < 1 {
		limit = 1
	}
	prefix := activity + " "

	filesPart := fmt.Sprintf("F%d/%d", completedFiles, totalFiles)
	batchPart := fmt.Sprintf("B%d/%d", batchIdx, batchCount)
	dataPart := fmt.Sprintf("%8.4f%% %s%s/%s", pctData, approx, humanSizeCompact(displayBytes), humanSizeCompact(totalBytes))
	ratePart := ""
	if rate > 0 {
		ratePart = humanRateCompact(rate)
		if approxRate {
			ratePart = "~" + ratePart
		}
	}

	// Start with the richest useful status. If the console is narrow, drop
	// secondary fields before sacrificing the bar. The bar receives all cells
	// that remain and therefore automatically scales with the actual terminal.
	tails := []string{
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, filesPart, batchPart, ratePart, eta), " ")),
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, filesPart, ratePart, eta), " ")),
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, filesPart, eta), " ")),
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, eta), " ")),
		dataPart,
		fmt.Sprintf("%8.4f%%", pctData),
	}

	chosenTail := tails[len(tails)-1]
	barW := limit - runeLen(prefix) - runeLen(" "+chosenTail) - 2 // [ and ]
	desiredBar := limit * 60 / 100
	if desiredBar < 24 {
		desiredBar = 24
	}
	if desiredBar > 120 {
		desiredBar = 120
	}
	for _, tail := range tails {
		candidate := " " + tail
		w := limit - runeLen(prefix) - runeLen(candidate) - 2 // [ and ]
		if w >= desiredBar {
			chosenTail = tail
			barW = w
			break
		}
	}
	if barW < 1 {
		barW = 1
	}
	bar := progressBarFloat(pctData, barW)
	line := prefix + bar + " " + chosenTail
	if runeLen(line) > limit {
		line = truncateMiddle(line, limit)
	}
	// Defensive guarantee for the live renderer: never emit control linefeeds.
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, "\n", "")
	return line
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func progressBarFloat(pct float64, w int) string {
	if w < 1 {
		w = 1
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	units := pct * float64(w) / 100.0
	full := int(units)
	frac := units - float64(full)
	partials := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	p := int(frac * 8)
	if p > 7 {
		p = 7
	}
	used := full
	mid := ""
	if p > 0 && full < w {
		mid = partials[p]
		used++
	}
	if full > w {
		full = w
		used = w
		mid = ""
	}
	return "[" + strings.Repeat("█", full) + mid + strings.Repeat("░", w-used) + "]"
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func humanSize(n int64) string {
	const kb = 1024
	const mb = 1024 * kb
	const gb = 1024 * mb
	const tb = 1024 * gb
	switch {
	case n >= tb:
		return fmt.Sprintf("%.2f TB", float64(n)/float64(tb))
	case n >= gb:
		return fmt.Sprintf("%.2f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func humanSizeCompact(n int64) string {
	const kb = 1024
	const mb = 1024 * kb
	const gb = 1024 * mb
	const tb = 1024 * gb
	switch {
	case n >= tb:
		return fmt.Sprintf("%.2fT", float64(n)/float64(tb))
	case n >= gb:
		return fmt.Sprintf("%.1fG", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1fM", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1fK", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func humanRateCompact(bytesPerSecond float64) string {
	if bytesPerSecond <= 0 {
		return ""
	}
	const kb = 1024.0
	const mb = 1024.0 * kb
	const gb = 1024.0 * mb
	switch {
	case bytesPerSecond >= gb:
		return fmt.Sprintf("%.2fG/s", bytesPerSecond/gb)
	case bytesPerSecond >= mb:
		return fmt.Sprintf("%.0fM/s", bytesPerSecond/mb)
	case bytesPerSecond >= kb:
		return fmt.Sprintf("%.0fK/s", bytesPerSecond/kb)
	default:
		return fmt.Sprintf("%.0fB/s", bytesPerSecond)
	}
}

func fmtDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d.Seconds())
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}

func truncateMiddle(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if runeLen(s) <= max {
		return s
	}
	if max <= 3 {
		head, _ := splitAtConsoleCells(s, max)
		return head
	}
	leftCells := (max - 1) / 2
	rightCells := max - 1 - leftCells
	left, _ := splitAtConsoleCells(s, leftCells)
	right := suffixAtConsoleCells(s, rightCells)
	return left + "…" + right
}

// runeLen returns displayed console cells, not the number of Unicode code
// points. CJK and emoji characters normally occupy two Windows-console cells;
// combining marks occupy none. The historic name is retained because this is
// the width primitive used throughout the existing progress renderer.
func runeLen(s string) int {
	width := 0
	for _, r := range s {
		width += consoleRuneWidth(r)
	}
	return width
}

func consoleRuneWidth(r rune) int {
	if r == 0 || r < 0x20 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) {
		return 0
	}
	if r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf) || (r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) || (r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) || (r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) || (r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

func splitAtConsoleCells(s string, max int) (head, tail string) {
	if max <= 0 || s == "" {
		return "", s
	}
	used := 0
	cut := 0
	for i, r := range s {
		w := consoleRuneWidth(r)
		if used+w > max {
			break
		}
		used += w
		cut = i + utf8.RuneLen(r)
	}
	if cut == 0 {
		// A two-cell glyph cannot fit in a one-cell remainder. Consume it and
		// show a one-cell continuation marker to guarantee forward progress.
		_, size := utf8.DecodeRuneInString(s)
		return "…", s[size:]
	}
	return s[:cut], s[cut:]
}

func suffixAtConsoleCells(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	used := 0
	start := len(runes)
	for start > 0 {
		w := consoleRuneWidth(runes[start-1])
		if used+w > max {
			break
		}
		used += w
		start--
	}
	return string(runes[start:])
}

func configDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("USERPROFILE")
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "SyzygyCheck")
}

func configPath() string { return filepath.Join(configDir(), "language.txt") }
func loadLanguage() (string, bool) {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return "en", false
	}
	s := strings.TrimSpace(string(b))
	if _, ok := texts[s]; !ok {
		return "en", false
	}
	return s, true
}
func saveLanguage(lang string) {
	p := configPath()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	_ = os.WriteFile(p, []byte(lang+"\n"), 0644)
}
func displayFor(code string) string {
	for _, x := range langOrder {
		if x.code == code {
			return x.display
		}
	}
	return code
}
func langIndex(code string) int {
	for i, x := range langOrder {
		if x.code == code {
			return i
		}
	}
	return 0
}
func readLine(r *bufio.Reader) string          { s, _ := r.ReadString('\n'); return strings.TrimSpace(s) }
func waitEnter(r *bufio.Reader, prompt string) { fmt.Print(prompt); _, _ = r.ReadString('\n') }
func waitContinueOrBack(r *bufio.Reader, prompt string) bool {
	fmt.Print(prompt)
	_, escaped := readMenuInput(r)
	return !escaped
}
func mapLabel(lang string) string {
	switch lang {
	case "de":
		return "Ordner"
	case "nl":
		return "Controlemap"
	case "fr":
		return "Dossier"
	case "es":
		return "Carpeta"
	case "zh":
		return "文件夹"
	case "ru":
		return "Папка"
	default:
		return "Folder"
	}
}
