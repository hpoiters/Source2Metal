package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var stdin = bufio.NewReader(os.Stdin)

func main() {
	initLanguage(len(os.Args) == 1)
	if err := releaseConsistencyCheck(); err != nil {
		fmt.Fprintln(os.Stderr, "RELEASE CONSISTENCY ERROR:", err)
		os.Exit(3)
	}
	cfg, err := parseConfig()
	if err != nil {
		fatal(cfg, err)
		return
	}
	if cfg.Interactive {
	menuLoop:
		for {
			action := startupMenu()
			switch action {
			case "run":
				break menuLoop
			case "extract-syzygy-package":
				p, err := extractSyzygyPackage()
				if err != nil {
					fmt.Println(L("ERROR:", "FEHLER:", "FOUT:", "ERREUR :", "ERROR:", "错误：", "ОШИБКА:"), err)
				} else {
					fmt.Println("\n"+L("SyzygyCheck package ready:", "SyzygyCheck-Paket fertig:", "SyzygyCheck-pakket klaar:", "Paquet SyzygyCheck prêt :", "Paquete SyzygyCheck listo:", "SyzygyCheck 包已就绪：", "Пакет SyzygyCheck готов:"), p)
					fmt.Println("SHA-256:", embeddedSyzygyPackageSHA256())
					fmt.Println(L("Only this ZIP file was written; Source2Metal does not run SyzygyCheck.", "Es wurde nur diese ZIP-Datei geschrieben; Source2Metal startet SyzygyCheck nicht.", "Alleen dit ZIP-bestand is geschreven; Source2Metal start SyzygyCheck niet.", "Seul ce fichier ZIP a été écrit ; Source2Metal ne lance pas SyzygyCheck.", "Solo se ha escrito este archivo ZIP; Source2Metal no ejecuta SyzygyCheck.", "只写出了这个 ZIP 文件；Source2Metal 不会运行 SyzygyCheck。", "Записан только этот ZIP-файл; Source2Metal не запускает SyzygyCheck."))
				}
				pause()
			case "buildinfo":
				fmt.Print("\n" + buildInfoText())
				pause()
			case "pagefile":
				fmt.Println("\n" + localizeUserText(pagefileHelpText()))
				pause()
			case "language":
				chooseLanguage(false)
			case "language-return":
				continue
			case "quit":
				return
			}
		}
	}
	if cfg.BuildInfo {
		fmt.Print(buildInfoText())
		return
	}
	if cfg.ExtractSyzygyPackage {
		p, err := extractSyzygyPackage()
		if err != nil {
			fatal(cfg, err)
			return
		}
		fmt.Println(L("SyzygyCheck package extracted from EXE:", "SyzygyCheck-Paket aus EXE extrahiert:", "SyzygyCheck-pakket uit EXE uitgepakt:", "Paquet SyzygyCheck extrait de l’EXE :", "Paquete SyzygyCheck extraído del EXE:", "已从 EXE 解压 SyzygyCheck 包：", "Пакет SyzygyCheck извлечён из EXE:"), p)
		fmt.Println("SHA-256:", embeddedSyzygyPackageSHA256())
		return
	}
	if cfg.PagefileHelp {
		fmt.Println(pagefileHelpText())
		return
	}
	if cfg.SelfTest {
		if err := selfTest(); err != nil {
			fatal(cfg, err)
			return
		}
		fmt.Println(L("Source2Metal self-test: OK", "Source2Metal-Selbsttest: OK", "Source2Metal selftest: OK", "Auto-test Source2Metal : OK", "Autoprueba Source2Metal: OK", "Source2Metal 自检：OK", "Самотест Source2Metal: OK"))
		return
	}
	if err := run(cfg); err != nil {
		fatal(cfg, err)
		return
	}
	if !cfg.NoPause && cfg.Interactive {
		pause()
	}
}

