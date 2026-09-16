package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var activeLanguage = "en"

func normalizeLanguage(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "en", "eng", "english":
		return "en"
	case "de", "deu", "ger", "german", "deutsch":
		return "de"
	case "nl", "nld", "dut", "dutch", "nederlands":
		return "nl"
	case "fr", "fra", "fre", "french", "français", "francais":
		return "fr"
	case "es", "spa", "spanish", "español", "espanol":
		return "es"
	case "zh", "zho", "chi", "chinese", "中文", "简体中文", "simplified chinese":
		return "zh"
	case "ru", "rus", "russian", "русский":
		return "ru"
	default:
		return ""
	}
}

func setLanguage(s string) {
	if n := normalizeLanguage(s); n != "" {
		activeLanguage = n
	} else {
		activeLanguage = "en"
	}
}

// L returns one user-facing string in the active language.
// Argument order: English, German, Dutch, French, Spanish, Simplified Chinese, Russian.
func L(en, de, nl, fr, es, zh, ru string) string {
	switch activeLanguage {
	case "de":
		return de
	case "nl":
		return nl
	case "fr":
		return fr
	case "es":
		return es
	case "zh":
		return zh
	case "ru":
		return ru
	default:
		return en
	}
}

func languageDisplayName(code string) string {
	switch normalizeLanguage(code) {
	case "de":
		return "Deutsch (German)"
	case "nl":
		return "Nederlands (Dutch)"
	case "fr":
		return "Français (French)"
	case "es":
		return "Español (Spanish)"
	case "zh":
		return "中文 (Chinese)"
	case "ru":
		return "Русский (Russian)"
	default:
		return "English (English)"
	}
}

func languageConfigPath() (string, error) {
	dir, err := applicationDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Source2Metal.ini"), nil
}

func readSavedLanguage() (string, bool) {
	p, err := languageConfigPath()
	if err != nil {
		return "", false
	}
	f, err := os.Open(p)
	if err != nil {
		return "", false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), "Language") {
			continue
		}
		if n := normalizeLanguage(v); n != "" {
			return n, true
		}
	}
	return "", false
}

func saveLanguage(code string) error {
	code = normalizeLanguage(code)
	if code == "" {
		code = "en"
	}
	p, err := languageConfigPath()
	if err != nil {
		return err
	}
	text := "# Source2Metal settings\r\nLanguage=" + code + "\r\n"
	return atomicWriteFile(p, []byte(text), 0644)
}

// initLanguage runs before the normal start menu. On a first interactive start,
// the selector is intentionally English so every installation begins from one
// predictable language. The choice is saved immediately as the default.
func initLanguage(interactive bool) {
	if code, ok := readSavedLanguage(); ok {
		setLanguage(code)
		return
	}
	setLanguage("en")
	if interactive {
		chooseLanguage(true)
	}
}

func chooseLanguage(firstStart bool) string {
	def := activeLanguage
	if firstStart || normalizeLanguage(def) == "" {
		def = "en"
	}
	defNum := map[string]int{"en": 1, "de": 2, "nl": 3, "fr": 4, "es": 5, "zh": 6, "ru": 7}[def]
	if firstStart {
		fmt.Println("Choose your default language")
		fmt.Println("Your choice will be remembered for the next start.")
	} else {
		fmt.Println("\n" + L("CHANGE LANGUAGE", "SPRACHE ÄNDERN", "TAAL WIJZIGEN", "CHANGER DE LANGUE", "CAMBIAR IDIOMA", "更改语言", "ИЗМЕНИТЬ ЯЗЫК"))
	}
	fmt.Println("  1 = English (English)")
	fmt.Println("  2 = Deutsch (German)")
	fmt.Println("  3 = Nederlands (Dutch)")
	fmt.Println("  4 = Français (French)")
	fmt.Println("  5 = Español (Spanish)")
	fmt.Println("  6 = 中文 (Chinese)")
	fmt.Println("  7 = Русский (Russian)")
	if firstStart {
		fmt.Printf("Choice [1-7] (Enter = %d): ", defNum)
	} else {
		fmt.Printf(L("Choice [1-7] (Enter = %d): ", "Auswahl [1-7] (Enter = %d): ", "Keuze [1-7] (Enter = %d): ", "Choix [1-7] (Entrée = %d) : ", "Elección [1-7] (Intro = %d): ", "选择 [1-7]（回车 = %d）：", "Выбор [1-7] (Enter = %d): "), defNum)
	}
	s, _ := stdin.ReadString('\n')
	s = strings.TrimSpace(s)
	n := defNum
	if s != "" {
		if x, err := strconv.Atoi(s); err == nil && x >= 1 && x <= 7 {
			n = x
		}
	}
	codes := []string{"", "en", "de", "nl", "fr", "es", "zh", "ru"}
	setLanguage(codes[n])
	_ = saveLanguage(activeLanguage)
	clearConsoleScreen()
	return activeLanguage
}

// U translates stable Dutch user-facing text. Technical identifiers, paths,
// hashes, PGN content and numerical values are deliberately left untouched.
func U(s string) string { return localizeUserText(s) }

func localizeTextFile(path string) error {
	if activeLanguage == "nl" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	localized := localizeReportText(string(b))
	if localized == string(b) {
		return nil
	}
	return atomicWriteFile(path, []byte(localized), 0644)
}

func localizeStatus(status string) string {
	switch status {
	case "GESLAAGD":
		return L("SUCCESS", "ERFOLGREICH", "GESLAAGD", "RÉUSSI", "CORRECTO", "成功", "УСПЕШНО")
	case "GESLAAGD_MET_OVERSLAGEN_BRONNEN":
		return L("SUCCESS WITH SKIPPED SOURCES", "ERFOLGREICH MIT ÜBERSPRUNGENEN QUELLEN", "GESLAAGD MET OVERGESLAGEN BRONNEN", "RÉUSSI AVEC SOURCES IGNORÉES", "CORRECTO CON FUENTES OMITIDAS", "成功（有跳过的源）", "УСПЕШНО С ПРОПУЩЕННЫМИ ИСТОЧНИКАМИ")
	case "GESLAAGD_MET_FOUTEN":
		return L("SUCCESS WITH ERRORS", "ERFOLGREICH MIT FEHLERN", "GESLAAGD MET FOUTEN", "RÉUSSI AVEC ERREURS", "CORRECTO CON ERRORES", "完成但有错误", "УСПЕШНО С ОШИБКАМИ")
	case "MISLUKT":
		return L("FAILED", "FEHLGESCHLAGEN", "MISLUKT", "ÉCHEC", "FALLIDO", "失败", "ОШИБКА")
	case "OVERGESLAGEN":
		return L("SKIPPED", "ÜBERSPRUNGEN", "OVERGESLAGEN", "IGNORÉ", "OMITIDO", "已跳过", "ПРОПУЩЕНО")
	case "KLAAR":
		return L("DONE", "FERTIG", "KLAAR", "TERMINÉ", "LISTO", "完成", "ГОТОВО")
	default:
		return strings.ReplaceAll(status, "_", " ")
	}
}

