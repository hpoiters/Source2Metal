package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const version = "2.0.7"

//go:embed tbcheck.exe
var tbcheckEXE []byte

type textSet struct {
	all, language, mainMenu, check, change, exit, choiceMain                                  string
	chooseDefault, remember, changeTitle, choiceLang                                          string
	found, onlyErrors, progress, checkTitle, done, mismatch, engineStart, noFiles, pressEnter string
}

var texts = map[string]textSet{
	"en": {
		"All-Language", "Language", "MAIN MENU", "Check Syzygy files [default]", "Change language (Изменить язык / 更改语言)", "Exit", "Choice [0-2] (Enter = 1): ",
		"Choose your default language", "Your choice will be remembered for the next start.", "CHANGE LANGUAGE", "Choice [1-7] (Enter = %d): ",
		"Found: %d file(s) (.rtbw/.rtbz)", "Only errors are shown below.", "Progress:", "Syzygy RTB checksum check", "Done: %d file(s) checked; %d error(s) found.", "ERROR: checksum mismatch - %s", "ERROR: the checksum engine could not be started: %v", "No .rtbw or .rtbz files were found in this folder.", "Press Enter to exit...",
	},
	"de": {
		"Alle Sprachen", "Sprache", "HAUPTMENÜ", "Syzygy-Dateien prüfen [Standard]", "Sprache ändern", "Beenden", "Auswahl [0-2] (Enter = 1): ",
		"Standardsprache wählen", "Die Auswahl wird für den nächsten Start gespeichert.", "SPRACHE ÄNDERN", "Auswahl [1-7] (Enter = %d): ",
		"Gefunden: %d Datei(en) (.rtbw/.rtbz)", "Nur Fehler werden unten angezeigt.", "Fortschritt:", "Syzygy RTB Prüfsummenprüfung", "Fertig: %d Datei(en) geprüft; %d Fehler gefunden.", "FEHLER: Prüfsumme stimmt nicht überein - %s", "FEHLER: Prüfsummen-Engine konnte nicht gestartet werden: %v", "Keine .rtbw- oder .rtbz-Dateien in diesem Ordner gefunden.", "Zum Beenden Enter drücken...",
	},
	"nl": {
		"Alle talen", "Taal", "HOOFDMENU", "Syzygy-bestanden controleren [standaard]", "Taal wijzigen", "Afsluiten", "Keuze [0-2] (Enter = 1): ",
		"Kies uw standaardtaal", "Uw keuze wordt voor de volgende start onthouden.", "TAAL WIJZIGEN", "Keuze [1-7] (Enter = %d): ",
		"Gevonden: %d bestand(en) (.rtbw/.rtbz)", "Alleen fouten worden hieronder getoond.", "Voortgang:", "Syzygy RTB checksumcontrole", "Klaar: %d bestand(en) gecontroleerd; %d fout(en) gevonden.", "FOUT: checksum komt niet overeen - %s", "FOUT: de checksum-engine kon niet worden gestart: %v", "Geen .rtbw- of .rtbz-bestanden in deze map gevonden.", "Druk op Enter om af te sluiten...",
	},
	"fr": {
		"Toutes langues", "Langue", "MENU PRINCIPAL", "Vérifier les fichiers Syzygy [défaut]", "Changer de langue", "Quitter", "Choix [0-2] (Entrée = 1) : ",
		"Choisissez votre langue par défaut", "Votre choix sera mémorisé pour le prochain démarrage.", "CHANGER DE LANGUE", "Choix [1-7] (Entrée = %d) : ",
		"Trouvé : %d fichier(s) (.rtbw/.rtbz)", "Seules les erreurs sont affichées ci-dessous.", "Progression :", "Contrôle de somme Syzygy RTB", "Terminé : %d fichier(s) vérifié(s) ; %d erreur(s) trouvée(s).", "ERREUR : la somme de contrôle ne correspond pas - %s", "ERREUR : le moteur de somme de contrôle n'a pas pu démarrer : %v", "Aucun fichier .rtbw ou .rtbz trouvé dans ce dossier.", "Appuyez sur Entrée pour quitter...",
	},
	"es": {
		"Todos los idiomas", "Idioma", "MENÚ PRINCIPAL", "Comprobar archivos Syzygy [predeterminado]", "Cambiar idioma", "Salir", "Opción [0-2] (Enter = 1): ",
		"Elija su idioma predeterminado", "Su elección se recordará en el próximo inicio.", "CAMBIAR IDIOMA", "Opción [1-7] (Enter = %d): ",
		"Encontrados: %d archivo(s) (.rtbw/.rtbz)", "Solo se muestran errores abajo.", "Progreso:", "Comprobación de suma Syzygy RTB", "Completado: %d archivo(s) comprobado(s); %d error(es) encontrado(s).", "ERROR: la suma de comprobación no coincide - %s", "ERROR: no se pudo iniciar el motor de comprobación: %v", "No se encontraron archivos .rtbw o .rtbz en esta carpeta.", "Pulse Enter para salir...",
	},
	"zh": {
		"全语言", "语言", "主菜单", "检查 Syzygy 文件 [默认]", "更改语言 (Change language / Изменить язык)", "退出", "选择 [0-2]（回车 = 1）：",
		"选择默认语言", "您的选择将在下次启动时保留。", "更改语言", "选择 [1-7]（回车 = %d）：",
		"找到：%d 个文件（.rtbw/.rtbz）", "下方只显示错误。", "进度：", "Syzygy RTB 校验和检查", "完成：已检查 %d 个文件；发现 %d 个错误。", "错误：校验和不匹配 - %s", "错误：无法启动校验和引擎：%v", "此文件夹中未找到 .rtbw 或 .rtbz 文件。", "按 Enter 退出...",
	},
	"ru": {
		"Все языки", "Язык", "ГЛАВНОЕ МЕНЮ", "Проверить файлы Syzygy [по умолчанию]", "Изменить язык (Change language / 更改语言)", "Выход", "Выбор [0-2] (Enter = 1): ",
		"Выберите язык по умолчанию", "Ваш выбор будет сохранён для следующего запуска.", "ИЗМЕНИТЬ ЯЗЫК", "Выбор [1-7] (Enter = %d): ",
		"Найдено: %d файл(ов) (.rtbw/.rtbz)", "Ниже показываются только ошибки.", "Прогресс:", "Проверка контрольных сумм Syzygy RTB", "Готово: проверено %d файл(ов); найдено ошибок: %d.", "ОШИБКА: контрольная сумма не совпадает - %s", "ОШИБКА: не удалось запустить механизм проверки: %v", "В этой папке не найдены файлы .rtbw или .rtbz.", "Нажмите Enter для выхода...",
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

func main() {
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
		in := readLine(r)
		if in == "" || in == "1" {
			runCheck(r, lang)
			return
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
	fmt.Printf("%s: %s\n\n", t.language, displayFor(lang))
	fmt.Println(t.mainMenu)
	fmt.Printf("  1 = %s\n", t.check)
	fmt.Printf("  2 = %s\n", t.change)
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
	in := readLine(r)
	if in == "" {
		return current
	}
	n, err := strconv.Atoi(in)
	if err == nil && n >= 1 && n <= len(langOrder) {
		return langOrder[n-1].code
	}
	return current
}

func runCheck(r *bufio.Reader, lang string) {
	t := texts[lang]
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("\n"+t.engineStart+"\n", err)
		waitEnter(r, t.pressEnter)
		return
	}
	dir := filepath.Dir(exe)
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("\n"+t.engineStart+"\n", err)
		waitEnter(r, t.pressEnter)
		return
	}
	type item struct {
		name, path string
		size       int64
	}
	var files []item
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".rtbw" && ext != ".rtbz" {
			continue
		}
		info, e2 := e.Info()
		if e2 != nil {
			continue
		}
		files = append(files, item{e.Name(), filepath.Join(dir, e.Name()), info.Size()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].name) < strings.ToLower(files[j].name) })
	if len(files) == 0 {
		fmt.Printf("\n%s\n", t.noFiles)
		waitEnter(r, t.pressEnter)
		return
	}

	helper, err := extractHelper()
	if err != nil {
		fmt.Printf("\n"+t.engineStart+"\n", err)
		waitEnter(r, t.pressEnter)
		return
	}
	defer os.Remove(helper)

	fmt.Printf("\n"+t.found+"\n", len(files))
	fmt.Println(t.onlyErrors)
	fmt.Printf("\n%s\n\n", t.progress)

	var completed int64
	var completedElapsed time.Duration
	var failures []string
	lastWidth := visibleConsoleWidth()

	for i, f := range files {
		fileStart := time.Now()
		done := make(chan engineResult, 1)
		go func(path string) { done <- checkOne(helper, path) }(f.path)

		ticker := time.NewTicker(250 * time.Millisecond)
		var res engineResult
		running := true
		for running {
			select {
			case res = <-done:
				running = false
			case <-ticker.C:
				w := visibleConsoleWidth()
				if w != lastWidth {
					lastWidth = w
					clearConsoleScreen()
					drawRunHeader(lang, dir, len(files))
				}
				renderProgress(lastWidth, completed, total, completedElapsed, i+1, len(files), f.name, f.size, fileStart)
			}
		}
		ticker.Stop()
		renderProgress(lastWidth, completed, total, completedElapsed, i+1, len(files), f.name, f.size, fileStart)

		if res.engineErr != nil && !res.fail {
			clearCurrentLine()
			fmt.Printf(t.engineStart+"\n", res.engineErr)
			failures = append(failures, fmt.Sprintf(t.mismatch, f.name))
		} else if res.fail {
			failures = append(failures, fmt.Sprintf(t.mismatch, f.name))
			// Brief live feedback, then resume a clean single progress line.
			clearCurrentLine()
			fmt.Print(fmt.Sprintf(t.mismatch, f.name))
			time.Sleep(900 * time.Millisecond)
			clearCurrentLine()
		}
		completed += f.size
		completedElapsed += time.Since(fileStart)
	}

	// A clean final screen is deliberate. It removes any visual remnants that
	// Windows Terminal may have reflowed while the user resized the window.
	clearConsoleScreen()
	fmt.Printf("SyzygyCheck v%s - %s\n", version, t.all)
	fmt.Printf("%s: %s\n\n", t.language, displayFor(lang))
	fmt.Println(t.checkTitle)
	fmt.Printf("%s: %s\n", mapLabel(lang), dir)
	fmt.Printf(t.found+"\n\n", len(files))
	fmt.Printf(t.done+"\n", len(files), len(failures))
	for _, f := range failures {
		fmt.Println(f)
	}
	fmt.Println()
	waitEnter(r, t.pressEnter)
}