func parseConfig() (Config, error) {
	c := Config{Mode: "", MaxPly: defaultMaxPly, MinPly: defaultMinPly}
	flag.StringVar(&c.Input, "input", "", L("input folder; default: EXE folder", "Eingabeordner; Standard: EXE-Ordner", "invoermap; standaard: EXE-map", "dossier d’entrée ; défaut : dossier EXE", "carpeta de entrada; predeterminado: carpeta EXE", "输入文件夹；默认：EXE 文件夹", "папка ввода; по умолчанию: папка EXE"))
	flag.StringVar(&c.Mode, "mode", "", "inventory|raw|all")
	flag.IntVar(&c.MaxPly, "max-ply", defaultMaxPly, L("maximum RAW depth 1..120", "maximale RAW-Tiefe 1..120", "maximale RAW-diepte 1..120", "profondeur RAW maximale 1..120", "profundidad RAW máxima 1..120", "最大 RAW 深度 1..120", "максимальная глубина RAW 1..120"))
	flag.IntVar(&c.MinPly, "min-ply", defaultMinPly, L("minimum game length", "Mindestlänge der Partie", "minimum partij-lengte", "longueur minimale de la partie", "longitud mínima de partida", "最短对局长度", "минимальная длина партии"))
	flag.IntVar(&c.MinElo, "min-elo", 0, L("0 or minimum Elo for both players", "0 oder Mindest-Elo für beide Spieler", "0 of minimum Elo voor beide spelers", "0 ou Elo minimum pour les deux joueurs", "0 o Elo mínimo para ambos jugadores", "0 或双方棋手最低 Elo", "0 или минимальный Elo для обоих игроков"))
	flag.IntVar(&c.MaxEloGap, "max-elo-gap", 0, L("0=off", "0=aus", "0=uit", "0=désactivé", "0=desactivado", "0=关闭", "0=выкл."))
	flag.IntVar(&c.Workers, "workers", 0, L("0=automatic (50% logical threads)", "0=automatisch (50% logische Threads)", "0=automatisch (50% logische threads)", "0=automatique (50% des threads logiques)", "0=automático (50% de hilos lógicos)", "0=自动（50% 逻辑线程）", "0=автоматически (50% логических потоков)"))
	flag.BoolVar(&c.NoPause, "no-pause", false, L("do not pause", "nicht pausieren", "niet pauzeren", "ne pas mettre en pause", "no pausar", "不暂停", "не делать паузу"))
	flag.BoolVar(&c.SelfTest, "selftest", false, L("internal test", "interner Test", "interne test", "test interne", "prueba interna", "内部测试", "внутренний тест"))
	flag.BoolVar(&c.ExtractSyzygyPackage, "extract-syzygy-package", false, L("extract the embedded SyzygyCheck utility next to the EXE", "eingebettetes SyzygyCheck-Hilfsprogramm neben der EXE entpacken", "pak de ingebedde SyzygyCheck-utility naast de EXE uit", "extraire l’utilitaire SyzygyCheck intégré à côté de l’EXE", "extraer la utilidad SyzygyCheck integrada junto al EXE", "将内嵌 SyzygyCheck 实用工具解压到 EXE 旁边", "извлечь встроенную утилиту SyzygyCheck рядом с EXE"))
	flag.BoolVar(&c.BuildInfo, "build-info", false, L("show build and source information", "Build- und Quellcodeinformationen anzeigen", "toon build- en broninformatie", "afficher les informations de compilation et de source", "mostrar información de compilación y fuente", "显示构建和源代码信息", "показать сведения о сборке и исходном коде"))
	flag.BoolVar(&c.PagefileHelp, "pagefile-help", false, L("show Windows pagefile advice", "Windows-Pagefile-Hinweis anzeigen", "toon Windows-pagefile advies", "afficher les conseils sur le fichier d’échange Windows", "mostrar consejos sobre el archivo de paginación de Windows", "显示 Windows 页面文件建议", "показать рекомендации по файлу подкачки Windows"))
	flag.Parse()
	c.Interactive = len(os.Args) == 1
	if c.Input == "" {
		exe, err := os.Executable()
		if err != nil {
			return c, err
		}
		c.Input = filepath.Dir(exe)
	}
	if c.MaxPly < 1 || c.MaxPly > maxAllowedPly {
		return c, fmt.Errorf(L("max-ply must be 1..%d", "max-ply muss 1..%d sein", "max-ply moet 1..%d zijn", "max-ply doit être compris entre 1 et %d", "max-ply debe ser 1..%d", "max-ply 必须为 1..%d", "max-ply должен быть 1..%d"), maxAllowedPly)
	}
	if c.MinPly < 1 || c.MinPly > maxAllowedPly {
		return c, fmt.Errorf(L("min-ply must be 1..%d", "min-ply muss 1..%d sein", "min-ply moet 1..%d zijn", "min-ply doit être compris entre 1 et %d", "min-ply debe ser 1..%d", "min-ply 必须为 1..%d", "min-ply должен быть 1..%d"), maxAllowedPly)
	}
	return c, nil
}

