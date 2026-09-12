package main

import (
	"bufio"
	"fmt"
	"strconv"
)

func normaliseDetectedThreads(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func halfThreads(n int) int {
	n = normaliseDetectedThreads(n)
	h := n / 2
	if h < 1 {
		h = 1
	}
	return h
}

func detectedThreadsLabel(lang string) string {
	switch lang {
	case "nl":
		return "Gedetecteerde threads"
	case "de":
		return "Erkannte Threads"
	case "fr":
		return "Threads détectés"
	case "es":
		return "Threads detectados"
	case "zh":
		return "检测到的线程"
	case "ru":
		return "Обнаружено потоков"
	default:
		return "Detected threads"
	}
}

func workersSummaryLabel(lang string) string {
	switch lang {
	case "nl":
		return "Werkers (threads)"
	case "de":
		return "Worker (Threads)"
	case "fr":
		return "Travailleurs (threads)"
	case "es":
		return "Trabajadores (threads)"
	case "zh":
		return "工作线程 (threads)"
	case "ru":
		return "Рабочие потоки (threads)"
	default:
		return "Workers (threads)"
	}
}

func threadChoiceLines(lang string, detected int) []string {
	detected = normaliseDetectedThreads(detected)
	half := halfThreads(detected)
	switch lang {
	case "nl":
		return []string{
			fmt.Sprintf("%s: %d", detectedThreadsLabel(lang), detected),
			"",
			fmt.Sprintf("  1 = Alle threads gebruiken (%d)", detected),
			fmt.Sprintf("  2 = Helft gebruiken (%d) [standaard]", half),
			"  3 = Zelf kiezen",
			"",
		}
	case "de":
		return []string{
			fmt.Sprintf("%s: %d", detectedThreadsLabel(lang), detected), "",
			fmt.Sprintf("  1 = Alle Threads verwenden (%d)", detected),
			fmt.Sprintf("  2 = Hälfte verwenden (%d) [Standard]", half),
			"  3 = Selbst wählen", "",
		}
	case "fr":
		return []string{
			fmt.Sprintf("%s : %d", detectedThreadsLabel(lang), detected), "",
			fmt.Sprintf("  1 = Utiliser tous les threads (%d)", detected),
			fmt.Sprintf("  2 = Utiliser la moitié (%d) [défaut]", half),
			"  3 = Choisir soi-même", "",
		}
	case "es":
		return []string{
			fmt.Sprintf("%s: %d", detectedThreadsLabel(lang), detected), "",
			fmt.Sprintf("  1 = Usar todos los threads (%d)", detected),
			fmt.Sprintf("  2 = Usar la mitad (%d) [predeterminado]", half),
			"  3 = Elegir manualmente", "",
		}
	case "zh":
		return []string{
			fmt.Sprintf("%s：%d", detectedThreadsLabel(lang), detected), "",
			fmt.Sprintf("  1 = 使用全部线程 (%d)", detected),
			fmt.Sprintf("  2 = 使用一半 (%d) [默认]", half),
			"  3 = 自行选择", "",
		}
	case "ru":
		return []string{
			fmt.Sprintf("%s: %d", detectedThreadsLabel(lang), detected), "",
			fmt.Sprintf("  1 = Использовать все потоки (%d)", detected),
			fmt.Sprintf("  2 = Использовать половину (%d) [по умолчанию]", half),
			"  3 = Выбрать вручную", "",
		}
	default:
		return []string{
			fmt.Sprintf("%s: %d", detectedThreadsLabel(lang), detected), "",
			fmt.Sprintf("  1 = Use all threads (%d)", detected),
			fmt.Sprintf("  2 = Use half (%d) [default]", half),
			"  3 = Choose manually", "",
		}
	}
}

func threadChoicePrompt(lang string) string {
	switch lang {
	case "nl":
		return "Keuze [1-3] (Enter = 2, standaard; Esc = terug): "
	case "de":
		return "Auswahl [1-3] (Enter = 2, Standard; Esc = zurück): "
	case "fr":
		return "Choix [1-3] (Entrée = 2, défaut ; Échap = retour) : "
	case "es":
		return "Opción [1-3] (Enter = 2, predeterminado; Esc = volver): "
	case "zh":
		return "选择 [1-3]（回车 = 2，默认；Esc = 返回）："
	case "ru":
		return "Выбор [1-3] (Enter = 2, по умолчанию; Esc = назад): "
	default:
		return "Choice [1-3] (Enter = 2, default; Esc = back): "
	}
}

func customThreadsPrompt(lang string, detected int) string {
	switch lang {
	case "nl":
		return fmt.Sprintf("Aantal threads [1-%d] (Esc = terug): ", detected)
	case "de":
		return fmt.Sprintf("Anzahl Threads [1-%d] (Esc = zurück): ", detected)
	case "fr":
		return fmt.Sprintf("Nombre de threads [1-%d] (Échap = retour) : ", detected)
	case "es":
		return fmt.Sprintf("Número de threads [1-%d] (Esc = volver): ", detected)
	case "zh":
		return fmt.Sprintf("线程数 [1-%d]（Esc = 返回）：", detected)
	case "ru":
		return fmt.Sprintf("Количество потоков [1-%d] (Esc = назад): ", detected)
	default:
		return fmt.Sprintf("Number of threads [1-%d] (Esc = back): ", detected)
	}
}

// chooseWorkerThreads is deliberately simple: Enter selects the balanced half
// of the logical processors detected for this process. Invalid input is asked
// again instead of silently changing the user's requested worker count.
func chooseWorkerThreads(r *bufio.Reader, lang string, detected int) (int, bool) {
	detected = normaliseDetectedThreads(detected)
	half := halfThreads(detected)
	for {
		fmt.Println()
		for _, line := range threadChoiceLines(lang, detected) {
			fmt.Println(line)
		}
		fmt.Print(threadChoicePrompt(lang))
		choice, escaped := readMenuInput(r)
		if escaped {
			return 0, false
		}
		switch choice {
		case "", "2":
			return half, true
		case "1":
			return detected, true
		case "3":
			for {
				fmt.Print(customThreadsPrompt(lang, detected))
				in, escaped := readMenuInput(r)
				if escaped {
					break
				}
				n, err := strconv.Atoi(in)
				if err == nil && n >= 1 && n <= detected {
					return n, true
				}
			}
		}
	}
}

func tbcheckArgs(workers int, names ...string) []string {
	if workers < 1 {
		workers = 1
	}
	args := make([]string, 0, 2+len(names))
	args = append(args, "--threads", strconv.Itoa(workers))
	args = append(args, names...)
	return args
}
