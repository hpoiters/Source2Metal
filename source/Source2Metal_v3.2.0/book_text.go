package main

import "fmt"

func bookPolicyText() string {
	return L(
		"BOOK RAW combines CTG and BIN. Complete legal move sequences are deduplicated within the merged BOOK file; prefixes and transpositions remain distinct. If results or headers differ, the first record is retained (CTG before BIN). Separate source files remain available. CTG results come from book statistics; BIN uses *. Neither is a played-game result. ALL SOURCES RAW combines GAME RAW and merged BOOK RAW when both exist, without cross-class deduplication. BIN does not produce METAL.",
		"BOOK RAW vereint CTG und BIN. Vollständige legale Zugfolgen werden in der zusammengeführten BOOK-Datei dedupliziert; Präfixe und Transpositionen bleiben getrennt. Bei unterschiedlichen Ergebnissen oder Kopfzeilen bleibt der erste Datensatz erhalten (CTG vor BIN). Einzeldateien bleiben verfügbar. CTG-Ergebnisse stammen aus Buchstatistiken; BIN verwendet *. Beides sind keine Ergebnisse gespielter Partien. ALL SOURCES RAW vereint GAME RAW und zusammengeführtes BOOK RAW, wenn beide vorhanden sind, ohne klassenübergreifende Deduplizierung. BIN erzeugt kein METAL.",
		"BOOK RAW verenigt CTG en BIN. In het samengevoegde BOOK-bestand worden exact gelijke volledige legale zettenreeksen ontdubbeld; prefixen en transposities blijven apart. Bij verschillende uitslagen of headers blijft het eerste record behouden (CTG vóór BIN). De afzonderlijke bronbestanden blijven beschikbaar. CTG-uitslagen komen uit boekstatistiek; BIN gebruikt *. Beide zijn geen uitslag van een gespeelde partij. ALL SOURCES RAW voegt GAME RAW en samengevoegde BOOK RAW samen wanneer beide bestaan, zonder ontdubbeling tussen die klassen. BIN levert geen METAL.",
		"BOOK RAW réunit CTG et BIN. Les séquences légales complètes identiques sont dédupliquées dans le fichier BOOK fusionné ; préfixes et transpositions restent distincts. Si les résultats ou en-têtes diffèrent, le premier enregistrement est conservé (CTG avant BIN). Les fichiers séparés restent disponibles. Les résultats CTG proviennent des statistiques du livre ; BIN utilise *. Ce ne sont pas des résultats de parties jouées. ALL SOURCES RAW réunit GAME RAW et BOOK RAW fusionné quand les deux existent, sans déduplication entre classes. BIN ne produit pas de METAL.",
		"BOOK RAW reúne CTG y BIN. Las secuencias legales completas idénticas se deduplican en el archivo BOOK combinado; los prefijos y transposiciones siguen separados. Si los resultados o encabezados difieren, se conserva el primer registro (CTG antes de BIN). Los archivos separados siguen disponibles. Los resultados CTG proceden de estadísticas del libro; BIN usa *. No son resultados de partidas jugadas. ALL SOURCES RAW reúne GAME RAW y BOOK RAW combinado cuando ambos existen, sin deduplicación entre clases. BIN no produce METAL.",
		"BOOK RAW 合并 CTG 和 BIN。在合并的 BOOK 文件中，仅对完全相同的合法完整走子序列去重；前缀和不同次序的转置变例保持独立。若结果或标签不同，保留首条记录（CTG 优先于 BIN）。各源文件仍保留。CTG 结果来自开局库统计；BIN 使用 *，均非实际对局结果。若 GAME RAW 和合并的 BOOK RAW 均存在，则生成 ALL SOURCES RAW，不进行跨类别去重。BIN 不生成 METAL。",
		"BOOK RAW объединяет CTG и BIN. В объединённом BOOK-файле удаляются дубликаты полных легальных последовательностей ходов; префиксы и транспозиции остаются раздельными. При разных результатах или заголовках сохраняется первая запись (CTG перед BIN). Отдельные исходные файлы сохраняются. Результаты CTG получены из статистики книги; BIN использует *. Это не результаты сыгранных партий. ALL SOURCES RAW объединяет GAME RAW и объединённый BOOK RAW при наличии обоих, без межклассовой дедупликации. BIN не создаёт METAL.")
}

func bookSummary(c Counters) string {
	return fmt.Sprintf(L("BOOK RAW: %d verified lines | %d duplicate lines removed\n", "BOOK RAW: %d verifizierte Linien | %d doppelte Linien entfernt\n", "BOOK RAW: %d geverifieerde lijnen | %d dubbele lijnen verwijderd\n", "BOOK RAW : %d lignes vérifiées | %d doublons supprimés\n", "BOOK RAW: %d líneas verificadas | %d duplicados eliminados\n", "BOOK RAW：%d 条已验证变例 | %d 条重复变例已删除\n", "BOOK RAW: %d проверенных линий | %d дубликатов удалено\n"), c.BookMergedGames, c.BookMergedDuplicates)
}

func rawLayoutText() string {
	return "RAW/1 - Separate Sources - RAW PGNs/GAME Sources/  (PGN, CBH, 2CBH)\n" +
		"RAW/1 - Separate Sources - RAW PGNs/BOOK Sources/  (CTG, BIN)\n" +
		"RAW/2 - Merged Sources - RAW PGNs/Source2Metal - Merged GAME Sources - RAW.pgn\n" +
		"RAW/2 - Merged Sources - RAW PGNs/Source2Metal - Merged BOOK Sources - RAW.pgn\n" +
		"RAW/2 - Merged Sources - RAW PGNs/Source2Metal - Merged ALL Sources - RAW.pgn\n"
}

func rawModeText() string {
	return L("  2 = RAW: PGN/CBH/2CBH -> GAME; CTG/BIN -> BOOK; separate and merged outputs",
		"  2 = RAW: PGN/CBH/2CBH -> GAME; CTG/BIN -> BOOK; einzelne und zusammengeführte Ausgaben",
		"  2 = RAW: PGN/CBH/2CBH -> GAME; CTG/BIN -> BOOK; afzonderlijk en samengevoegd",
		"  2 = RAW : PGN/CBH/2CBH -> GAME ; CTG/BIN -> BOOK ; sorties séparées et fusionnées",
		"  2 = RAW: PGN/CBH/2CBH -> GAME; CTG/BIN -> BOOK; salidas separadas y combinadas",
		"  2 = RAW：PGN/CBH/2CBH -> GAME；CTG/BIN -> BOOK；各源及合并输出",
		"  2 = RAW: PGN/CBH/2CBH -> GAME; CTG/BIN -> BOOK; отдельные и объединённые результаты")
}