// localizeUserText is used for generated text reports that are assembled in
// Dutch by the proven processing code. It translates the stable user-facing
// vocabulary without touching data, paths, hashes, PGN or numerical results.
func localizeUserText(s string) string {
	if activeLanguage == "nl" {
		return s
	}
	var pairs []string
	add := func(nl, en, de, fr, es, zh, ru string) {
		tr := L(en, de, nl, fr, es, zh, ru)
		if tr != nl {
			pairs = append(pairs, nl, tr)
		}
	}
	// Longer phrases first.
	add("PARALLELLE VERWERKING", "PARALLEL PROCESSING", "PARALLELE VERARBEITUNG", "TRAITEMENT PARALLÈLE", "PROCESAMIENTO PARALELO", "并行处理", "ПАРАЛЛЕЛЬНАЯ ОБРАБОТКА")
	add("HARDWAREDETECTIE", "HARDWARE DETECTION", "HARDWAREERKENNUNG", "DÉTECTION DU MATÉRIEL", "DETECCIÓN DE HARDWARE", "硬件检测", "ОПРЕДЕЛЕНИЕ ОБОРУДОВАНИЯ")
	add("Bronnen inventariseren", "Inventory sources", "Quellen inventarisieren", "Inventorier les sources", "Inventariar fuentes", "盘点源文件", "Инвентаризация источников")
	add("Bronnen valideren", "Validate sources", "Quellen validieren", "Valider les sources", "Validar fuentes", "验证源文件", "Проверка источников")
	add("RAW bouwen", "Build RAW", "RAW erstellen", "Construire RAW", "Crear RAW", "构建 RAW", "Создание RAW")
	add("METAL bouwen", "Build METAL", "METAL erstellen", "Construire METAL", "Crear METAL", "构建 METAL", "Создание METAL")
	add("Rapporten + checksums", "Reports + checksums", "Berichte + Prüfsummen", "Rapports + sommes de contrôle", "Informes + sumas de verificación", "报告 + 校验和", "Отчёты + контрольные суммы")
	add("Tijdelijke bestanden opruimen", "Clean temporary files", "Temporäre Dateien aufräumen", "Nettoyer les fichiers temporaires", "Limpiar archivos temporales", "清理临时文件", "Очистка временных файлов")
	add("overgeslagen", "skipped", "übersprungen", "ignoré", "omitido", "已跳过", "пропущено")
	add("gemaakt", "created", "erstellt", "créé", "creado", "已创建", "создано")
	add("niet nodig", "not required", "nicht erforderlich", "inutile", "no es necesario", "不需要", "не требуется")
	add("Geen bruikbare PGN/CBH/2CBH GAME-bronnen", "No usable PGN/CBH/2CBH GAME sources", "Keine nutzbaren PGN/CBH/2CBH-GAME-Quellen", "Aucune source GAME PGN/CBH/2CBH utilisable", "No hay fuentes GAME PGN/CBH/2CBH utilizables", "没有可用的 PGN/CBH/2CBH GAME 源", "Нет пригодных источников GAME PGN/CBH/2CBH")
	add("Geen complete CTG-sets", "No complete CTG sets", "Keine vollständigen CTG-Sets", "Aucun ensemble CTG complet", "No hay conjuntos CTG completos", "没有完整的 CTG 集", "Нет полных наборов CTG")
	add("ALLE SOURCES RAW niet nodig: daarvoor zijn zowel GAME RAW als CTG RAW nodig.", "ALL SOURCES RAW is not required: both GAME RAW and CTG RAW are needed.", "ALL SOURCES RAW ist nicht erforderlich: dafür werden sowohl GAME RAW als auch CTG RAW benötigt.", "ALL SOURCES RAW n’est pas nécessaire : GAME RAW et CTG RAW sont tous deux requis.", "ALL SOURCES RAW no es necesario: se necesitan tanto GAME RAW como CTG RAW.", "不需要 ALL SOURCES RAW：必须同时存在 GAME RAW 和 CTG RAW。", "ALL SOURCES RAW не требуется: необходимы и GAME RAW, и CTG RAW.")
	add("Geen GAME-bronnen voor GAME-METAL", "No GAME sources for GAME-METAL", "Keine GAME-Quellen für GAME-METAL", "Aucune source GAME pour GAME-METAL", "No hay fuentes GAME para GAME-METAL", "没有用于 GAME-METAL 的 GAME 源", "Нет GAME-источников для GAME-METAL")
	add("Werkmap", "Working folder", "Arbeitsordner", "Dossier de travail", "Carpeta de trabajo", "工作文件夹", "Рабочая папка")
	add("Platform", "Platform", "Plattform", "Plateforme", "Plataforma", "平台", "Платформа")
	add("Logische CPU-threads", "Logical CPU threads", "Logische CPU-Threads", "Threads CPU logiques", "Hilos lógicos de CPU", "逻辑 CPU 线程", "Логические потоки CPU")
	add("Aanbevolen workers", "Recommended workers", "Empfohlene Worker", "Workers recommandés", "Workers recomendados", "推荐 worker 数", "Рекомендуемые рабочие потоки")
	add("Aantal workers", "Number of workers", "Anzahl Worker", "Nombre de workers", "Número de workers", "worker 数量", "Количество рабочих потоков")
	add("alleen inventaris/rapport", "inventory/report only", "nur Inventar/Bericht", "inventaire/rapport uniquement", "solo inventario/informe", "仅清单/报告", "только инвентаризация/отчёт")
	add("RAW + METAL [standaard]", "RAW + METAL [default]", "RAW + METAL [Standard]", "RAW + METAL [défaut]", "RAW + METAL [predeterminado]", "RAW + METAL [默认]", "RAW + METAL [по умолчанию]")
	add("Keuze", "Choice", "Auswahl", "Choix", "Elección", "选择", "Выбор")
	add("Maximale RAW-diepte", "Maximum RAW depth", "Maximale RAW-Tiefe", "Profondeur RAW maximale", "Profundidad RAW máxima", "最大 RAW 深度", "Максимальная глубина RAW")
	add("Minimumlengte partij", "Minimum game length", "Mindestlänge der Partie", "Longueur minimale de la partie", "Longitud mínima de partida", "最短对局长度", "Минимальная длина партии")
	add("Extra Elo-filter", "Additional Elo filter", "Zusätzlicher Elo-Filter", "Filtre Elo supplémentaire", "Filtro Elo adicional", "附加 Elo 过滤器", "Дополнительный фильтр Elo")
	add("bronselectie behouden", "keep source selection", "Quellenauswahl beibehalten", "conserver la sélection source", "mantener selección de origen", "保留源选择", "сохранить исходный отбор")
	add("beide spelers", "both players", "beide Spieler", "les deux joueurs", "ambos jugadores", "双方棋手", "оба игрока")
	add("Maximaal Elo-verschil", "Maximum Elo difference", "Maximale Elo-Differenz", "Écart Elo maximal", "Diferencia Elo máxima", "最大 Elo 差", "Максимальная разница Elo")
	add("geen", "none", "keine", "aucun", "ninguno", "无", "нет")
	add("Ongeldige invoer.", "Invalid input.", "Ungültige Eingabe.", "Saisie invalide.", "Entrada no válida.", "输入无效。", "Неверный ввод.")
	add("ongeldige modus", "invalid mode", "ungültiger Modus", "mode invalide", "modo no válido", "模式无效", "неверный режим")
	add("TEMP opgeruimd.", "TEMP cleaned.", "TEMP aufgeräumt.", "TEMP nettoyé.", "TEMP limpiado.", "TEMP 已清理。", "TEMP очищен.")
	add("Klaar. Resultaten", "Done. Results", "Fertig. Ergebnisse", "Terminé. Résultats", "Listo. Resultados", "完成。结果", "Готово. Результаты")
	add("Workers gebruikt", "Workers used", "Verwendete Worker", "Workers utilisés", "Workers usados", "使用的 worker", "Использовано рабочих потоков")
	add("ChessBase-adapters", "ChessBase adapters", "ChessBase-Adapter", "Adaptateurs ChessBase", "Adaptadores ChessBase", "ChessBase 适配器", "Адаптеры ChessBase")
	add("OVERGESLAGEN", "SKIPPED", "ÜBERSPRUNGEN", "IGNORÉ", "OMITIDO", "已跳过", "ПРОПУЩЕНО")
	add("MISLUKT", "FAILED", "FEHLGESCHLAGEN", "ÉCHEC", "FALLIDO", "失败", "ОШИБКА")
	add("Adapter klaar", "Adapter done", "Adapter fertig", "Adaptateur terminé", "Adaptador listo", "适配器完成", "Адаптер готов")
	add("partijen", "games", "Partien", "parties", "partidas", "对局", "партий")
	add("fysieke records", "physical records", "physische Datensätze", "enregistrements physiques", "registros físicos", "物理记录", "физических записей")
	add("niet-actief", "inactive", "inaktiv", "inactif", "inactivo", "非活动", "неактивных")
	add("Per-bron resultaat", "Per-source result", "Ergebnis pro Quelle", "Résultat par source", "Resultado por fuente", "每源结果", "Результат по источнику")
	add("controle OK", "check OK", "Prüfung OK", "contrôle OK", "comprobación OK", "检查通过", "проверка OK")
	add("decodefouten", "decode errors", "Dekodierfehler", "erreurs de décodage", "errores de decodificación", "解码错误", "ошибок декодирования")
	add("Analyse klaar", "Analysis done", "Analyse fertig", "Analyse terminée", "Análisis listo", "分析完成", "Анализ завершён")
	add("Selectie klaar", "Selection done", "Auswahl fertig", "Sélection terminée", "Selección lista", "选择完成", "Отбор завершён")
	add("echte modelpartijen", "real model games", "echte Modellpartien", "parties modèles réelles", "partidas modelo reales", "真实模型对局", "реальных модельных партий")
	add("samengevoegd", "merged", "zusammengeführt", "fusionné", "combinado", "已合并", "объединено")
	add("Automatisch samengevoegd naar", "Automatically merged to", "Automatisch zusammengeführt nach", "Fusionné automatiquement vers", "Combinado automáticamente en", "自动合并到", "Автоматически объединено в")
	add("CPU", "CPU", "CPU", "CPU", "CPU", "CPU", "CPU")
	add("niet betrouwbaar uitgelezen", "could not be read reliably", "konnte nicht zuverlässig ausgelesen werden", "lecture non fiable", "no se pudo leer con fiabilidad", "无法可靠读取", "не удалось надёжно определить")
	add("Vrije uitvoerruimte", "Free output space", "Freier Ausgabespeicher", "Espace de sortie libre", "Espacio libre de salida", "可用输出空间", "Свободное место для вывода")
	add("Standaard workers", "Default workers", "Standard-Worker", "Workers par défaut", "Workers predeterminados", "默认 worker 数", "Рабочие потоки по умолчанию")
	add("WAARSCHUWING", "WARNING", "WARNUNG", "AVERTISSEMENT", "ADVERTENCIA", "警告", "ПРЕДУПРЕЖДЕНИЕ")
	add("Broninventaris", "Source inventory", "Quelleninventar", "Inventaire des sources", "Inventario de fuentes", "源清单", "Инвентаризация источников")
	add("Geen ondersteunde bronbestanden gevonden.", "No supported source files found.", "Keine unterstützten Quelldateien gefunden.", "Aucun fichier source pris en charge trouvé.", "No se encontraron archivos fuente compatibles.", "未找到支持的源文件。", "Поддерживаемые исходные файлы не найдены.")
	add("Er is bewust geen leeg of misleidend productbestand aangemaakt.", "No empty or misleading product file was created.", "Es wurde bewusst keine leere oder irreführende Produktdatei erstellt.", "Aucun fichier produit vide ou trompeur n'a été créé.", "No se creó ningún archivo de producto vacío o engañoso.", "未创建空的或误导性的结果文件。", "Пустой или вводящий в заблуждение файл результата не создавался.")
	add("De originele bronbestanden zijn niet gewijzigd.", "The original source files were not modified.", "Die ursprünglichen Quelldateien wurden nicht verändert.", "Les fichiers source d'origine n'ont pas été modifiés.", "Los archivos fuente originales no se modificaron.", "原始源文件未被修改。", "Исходные файлы не изменялись.")
	add("Overwinningen + Verliespartijen AAN; Wit/Zwart/Speler UIT; spelernaam leeg; alle partijen", "Wins + Losses ON; White/Black/Player OFF; player name empty; all games", "Siege + Niederlagen EIN; Weiß/Schwarz/Spieler AUS; Spielername leer; alle Partien", "Victoires + Défaites ACTIVÉES ; Blancs/Noirs/Joueur DÉSACTIVÉS ; nom du joueur vide ; toutes les parties", "Victorias + Derrotas ACTIVADAS; Blancas/Negras/Jugador DESACTIVADOS; nombre vacío; todas las partidas", "胜局 + 负局开启；白方/黑方/棋手关闭；棋手名留空；所有对局", "Победы + Поражения ВКЛ.; Белые/Чёрные/Игрок ВЫКЛ.; имя игрока пустое; все партии")
	add("Bronnen worden alleen gelezen; originele bronbestanden worden niet gewijzigd.", "Sources are read-only; original source files are never modified.", "Quellen werden nur gelesen; Originaldateien werden nie verändert.", "Les sources sont en lecture seule ; les fichiers d'origine ne sont jamais modifiés.", "Las fuentes son de solo lectura; los archivos originales nunca se modifican.", "源文件只读；原始文件绝不会被修改。", "Источники читаются только для чтения; исходные файлы никогда не изменяются.")
	add("Bronzoektocht blijft binnen de gekozen root en onderliggende mappen; uitvoer- en tempmappen worden overgeslagen.", "Source discovery stays inside the selected root and its subfolders; output and temporary folders are skipped.", "Die Quellensuche bleibt im gewählten Stammordner und dessen Unterordnern; Ausgabe- und Temp-Ordner werden übersprungen.", "La recherche des sources reste dans le dossier racine choisi et ses sous-dossiers ; les dossiers de sortie et temporaires sont ignorés.", "La búsqueda de fuentes permanece dentro de la raíz elegida y sus subcarpetas; se omiten las carpetas de salida y temporales.", "源搜索仅限于所选根目录及其子目录；输出和临时目录会被跳过。", "Поиск источников выполняется только в выбранной корневой папке и её подпапках; выходные и временные папки пропускаются.")
	add("GAME-METAL gebruikt volledige legale schaakbordcontrole en schrijft complete modelpartijen met originele uitslag.", "GAME-METAL performs full legal-board validation and writes complete model games with their original result.", "GAME-METAL führt eine vollständige Legalitätsprüfung durch und schreibt komplette Modellpartien mit Originalergebnis.", "GAME-METAL effectue une validation complète de la légalité et écrit des parties modèles complètes avec leur résultat d'origine.", "GAME-METAL realiza una validación completa de legalidad y escribe partidas modelo completas con su resultado original.", "GAME-METAL 进行完整的合法棋局验证，并写出带原始结果的完整模型对局。", "GAME-METAL выполняет полную проверку легальности и записывает полные модельные партии с исходным результатом.")
	add("CTG RAW laat de uitvoergrootte door de werkelijk decodeerbare broninhoud bepalen.", "CTG RAW lets the actually decodable source content determine output size.", "Bei CTG RAW bestimmt der tatsächlich dekodierbare Quellinhalt die Ausgabegröße.", "CTG RAW laisse le contenu réellement décodable déterminer la taille de sortie.", "CTG RAW deja que el contenido realmente decodificable determine el tamaño de salida.", "CTG RAW 的输出大小由实际可解码的源内容决定。", "Размер CTG RAW определяется фактически декодируемым содержимым источника.")
	add("GAME-METAL en CTG-METAL blijven herkenbaar gescheiden omdat zij verschillende soorten evidence coderen.", "GAME-METAL and CTG-METAL remain clearly separated because they encode different kinds of evidence.", "GAME-METAL und CTG-METAL bleiben klar getrennt, da sie unterschiedliche Evidenzarten codieren.", "GAME-METAL et CTG-METAL restent clairement séparés car ils codent des types de preuves différents.", "GAME-METAL y CTG-METAL permanecen claramente separados porque codifican tipos de evidencia distintos.", "GAME-METAL 与 CTG-METAL 保持明确分离，因为它们编码不同类型的证据。", "GAME-METAL и CTG-METAL остаются чётко разделёнными, поскольку кодируют разные типы данных.")
	add("Fritz-inleerinstelling:", "Fritz learning setting:", "Fritz-Lerneinstellung:", "Réglage d'apprentissage Fritz :", "Ajuste de aprendizaje de Fritz:", "Fritz 学习设置：", "Настройка обучения Fritz:")
	add("TECHNISCHE CONTROLES", "TECHNICAL CHECKS", "TECHNISCHE PRÜFUNGEN", "CONTRÔLES TECHNIQUES", "CONTROLES TÉCNICOS", "技术检查", "ТЕХНИЧЕСКИЕ ПРОВЕРКИ")
	add("BRONINVENTARIS", "SOURCE INVENTORY", "QUELLENINVENTAR", "INVENTAIRE DES SOURCES", "INVENTARIO DE FUENTES", "源清单", "ИНВЕНТАРЬ ИСТОЧНИКОВ")
	add("BRONNEN", "SOURCES", "QUELLEN", "SOURCES", "FUENTES", "源", "ИСТОЧНИКИ")
	add("INSTELLINGEN", "SETTINGS", "EINSTELLUNGEN", "PARAMÈTRES", "AJUSTES", "设置", "НАСТРОЙКИ")
	add("RAPPORT", "REPORT", "BERICHT", "RAPPORT", "INFORME", "报告", "ОТЧЁТ")
	add("VOLLEDIG PROCESRAPPORT", "FULL PROCESS REPORT", "VOLLSTÄNDIGER PROZESSBERICHT", "RAPPORT COMPLET DU PROCESSUS", "INFORME COMPLETO DEL PROCESO", "完整处理报告", "ПОЛНЫЙ ОТЧЁТ О ПРОЦЕССЕ")
	add("SHA-256 VAN DEFINITIEVE RUNBESTANDEN", "SHA-256 OF FINAL RUN FILES", "SHA-256 DER ENDGÜLTIGEN LAUFDATEIEN", "SHA-256 DES FICHIERS FINAUX", "SHA-256 DE LOS ARCHIVOS FINALES", "最终运行文件的 SHA-256", "SHA-256 ИТОГОВЫХ ФАЙЛОВ ЗАПУСКА")
	add("Gebouwd/overgeslagen/fout", "Built/skipped/failed", "Erstellt/übersprungen/fehlgeschlagen", "Créé/ignoré/échec", "Creado/omitido/fallido", "已生成/已跳过/失败", "Создано/пропущено/ошибка")
	add("geslaagd", "successful", "erfolgreich", "réussi", "correcto", "成功", "успешно")
	add("overgeslagen", "skipped", "übersprungen", "ignoré", "omitido", "已跳过", "пропущено")
	add("mislukt", "failed", "fehlgeschlagen", "échec", "fallido", "失败", "ошибка")
	add("Geverifieerd", "Verified", "Verifiziert", "Vérifié", "Verificado", "已验证", "Проверено")
	add("Geaccepteerd", "Accepted", "Akzeptiert", "Accepté", "Aceptado", "已接受", "Принято")
	add("Partijen gezien", "Games seen", "Gesehene Partien", "Parties vues", "Partidas vistas", "已查看对局", "Просмотрено партий")
	add("Partijen", "Games", "Partien", "Parties", "Partidas", "对局", "Партии")
	add("Modelpartijen", "Model games", "Modellpartien", "Parties modèles", "Partidas modelo", "模型对局", "Модельные партии")
	add("Ongeldige partijen", "Invalid games", "Ungültige Partien", "Parties invalides", "Partidas no válidas", "无效对局", "Недопустимые партии")
	add("Duplicaten", "Duplicates", "Duplikate", "Doublons", "Duplicados", "重复项", "Дубликаты")
	add("Posities", "Positions", "Positionen", "Positions", "Posiciones", "局面", "Позиции")
	add("Zetkandidaten", "Move candidates", "Zugkandidaten", "Coups candidats", "Jugadas candidatas", "候选着法", "Кандидаты ходов")
	add("Gekwalificeerd", "Qualified", "Qualifiziert", "Qualifié", "Calificado", "符合条件", "Квалифицировано")
	add("Voorkeurposities", "Preference positions", "Präferenzpositionen", "Positions préférées", "Posiciones preferidas", "偏好局面", "Позиции предпочтения")
	add("Voorkeurszetten", "Preferred moves", "Bevorzugte Züge", "Coups préférés", "Jugadas preferidas", "偏好着法", "Предпочтительные ходы")
	add("Decodefouten", "Decode errors", "Dekodierfehler", "Erreurs de décodage", "Errores de decodificación", "解码错误", "Ошибки декодирования")
	add("Duur", "Duration", "Dauer", "Durée", "Duración", "耗时", "Длительность")
	add("Modus", "Mode", "Modus", "Mode", "Modo", "模式", "Режим")
	add("Max diepte", "Max depth", "Max. Tiefe", "Profondeur max.", "Profundidad máx.", "最大深度", "Макс. глубина")
	add("Minimum partij", "Minimum game", "Mindestpartie", "Partie minimale", "Partida mínima", "最短对局", "Минимальная партия")
	add("Minimum Elo", "Minimum Elo", "Mindest-Elo", "Elo minimum", "Elo mínimo", "最低 Elo", "Минимальный Elo")
	add("Max Elo-verschil", "Max Elo difference", "Max. Elo-Differenz", "Écart Elo max.", "Diferencia Elo máx.", "最大 Elo 差", "Макс. разница Elo")
	add("Logische threads", "Logical threads", "Logische Threads", "Threads logiques", "Hilos lógicos", "逻辑线程", "Логические потоки")
	add("Vrij bij start", "Free at start", "Frei beim Start", "Libre au démarrage", "Libre al inicio", "启动时可用", "Свободно при старте")
	add("Bron", "Source", "Quelle", "Source", "Fuente", "源", "Источник")
	add("Formaat", "Format", "Format", "Format", "Formato", "格式", "Формат")
	add("Status", "Status", "Status", "Statut", "Estado", "状态", "Статус")
	add("compleet", "complete", "vollständig", "complet", "completo", "完整", "полный")
	add("bijbestand", "side file", "Begleitdatei", "fichier associé", "archivo asociado", "附属文件", "сопутствующий файл")
	add("ONVOLLEDIG", "INCOMPLETE", "UNVOLLSTÄNDIG", "INCOMPLET", "INCOMPLETO", "不完整", "НЕПОЛНЫЙ")
	add("OPMERKING", "NOTE", "HINWEIS", "REMARQUE", "NOTA", "说明", "ПРИМЕЧАНИЕ")
	return strings.NewReplacer(pairs...).Replace(s)
}