type engineResult struct {
	fail      bool
	engineErr error
}

func checkOne(helper, file string) engineResult {
	cmd := exec.Command(helper, file)
	cmd.Dir = filepath.Dir(file)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	err := cmd.Run()
	out := b.String()
	if strings.Contains(out, "FAIL!") {
		return engineResult{fail: true}
	}
	if strings.Contains(out, "OK!") && err == nil {
		return engineResult{}
	}
	if err != nil {
		return engineResult{engineErr: fmt.Errorf("%v", err)}
	}
	if strings.Contains(out, "ERROR:") {
		return engineResult{engineErr: fmt.Errorf("tbcheck error")}
	}
	// A helper that exits cleanly without the expected OK marker is treated as
	// an engine error rather than silently accepting the file.
	return engineResult{engineErr: fmt.Errorf("unexpected tbcheck result")}
}

func extractHelper() (string, error) {
	f, err := os.CreateTemp("", "SyzygyCheck_tbcheck_*.exe")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err = f.Write(tbcheckEXE); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err = f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func drawRunHeader(lang, dir string, n int) {
	t := texts[lang]
	fmt.Printf("SyzygyCheck v%s - %s\n", version, t.all)
	fmt.Printf("%s: %s\n\n", t.language, displayFor(lang))
	fmt.Printf(t.found+"\n", n)
	fmt.Println(t.onlyErrors)
	fmt.Printf("\n%s\n\n", t.progress)
}

func renderProgress(width int, completed, total int64, completedElapsed time.Duration, idx, n int, name string, size int64, fileStart time.Time) {
	if width < 20 {
		width = 20
	}
	pct := 0
	if total > 0 {
		pct = int(float64(completed) * 100 / float64(total))
		if pct > 100 {
			pct = 100
		}
	}
	eta := "ETA ..."
	if completed > 0 && completedElapsed > 0 {
		rate := float64(completed) / completedElapsed.Seconds()
		if rate > 0 {
			remain := total - completed
			if remain < 0 {
				remain = 0
			}
			eta = "ETA ca. " + fmtDuration(time.Duration(float64(remain)/rate)*time.Second)
		}
	}
	elapsedFile := fmtDuration(time.Since(fileStart))
	sz := humanSize(size)
	barW := 20
	full := func(bw int, nm string, withSize bool) string {
		bar := progressBar(pct, bw)
		if withSize {
			return fmt.Sprintf("%s %d%% data | %d/%d | %s | %s | %s | %s", bar, pct, idx, n, nm, sz, elapsedFile, eta)
		}
		return fmt.Sprintf("%s %d%% data | %d/%d | %s | %s | %s", bar, pct, idx, n, nm, elapsedFile, eta)
	}
	limit := width - 1
	line := full(barW, name, true)
	if runeLen(line) > limit {
		available := limit - (runeLen(full(barW, "", true)))
		if available < 8 {
			available = 8
		}
		line = full(barW, truncateMiddle(name, available), true)
	}
	if runeLen(line) > limit {
		barW = 10
		available := limit - runeLen(full(barW, "", false))
		if available < 8 {
			available = 8
		}
		line = full(barW, truncateMiddle(name, available), false)
	}
	if runeLen(line) > limit {
		base := fmt.Sprintf("%d%% | %d/%d | %s | %s", pct, idx, n, name, eta)
		over := runeLen(base) - limit
		if over > 0 {
			name = truncateMiddle(name, maxInt(6, runeLen(name)-over))
		}
		line = fmt.Sprintf("%d%% | %d/%d | %s | %s", pct, idx, n, name, eta)
	}
	if runeLen(line) > limit {
		line = fmt.Sprintf("%d/%d | %s", idx, n, truncateMiddle(name, maxInt(4, limit-10)))
	}
	clearCurrentLine()
	fmt.Print(line)
}

func progressBar(pct, w int) string {
	if w < 1 {
		w = 1
	}
	fill := pct * w / 100
	if fill > w {
		fill = w
	}
	return "[" + strings.Repeat("█", fill) + strings.Repeat("░", w-fill) + "]"
}

func humanSize(n int64) string {
	const mb = 1024 * 1024
	const gb = 1024 * mb
	if n >= gb {
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	}
	return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
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
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	left := (max - 1) / 2
	right := max - 1 - left
	return string(r[:left]) + "…" + string(r[len(r)-right:])
}
func runeLen(s string) int { return utf8.RuneCountInString(s) }
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func configPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("USERPROFILE")
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "SyzygyCheck", "language.txt")
}
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
func mapLabel(lang string) string {
	switch lang {
	case "de":
		return "Ordner"
	case "nl":
		return "Map"
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

func init() { _ = runtime.GOOS }
