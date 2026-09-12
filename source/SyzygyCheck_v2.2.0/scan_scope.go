package main

import (
	"bufio"
	"fmt"
	"strings"
)

type scanScopeText struct {
	heading, current, recursive, boundary, prompt string
}

func scanScopeTexts(lang string) scanScopeText {
	switch lang {
	case "nl":
		return scanScopeText{"Welke mappen wilt u checken?", "Alleen deze Controlemap [standaard]", "Deze Controlemap en alle submappen", "Er wordt nooit in hogere mappen gezocht.", "Keuze [1-2] (Enter = 1; Esc = terug): "}
	case "de":
		return scanScopeText{"Welche Ordner sollen geprüft werden?", "Nur dieser Prüfungsordner [Standard]", "Dieser Prüfungsordner und alle Unterordner", "Übergeordnete Ordner werden niemals durchsucht.", "Auswahl [1-2] (Enter = 1; Esc = zurück): "}
	case "fr":
		return scanScopeText{"Quels dossiers voulez-vous vérifier ?", "Uniquement ce dossier de contrôle [défaut]", "Ce dossier de contrôle et tous ses sous-dossiers", "Les dossiers parents ne sont jamais parcourus.", "Choix [1-2] (Entrée = 1 ; Échap = retour) : "}
	case "es":
		return scanScopeText{"¿Qué carpetas desea comprobar?", "Solo esta carpeta de control [predeterminado]", "Esta carpeta de control y todas sus subcarpetas", "Nunca se buscan carpetas superiores.", "Opción [1-2] (Enter = 1; Esc = volver): "}
	case "zh":
		return scanScopeText{"您要检查哪些文件夹？", "仅此检查文件夹 [默认]", "此检查文件夹及其所有子文件夹", "绝不会搜索上级文件夹。", "选择 [1-2]（回车 = 1；Esc = 返回）："}
	case "ru":
		return scanScopeText{"Какие папки проверить?", "Только эту проверяемую папку [по умолчанию]", "Эту папку и все её подпапки", "Папки верхнего уровня никогда не сканируются.", "Выбор [1-2] (Enter = 1; Esc = назад): "}
	default:
		return scanScopeText{"Which folders do you want to check?", "This control folder only [default]", "This control folder and all subfolders", "Parent folders are never scanned.", "Choice [1-2] (Enter = 1; Esc = back): "}
	}
}

func chooseScanScope(r *bufio.Reader, lang string) (recursive, ok bool) {
	t := scanScopeTexts(lang)
	for {
		fmt.Println()
		printWrappedStatic(t.heading)
		printWrappedPrefixedStatic("  1 = ", t.current)
		printWrappedPrefixedStatic("  2 = ", t.recursive)
		printWrappedStatic(t.boundary)
		printWrappedPrompt(t.prompt)
		in, escaped := readMenuInput(r)
		if escaped {
			return false, false
		}
		switch strings.TrimSpace(in) {
		case "", "1":
			return false, true
		case "2":
			return true, true
		}
	}
}

func scanScopeLabel(lang string) string {
	switch lang {
	case "nl":
		return "Checkbereik"
	case "de":
		return "Prüfbereich"
	case "fr":
		return "Étendue du contrôle"
	case "es":
		return "Ámbito del control"
	case "zh":
		return "检查范围"
	case "ru":
		return "Область проверки"
	default:
		return "Check scope"
	}
}

func scanScopeDescription(lang string, recursive bool) string {
	t := scanScopeTexts(lang)
	if recursive {
		return t.recursive
	}
	return t.current
}

func noFilesInScope(lang string, recursive bool) string {
	if !recursive {
		return texts[lang].noFiles
	}
	switch lang {
	case "nl":
		return "Geen .rtbw- of .rtbz-files gevonden in deze Controlemap of de submappen."
	case "de":
		return "Keine .rtbw- oder .rtbz-Dateien in diesem Prüfungsordner oder seinen Unterordnern gefunden."
	case "fr":
		return "Aucun fichier .rtbw ou .rtbz trouvé dans ce dossier de contrôle ou ses sous-dossiers."
	case "es":
		return "No se encontraron archivos .rtbw o .rtbz en esta carpeta de control ni en sus subcarpetas."
	case "zh":
		return "此检查文件夹及其子文件夹中未找到 .rtbw 或 .rtbz 文件。"
	case "ru":
		return "В этой проверяемой папке и её подпапках не найдены файлы .rtbw или .rtbz."
	default:
		return "No .rtbw or .rtbz files were found in this control folder or its subfolders."
	}
}