// localizeReportText translates report labels and fixed explanatory sentences
// without rewriting payload data such as paths, hashes, source names or PGN.
func localizeReportText(s string) string {
	if activeLanguage == "nl" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = localizeReportLine(line)
	}
	return strings.Join(lines, "\n")
}

func localizeReportLine(line string) string {
	trim := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Versioned report titles.
	titles := []struct{ nl, en, de, fr, es, zh, ru string }{
		{" - RAPPORT", " - REPORT", " - BERICHT", " - RAPPORT", " - INFORME", " - 报告", " - ОТЧЁТ"},
		{" - VOLLEDIG PROCESRAPPORT", " - FULL PROCESS REPORT", " - VOLLSTÄNDIGER PROZESSBERICHT", " - RAPPORT COMPLET DU PROCESSUS", " - INFORME COMPLETO DEL PROCESO", " - 完整处理报告", " - ПОЛНЫЙ ОТЧЁТ О ПРОЦЕССЕ"},
		{" - INSTELLINGEN", " - SETTINGS", " - EINSTELLUNGEN", " - PARAMÈTRES", " - AJUSTES", " - 设置", " - НАСТРОЙКИ"},
		{" - BRONINVENTARIS", " - SOURCE INVENTORY", " - QUELLENINVENTAR", " - INVENTAIRE DES SOURCES", " - INVENTARIO DE FUENTES", " - 源清单", " - ИНВЕНТАРЬ ИСТОЧНИКОВ"},
		{" - SHA-256 VAN DEFINITIEVE RUNBESTANDEN", " - SHA-256 OF FINAL RUN FILES", " - SHA-256 DER ENDGÜLTIGEN LAUFDATEIEN", " - SHA-256 DES FICHIERS FINAUX", " - SHA-256 DE LOS ARCHIVOS FINALES", " - 最终运行文件的 SHA-256", " - SHA-256 ИТОГОВЫХ ФАЙЛОВ ЗАПУСКА"},
		{" - METAL ROUTES", " - METAL ROUTES", " - METAL-ROUTEN", " - VOIES METAL", " - RUTAS METAL", " - METAL 路径", " - ВЕТКИ METAL"},
		{" adapterrapport", " adapter report", " Adapterbericht", " rapport d’adaptateur", " informe del adaptador", " 适配器报告", " отчёт адаптера"},
	}
	if strings.HasPrefix(trim, "Source2Metal v") {
		for _, t := range titles {
			if strings.HasSuffix(trim, t.nl) {
				return indent + strings.TrimSuffix(trim, t.nl) + L(t.en, t.de, t.nl, t.fr, t.es, t.zh, t.ru)
			}
		}
	}

	exact := map[string][6]string{
		"BRONNEN":                       {"SOURCES", "QUELLEN", "SOURCES", "FUENTES", "源", "ИСТОЧНИКИ"},
		"CHESSBASE-ADAPTERS":            {"CHESSBASE ADAPTERS", "CHESSBASE-ADAPTER", "ADAPTATEURS CHESSBASE", "ADAPTADORES CHESSBASE", "CHESSBASE 适配器", "АДАПТЕРЫ CHESSBASE"},
		"TECHNISCHE CONTROLES":          {"TECHNICAL CHECKS", "TECHNISCHE PRÜFUNGEN", "CONTRÔLES TECHNIQUES", "CONTROLES TÉCNICOS", "技术检查", "ТЕХНИЧЕСКИЕ ПРОВЕРКИ"},
		"PROCESKETEN":                   {"PROCESS FLOW", "PROZESSKETTE", "CHAÎNE DE TRAITEMENT", "FLUJO DEL PROCESO", "处理流程", "ЦЕПОЧКА ОБРАБОТКИ"},
		"SAMENGEVATTE RESULTATEN":       {"SUMMARY RESULTS", "ZUSAMMENGEFASSTE ERGEBNISSE", "RÉSULTATS RÉSUMÉS", "RESULTADOS RESUMIDOS", "结果摘要", "СВОДНЫЕ РЕЗУЛЬТАТЫ"},
		"HERLEIDBAARHEID PER BRON":      {"TRACEABILITY PER SOURCE", "RÜCKVERFOLGBARKEIT PRO QUELLE", "TRAÇABILITÉ PAR SOURCE", "TRAZABILIDAD POR FUENTE", "每源可追溯性", "ТРАССИРУЕМОСТЬ ПО ИСТОЧНИКАМ"},
		"CONTROLES EN VANGRAILS":        {"CHECKS AND GUARDRAILS", "PRÜFUNGEN UND SCHUTZREGELN", "CONTRÔLES ET GARDE-FOUS", "CONTROLES Y SALVAGUARDAS", "检查与保护措施", "ПРОВЕРКИ И ОГРАНИЧИТЕЛИ"},
		"OVERGESLAGEN - REDENEN":        {"SKIPPED - REASONS", "ÜBERSPRUNGEN - GRÜNDE", "IGNORÉ - RAISONS", "OMITIDO - MOTIVOS", "已跳过 - 原因", "ПРОПУЩЕНО - ПРИЧИНЫ"},
		"WAAROM GESCHEIDEN?":            {"WHY SEPARATE?", "WARUM GETRENNT?", "POURQUOI SÉPARER ?", "¿POR QUÉ SEPARADOS?", "为什么分开？", "ПОЧЕМУ РАЗДЕЛЬНО?"},
		"FRITZ - BIJLEREN UIT DATABASE": {"FRITZ - LEARN FROM DATABASE", "FRITZ - AUS DATENBANK LERNEN", "FRITZ - APPRENDRE DEPUIS UNE BASE", "FRITZ - APRENDER DE BASE DE DATOS", "FRITZ - 从数据库学习", "FRITZ - ОБУЧЕНИЕ ИЗ БАЗЫ"},
	}
	if v, ok := exact[trim]; ok {
		return indent + pick6(v)
	}

	// Fixed explanatory lines. These are matched exactly, so source data cannot
	// accidentally be translated.
	fixed := map[string][6]string{
		"Geen producttrace beschikbaar in deze modus.":                                                                                             {"No product trace is available in this mode.", "In diesem Modus ist keine Produktverfolgung verfügbar.", "Aucune trace de produit n’est disponible dans ce mode.", "No hay trazabilidad de producto disponible en este modo.", "此模式下没有可用的结果追踪。", "В этом режиме трассировка результата недоступна."},
		`PGN -------------------------------> per-bron GAME RAW ----\`:                                                                             {`PGN -------------------------------> per-source GAME RAW ----\`, `PGN -------------------------------> GAME RAW pro Quelle ----\`, `PGN -------------------------------> GAME RAW par source ----\`, `PGN -------------------------------> GAME RAW por fuente ----\`, `PGN -------------------------------> 每源 GAME RAW ----\`, `PGN -------------------------------> GAME RAW по источнику ----\`},
		"CBH  -> tijdelijke adapter ----------> per-bron GAME RAW -----+--> samengevoegde GAME RAW":                                                {"CBH  -> temporary adapter ----------> per-source GAME RAW -----+--> merged GAME RAW", "CBH  -> temporärer Adapter ----------> GAME RAW pro Quelle -----+--> zusammengeführtes GAME RAW", "CBH  -> adaptateur temporaire -------> GAME RAW par source -----+--> GAME RAW fusionné", "CBH  -> adaptador temporal -----------> GAME RAW por fuente -----+--> GAME RAW combinado", "CBH  -> 临时适配器 --------------------> 每源 GAME RAW ----------+--> 合并 GAME RAW", "CBH  -> временный адаптер -----------> GAME RAW по источнику ----+--> объединённый GAME RAW"},
		"2CBH -> tijdelijke adapter ----------> per-bron GAME RAW ----/":                                                                           {"2CBH -> temporary adapter ----------> per-source GAME RAW ----/", "2CBH -> temporärer Adapter ----------> GAME RAW pro Quelle ----/", "2CBH -> adaptateur temporaire -------> GAME RAW par source ----/", "2CBH -> adaptador temporal -----------> GAME RAW por fuente ----/", "2CBH -> 临时适配器 --------------------> 每源 GAME RAW ---------/", "2CBH -> временный адаптер -----------> GAME RAW по источнику ---/"},
		"PGN/CBH/2CBH -> legale positie-analyse -> gezamenlijke GAME evidence -> echte modelpartijen -> GAME METAL":                                {"PGN/CBH/2CBH -> legal position analysis -> combined GAME evidence -> real model games -> GAME METAL", "PGN/CBH/2CBH -> legale Positionsanalyse -> gemeinsame GAME-Evidenz -> echte Modellpartien -> GAME METAL", "PGN/CBH/2CBH -> analyse de positions légales -> preuve GAME combinée -> vraies parties modèles -> GAME METAL", "PGN/CBH/2CBH -> análisis legal de posiciones -> evidencia GAME combinada -> partidas modelo reales -> GAME METAL", "PGN/CBH/2CBH -> 合法局面分析 -> 合并 GAME 证据 -> 真实模型对局 -> GAME METAL", "PGN/CBH/2CBH -> анализ легальных позиций -> объединённые данные GAME -> реальные модельные партии -> GAME METAL"},
		"CTG/CTO/CTB -> CTG RAW-extractor -------------------------------------------> per-bron CTG RAW":                                           {"CTG/CTO/CTB -> CTG RAW extractor -------------------------------------------> per-source CTG RAW", "CTG/CTO/CTB -> CTG-RAW-Extraktor ------------------------------------------> CTG RAW pro Quelle", "CTG/CTO/CTB -> extracteur CTG RAW -----------------------------------------> CTG RAW par source", "CTG/CTO/CTB -> extractor CTG RAW ------------------------------------------> CTG RAW por fuente", "CTG/CTO/CTB -> CTG RAW 提取器 ---------------------------------------------> 每源 CTG RAW", "CTG/CTO/CTB -> экстрактор CTG RAW ----------------------------------------> CTG RAW по источнику"},
		"CTG/CTO/CTB -> strikte CTG Metal-selectie -> evt. zichtbare voorzichtige fallback -> per-bron CTG METAL":                                  {"CTG/CTO/CTB -> strict CTG Metal selection -> optional visible cautious fallback -> per-source CTG METAL", "CTG/CTO/CTB -> strikte CTG-Metal-Auswahl -> ggf. sichtbarer vorsichtiger Fallback -> CTG METAL pro Quelle", "CTG/CTO/CTB -> sélection CTG Metal stricte -> repli prudent visible éventuel -> CTG METAL par source", "CTG/CTO/CTB -> selección CTG Metal estricta -> posible alternativa prudente visible -> CTG METAL por fuente", "CTG/CTO/CTB -> 严格 CTG Metal 选择 -> 可选且明确显示的谨慎后备方案 -> 每源 CTG METAL", "CTG/CTO/CTB -> строгий отбор CTG Metal -> при необходимости явно показанный осторожный резерв -> CTG METAL по источнику"},
		"GAME RAW + CTG RAW -> samengevoegde ALLE Sources RAW (zonder cross-class dedup)":                                                          {"GAME RAW + CTG RAW -> merged ALL Sources RAW (without cross-class dedup)", "GAME RAW + CTG RAW -> zusammengeführtes ALL Sources RAW (ohne klassenübergreifende Deduplizierung)", "GAME RAW + CTG RAW -> ALL Sources RAW fusionné (sans déduplication interclasse)", "GAME RAW + CTG RAW -> ALL Sources RAW combinado (sin deduplicación entre clases)", "GAME RAW + CTG RAW -> 合并 ALL Sources RAW（不进行跨类别去重）", "GAME RAW + CTG RAW -> объединённый ALL Sources RAW (без межклассовой дедупликации)"},
		"Methode: transpositie-bewuste positie/zet-statistiek; uitvoer bestaat uit volledige legale bronpartijen met hun oorspronkelijke uitslag.": {"Method: transposition-aware position/move statistics; output consists of complete legal source games with their original result.", "Methode: transpositionsbewusste Positions-/Zugstatistik; die Ausgabe besteht aus vollständigen legalen Quellpartien mit Originalergebnis.", "Méthode : statistiques de positions/coups tenant compte des transpositions ; la sortie contient des parties source légales complètes avec leur résultat d’origine.", "Método: estadísticas de posición/jugada con transposiciones; la salida contiene partidas fuente legales completas con su resultado original.", "方法：转置感知的局面/着法统计；输出为保留原始结果的完整合法源对局。", "Метод: статистика позиций/ходов с учётом транспозиций; выход содержит полные легальные исходные партии с исходным результатом."},
		"- Bronnen worden alleen gelezen; originele bronbestanden worden niet gewijzigd.":                                                          {"- Sources are read-only; original source files are never modified.", "- Quellen werden nur gelesen; Originaldateien werden nie verändert.", "- Les sources sont en lecture seule ; les fichiers d’origine ne sont jamais modifiés.", "- Las fuentes son de solo lectura; los archivos originales nunca se modifican.", "- 源文件只读；原始文件绝不会被修改。", "- Источники читаются только для чтения; исходные файлы никогда не изменяются."},
		"- Bronzoektocht blijft binnen de gekozen root en onderliggende mappen; uitvoer- en tempmappen worden overgeslagen.":                       {"- Source discovery stays inside the selected root and its subfolders; output and temporary folders are skipped.", "- Die Quellensuche bleibt im gewählten Stammordner und dessen Unterordnern; Ausgabe- und Temp-Ordner werden übersprungen.", "- La recherche des sources reste dans le dossier racine choisi et ses sous-dossiers ; les dossiers de sortie et temporaires sont ignorés.", "- La búsqueda de fuentes permanece dentro de la raíz elegida y sus subcarpetas; se omiten las carpetas de salida y temporales.", "- 源搜索仅限于所选根目录及其子目录；输出和临时目录会被跳过。", "- Поиск источников выполняется только в выбранной корневой папке и её подпапках; выходные и временные папки пропускаются."},
		"- Bronzoektocht gaat alleen vanuit de gekozen root naar beneden; output- en tempmappen worden overgeslagen.":                              {"- Source discovery only descends from the selected root; output and temporary folders are skipped.", "- Die Quellensuche geht nur vom gewählten Stammordner abwärts; Ausgabe- und Temp-Ordner werden übersprungen.", "- La recherche des sources descend uniquement depuis le dossier racine choisi ; les dossiers de sortie et temporaires sont ignorés.", "- La búsqueda de fuentes solo desciende desde la raíz elegida; se omiten las carpetas de salida y temporales.", "- 源搜索仅从所选根目录向下进行；输出和临时目录会被跳过。", "- Поиск источников идёт только вниз от выбранной корневой папки; выходные и временные папки пропускаются."},
		"- ZIP/RAR/7z-archieven worden niet als bron geopend.":                                                                                     {"- ZIP/RAR/7z archives are not opened as sources.", "- ZIP/RAR/7z-Archive werden nicht als Quellen geöffnet.", "- Les archives ZIP/RAR/7z ne sont pas ouvertes comme sources.", "- Los archivos ZIP/RAR/7z no se abren como fuentes.", "- ZIP/RAR/7z 压缩包不会作为源打开。", "- Архивы ZIP/RAR/7z не открываются как источники."},
		"- GAME-METAL gebruikt volledige legale schaakbordcontrole en schrijft complete modelpartijen met originele uitslag.":                      {"- GAME-METAL performs full legal-board validation and writes complete model games with their original result.", "- GAME-METAL führt eine vollständige Legalitätsprüfung durch und schreibt komplette Modellpartien mit Originalergebnis.", "- GAME-METAL effectue une validation complète de la légalité et écrit des parties modèles complètes avec leur résultat d’origine.", "- GAME-METAL realiza una validación completa de legalidad y escribe partidas modelo completas con su resultado original.", "- GAME-METAL 进行完整的合法棋局验证，并写出带原始结果的完整模型对局。", "- GAME-METAL выполняет полную проверку легальности и записывает полные модельные партии с исходным результатом."},
		"- CTG RAW laat de uitvoergrootte door de werkelijk decodeerbare broninhoud bepalen.":                                                      {"- CTG RAW lets the actually decodable source content determine output size.", "- Bei CTG RAW bestimmt der tatsächlich dekodierbare Quellinhalt die Ausgabegröße.", "- CTG RAW laisse le contenu réellement décodable déterminer la taille de sortie.", "- CTG RAW deja que el contenido realmente decodificable determine el tamaño de salida.", "- CTG RAW 的输出大小由实际可解码的源内容决定。", "- Размер CTG RAW определяется фактически декодируемым содержимым источника."},
		"- GAME-METAL en CTG-METAL blijven herkenbaar gescheiden omdat zij verschillende soorten evidence coderen.":                                {"- GAME-METAL and CTG-METAL remain clearly separated because they encode different kinds of evidence.", "- GAME-METAL und CTG-METAL bleiben klar getrennt, da sie unterschiedliche Evidenzarten codieren.", "- GAME-METAL et CTG-METAL restent clairement séparés car ils codent des types de preuves différents.", "- GAME-METAL y CTG-METAL permanecen claramente separados porque codifican tipos de evidencia distintos.", "- GAME-METAL 与 CTG-METAL 保持明确分离，因为它们编码不同类型的证据。", "- GAME-METAL и CTG-METAL остаются чётко разделёнными, поскольку кодируют разные типы данных."},
		"- Fritz-inleerinstelling: Overwinningen + Verliespartijen AAN; Wit/Zwart/Speler UIT; spelernaam leeg; alle partijen.":                     {"- Fritz learning setting: Wins + Losses ON; White/Black/Player OFF; player name empty; all games.", "- Fritz-Lerneinstellung: Siege + Niederlagen EIN; Weiß/Schwarz/Spieler AUS; Spielername leer; alle Partien.", "- Réglage d’apprentissage Fritz : Victoires + Défaites ACTIVÉES ; Blancs/Noirs/Joueur DÉSACTIVÉS ; nom du joueur vide ; toutes les parties.", "- Ajuste de aprendizaje de Fritz: Victorias + Derrotas ACTIVADAS; Blancas/Negras/Jugador DESACTIVADOS; nombre vacío; todas las partidas.", "- Fritz 学习设置：胜局 + 负局开启；白方/黑方/棋手关闭；棋手名留空；所有对局。", "- Настройка обучения Fritz: Победы + Поражения ВКЛ.; Белые/Чёрные/Игрок ВЫКЛ.; имя игрока пустое; все партии."},
		"Overwinningen + Verliespartijen AAN; Wit/Zwart/Speler UIT; spelernaam leeg;":                                                              {"Wins + Losses ON; White/Black/Player OFF; player name empty;", "Siege + Niederlagen EIN; Weiß/Schwarz/Spieler AUS; Spielername leer;", "Victoires + Défaites ACTIVÉES ; Blancs/Noirs/Joueur DÉSACTIVÉS ; nom du joueur vide ;", "Victorias + Derrotas ACTIVADAS; Blancas/Negras/Jugador DESACTIVADOS; nombre vacío;", "胜局 + 负局开启；白方/黑方/棋手关闭；棋手名留空；", "Победы + Поражения ВКЛ.; Белые/Чёрные/Игрок ВЫКЛ.; имя игрока пустое;"},
		"alle partijen.": {"all games.", "alle Partien.", "toutes les parties.", "todas las partidas.", "所有对局。", "все партии."},
	}
	if v, ok := fixed[trim]; ok {
		return indent + pick6(v)
	}

	// Translate left-hand report labels only; keep values untouched.
	if idx := strings.Index(line, ":"); idx >= 0 {
		left := strings.TrimSpace(line[:idx])
		rest := line[idx+1:]
		if tr, ok := reportLabel(left); ok {
			rest = localizeReportValue(left, rest)
			out := indent + tr + ":" + rest
			return localizeStructuredReportLine(out)
		}
	}

	// Source inventory field inside a pipe-delimited line.
	if strings.Contains(line, " | compleet=") {
		parts := strings.Split(line, " | ")
		for i, p := range parts {
			if strings.HasPrefix(p, "compleet=") {
				parts[i] = L("complete=", "vollständig=", "compleet=", "complet=", "completo=", "完整=", "полный=") + strings.TrimPrefix(p, "compleet=")
			}
		}
		return strings.Join(parts, " | ")
	}

	return localizeStructuredReportLine(line)
}

func localizeReportValue(label, rest string) string {
	switch label {
	case "Workers":
		rest = strings.ReplaceAll(rest, "logische threads", L("logical threads", "logische Threads", "logische threads", "threads logiques", "hilos lógicos", "逻辑线程", "логических потоков"))
	case "Elo-filter":
		rest = strings.ReplaceAll(rest, "bronselectie behouden", L("keep source selection", "Quellenauswahl beibehalten", "bronselectie behouden", "conserver la sélection source", "mantener selección de origen", "保留源选择", "сохранить исходный отбор"))
		rest = strings.ReplaceAll(rest, "beide spelers", L("both players", "beide Spieler", "beide spelers", "les deux joueurs", "ambos jugadores", "双方棋手", "оба игрока"))
	case "Max Elo-verschil":
		rest = strings.ReplaceAll(rest, "geen", L("none", "keine", "geen", "aucun", "ninguno", "无", "нет"))
	case "Modus":
		rest = strings.ReplaceAll(rest, "inventory", L("inventory", "Inventar", "inventory", "inventaire", "inventario", "清单", "инвентаризация"))
	}
	return rest
}

func reportLabel(nl string) (string, bool) {
	labels := map[string][6]string{
		"Status":                    {"Status", "Status", "Statut", "Estado", "状态", "Статус"},
		"Runstatus":                 {"Run status", "Laufstatus", "Statut du run", "Estado del run", "运行状态", "Статус запуска"},
		"Run-ID":                    {"Run ID", "Lauf-ID", "ID du run", "ID de ejecución", "运行 ID", "ID запуска"},
		"Start":                     {"Start", "Start", "Début", "Inicio", "开始", "Начало"},
		"Einde":                     {"End", "Ende", "Fin", "Fin", "结束", "Окончание"},
		"Duur":                      {"Duration", "Dauer", "Durée", "Duración", "耗时", "Длительность"},
		"Root":                      {"Root", "Root", "Racine", "Raíz", "根目录", "Корень"},
		"Modus":                     {"Mode", "Modus", "Mode", "Modo", "模式", "Режим"},
		"Workers":                   {"Workers", "Worker", "Workers", "Workers", "Workers", "Рабочие потоки"},
		"Vrij bij start":            {"Free at start", "Frei beim Start", "Libre au démarrage", "Libre al inicio", "启动时可用", "Свободно при старте"},
		"Max diepte":                {"Max depth", "Max. Tiefe", "Profondeur max.", "Profundidad máx.", "最大深度", "Макс. глубина"},
		"Minimum partij":            {"Minimum game", "Mindestpartie", "Partie minimale", "Partida mínima", "最短对局", "Минимальная партия"},
		"Minimum Elo":               {"Minimum Elo", "Mindest-Elo", "Elo minimum", "Elo mínimo", "最低 Elo", "Минимальный Elo"},
		"Elo-filter":                {"Elo filter", "Elo-Filter", "Filtre Elo", "Filtro Elo", "Elo 过滤器", "Фильтр Elo"},
		"Max Elo-verschil":          {"Max Elo difference", "Max. Elo-Differenz", "Écart Elo max.", "Diferencia Elo máx.", "最大 Elo 差", "Макс. разница Elo"},
		"Logische threads":          {"Logical threads", "Logische Threads", "Threads logiques", "Hilos lógicos", "逻辑线程", "Логические потоки"},
		"Partijen gezien":           {"Games seen", "Gesehene Partien", "Parties vues", "Partidas vistas", "已查看对局", "Просмотрено партий"},
		"Geaccepteerd":              {"Accepted", "Akzeptiert", "Accepté", "Aceptado", "已接受", "Принято"},
		"Geverifieerd":              {"Verified", "Verifiziert", "Vérifié", "Verificado", "已验证", "Проверено"},
		"Exact duplicaat":           {"Exact duplicate", "Exaktes Duplikat", "Doublon exact", "Duplicado exacto", "完全重复", "Точный дубликат"},
		"Geen geldige uitslag":      {"No valid result", "Kein gültiges Ergebnis", "Aucun résultat valide", "Sin resultado válido", "无有效结果", "Нет допустимого результата"},
		"Te kort":                   {"Too short", "Zu kurz", "Trop court", "Demasiado corta", "过短", "Слишком короткие"},
		"Elo-verschil":              {"Elo difference", "Elo-Differenz", "Écart Elo", "Diferencia Elo", "Elo 差", "Разница Elo"},
		"Variatietakken verwijderd": {"Variation branches removed", "Variantenäste entfernt", "Branches de variantes supprimées", "Ramas de variantes eliminadas", "已删除变化分支", "Удалено ветвей вариантов"},
		"Commentaren verwijderd":    {"Comments removed", "Kommentare entfernt", "Commentaires supprimés", "Comentarios eliminados", "已删除注释", "Удалено комментариев"},
		"Ply geschreven":            {"Ply written", "Ply geschrieben", "Ply écrits", "Ply escritos", "已写入 ply", "Записано ply"},
		"Methode":                   {"Method", "Methode", "Méthode", "Método", "方法", "Метод"},
		"Unieke legale partijen":    {"Unique legal games", "Eindeutige legale Partien", "Parties légales uniques", "Partidas legales únicas", "唯一合法对局", "Уникальные легальные партии"},
		"Ongeldige partijen":        {"Invalid games", "Ungültige Partien", "Parties invalides", "Partidas no válidas", "无效对局", "Недопустимые партии"},
		"Duplicaten":                {"Duplicates", "Duplikate", "Doublons", "Duplicados", "重复项", "Дубликаты"},
		"Posities":                  {"Positions", "Positionen", "Positions", "Posiciones", "局面", "Позиции"},
		"Zetkandidaten":             {"Move candidates", "Zugkandidaten", "Coups candidats", "Jugadas candidatas", "候选着法", "Кандидаты ходов"},
		"Gekwalificeerd":            {"Qualified", "Qualifiziert", "Qualifié", "Calificado", "符合条件", "Квалифицировано"},
		"Voorkeurposities":          {"Preference positions", "Präferenzpositionen", "Positions préférées", "Posiciones preferidas", "偏好局面", "Позиции предпочтения"},
		"Voorkeurszetten":           {"Preferred moves", "Bevorzugte Züge", "Coups préférés", "Jugadas preferidas", "偏好着法", "Предпочтительные ходы"},
		"Modelpartijen":             {"Model games", "Modellpartien", "Parties modèles", "Partidas modelo", "模型对局", "Модельные партии"},
		"Gebouwd/overgeslagen/fout": {"Built/skipped/failed", "Erstellt/übersprungen/fehlgeschlagen", "Créé/ignoré/échec", "Creado/omitido/fallido", "已生成/已跳过/失败", "Создано/пропущено/ошибка"},
		"Partijen":                  {"Games", "Partien", "Parties", "Partidas", "对局", "Партии"},
		"Decodefouten":              {"Decode errors", "Dekodierfehler", "Erreurs de décodage", "Errores de decodificación", "解码错误", "Ошибки декодирования"},
		"Records":                   {"Records", "Datensätze", "Enregistrements", "Registros", "记录", "Записи"},
		"Bron":                      {"Source", "Quelle", "Source", "Fuente", "源", "Источник"},
		"Formaat":                   {"Format", "Format", "Format", "Formato", "格式", "Формат"},
		"Set compleet":              {"Set complete", "Set vollständig", "Ensemble complet", "Conjunto completo", "集合完整", "Набор полный"},
		"Records fysiek":            {"Physical records", "Physische Datensätze", "Enregistrements physiques", "Registros físicos", "物理记录", "Физические записи"},
		"Niet-actief":               {"Inactive", "Inaktiv", "Inactif", "Inactivo", "非活动", "Неактивно"},
		"Geconverteerd":             {"Converted", "Konvertiert", "Converti", "Convertido", "已转换", "Преобразовано"},
		"Overgeslagen":              {"Skipped", "Übersprungen", "Ignoré", "Omitido", "已跳过", "Пропущено"},
		"Variatietakken":            {"Variation branches", "Variantenäste", "Branches de variantes", "Ramas de variantes", "变化分支", "Ветви вариантов"},
		"Fout":                      {"Error", "Fehler", "Erreur", "Error", "错误", "Ошибка"},
		"RAW resultaat":             {"RAW result", "RAW-Ergebnis", "Résultat RAW", "Resultado RAW", "RAW 结果", "Результат RAW"},
		"RAW geproduceerd":          {"RAW produced", "RAW erzeugt", "RAW produit", "RAW producido", "已生成 RAW", "RAW создано"},
		"RAW gezien":                {"RAW seen", "RAW gesehen", "RAW vus", "RAW vistos", "已查看 RAW", "RAW просмотрено"},
		"RAW per bron":              {"RAW per source", "RAW pro Quelle", "RAW par source", "RAW por fuente", "每源 RAW", "RAW по источнику"},
		"METAL resultaat":           {"METAL result", "METAL-Ergebnis", "Résultat METAL", "Resultado METAL", "METAL 结果", "Результат METAL"},
		"METAL geselecteerd":        {"METAL selected", "METAL ausgewählt", "METAL sélectionné", "METAL seleccionado", "已选择 METAL", "METAL отобрано"},
		"METAL gezien":              {"METAL seen", "METAL gesehen", "METAL vus", "METAL vistos", "已查看 METAL", "METAL просмотрено"},
	}
	v, ok := labels[nl]
	if !ok {
		return "", false
	}
	return pick6(v), true
}

func pick6(v [6]string) string {
	switch activeLanguage {
	case "de":
		return v[1]
	case "fr":
		return v[2]
	case "es":
		return v[3]
	case "zh":
		return v[4]
	case "ru":
		return v[5]
	default:
		return v[0]
	}
}

func localizeStructuredReportLine(line string) string {
	// Only rewrite result vocabulary on known summary/status lines; do not touch
	// arbitrary source/path lines.
	trim := strings.TrimSpace(line)
	known := strings.HasPrefix(trim, "GAME RAW") || strings.HasPrefix(trim, "CTG RAW") || strings.HasPrefix(trim, "GAME METAL") || strings.HasPrefix(trim, "CTG METAL") || strings.HasPrefix(trim, "ALLE SOURCES RAW") || strings.HasPrefix(trim, "CBH  ") || strings.HasPrefix(trim, "2CBH")
	if !known {
		return line
	}
	// Preserve a trailing output path after the final " | " when present.
	main, tail := line, ""
	if strings.Count(line, " | ") >= 2 {
		if idx := strings.LastIndex(line, " | "); idx >= 0 {
			candidate := line[idx+3:]
			if strings.Contains(candidate, "/") || strings.Contains(candidate, "\\") || strings.HasSuffix(strings.ToLower(candidate), ".pgn") {
				main, tail = line[:idx], line[idx:]
			}
		}
	}
	repls := [][7]string{
		{"geslaagd", "successful", "erfolgreich", "réussi", "correcto", "成功", "успешно"},
		{"gebouwd", "built", "erstellt", "créé", "creado", "已生成", "создано"},
		{"overgeslagen", "skipped", "übersprungen", "ignoré", "omitido", "已跳过", "пропущено"},
		{"mislukt", "failed", "fehlgeschlagen", "échec", "fallido", "失败", "ошибка"},
		{"geverifieerd", "verified", "verifiziert", "vérifié", "verificado", "已验证", "проверено"},
		{"modelpartijen", "model games", "Modellpartien", "parties modèles", "partidas modelo", "模型对局", "модельных партий"},
		{"adapterpartijen", "adapter games", "Adapterpartien", "parties adaptateur", "partidas del adaptador", "适配器对局", "партий адаптера"},
	}
	for _, r := range repls {
		tr := L(r[1], r[2], r[0], r[3], r[4], r[5], r[6])
		main = strings.ReplaceAll(main, r[0], tr)
	}
	return main + tail
}