func run(cfg Config) error {
	root, err := filepath.Abs(cfg.Input)
	if err != nil {
		return err
	}
	hw := detectHardware(root)
	if cfg.Workers <= 0 {
		cfg.Workers = defaultWorkers(hw.LogicalThreads)
	}
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Workers > hw.LogicalThreads {
		cfg.Workers = hw.LogicalThreads
	}

	fmt.Printf("Source2Metal v%s\n", version)
	fmt.Printf("Platform         : %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf(U("Werkmap          : %s\n"), root)
	printHardware(hw)

	fmt.Println("\n" + U("1/6 Bronnen inventariseren..."))
	inv, err := scanSources(root)
	if err != nil {
		return err
	}
	printInventory(inv)
	if cfg.Interactive {
		cfg, err = interactiveChoices(cfg, inv, hw)
		if err != nil {
			return err
		}
	}
	if cfg.Mode == "" {
		cfg.Mode = "inventory"
	}

	stamp := time.Now().Format("2006-01-02_15-04-05")
	layout, err := createLayout(root, stamp)
	if err != nil {
		return err
	}
	started := time.Now()
	c := Counters{Started: started}
	for _, s := range inv.Sources {
		switch s.Kind {
		case KindPGN:
			c.FilesPGN++
		case KindCTG:
			c.SetsCTG++
		case KindCBH:
			c.FilesCBH++
		case Kind2CBH:
			c.Files2CBH++
		}
	}

	fmt.Println("\n" + U("2/6 Bronnen valideren..."))
	fmt.Println(L("PGN, CBH and 2CBH sources are GAME sources: they build RAW and, in mode 3, also feed GAME-METAL.", "PGN-, CBH- und 2CBH-Quellen sind GAME-Quellen: Sie erstellen RAW und speisen in Modus 3 auch GAME-METAL.", "PGN-, CBH- en 2CBH-bronnen worden GAME-bronnen: zij bouwen RAW en voeden in modus 3 ook GAME-METAL.", "Les sources PGN, CBH et 2CBH sont des sources GAME : elles construisent RAW et alimentent aussi GAME-METAL en mode 3.", "Las fuentes PGN, CBH y 2CBH son fuentes GAME: crean RAW y, en modo 3, también alimentan GAME-METAL.", "PGN、CBH 和 2CBH 源属于 GAME 源：它们构建 RAW，并在模式 3 中同时供给 GAME-METAL。", "Источники PGN, CBH и 2CBH являются GAME-источниками: они создают RAW и в режиме 3 также питают GAME-METAL."))
	fmt.Println(L("Complete CTG/CTO/CTB sets build separate CTG RAW and feed the CTG-METAL route.", "Vollständige CTG/CTO/CTB-Sets erstellen separates CTG RAW und speisen die CTG-METAL-Route.", "Complete CTG/CTO/CTB-sets bouwen afzonderlijke CTG RAW en voeden de CTG-METAL-route.", "Les ensembles CTG/CTO/CTB complets construisent un CTG RAW séparé et alimentent la voie CTG-METAL.", "Los conjuntos CTG/CTO/CTB completos crean CTG RAW separado y alimentan la ruta CTG-METAL.", "完整的 CTG/CTO/CTB 集会构建独立的 CTG RAW，并供给 CTG-METAL 路径。", "Полные наборы CTG/CTO/CTB создают отдельный CTG RAW и питают ветку CTG-METAL."))

	var rawGamesPath string
	fmt.Println(binScopeText())
	var rawBookPaths []string
	var cbSources []Source
	if cfg.Mode == "raw" || cfg.Mode == "all" {
		fmt.Println("\n" + U("3/6 RAW bouwen..."))
		cbSources = prepareChessBaseSources(inv, layout, &c)
		if c.FilesPGN > 0 || len(cbSources) > 0 {
			rawGamesPath, err = buildRawGames(inv, layout.RawGamesFile, cfg, &c, cbSources...)
			if err != nil {
				c.Elapsed = time.Since(started)
				pruneEmptyOutputDirs(layout)
				_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
				return err
			}
			fmt.Println(U("GAME RAW gemaakt:"), rawGamesPath)
		} else {
			fmt.Println(U("Geen bruikbare PGN/CBH/2CBH GAME-bronnen: GAME RAW overgeslagen."))
		}

		rawBookPaths, err = buildRawBooks(inv, layout, cfg, &c)
		if err != nil {
			c.Elapsed = time.Since(started)
			pruneEmptyOutputDirs(layout)
			_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
			return err
		}
		binPaths, binErr := buildBINRaw(inv, layout, cfg, &c)
		if binErr != nil {
			c.Elapsed = time.Since(started)
			_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
			return binErr
		}
		rawBookPaths = append(rawBookPaths, binPaths...)
		bookPath, mergeErr := mergeBookRAW(rawBookPaths, layout, &c)
		if mergeErr != nil {
			c.Elapsed = time.Since(started)
			_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
			return mergeErr
		}
		rawBookPaths = nil
		if bookPath != "" {
			rawBookPaths = []string{bookPath}
		}
		if combined, e := buildCombined(rawGamesPath, rawBookPaths, layout, &c); e != nil {
			c.Elapsed = time.Since(started)
			pruneEmptyOutputDirs(layout)
			_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
			return e
		} else if combined != "" {
			fmt.Println(U("ALLE SOURCES RAW gemaakt:"), combined)
		} else {
			fmt.Println(bookPolicyText())
		}
	} else {
		fmt.Println("\n" + U("3/6 RAW bouwen: overgeslagen"))
	}

	if cfg.Mode == "all" {
		fmt.Println("\n" + U("4/6 METAL bouwen..."))
		fmt.Println(L("GAME-METAL: PGN/CBH/2CBH are evaluated with transposition awareness; output consists of real source games with their original result.", "GAME-METAL: PGN/CBH/2CBH werden transpositionsbewusst bewertet; die Ausgabe besteht aus echten Quellpartien mit Originalergebnis.", "GAME-METAL: PGN/CBH/2CBH worden transpositie-bewust beoordeeld; uitvoer bestaat uit echte bronpartijen met hun originele uitslag.", "GAME-METAL : PGN/CBH/2CBH sont évalués en tenant compte des transpositions ; la sortie contient de vraies parties source avec leur résultat d’origine.", "GAME-METAL: PGN/CBH/2CBH se evalúan teniendo en cuenta las transposiciones; la salida contiene partidas fuente reales con su resultado original.", "GAME-METAL：PGN/CBH/2CBH 按转置感知方式评估；输出为保留原始结果的真实源对局。", "GAME-METAL: PGN/CBH/2CBH оцениваются с учётом транспозиций; выход состоит из реальных исходных партий с их исходным результатом."))
		if err := buildMetalFromGames(inv, cbSources, layout, cfg, &c); err != nil {
			if c.GameMetalFailed == 0 {
				c.GameMetalFailed++
			}
			c.Elapsed = time.Since(started)
			pruneEmptyOutputDirs(layout)
			_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
			return err
		}
		fmt.Println("\n" + L("CTG-METAL: each complete CTG set is processed separately by the book-oriented Metal route.", "CTG-METAL: Jedes vollständige CTG-Set wird separat durch die buchorientierte Metal-Route verarbeitet.", "CTG-METAL: iedere complete CTG-set wordt afzonderlijk door de boekgerichte Metal-route verwerkt.", "CTG-METAL : chaque ensemble CTG complet est traité séparément par la voie Metal orientée livre.", "CTG-METAL: cada conjunto CTG completo se procesa por separado mediante la ruta Metal orientada a libros.", "CTG-METAL：每个完整 CTG 集都由面向开局库的 Metal 路径单独处理。", "CTG-METAL: каждый полный набор CTG отдельно обрабатывается книжной веткой Metal."))
		if err := buildMetalFromCTG(inv, layout, cfg, &c); err != nil {
			c.Elapsed = time.Since(started)
			pruneEmptyOutputDirs(layout)
			_ = writeReports(layout, inv, cfg, hw, c, "MISLUKT")
			return err
		}
	} else {
		fmt.Println("\n" + U("4/6 METAL bouwen: overgeslagen"))
	}

	fmt.Println("\n" + U("5/6 Rapporten + checksums..."))
	c.Elapsed = time.Since(started)
	pruneEmptyOutputDirs(layout)
	finalStatus := classifyRunStatus(cfg, c)
	if err := writeReports(layout, inv, cfg, hw, c, finalStatus); err != nil {
		return err
	}

	fmt.Println("\n" + U("6/6 Tijdelijke bestanden opruimen..."))
	_ = os.RemoveAll(layout.TempDir)
	pruneEmptyOutputDirs(layout)
	fmt.Println(U("TEMP opgeruimd."))

	fmt.Println("\n"+U("Klaar. Resultaten:"), layout.RunDir)
	fmt.Println(L("RUN STATUS      :", "LAUFSTATUS      :", "RUNSTATUS       :", "STATUT DU RUN   :", "ESTADO DEL RUN  :", "运行状态        :", "СТАТУС ЗАПУСКА  :"), localizeStatus(finalStatus))
	fmt.Printf(L("Workers used    : %d / %d logical threads\n", "Worker verwendet: %d / %d logische Threads\n", "Workers gebruikt: %d / %d logische threads\n", "Workers utilisés : %d / %d threads logiques\n", "Workers usados   : %d / %d hilos lógicos\n", "使用的 worker    ：%d / %d 逻辑线程\n", "Рабочие потоки  : %d / %d логических потоков\n"), cfg.Workers, hw.LogicalThreads)
	fmt.Printf(L("CBH adapter     : %d successful | %d failed\n", "CBH-Adapter     : %d erfolgreich | %d fehlgeschlagen\n", "CBH adapter     : %d geslaagd | %d mislukt\n", "Adaptateur CBH  : %d réussi | %d échec\n", "Adaptador CBH   : %d correcto | %d fallido\n", "CBH 适配器      ：%d 成功 | %d 失败\n", "Адаптер CBH    : %d успешно | %d ошибок\n"), c.CBHSetsBuilt, c.CBHSetsFailed)
	fmt.Printf(L("2CBH adapter    : %d successful | %d failed\n", "2CBH-Adapter    : %d erfolgreich | %d fehlgeschlagen\n", "2CBH adapter    : %d geslaagd | %d mislukt\n", "Adaptateur 2CBH : %d réussi | %d échec\n", "Adaptador 2CBH  : %d correcto | %d fallido\n", "2CBH 适配器     ：%d 成功 | %d 失败\n", "Адаптер 2CBH   : %d успешно | %d ошибок\n"), c.TwoCBHSetsBuilt, c.TwoCBHSetsFailed)
	fmt.Print(binSummary(c), bookSummary(c))
	fmt.Printf(L("CTG RAW         : %d successful | %d skipped | %d failed\n", "CTG RAW         : %d erfolgreich | %d übersprungen | %d fehlgeschlagen\n", "CTG RAW         : %d geslaagd | %d overgeslagen | %d mislukt\n", "CTG RAW         : %d réussi | %d ignoré | %d échec\n", "CTG RAW         : %d correcto | %d omitido | %d fallido\n", "CTG RAW         ：%d 成功 | %d 已跳过 | %d 失败\n", "CTG RAW         : %d успешно | %d пропущено | %d ошибок\n"), c.CTGRawBuilt, c.CTGRawSkipped, c.CTGRawFailed)
	fmt.Printf(L("GAME METAL      : %d successful | %d skipped | %d failed | %s model games\n", "GAME METAL      : %d erfolgreich | %d übersprungen | %d fehlgeschlagen | %s Modellpartien\n", "GAME METAL      : %d geslaagd | %d overgeslagen | %d mislukt | %s modelpartijen\n", "GAME METAL      : %d réussi | %d ignoré | %d échec | %s parties modèles\n", "GAME METAL      : %d correcto | %d omitido | %d fallido | %s partidas modelo\n", "GAME METAL      ：%d 成功 | %d 已跳过 | %d 失败 | %s 模型对局\n", "GAME METAL      : %d успешно | %d пропущено | %d ошибок | %s модельных партий\n"), c.GameMetalBuilt, c.GameMetalSkipped, c.GameMetalFailed, fmtInt(c.GameMetalSelected))
	fmt.Printf(L("CTG METAL       : %d successful | %d skipped | %d failed\n", "CTG METAL       : %d erfolgreich | %d übersprungen | %d fehlgeschlagen\n", "CTG METAL       : %d geslaagd | %d overgeslagen | %d mislukt\n", "CTG METAL       : %d réussi | %d ignoré | %d échec\n", "CTG METAL       : %d correcto | %d omitido | %d fallido\n", "CTG METAL       ：%d 成功 | %d 已跳过 | %d 失败\n", "CTG METAL       : %d успешно | %d пропущено | %d ошибок\n"), c.CTGMetalBuilt, c.CTGMetalSkipped, c.CTGMetalFailed)
	fmt.Println(L("Source code: separate download on the Source2Metal GitHub release page", "Quellcode: separater Download auf der Source2Metal-GitHub-Versionsseite", "Broncode: afzonderlijke download op de Source2Metal-releasepagina van GitHub", "Code source : téléchargement séparé sur la page de version Source2Metal de GitHub", "Código fuente: descarga independiente en la página de la versión Source2Metal de GitHub", "源代码：可在 GitHub 的 Source2Metal 发布页单独下载", "Исходный код: отдельная загрузка на странице выпуска Source2Metal в GitHub"))
	return nil
}

func classifyRunStatus(cfg Config, c Counters) string {
	if c.BINRawFailed > 0 {
		return "MISLUKT"
	}
	if c.CBHSetsFailed > 0 || c.TwoCBHSetsFailed > 0 || c.CTGRawFailed > 0 || c.CTGMetalFailed > 0 || c.GameMetalFailed > 0 {
		return "GESLAAGD_MET_FOUTEN"
	}
	if (cfg.Mode == "raw" || cfg.Mode == "all") && c.CTGRawSkipped > 0 {
		return "GESLAAGD_MET_OVERSLAGEN_BRONNEN"
	}
	if cfg.Mode == "all" && (c.CTGMetalSkipped > 0 || c.GameMetalSkipped > 0) {
		return "GESLAAGD_MET_OVERSLAGEN_BRONNEN"
	}
	return "GESLAAGD"
}

func interactiveChoices(c Config, inv Inventory, hw HardwareInfo) (Config, error) {
	fmt.Println("\n" + U("PARALLELLE VERWERKING"))
	def := defaultWorkers(hw.LogicalThreads)
	fmt.Printf(U("Logische CPU-threads : %d\n"), hw.LogicalThreads)
	fmt.Printf(L("Recommended workers  : %d (50%%)\n", "Empfohlene Worker   : %d (50%%)\n", "Aanbevolen workers   : %d (50%%)\n", "Workers recommandés : %d (50%%)\n", "Workers recomendados: %d (50%%)\n", "推荐 worker 数       ：%d（50%%）\n", "Рекомендуемые потоки: %d (50%%)\n"), def)
	c.Workers = askInt(fmt.Sprintf(U("Aantal workers [1-%d] (Enter = %d): "), hw.LogicalThreads, def), def, 1, hw.LogicalThreads)

	fmt.Println("\n" + L("MODE", "MODUS", "MODUS", "MODE", "MODO", "模式", "РЕЖИМ"))
	fmt.Println("  1 = " + U("alleen inventaris/rapport"))
	fmt.Println(rawModeText())
	fmt.Println(binScopeText())
	fmt.Println(L("  3 = RAW + METAL [default] - PGN/CBH/2CBH/CTG/BIN integration active", "  3 = RAW + METAL [Standard] - PGN/CBH/2CBH/CTG/BIN-Integration aktiv", "  3 = RAW + METAL [standaard] - PGN/CBH/2CBH/CTG/BIN-integratie actief", "  3 = RAW + METAL [défaut] - intégration PGN/CBH/2CBH/CTG/BIN active", "  3 = RAW + METAL [predeterminado] - integración PGN/CBH/2CBH/CTG/BIN activa", "  3 = RAW + METAL [默认] - PGN/CBH/2CBH/CTG/BIN 集成已启用", "  3 = RAW + METAL [по умолчанию] - интеграция PGN/CBH/2CBH/CTG/BIN активна"))
	fmt.Print(L("Choice [1-3] (Enter = 3): ", "Auswahl [1-3] (Enter = 3): ", "Keuze [1-3] (Enter = 3): ", "Choix [1-3] (Entrée = 3) : ", "Elección [1-3] (Intro = 3): ", "选择 [1-3]（回车 = 3）：", "Выбор [1-3] (Enter = 3): "))
	s, _ := stdin.ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		s = "3"
	}
	switch s {
	case "1":
		c.Mode = "inventory"
	case "2":
		c.Mode = "raw"
	case "3":
		c.Mode = "all"
	default:
		return c, errors.New(U("ongeldige modus"))
	}
	if c.Mode != "inventory" {
		c.MaxPly = askInt(fmt.Sprintf(U("Maximale RAW-diepte [1-%d] (Enter = %d): "), maxAllowedPly, defaultMaxPly), defaultMaxPly, 1, maxAllowedPly)
		c.MinPly = askInt(fmt.Sprintf(U("Minimumlengte partij [1-%d] (Enter = %d): "), maxAllowedPly, defaultMinPly), defaultMinPly, 1, maxAllowedPly)
		fmt.Println(L("Additional Elo filter: 0=keep source selection, 1=both >=2500, 2=both >=2400", "Zusätzlicher Elo-Filter: 0=Quellenauswahl beibehalten, 1=beide >=2500, 2=beide >=2400", "Extra Elo-filter: 0=bronselectie behouden, 1=beide >=2500, 2=beide >=2400", "Filtre Elo supplémentaire : 0=conserver la sélection source, 1=les deux >=2500, 2=les deux >=2400", "Filtro Elo adicional: 0=mantener selección de origen, 1=ambos >=2500, 2=ambos >=2400", "附加 Elo 过滤器：0=保留源选择，1=双方 >=2500，2=双方 >=2400", "Дополнительный фильтр Elo: 0=сохранить исходный отбор, 1=оба >=2500, 2=оба >=2400"))
		e := askInt(L("Choice [0-2] (Enter = 0): ", "Auswahl [0-2] (Enter = 0): ", "Keuze [0-2] (Enter = 0): ", "Choix [0-2] (Entrée = 0) : ", "Elección [0-2] (Intro = 0): ", "选择 [0-2]（回车 = 0）：", "Выбор [0-2] (Enter = 0): "), 0, 0, 2)
		if e == 1 {
			c.MinElo = 2500
		} else if e == 2 {
			c.MinElo = 2400
		} else {
			c.MinElo = 0
		}
		c.MaxEloGap = askInt(L("Maximum Elo difference (0=none, Enter=0): ", "Maximale Elo-Differenz (0=keine, Enter=0): ", "Maximaal Elo-verschil (0=geen, Enter=0): ", "Écart Elo maximal (0=aucun, Entrée=0) : ", "Diferencia Elo máxima (0=ninguna, Intro=0): ", "最大 Elo 差（0=无，回车=0）：", "Максимальная разница Elo (0=нет, Enter=0): "), 0, 0, 2000)
	}
	return c, nil
}

