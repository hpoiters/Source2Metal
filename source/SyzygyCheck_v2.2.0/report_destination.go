package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	browseForFolderFn = browseForFolder
	desktopPathFn     = desktopPath
)

type reportDestinationText struct {
	heading, subfolder, desktop, control, choose, defaultWord, prompt, previous, browseError, previousUnavailable string
}

func reportDestinationTexts(lang string) reportDestinationText {
	switch lang {
	case "nl":
		return reportDestinationText{"Waar wilt u het resultaatrapport opslaan?", "De resultaten komen daar in de map " + reportFolderName(lang) + ".", "Resultaattekst naar Desktop", "Resultaattekst naar Controlemap", "Resultaattekst naar map naar keuze (Verkenner)", "standaard", "Keuze [1-3] (Enter = %d): ", "laatst gekozen: %s", "De map kon niet worden gekozen: %v", "De eerder gekozen map is niet meer beschikbaar: %s. Desktop is daarom nu de standaardbestemming."}
	case "de":
		return reportDestinationText{"Wo soll der Ergebnisbericht gespeichert werden?", "Die Ergebnisse werden dort im Ordner " + reportFolderName(lang) + " gespeichert.", "Ergebnistext auf dem Desktop", "Ergebnistext im Prüfungsordner", "Ergebnistext in einem Ordner Ihrer Wahl (Explorer)", "Standard", "Auswahl [1-3] (Enter = %d): ", "zuletzt gewählt: %s", "Der Ordner konnte nicht ausgewählt werden: %v", "Der zuvor gewählte Ordner ist nicht mehr verfügbar: %s. Der Desktop ist daher jetzt das Standardziel."}
	case "fr":
		return reportDestinationText{"Où souhaitez-vous enregistrer le rapport de résultat ?", "Les résultats y seront placés dans le dossier " + reportFolderName(lang) + ".", "Texte du résultat sur le Bureau", "Texte du résultat dans le dossier contrôlé", "Texte du résultat dans un dossier au choix (Explorateur)", "défaut", "Choix [1-3] (Entrée = %d) : ", "dernier choix : %s", "Le dossier n’a pas pu être sélectionné : %v", "Le dossier choisi précédemment n’est plus disponible : %s. Le Bureau est donc maintenant la destination par défaut."}
	case "es":
		return reportDestinationText{"¿Dónde desea guardar el informe de resultados?", "Los resultados se guardarán allí en la carpeta " + reportFolderName(lang) + ".", "Texto del resultado en el Escritorio", "Texto del resultado en la carpeta comprobada", "Texto del resultado en una carpeta elegida (Explorador)", "predeterminado", "Opción [1-3] (Enter = %d): ", "última elección: %s", "No se pudo seleccionar la carpeta: %v", "La carpeta seleccionada anteriormente ya no está disponible: %s. Por tanto, el Escritorio es ahora el destino predeterminado."}
	case "zh":
		return reportDestinationText{"您要将结果报告保存在哪里？", "结果将保存在所选位置的 " + reportFolderName(lang) + " 文件夹中。", "将结果文本保存到桌面", "将结果文本保存到检查文件夹", "将结果文本保存到自选文件夹（资源管理器）", "默认", "选择 [1-3]（回车 = %d）：", "上次选择：%s", "无法选择文件夹：%v", "先前选择的文件夹已不可用：%s。因此，桌面现在是默认保存位置。"}
	case "ru":
		return reportDestinationText{"Куда сохранить отчёт о результатах?", "Результаты будут сохранены там в папке " + reportFolderName(lang) + ".", "Текст результата на рабочий стол", "Текст результата в проверяемую папку", "Текст результата в выбранную папку (Проводник)", "по умолчанию", "Выбор [1-3] (Enter = %d): ", "последний выбор: %s", "Не удалось выбрать папку: %v", "Ранее выбранная папка больше недоступна: %s. Поэтому рабочий стол теперь используется по умолчанию."}
	default:
		return reportDestinationText{"Where do you want to save the result report?", "Results will be stored there in the " + reportFolderName(lang) + " folder.", "Result text to Desktop", "Result text to control folder", "Result text to a folder of your choice (Explorer)", "default", "Choice [1-3] (Enter = %d): ", "last chosen: %s", "The folder could not be selected: %v", "The previously selected folder is no longer available: %s. Desktop is therefore now the default destination."}
	}
}

const reportDestinationPreferenceFile = "report_destination.json"

type reportDestinationPreference struct {
	Mode       string `json:"mode"`
	CustomPath string `json:"custom_path,omitempty"`
}

