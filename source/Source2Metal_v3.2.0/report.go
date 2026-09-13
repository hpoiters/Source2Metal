package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func writeReports(layout OutputLayout, inv Inventory, cfg Config, hw HardwareInfo, c Counters, status string) error {
	if err := os.MkdirAll(layout.ReportDir, 0755); err != nil {
		return err
	}
	if err := writeInventoryReport(filepath.Join(layout.ReportDir, "Source2Metal_Report.txt"), inv, cfg, hw, c, status); err != nil {
		return err
	}
	if err := writeSettings(filepath.Join(layout.ReportDir, "Source2Metal_Settings.txt"), inv, cfg, hw); err != nil {
		return err
	}
	if err := writeSources(filepath.Join(layout.ReportDir, "Source2Metal_Sources.txt"), inv); err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(layout.ReportDir, "Source2Metal_BuildInfo.txt"), []byte(buildInfoText()), 0644); err != nil {
		return err
	}
	sourceInfo := fmt.Sprintf(L(`Source2Metal v%s - SOURCE CODE

The source code is deliberately not duplicated inside the Windows EXE or user package.
Download it separately from the Source2Metal release page on GitHub.
`, `Source2Metal v%s - QUELLCODE

Der Quellcode wird bewusst nicht in der Windows-EXE oder im Benutzerpaket dupliziert.
Laden Sie ihn separat von der Source2Metal-Versionsseite auf GitHub herunter.
`, `Source2Metal v%s - BRONCODE

De broncode wordt bewust niet dubbel opgenomen in de Windows-EXE of het gebruikerspakket.
Download hem afzonderlijk van de Source2Metal-releasepagina op GitHub.
`, `Source2Metal v%s - CODE SOURCE

Le code source n'est volontairement pas dupliqué dans l'EXE Windows ou le paquet utilisateur.
Téléchargez-le séparément depuis la page de version Source2Metal sur GitHub.
`, `Source2Metal v%s - CÓDIGO FUENTE

El código fuente no se duplica deliberadamente dentro del EXE de Windows ni del paquete de usuario.
Descárguelo por separado desde la página de la versión Source2Metal en GitHub.
`, `Source2Metal v%s - 源代码

源代码不会在 Windows EXE 或用户包内重复保存。
请从 GitHub 的 Source2Metal 发布页单独下载。
`, `Source2Metal v%s - ИСХОДНЫЙ КОД

Исходный код намеренно не дублируется внутри Windows EXE или пользовательского пакета.
Загрузите его отдельно со страницы выпуска Source2Metal на GitHub.
`), version)
	if err := atomicWriteFile(filepath.Join(layout.ReportDir, "Source2Metal_SourceInfo.txt"), []byte(sourceInfo), 0644); err != nil {
		return err
	}
	statusText := fmt.Sprintf("Source2Metal v%s\nStatus: %s\nRun-ID: %s\nModus: %s\nWorkers: %d/%d\nGAME RAW: %d records / %d geverifieerd\nCTG RAW: %d gebouwd / %d overgeslagen / %d mislukt / %d records\nGAME METAL: %d gebouwd / %d overgeslagen / %d mislukt / %d modelpartijen\nCTG METAL: %d gebouwd / %d overgeslagen / %d mislukt\nDuur: %s\n", version, localizeStatus(status), runID(c), cfg.Mode, cfg.Workers, hw.LogicalThreads, c.GamesAccepted, c.RawGamesVerified, c.CTGRawBuilt, c.CTGRawSkipped, c.CTGRawFailed, c.CTGRawGames, c.GameMetalBuilt, c.GameMetalSkipped, c.GameMetalFailed, c.GameMetalSelected, c.CTGMetalBuilt, c.CTGMetalSkipped, c.CTGMetalFailed, c.Elapsed)
	statusText += bookSummary(c) + binSummary(c) + binScopeText() + "\n"
	if err := atomicWriteFile(filepath.Join(layout.ReportDir, "STATUS.txt"), []byte(localizeReportText(statusText)), 0644); err != nil {
		return err
	}
	if err := writeOutputExplanations(layout, cfg, c); err != nil {
		return err
	}
	if err := writeFullProcessReport(filepath.Join(layout.ReportDir, "Source2Metal - Full Process Report.txt"), layout, inv, cfg, c, status); err != nil {
		return err
	}
	if _, err := os.Stat(layout.MetalDir); err == nil {
		if err := writeMetalStatus(layout.MetalDir); err != nil {
			return err
		}
	}
	return writeChecksums(filepath.Join(layout.ReportDir, "Source2Metal_SHA256.txt"), layout.RunDir)
}

func runID(c Counters) string {
	if c.Started.IsZero() {
		return "onbekend"
	}
	return c.Started.Format("20060102-150405")
}