func startupMenu() string {
	fmt.Printf("Source2Metal v%s\n", version)
	fmt.Println(L("Main development : OpenAI ChatGPT (GPT-5.6 Sol)", "Hauptentwicklung : OpenAI ChatGPT (GPT-5.6 Sol)", "Hoofdontwikkeling: OpenAI ChatGPT (GPT-5.6 Sol)", "Développement principal : OpenAI ChatGPT (GPT-5.6 Sol)", "Desarrollo principal: OpenAI ChatGPT (GPT-5.6 Sol)", "主要开发：OpenAI ChatGPT (GPT-5.6 Sol)", "Основная разработка: OpenAI ChatGPT (GPT-5.6 Sol)"))
	fmt.Println(L("Practice & testing: Nocompany, Netherlands - computer chess player", "Praxis & Tests    : Nocompany, Niederlande - Computerschachspieler", "Praktijk & testen: Nocompany, Nederland - computerschaker", "Pratique & tests : Nocompany, Pays-Bas - joueur d'échecs informatiques", "Práctica y pruebas: Nocompany, Países Bajos - jugador de ajedrez informático", "实践与测试：Nocompany，荷兰 - 计算机国际象棋棋手", "Практика и тесты: Nocompany, Нидерланды - компьютерный шахматист"))
	fmt.Println("\n" + L("MAIN MENU", "HAUPTMENÜ", "HOOFDMENU", "MENU PRINCIPAL", "MENÚ PRINCIPAL", "主菜单", "ГЛАВНОЕ МЕНЮ"))
	fmt.Println(L("  1 = Process sources [default]", "  1 = Quellen verarbeiten [Standard]", "  1 = Bronnen verwerken [standaard]", "  1 = Traiter les sources [défaut]", "  1 = Procesar fuentes [predeterminado]", "  1 = 处理源文件 [默认]", "  1 = Обработать источники [по умолчанию]"))
	fmt.Println(L("  2 = SyzygyCheck / pagefile", "  2 = SyzygyCheck / Pagefile", "  2 = SyzygyCheck / pagefile", "  2 = SyzygyCheck / fichier d’échange", "  2 = SyzygyCheck / archivo de paginación", "  2 = SyzygyCheck / 页面文件", "  2 = SyzygyCheck / файл подкачки"))
	fmt.Println(L("  3 = Change language (Изменить язык / 更改语言)", "  3 = Sprache ändern", "  3 = Taal wijzigen", "  3 = Changer de langue", "  3 = Cambiar idioma", "  3 = 更改语言 (Change language / Изменить язык)", "  3 = Изменить язык (Change language / 更改语言)"))
	fmt.Println(L("  4 = Information", "  4 = Informationen", "  4 = Informatie", "  4 = Informations", "  4 = Información", "  4 = 信息", "  4 = Информация"))
	fmt.Println(L("  0 = Exit", "  0 = Beenden", "  0 = Afsluiten", "  0 = Quitter", "  0 = Salir", "  0 = 退出", "  0 = Выход"))
	fmt.Print(L("Choice [0-4] (Enter = 1): ", "Auswahl [0-4] (Enter = 1): ", "Keuze [0-4] (Enter = 1): ", "Choix [0-4] (Entrée = 1) : ", "Elección [0-4] (Intro = 1): ", "选择 [0-4]（回车 = 1）：", "Выбор [0-4] (Enter = 1): "))
	x, _ := stdin.ReadString('\n')
	x = strings.TrimSpace(x)
	if x == "" || x == "1" {
		return "run"
	}
	switch x {
	case "2":
		return syzygyAndPagefileMenu()
	case "3":
		return "language"
	case "4":
		return informationMenu()
	case "0":
		return "quit"
	default:
		fmt.Println(L("Invalid choice; source processing will start.", "Ungültige Auswahl; die Quellverarbeitung wird gestartet.", "Ongeldige keuze; bronnen verwerken wordt gestart.", "Choix invalide ; le traitement des sources va démarrer.", "Elección no válida; se iniciará el procesamiento de fuentes.", "选择无效；将开始处理源文件。", "Неверный выбор; будет запущена обработка источников."))
		return "run"
	}
}

