package main

import (
	"bufio"
	"fmt"
	"strings"
)

// pagefileSafetyGate deliberately stays short on-screen. The detailed Windows
// procedure belongs in the manual so the checker UI remains calm and readable.
// Enter continues; A aborts before the self-test or real tablebase scan begins.
func pagefileSafetyGate(r *bufio.Reader, lang string) bool {
	for {
		fmt.Println()
		for _, line := range pagefileWarningLines(lang) {
			fmt.Println(line)
		}
		fmt.Print(pagefileChoicePrompt(lang))
		choice, escaped := readMenuInput(r)
		if escaped {
			return false
		}
		choice = strings.ToLower(strings.TrimSpace(choice))
		switch choice {
		case "":
			return true
		case "a":
			return false
		default:
			// Keep the choice deliberately narrow and predictable. Reprint the
			// short gate instead of silently interpreting another key.
			fmt.Println(pagefileInvalidChoice(lang))
		}
	}
}

func pagefileWarningLines(lang string) []string {
	switch lang {
	case "nl":
		return []string{
			"LET OP BIJ ZEER GROTE SYZYGY-DATABASES:",
			"De controle kan veel virtueel geheugen (pagefile/wisselbestand) vereisen.",
			"Advies: laat Windows het wisselbestand automatisch beheren.",
			"Zie de uitgebreide handleiding voor de Windows-instellingen.",
		}
	case "de":
		return []string{
			"HINWEIS BEI SEHR GROSSEN SYZYGY-DATENBANKEN:",
			"Die Prüfung kann viel virtuellen Speicher (Auslagerungsdatei) benötigen.",
			"Empfehlung: Windows die Auslagerungsdatei automatisch verwalten lassen.",
			"Siehe ausführliche Anleitung für die Windows-Einstellungen.",
		}
	case "fr":
		return []string{
			"ATTENTION POUR LES TRÈS GRANDES BASES SYZYGY :",
			"La vérification peut nécessiter beaucoup de mémoire virtuelle (fichier d'échange).",
			"Conseil : laissez Windows gérer automatiquement le fichier d'échange.",
			"Consultez le manuel détaillé pour les réglages Windows.",
		}
	case "es":
		return []string{
			"ATENCIÓN CON BASES SYZYGY MUY GRANDES:",
			"La comprobación puede requerir mucha memoria virtual (archivo de paginación).",
			"Consejo: deje que Windows administre automáticamente el archivo de paginación.",
			"Consulte el manual detallado para la configuración de Windows.",
		}
	case "zh":
		return []string{
			"超大型 SYZYGY 数据库注意事项：",
			"校验可能需要大量虚拟内存（页面文件）。",
			"建议：让 Windows 自动管理页面文件。",
			"Windows 设置步骤请参阅详细说明文档。",
		}
	case "ru":
		return []string{
			"ВНИМАНИЕ ДЛЯ ОЧЕНЬ БОЛЬШИХ БАЗ SYZYGY:",
			"Проверка может потребовать большой объём виртуальной памяти (файл подкачки).",
			"Рекомендуется позволить Windows автоматически управлять файлом подкачки.",
			"Подробные настройки Windows приведены в руководстве.",
		}
	default:
		return []string{
			"NOTE FOR VERY LARGE SYZYGY DATABASES:",
			"The check can require substantial virtual memory (pagefile).",
			"Recommendation: let Windows automatically manage the paging file.",
			"See the detailed manual for the Windows settings.",
		}
	}
}

func pagefileChoicePrompt(lang string) string {
	switch lang {
	case "nl":
		return "\nEnter = doorgaan   A/Esc = terug om de instellingen te controleren/wijzigen: "
	case "de":
		return "\nEnter = fortfahren   A/Esc = zurück, um die Einstellungen zu prüfen/ändern: "
	case "fr":
		return "\nEntrée = continuer   A/Échap = retour pour vérifier/modifier les réglages : "
	case "es":
		return "\nEnter = continuar   A/Esc = volver para revisar/cambiar la configuración: "
	case "zh":
		return "\nEnter = 继续   A/Esc = 返回以检查/修改设置："
	case "ru":
		return "\nEnter = продолжить   A/Esc = назад для проверки/изменения настроек: "
	default:
		return "\nEnter = continue   A/Esc = back to check/change the settings: "
	}
}

func pagefileInvalidChoice(lang string) string {
	switch lang {
	case "nl":
		return "Kies Enter om door te gaan of A/Esc om terug te gaan."
	case "de":
		return "Drücken Sie Enter zum Fortfahren oder A/Esc, um zurückzugehen."
	case "fr":
		return "Appuyez sur Entrée pour continuer ou A/Échap pour revenir."
	case "es":
		return "Pulse Enter para continuar o A/Esc para volver."
	case "zh":
		return "请按 Enter 继续，或按 A/Esc 返回。"
	case "ru":
		return "Нажмите Enter для продолжения или A/Esc для возврата."
	default:
		return "Press Enter to continue or A/Esc to go back."
	}
}