func reportDestinationPreferencePath() string {
	return filepath.Join(configDir(), reportDestinationPreferenceFile)
}

func loadReportDestinationPreference() (reportDestinationPreference, string) {
	fallback := reportDestinationPreference{Mode: "desktop"}
	b, err := os.ReadFile(reportDestinationPreferencePath())
	if err != nil {
		return fallback, ""
	}
	var pref reportDestinationPreference
	if json.Unmarshal(b, &pref) != nil {
		return fallback, ""
	}
	switch pref.Mode {
	case "desktop", "control":
		return pref, ""
	case "custom":
		info, err := os.Stat(pref.CustomPath)
		if err == nil && info.IsDir() {
			pref.CustomPath = filepath.Clean(pref.CustomPath)
			return pref, ""
		}
		return fallback, strings.TrimSpace(pref.CustomPath)
	}
	return fallback, ""
}

func saveReportDestinationPreference(pref reportDestinationPreference) {
	p := reportDestinationPreferencePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return
	}
	b, err := json.MarshalIndent(pref, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(p, append(b, '\n'), 0644)
}

func reportDestinationChoice(pref reportDestinationPreference) int {
	switch pref.Mode {
	case "control":
		return 2
	case "custom":
		return 3
	default:
		return 1
	}
}

func markDefault(label, word string, choice, defaultChoice int) string {
	if choice == defaultChoice {
		return label + " [" + word + "]"
	}
	return label
}

func reportFolderName(lang string) string {
	switch lang {
	case "zh":
		return "检查结果"
	case "ru":
		return "РезультатПроверки"
	default:
		return "CheckResult"
	}
}

// chooseReportFolder is deliberately called only after verification has
// finished and immediately before the report is written. Enter uses the last
// selected destination (Desktop on first use). Cancelling Explorer returns to
// this menu so a completed result is never silently discarded.
func chooseReportFolder(r *bufio.Reader, lang, controlDir string) (string, bool) {
	t := reportDestinationTexts(lang)
	pref, unavailableCustomPath := loadReportDestinationPreference()
	showUnavailableWarning := unavailableCustomPath != ""
	for {
		defaultChoice := reportDestinationChoice(pref)
		fmt.Println()
		if showUnavailableWarning {
			printWrappedStatic(fmt.Sprintf(t.previousUnavailable, unavailableCustomPath))
			fmt.Println()
			showUnavailableWarning = false
		}
		printWrappedStatic(t.heading)
		printWrappedStatic(t.subfolder)
		printWrappedPrefixedStatic("  1 = ", markDefault(t.desktop, t.defaultWord, 1, defaultChoice))
		printWrappedPrefixedStatic("  2 = ", markDefault(t.control, t.defaultWord, 2, defaultChoice))
		printWrappedPrefixedStatic("  3 = ", markDefault(t.choose, t.defaultWord, 3, defaultChoice))
		if pref.Mode == "custom" && pref.CustomPath != "" {
			printWrappedPrefixedStatic("      ", fmt.Sprintf(t.previous, pref.CustomPath))
		}
		printWrappedPrompt(fmt.Sprintf(t.prompt, defaultChoice))
		in, escaped := readMenuInput(r)
		if escaped {
			in = ""
		}
		choice := strings.TrimSpace(in)
		if choice == "" {
			choice = fmt.Sprintf("%d", defaultChoice)
			if choice == "3" && pref.CustomPath != "" {
				return filepath.Join(pref.CustomPath, reportFolderName(lang)), true
			}
		}
		switch choice {
		case "1":
			saveReportDestinationPreference(reportDestinationPreference{Mode: "desktop"})
			if desktop, err := desktopPathFn(); err == nil && desktop != "" {
				return filepath.Join(desktop, reportFolderName(lang)), true
			}
			return filepath.Join(controlDir, reportFolderName(lang)), true
		case "2":
			saveReportDestinationPreference(reportDestinationPreference{Mode: "control"})
			return filepath.Join(controlDir, reportFolderName(lang)), true
		case "3":
			folder, err := browseForFolderFn(t.heading)
			if err == nil && folder != "" {
				if absolute, absErr := filepath.Abs(folder); absErr == nil {
					folder = absolute
				}
				folder = filepath.Clean(folder)
				pref = reportDestinationPreference{Mode: "custom", CustomPath: folder}
				saveReportDestinationPreference(pref)
				return filepath.Join(folder, reportFolderName(lang)), true
			}
			if err != nil && !isFolderSelectionCancelled(err) {
				fmt.Printf(t.browseError+"\n", err)
			}
		}
	}
}