func syzygyAndPagefileMenu() string {
	fmt.Println("\n" + L("SYZYGYCHECK / PAGEFILE", "SYZYGYCHECK / PAGEFILE", "SYZYGYCHECK / PAGEFILE", "SYZYGYCHECK / FICHIER D’ÉCHANGE", "SYZYGYCHECK / ARCHIVO DE PAGINACIÓN", "SYZYGYCHECK / 页面文件", "SYZYGYCHECK / ФАЙЛ ПОДКАЧКИ"))
	fmt.Println(L("  1 = Extract separate SyzygyCheck package (ZIP)", "  1 = Separates SyzygyCheck-Paket (ZIP) entpacken", "  1 = Los SyzygyCheck-pakket uitpakken (ZIP)", "  1 = Extraire le paquet SyzygyCheck séparé (ZIP)", "  1 = Extraer paquete SyzygyCheck independiente (ZIP)", "  1 = 解出独立 SyzygyCheck 包（ZIP）", "  1 = Извлечь отдельный пакет SyzygyCheck (ZIP)"))
	fmt.Println(L("  2 = Pagefile information", "  2 = Pagefile-Information", "  2 = Pagefile-informatie", "  2 = Informations sur le fichier d'échange", "  2 = Información del archivo de paginación", "  2 = 页面文件信息", "  2 = Информация о файле подкачки"))
	fmt.Println(L("  0 = Back", "  0 = Zurück", "  0 = Terug", "  0 = Retour", "  0 = Volver", "  0 = 返回", "  0 = Назад"))
	fmt.Print(L("Choice [0-2]: ", "Auswahl [0-2]: ", "Keuze [0-2]: ", "Choix [0-2] : ", "Elección [0-2]: ", "选择 [0-2]：", "Выбор [0-2]: "))
	x, _ := stdin.ReadString('\n')
	x = strings.TrimSpace(x)
	switch x {
	case "1":
		return "extract-syzygy-package"
	case "2":
		return "pagefile"
	default:
		return "language-return"
	}
}

