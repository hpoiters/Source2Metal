package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// printWrappedStatic keeps explanatory/static text readable when the console
// is narrow. Live progress has its own stricter one-line renderer.
func printWrappedStatic(text string) {
	width := visibleConsoleWidth() - 1
	if width < 11 {
		width = 11
	}
	for _, line := range wrapStaticText(text, width) {
		fmt.Println(line)
	}
}

// printWrappedPrefixedStatic gives menu items a hanging indent and keeps every
// emitted line inside the current visible console width. Long path tokens are
// split safely instead of relying on the console host's automatic wrapping.
func printWrappedPrefixedStatic(prefix, text string) {
	width := visibleConsoleWidth() - 1
	if width < 11 {
		width = 11
	}
	textWidth := width - runeLen(prefix)
	if textWidth < 1 {
		textWidth = 1
	}
	lines := wrapStaticText(text, textWidth)
	indent := strings.Repeat(" ", runeLen(prefix))
	for i, line := range lines {
		if i == 0 {
			fmt.Println(prefix + line)
		} else {
			fmt.Println(indent + line)
		}
	}
}

// printWrappedPrompt wraps a dynamic prompt but deliberately leaves its last
// line open so redirected input and a physical Windows console behave alike.
func printWrappedPrompt(text string) {
	width := visibleConsoleWidth() - 1
	if width < 11 {
		width = 11
	}
	// Reserve one final cell for the visible space between the colon and input.
	lines := wrapStaticText(strings.TrimSpace(text), width-1)
	for i, line := range lines {
		if i == len(lines)-1 {
			fmt.Print(line + " ")
		} else {
			fmt.Println(line)
		}
	}
}

func wrapStaticText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	current := ""
	flush := func() {
		if current != "" {
			out = append(out, current)
			current = ""
		}
	}
	for _, word := range words {
		// Hard-split an exceptionally long token (e.g. a path-like token) so
		// static text still never relies on the console's automatic wrapping.
		for runeLen(word) > width {
			flush()
			head, tail := splitAtConsoleCells(word, width)
			out = append(out, head)
			word = tail
		}
		if current == "" {
			current = word
			continue
		}
		if runeLen(current)+1+runeLen(word) <= width {
			current += " " + word
		} else {
			flush()
			current = word
		}
	}
	flush()
	return out
}

func pathIsDesktop(path string) bool {
	desktop, err := desktopPath()
	if err != nil || desktop == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(filepath.Dir(path)), filepath.Clean(desktop))
}

func pathIsDesktopReportFolder(lang, path string) bool {
	desktop, err := desktopPath()
	if err != nil || desktop == "" {
		return false
	}
	want := filepath.Join(desktop, reportFolderName(lang))
	return strings.EqualFold(filepath.Clean(filepath.Dir(path)), filepath.Clean(want))
}

// reportSavedScreenLines deliberately gives both a human location and the
// exact path. A raw C:\\...\\Desktop path is not equally obvious to every user.
func reportSavedScreenLines(lang, path string) []string {
	if pathIsDesktopReportFolder(lang, path) {
		switch lang {
		case "nl":
			return []string{"Resultaatrapport opgeslagen in de map " + reportFolderName(lang) + " op uw Windows-bureaublad (Desktop).", "Volledig pad: " + path}
		case "de":
			return []string{"Ergebnisbericht im Ordner " + reportFolderName(lang) + " auf Ihrem Windows-Desktop gespeichert.", "Vollständiger Pfad: " + path}
		case "fr":
			return []string{"Rapport de résultat enregistré dans le dossier " + reportFolderName(lang) + " sur votre Bureau Windows (Desktop).", "Chemin complet : " + path}
		case "es":
			return []string{"Informe de resultados guardado en la carpeta " + reportFolderName(lang) + " de su Escritorio de Windows (Desktop).", "Ruta completa: " + path}
		case "zh":
			return []string{"结果报告已保存到 Windows 桌面 (Desktop) 上的 " + reportFolderName(lang) + " 文件夹。", "完整路径：" + path}
		case "ru":
			return []string{"Отчёт о результатах сохранён в папке " + reportFolderName(lang) + " на рабочем столе Windows (Desktop).", "Полный путь: " + path}
		default:
			return []string{"Result report saved in the " + reportFolderName(lang) + " folder on your Windows desktop (Desktop).", "Full path: " + path}
		}
	}
	onDesktop := pathIsDesktop(path)
	if onDesktop {
		switch lang {
		case "nl":
			return []string{"Resultaatrapport opgeslagen op uw Windows-bureaublad (Desktop).", "Volledig pad: " + path}
		case "de":
			return []string{"Ergebnisbericht auf Ihrem Windows-Desktop gespeichert.", "Vollständiger Pfad: " + path}
		case "fr":
			return []string{"Rapport de résultat enregistré sur votre Bureau Windows (Desktop).", "Chemin complet : " + path}
		case "es":
			return []string{"Informe de resultados guardado en su Escritorio de Windows (Desktop).", "Ruta completa: " + path}
		case "zh":
			return []string{"结果报告已保存到您的 Windows 桌面 (Desktop)。", "完整路径：" + path}
		case "ru":
			return []string{"Отчёт о результатах сохранён на рабочем столе Windows (Desktop).", "Полный путь: " + path}
		default:
			return []string{"Result report saved on your Windows desktop (Desktop).", "Full path: " + path}
		}
	}
	switch lang {
	case "nl":
		return []string{"Resultaatrapport opgeslagen.", "Volledig pad: " + path}
	case "de":
		return []string{"Ergebnisbericht gespeichert.", "Vollständiger Pfad: " + path}
	case "fr":
		return []string{"Rapport de résultat enregistré.", "Chemin complet : " + path}
	case "es":
		return []string{"Informe de resultados guardado.", "Ruta completa: " + path}
	case "zh":
		return []string{"结果报告已保存。", "完整路径：" + path}
	case "ru":
		return []string{"Отчёт о результатах сохранён.", "Полный путь: " + path}
	default:
		return []string{"Result report saved.", "Full path: " + path}
	}
}

func selfTestFilesUntouchedLine(lang string) string {
	switch lang {
	case "nl":
		return "Deze TEST-FAIL veranderde niets aan uw eigen bestanden."
	case "de":
		return "Dieser TEST-FEHLER hat an Ihren eigenen Dateien nichts verändert."
	case "fr":
		return "Cet ÉCHEC DE TEST n'a rien modifié dans vos propres fichiers."
	case "es":
		return "Este FALLO DE PRUEBA no modificó ninguno de sus propios archivos."
	case "zh":
		return "此测试失败未更改您的任何文件。"
	case "ru":
		return "Этот ТЕСТОВЫЙ СБОЙ ничего не изменил в ваших собственных файлах."
	default:
		return "This TEST-FAIL changed nothing in your own files."
	}
}

// languageEscapeLabel keeps the route back to the language menu recognisable
// even when the user has accidentally selected an unfamiliar script. English,
// Chinese and Russian are used as bridge labels, following the Source2Metal
// convention. The currently selected bridge language is not repeated.
func languageEscapeLabel(lang, local string) string {
	var bridges []string
	if lang != "en" {
		bridges = append(bridges, "Change language")
	}
	if lang != "zh" {
		bridges = append(bridges, "更改语言")
	}
	if lang != "ru" {
		bridges = append(bridges, "Изменить язык")
	}
	if len(bridges) == 0 {
		return local
	}
	return local + " [" + strings.Join(bridges, " / ") + "]"
}