func writeInventoryReport(path string, inv Inventory, cfg Config, hw HardwareInfo, c Counters, status string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Source2Metal v%s - RAPPORT\n\n", version)
	fmt.Fprintf(&b, "Status            : %s\n", localizeStatus(status))
	fmt.Fprintf(&b, "Run-ID            : %s\n", runID(c))
	if !c.Started.IsZero() {
		fmt.Fprintf(&b, "Start             : %s\n", c.Started.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&b, "Einde             : %s\n", c.Started.Add(c.Elapsed).Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(&b, "Duur              : %s\n", c.Elapsed)
	fmt.Fprintf(&b, "Root              : %s\n", inv.Root)
	fmt.Fprintf(&b, "Modus             : %s\n", cfg.Mode)
	fmt.Fprintf(&b, "Workers           : %d / %d logische threads\n", cfg.Workers, hw.LogicalThreads)
	if hw.CPUName != "" {
		fmt.Fprintf(&b, "CPU               : %s\n", hw.CPUName)
	}
	if hw.RAMKnown {
		fmt.Fprintf(&b, "RAM               : %.1f GB\n", gib(hw.RAMBytes))
	}
	if hw.PagefileKnown {
		fmt.Fprintf(&b, "Pagefile          : %.1f GB", gib(hw.PagefileBytes))
		if hw.PagefilePaths != "" {
			fmt.Fprintf(&b, " | %s", hw.PagefilePaths)
		}
		fmt.Fprintln(&b)
	}
	if hw.FreeKnown {
		fmt.Fprintf(&b, "Vrij bij start    : %.1f GB\n", gib(hw.FreeBytes))
	}
	fmt.Fprintf(&b, "Max diepte        : %d ply\n", cfg.MaxPly)
	fmt.Fprintf(&b, "Minimum partij    : %d ply\n", cfg.MinPly)
	if cfg.MinElo == 0 {
		fmt.Fprintln(&b, "Elo-filter        : bronselectie behouden")
	} else {
		fmt.Fprintf(&b, "Elo-filter        : beide spelers >= %d\n", cfg.MinElo)
	}
	if cfg.MaxEloGap == 0 {
		fmt.Fprintln(&b, "Max Elo-verschil : geen")
	} else {
		fmt.Fprintf(&b, "Max Elo-verschil : %d\n", cfg.MaxEloGap)
	}

	fmt.Fprintln(&b, "\nBRONNEN")
	for _, s := range inv.Sources {
		state := ""
		if !s.Complete {
			state = " | ONVOLLEDIG"
		}
		fmt.Fprintf(&b, "%-4s | %s%s\n", s.Kind, s.Path, state)
	}

	fmt.Fprintln(&b, "\nGAME RAW")
	fmt.Fprintf(&b, "Partijen gezien            : %s\n", fmtInt(c.GamesSeen))
	fmt.Fprintf(&b, "Geaccepteerd               : %s\n", fmtInt(c.GamesAccepted))
	fmt.Fprintf(&b, "Geverifieerd               : %s\n", fmtInt(c.RawGamesVerified))
	fmt.Fprintf(&b, "Exact duplicaat            : %s\n", fmtInt(c.GamesDuplicate))
	fmt.Fprintf(&b, "Geen geldige uitslag       : %s\n", fmtInt(c.GamesNoResult))
	fmt.Fprintf(&b, "FEN/SetUp                  : %s\n", fmtInt(c.GamesSetup))
	fmt.Fprintf(&b, "Te kort                    : %s\n", fmtInt(c.GamesShort))
	fmt.Fprintf(&b, "Elo-filter                 : %s\n", fmtInt(c.GamesElo))
	fmt.Fprintf(&b, "Elo-verschil               : %s\n", fmtInt(c.GamesEloGap))
	fmt.Fprintf(&b, "Variatietakken verwijderd  : %s\n", fmtInt(c.VariationsStripped))
	fmt.Fprintf(&b, "Commentaren verwijderd     : %s\n", fmtInt(c.CommentsStripped))
	fmt.Fprintf(&b, "Ply geschreven             : %s\n", fmtInt(c.PlyWritten))
	if c.OutputBytes > 0 {
		fmt.Fprintf(&b, "Bytes                       : %s\n", fmtInt(c.OutputBytes))
		fmt.Fprintf(&b, "SHA-256                     : %s\n", c.OutputSHA256)
	}

	fmt.Fprintln(&b, "\nCHESSBASE-ADAPTERS")
	fmt.Fprintf(&b, "CBH  : %s geslaagd / %s mislukt | %s adapterpartijen\n", fmtInt(c.CBHSetsBuilt), fmtInt(c.CBHSetsFailed), fmtInt(c.CBHConverted))
	fmt.Fprintf(&b, "2CBH : %s geslaagd / %s mislukt | %s adapterpartijen\n", fmtInt(c.TwoCBHSetsBuilt), fmtInt(c.TwoCBHSetsFailed), fmtInt(c.TwoCBHConverted))

	fmt.Fprintln(&b, "\nGAME METAL")
	fmt.Fprintln(&b, "Methode: transpositie-bewuste positie/zet-statistiek; uitvoer bestaat uit volledige legale bronpartijen met hun oorspronkelijke uitslag.")
	fmt.Fprintf(&b, "Unieke legale partijen      : %s\n", fmtInt(c.GameMetalGamesAccepted))
	fmt.Fprintf(&b, "Ongeldige partijen          : %s\n", fmtInt(c.GameMetalInvalid))
	fmt.Fprintf(&b, "Duplicaten                  : %s\n", fmtInt(c.GameMetalDuplicates))
	fmt.Fprintf(&b, "Posities                    : %s\n", fmtInt(c.GameMetalPositions))
	fmt.Fprintf(&b, "Zetkandidaten               : %s\n", fmtInt(c.GameMetalCandidates))
	fmt.Fprintf(&b, "Gekwalificeerd              : %s\n", fmtInt(c.GameMetalQualified))
	fmt.Fprintf(&b, "Voorkeurposities            : %s\n", fmtInt(c.GameMetalAnchorPositions))
	fmt.Fprintf(&b, "Voorkeurszetten             : %s\n", fmtInt(c.GameMetalAnchorSignals))
	fmt.Fprintf(&b, "Modelpartijen               : %s\n", fmtInt(c.GameMetalSelected))
	fmt.Fprintf(&b, "Geverifieerd                : %s\n", fmtInt(c.GameMetalVerified))
	fmt.Fprintf(&b, "Gebouwd/overgeslagen/fout   : %s / %s / %s\n", fmtInt(c.GameMetalBuilt), fmtInt(c.GameMetalSkipped), fmtInt(c.GameMetalFailed))
	if c.GameMetalBytes > 0 {
		fmt.Fprintf(&b, "Bytes                       : %s\n", fmtInt(c.GameMetalBytes))
		fmt.Fprintf(&b, "SHA-256                     : %s\n", c.GameMetalSHA256)
	}

	fmt.Fprintln(&b, binScopeText())
	fmt.Fprint(&b, binSummary(c), bookSummary(c))
	fmt.Fprintln(&b, bookPolicyText())
	fmt.Fprintln(&b, "\nCTG RAW")
	fmt.Fprintf(&b, "Gebouwd/overgeslagen/fout   : %s / %s / %s\n", fmtInt(c.CTGRawBuilt), fmtInt(c.CTGRawSkipped), fmtInt(c.CTGRawFailed))
	fmt.Fprintf(&b, "Partijen                    : %s\n", fmtInt(c.CTGRawGames))
	fmt.Fprintf(&b, "Geverifieerd                : %s\n", fmtInt(c.RawBooksVerified))
	fmt.Fprintf(&b, "Ply                         : %s\n", fmtInt(c.CTGRawPlies))
	fmt.Fprintf(&b, "Decodefouten                : %s\n", fmtInt(c.CTGDecodeErrors))
	fmt.Fprintf(&b, "Bytes                       : %s\n", fmtInt(c.CTGRawBytes))

	fmt.Fprintln(&b, "\nCTG METAL")
	fmt.Fprintf(&b, "Gebouwd/overgeslagen/fout   : %s / %s / %s\n", fmtInt(c.CTGMetalBuilt), fmtInt(c.CTGMetalSkipped), fmtInt(c.CTGMetalFailed))
	fmt.Fprintf(&b, "Bytes                       : %s\n", fmtInt(c.CTGMetalBytes))

	if c.CombinedBytes > 0 {
		fmt.Fprintln(&b, "\nALLE SOURCES RAW")
		fmt.Fprintf(&b, "Records                     : %s\n", fmtInt(c.CombinedGames))
		fmt.Fprintf(&b, "Geverifieerd                : %s\n", fmtInt(c.CombinedVerified))
		fmt.Fprintf(&b, "Bytes                       : %s\n", fmtInt(c.CombinedBytes))
		fmt.Fprintf(&b, "SHA-256                     : %s\n", c.CombinedSHA256)
	}

	fmt.Fprintln(&b, "\nTECHNISCHE CONTROLES")
	fmt.Fprintln(&b, "- Bronnen worden alleen gelezen; originele bronbestanden worden niet gewijzigd.")
	fmt.Fprintln(&b, "- Bronzoektocht blijft binnen de gekozen root en onderliggende mappen; uitvoer- en tempmappen worden overgeslagen.")
	fmt.Fprintln(&b, "- GAME-METAL gebruikt volledige legale schaakbordcontrole en schrijft complete modelpartijen met originele uitslag.")
	fmt.Fprintln(&b, "- CTG RAW laat de uitvoergrootte door de werkelijk decodeerbare broninhoud bepalen.")
	fmt.Fprintln(&b, L("- CTG METAL never lowers selection thresholds silently; any cautious fallback is reported visibly.", "- CTG METAL senkt Auswahlschwellen niemals still ab; ein vorsichtiger Fallback wird sichtbar gemeldet.", "- CTG METAL verlaagt selectiedrempels niet stilzwijgend; een eventuele voorzichtige fallback wordt zichtbaar gemeld.", "- CTG METAL n’abaisse jamais silencieusement les seuils de sélection ; tout repli prudent est signalé clairement.", "- CTG METAL nunca reduce silenciosamente los umbrales de selección; cualquier alternativa prudente se informa claramente.", "- CTG METAL 不会静默降低选择阈值；任何谨慎的后备模式都会明确报告。", "- CTG METAL никогда не снижает пороги отбора скрытно; любой осторожный резервный режим явно отображается."))
	fmt.Fprintln(&b, "- GAME-METAL en CTG-METAL blijven herkenbaar gescheiden omdat zij verschillende soorten evidence coderen.")
	fmt.Fprintln(&b, "- Fritz-inleerinstelling: Overwinningen + Verliespartijen AAN; Wit/Zwart/Speler UIT; spelernaam leeg; alle partijen.")
	return atomicWriteFile(path, []byte(localizeReportText(b.String())), 0644)
}

func writeSettings(path string, inv Inventory, cfg Config, hw HardwareInfo) error {
	text := fmt.Sprintf(`Source2Metal v%s - INSTELLINGEN

Root              : %s
Modus             : %s
Max diepte        : %d ply
Minimum partij    : %d ply
Minimum Elo       : %d
Max Elo-verschil  : %d
Workers           : %d
Logische threads  : %d
RAM bytes         : %d
Pagefile bytes    : %d
`, version, inv.Root, cfg.Mode, cfg.MaxPly, cfg.MinPly, cfg.MinElo, cfg.MaxEloGap, cfg.Workers, hw.LogicalThreads, hw.RAMBytes, hw.PagefileBytes)
	return atomicWriteFile(path, []byte(localizeReportText(text)), 0644)
}

func writeSources(path string, inv Inventory) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Source2Metal v%s - BRONINVENTARIS\nRoot: %s\n\n", version, inv.Root)
	for i, s := range inv.Sources {
		fmt.Fprintf(&b, "%03d | %-4s | %d bytes | compleet=%v | %s\n", i+1, s.Kind, s.Bytes, s.Complete, s.Path)
		for _, a := range s.Aux {
			fmt.Fprintf(&b, "      bijbestand: %s\n", a)
		}
	}
	return atomicWriteFile(path, []byte(localizeReportText(b.String())), 0644)
}

func writeChecksums(path, runDir string) error {
	var files []string
	err := filepath.Walk(runDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Clean(p) == filepath.Clean(path) {
			return nil
		}
		rel, _ := filepath.Rel(runDir, p)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 0 && strings.EqualFold(parts[0], "TEMP") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(p), ".source2metal-tmp-") {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)
	var b strings.Builder
	fmt.Fprintf(&b, "Source2Metal v%s - SHA-256 VAN DEFINITIEVE RUNBESTANDEN\n\n", version)
	for _, p := range files {
		h, err := fileSHA256(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(runDir, p)
		fmt.Fprintf(&b, "%s  %s\n", h, rel)
	}
	return atomicWriteFile(path, []byte(localizeReportText(b.String())), 0644)
}

func writeMetalStatus(metalDir string) error {
	text := fmt.Sprintf(`Source2Metal v%s - METAL ROUTES

GAME-METAL
PGN, CBH en 2CBH voeden gezamenlijk een transpositie-bewust GAME-model.
De uitvoer bestaat uit volledige, legaal gecontroleerde echte bronpartijen
met hun oorspronkelijke uitslag.

CTG-METAL
Iedere CTG blijft een afzonderlijke boekgerichte bewijsbron. Eerst wordt de
strikte selectie geprobeerd. Alleen wanneer de diagnostiek voldoende potentieel
ziet, kan zichtbaar een voorzichtige Kleine/Brede-fallback worden geprobeerd.

WAAROM GESCHEIDEN?
GAME-METAL en CTG-METAL vertegenwoordigen verschillende soorten leer-evidence.
Ze blijven daarom traceerbaar gescheiden en worden niet blind samengevoegd.

FRITZ - BIJLEREN UIT DATABASE
Overwinningen + Verliespartijen AAN; Wit/Zwart/Speler UIT; spelernaam leeg;
alle partijen.
`, version)
	return atomicWriteFile(filepath.Join(metalDir, "INFO - METAL ROUTES.txt"), []byte(localizeReportText(text)), 0644)
}