func informationMenu() string {
	fmt.Println("\n" + L("INFORMATION", "INFORMATIONEN", "INFORMATIE", "INFORMATIONS", "INFORMACIÓN", "信息", "ИНФОРМАЦИЯ"))
	fmt.Println(L("  1 = Build information", "  1 = Buildinformationen", "  1 = Buildinformatie", "  1 = Informations de compilation", "  1 = Información de compilación", "  1 = 构建信息", "  1 = Информация о сборке"))
	fmt.Println(L("  0 = Back", "  0 = Zurück", "  0 = Terug", "  0 = Retour", "  0 = Volver", "  0 = 返回", "  0 = Назад"))
	fmt.Print(L("Choice [0-1]: ", "Auswahl [0-1]: ", "Keuze [0-1]: ", "Choix [0-1] : ", "Elección [0-1]: ", "选择 [0-1]：", "Выбор [0-1]: "))
	x, _ := stdin.ReadString('\n')
	x = strings.TrimSpace(x)
	if x == "1" {
		return "buildinfo"
	}
	return "language-return"
}

func askInt(prompt string, def, min, max int) int {
	for {
		fmt.Print(prompt)
		s, _ := stdin.ReadString('\n')
		s = strings.TrimSpace(s)
		if s == "" {
			return def
		}
		n, e := strconv.Atoi(s)
		if e == nil && n >= min && n <= max {
			return n
		}
		fmt.Println(U("Ongeldige invoer."))
	}
}

func fatal(c Config, err error) {
	fmt.Fprintln(os.Stderr, "\n"+L("ERROR:", "FEHLER:", "FOUT:", "ERREUR :", "ERROR:", "错误：", "ОШИБКА:"), err)
	if !c.NoPause && c.Interactive {
		pause()
	}
	os.Exit(2)
}

func pause() {
	fmt.Print("\n" + L("Press Enter to continue...", "Drücken Sie Enter, um fortzufahren...", "Druk op Enter om verder te gaan...", "Appuyez sur Entrée pour continuer...", "Pulse Intro para continuar...", "按回车继续……", "Нажмите Enter, чтобы продолжить..."))
	_, _ = stdin.ReadString('\n')
}
