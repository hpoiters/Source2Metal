package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func relOutput(layout OutputLayout, p string) string {
	if p == "" {
		return "-"
	}
	r, err := filepath.Rel(layout.RunDir, p)
	if err != nil {
		return p
	}
	return r
}

func writeOutputExplanations(layout OutputLayout, cfg Config, c Counters) error {
	if _, err := os.Stat(layout.RawDir); err == nil {
		rawText := fmt.Sprintf("Source2Metal v%s - RAW\n\n", version) + rawLayoutText() + "\n" + bookPolicyText() + "\n\n" + binScopeText() + "\n" + binSummary(c) + bookSummary(c)
		if err := atomicWriteFile(filepath.Join(layout.RawDir, "INFO - RAW RESULTS.txt"), []byte(rawText), 0644); err != nil {
			return err
		}
	}

	if _, err := os.Stat(layout.MetalDir); err == nil {
		metalTemplate := L(
			`Source2Metal v%s - METAL RESULTS EXPLAINED

PER-SOURCE RESULTS
For each source, the METAL result actually produced remains visible. A source
without a responsible result receives no empty PGN; instead STATUS.txt explains
the concrete reason and, where available, points to the technical detail report.

GAME METAL
- board positions and transpositions are counted together;
- win/draw/loss is evaluated from the side to move;
- Balanced profile: at least 32 observations and 4 decisive results, with at
  most 2 nearly equivalent preferred moves per position;
- a model game must hit at least 3 preference positions and at least 80%% coverage;
- output is the complete legal source game with its original result; no AnchorPly
  pseudo-games and no fictitious result.

CTG METAL
All decoded reachable book moves are preserved through neutral RAW lines, with result *.
No frequency filter, recommendations, learned weights or artificial results are imported.
METAL/Metal.pgn automatically combines CTG lines and selected GAME records without personal headers.
Use CTG lines for book import; they are not win/loss learning signals.
See the CTG JSON report for coverage, depth cutoffs and decoder limitations.
`,
			`Source2Metal v%s - ERKLÄRUNG DER METAL-ERGEBNISSE

ERGEBNISSE PRO QUELLE
Für jede Quelle bleibt sichtbar, welches METAL-Ergebnis tatsächlich erzeugt
wurde. Eine Quelle ohne verantwortbares Ergebnis erhält keine leere PGN, sondern
eine STATUS.txt mit konkreter Begründung und soweit vorhanden technischem Bericht.

GAME METAL
- Brettpositionen und Transpositionen werden gemeinsam gezählt;
- Sieg/Remis/Niederlage wird aus Sicht der am Zug befindlichen Farbe bewertet;
- Profil Ausgewogen: mindestens 32 Beobachtungen und 4 entschiedene Ergebnisse,
  maximal 2 nahezu gleichwertige Vorzugszüge pro Position;
- eine Modellpartie muss mindestens 3 Präferenzpositionen und 80%% Abdeckung treffen;
- ausgegeben wird die vollständige legale Quellpartie mit Originalergebnis; keine
  AnchorPly-Pseudopartien und kein fiktives Ergebnis.

CTG METAL
Alle dekodierten erreichbaren Buchzüge bleiben als neutrale RAW-Linien mit Ergebnis * erhalten.
Keine Häufigkeitsfilter, Empfehlungen, Lerngewichte oder künstlichen Ergebnisse.
METAL/Metal.pgn vereint CTG-Linien und ausgewählte GAME-Partien ohne persönliche Kopfzeilen.
CTG-Linien dienen dem Buchimport, nicht als Sieg/Niederlage-Lernsignale.
Der CTG-JSON-Bericht nennt Abdeckung, Tiefenabbrüche und Decodergrenzen.
`,
			`Source2Metal v%s - UITLEG METAL RESULTATEN

PER-BRON RESULTATEN
Per bron is zichtbaar welk METAL-resultaat werkelijk is geproduceerd. Een bron
zonder verantwoord resultaat krijgt geen lege PGN maar een STATUS.txt met de
concrete reden en, waar beschikbaar, het technische detailrapport.

GAME METAL
- bordposities en transposities worden gezamenlijk geteld;
- winst/remise/verlies wordt vanuit de kleur aan zet beoordeeld;
- profiel Gebalanceerd: minimaal 32 waarnemingen en 4 beslissende resultaten,
  maximaal 2 bijna-gelijkwaardige voorkeurszetten per positie;
- een modelpartij moet minimaal 3 voorkeursposities treffen en minimaal 80%%
  dekking halen;
- de uitvoer bestaat uit de volledige legale bronpartij met oorspronkelijke
  uitslag; geen AnchorPly-pseudopartijen en geen fictieve uitslag.

CTG METAL
Alle gedecodeerde bereikbare boekzetten blijven behouden via neutrale RAW-lijnen met uitslag *.
Geen frequentiefilter, aanbevelingen, leergewichten of kunstmatige uitslagen.
METAL/Metal.pgn voegt CTG-lijnen en geselecteerde GAME-partijen automatisch samen, zonder persoonlijke headers.
Gebruik CTG-lijnen voor boekimport; het zijn geen winst/verlies-leersignalen.
Het CTG-JSON-rapport vermeldt dekking, diepteafbrekingen en decoderbeperkingen.
`,
			`Source2Metal v%s - EXPLICATION DES RÉSULTATS METAL

RÉSULTATS PAR SOURCE
Pour chaque source, le résultat METAL réellement produit reste visible. Une
source sans résultat responsable ne reçoit pas de PGN vide, mais un STATUS.txt
avec la raison concrète et, si disponible, le rapport technique détaillé.

GAME METAL
- positions et transpositions sont comptées ensemble ;
- victoire/nulle/défaite est évaluée du point de vue du camp au trait ;
- profil Équilibré : au moins 32 observations et 4 résultats décisifs, avec au
  maximum 2 coups préférés presque équivalents par position ;
- une partie modèle doit rencontrer au moins 3 positions préférées et 80%% de couverture ;
- la sortie est la partie source légale complète avec son résultat d’origine ;
  aucune pseudo-partie AnchorPly ni résultat fictif.

CTG METAL
Tous les coups de livre accessibles et décodés sont conservés dans des lignes RAW neutres, résultat *.
Sans filtre de fréquence, recommandations, poids appris ou résultats artificiels.
METAL/Metal.pgn réunit les lignes CTG et les parties GAME sélectionnées sans en-têtes personnels.
Les lignes CTG servent à importer un livre, pas à fournir des signaux de victoire/défaite.
Le rapport JSON indique couverture, limites de profondeur et restrictions du décodeur.
`,
			`Source2Metal v%s - EXPLICACIÓN DE RESULTADOS METAL

RESULTADOS POR FUENTE
Para cada fuente queda visible el resultado METAL realmente producido. Una fuente
sin resultado responsable no recibe un PGN vacío, sino STATUS.txt con el motivo
concreto y, cuando existe, el informe técnico detallado.

GAME METAL
- las posiciones y transposiciones se cuentan conjuntamente;
- victoria/tablas/derrota se evalúa desde el lado que mueve;
- perfil Equilibrado: mínimo 32 observaciones y 4 resultados decisivos, máximo
  2 jugadas preferidas casi equivalentes por posición;
- una partida modelo debe alcanzar al menos 3 posiciones preferidas y 80%% de cobertura;
- la salida es la partida fuente legal completa con su resultado original; sin
  pseudo-partidas AnchorPly ni resultados ficticios.

CTG METAL
Se conservan todas las jugadas accesibles y decodificadas en líneas RAW neutras, resultado *.
Sin filtros de frecuencia, recomendaciones, pesos aprendidos ni resultados artificiales.
METAL/Metal.pgn combina líneas CTG y partidas GAME seleccionadas sin cabeceras personales.
Las líneas CTG sirven para importar el libro, no como señales de victoria/derrota.
El informe JSON indica cobertura, cortes por profundidad y límites del decodificador.
`,
			`Source2Metal v%s - METAL 结果说明

每源结果
每个源实际产生的 METAL 结果都会保留可见。没有负责任结果的源不会得到空 PGN，
而是得到 STATUS.txt，说明具体原因，并在可用时指向技术详细报告。

GAME METAL
- 棋盘局面和转置一起统计；
- 胜/和/负从当前行棋方角度评估；
- “平衡”配置：至少 32 次观察和 4 个决定性结果，每个局面最多 2 个几乎等价的
  偏好着法；
- 模型对局必须命中至少 3 个偏好局面并达到至少 80%% 覆盖；
- 输出为完整、合法、真实的源对局并保留原始结果；不使用 AnchorPly 伪对局，
  不使用虚构结果。

CTG METAL
保留所有可达且已解码的开局库着法，输出中性 RAW 变例，结果为 *。
不使用频率筛选、推荐、学习权重或虚构结果。
METAL/Metal.pgn 自动合并 CTG 变例及筛选后的 GAME 对局，并移除个人头信息。
CTG 变例用于导入开局库，不是胜负学习信号。
JSON 报告列出覆盖率、深度截断和解码器限制。
`,
			`Source2Metal v%s - ПОЯСНЕНИЕ РЕЗУЛЬТАТОВ METAL

РЕЗУЛЬТАТЫ ПО ИСТОЧНИКАМ
Для каждого источника виден фактически созданный результат METAL. Источник без
ответственного результата не получает пустой PGN; вместо этого STATUS.txt
объясняет конкретную причину и, при наличии, указывает технический отчёт.

GAME METAL
- позиции и транспозиции учитываются совместно;
- победа/ничья/поражение оценивается со стороны игрока, которому принадлежит ход;
- профиль «Сбалансированный»: минимум 32 наблюдения и 4 решающих результата,
  максимум 2 почти равноценных предпочтительных хода на позицию;
- модельная партия должна встретить минимум 3 предпочтительные позиции и 80%% покрытия;
- выводится полная легальная исходная партия с исходным результатом; без
  AnchorPly-псевдопартий и без фиктивных результатов.

CTG METAL
Все декодированные доступные ходы сохраняются в нейтральных линиях RAW с результатом *.
Без фильтра частоты, рекомендаций, обученных весов и вымышленных результатов.
METAL/Metal.pgn объединяет линии CTG и выбранные партии GAME без персональных заголовков.
Линии CTG предназначены для импорта книги, а не для обучения по победам и поражениям.
JSON-отчёт содержит покрытие, ограничения глубины и ограничения декодера.
`)
		metalText := fmt.Sprintf(metalTemplate, version)
		if err := atomicWriteFile(filepath.Join(layout.MetalDir, "INFO - METAL RESULTS.txt"), []byte(metalText), 0644); err != nil {
			return err
		}
		if _, err := os.Stat(layout.MetalMergedDir); err == nil {
			mergedTemplate := L(
				`Source2Metal v%s - MERGED METAL RESULTS

The merged GAME-METAL is exactly the sum of selected per-source GAME contributions
after global deduplication. Statistical preferences are calculated over all valid
GAME sources together, allowing transpositions across PGN, CBH and 2CBH to supply
evidence together.

CTG-METAL remains traceable per CTG source and is not blindly mixed with real
GAME model games.

Merged GAME-METAL records : %s
Verified                  : %s
SHA-256                   : %s
`,
				`Source2Metal v%s - ZUSAMMENGEFÜHRTE METAL-ERGEBNISSE

Das zusammengeführte GAME-METAL ist nach globaler Deduplizierung exakt die Summe
der ausgewählten GAME-Beiträge pro Quelle. Statistische Präferenzen werden über
alle gültigen GAME-Quellen gemeinsam berechnet, sodass Transpositionen zwischen
PGN, CBH und 2CBH gemeinsam Evidenz liefern können.

CTG-METAL bleibt pro CTG-Quelle nachvollziehbar und wird nicht blind mit echten
GAME-Modellpartien gemischt.

GAME-METAL Records zusammengeführt : %s
Verifiziert                         : %s
SHA-256                             : %s
`,
				`Source2Metal v%s - UITLEG SAMENGEVOEGDE METAL RESULTATEN

De samengevoegde GAME-METAL is exact de som van de geselecteerde per-bron
GAME-bijdragen na globale ontdubbeling. De statistische voorkeuren worden over
alle geldige GAME-bronnen gezamenlijk berekend, zodat transposities tussen PGN,
CBH en 2CBH samen bewijs kunnen leveren.

CTG-METAL blijft per CTG-bron traceerbaar en wordt niet blind met echte GAME-
modelpartijen gemengd.

Samengevoegde GAME-METAL records : %s
Geverifieerd                     : %s
SHA-256                          : %s
`,
				`Source2Metal v%s - RÉSULTATS METAL FUSIONNÉS

Le GAME-METAL fusionné est exactement la somme des contributions GAME sélectionnées
par source après déduplication globale. Les préférences statistiques sont calculées
sur l’ensemble des sources GAME valides, de sorte que les transpositions entre PGN,
CBH et 2CBH puissent fournir des preuves conjointement.

CTG-METAL reste traçable par source CTG et n’est pas mélangé aveuglément avec les
vraies parties modèles GAME.

Enregistrements GAME-METAL fusionnés : %s
Vérifiés                            : %s
SHA-256                             : %s
`,
				`Source2Metal v%s - RESULTADOS METAL COMBINADOS

El GAME-METAL combinado es exactamente la suma de las contribuciones GAME
seleccionadas por fuente después de la deduplicación global. Las preferencias
estadísticas se calculan conjuntamente sobre todas las fuentes GAME válidas, de
modo que las transposiciones entre PGN, CBH y 2CBH aporten evidencia conjuntamente.

CTG-METAL sigue siendo trazable por fuente CTG y no se mezcla ciegamente con
partidas modelo GAME reales.

Registros GAME-METAL combinados : %s
Verificados                     : %s
SHA-256                         : %s
`,
				`Source2Metal v%s - 合并 METAL 结果

合并后的 GAME-METAL 在全局去重之后，正好等于各源已选择 GAME 贡献之和。统计偏好
在所有有效 GAME 源上联合计算，因此 PGN、CBH 和 2CBH 之间的转置可以共同提供证据。

CTG-METAL 仍按每个 CTG 源保持可追溯，不会与真实 GAME 模型对局盲目混合。

合并 GAME-METAL 记录 : %s
已验证              : %s
SHA-256             : %s
`,
				`Source2Metal v%s - ОБЪЕДИНЁННЫЕ РЕЗУЛЬТАТЫ METAL

Объединённый GAME-METAL после глобальной дедупликации точно равен сумме выбранных
GAME-вкладов по источникам. Статистические предпочтения рассчитываются совместно
по всем допустимым GAME-источникам, поэтому транспозиции между PGN, CBH и 2CBH
могут совместно давать данные.

CTG-METAL остаётся трассируемым по каждому CTG-источнику и не смешивается вслепую
с реальными модельными партиями GAME.

Объединённые записи GAME-METAL : %s
Проверено                     : %s
SHA-256                       : %s
`)
			mergedText := fmt.Sprintf(mergedTemplate, version, fmtInt(c.GameMetalSelected), fmtInt(c.GameMetalVerified), c.GameMetalSHA256)
			if err := atomicWriteFile(filepath.Join(layout.MetalMergedDir, "INFO - MERGED METAL RESULTS.txt"), []byte(mergedText), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFullProcessReport(path string, layout OutputLayout, inv Inventory, cfg Config, c Counters, status string) error {
	var b strings.Builder
	fmt.Fprintf(&b, L("Source2Metal v%s - FULL PROCESS REPORT\n", "Source2Metal v%s - VOLLSTÄNDIGER PROZESSBERICHT\n", "Source2Metal v%s - VOLLEDIG PROCESRAPPORT\n", "Source2Metal v%s - RAPPORT COMPLET DU PROCESSUS\n", "Source2Metal v%s - INFORME COMPLETO DEL PROCESO\n", "Source2Metal v%s - 完整处理报告\n", "Source2Metal v%s - ПОЛНЫЙ ОТЧЁТ О ПРОЦЕССЕ\n"), version)
	fmt.Fprintln(&b, strings.Repeat("=", 62))
	fmt.Fprintf(&b, L("Run status          : %s\n", "Laufstatus         : %s\n", "Runstatus          : %s\n", "Statut du run       : %s\n", "Estado del run      : %s\n", "运行状态            ：%s\n", "Статус запуска      : %s\n"), localizeStatus(status))
	fmt.Fprintf(&b, L("Run ID              : %s\n", "Lauf-ID            : %s\n", "Run-ID             : %s\n", "ID du run           : %s\n", "ID de ejecución     : %s\n", "运行 ID             ：%s\n", "ID запуска          : %s\n"), runID(c))
	if !c.Started.IsZero() {
		fmt.Fprintf(&b, L("Start               : %s\n", "Start              : %s\n", "Start              : %s\n", "Début               : %s\n", "Inicio              : %s\n", "开始                ：%s\n", "Начало              : %s\n"), c.Started.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&b, L("End                 : %s\n", "Ende               : %s\n", "Einde              : %s\n", "Fin                 : %s\n", "Fin                 : %s\n", "结束                ：%s\n", "Окончание           : %s\n"), c.Started.Add(c.Elapsed).Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(&b, "Root               : %s\n", inv.Root)
	fmt.Fprintf(&b, L("Mode               : %s\n", "Modus              : %s\n", "Modus              : %s\n", "Mode                : %s\n", "Modo                : %s\n", "模式                ：%s\n", "Режим               : %s\n"), cfg.Mode)
	fmt.Fprintf(&b, "Workers            : %d\n", cfg.Workers)
	fmt.Fprintf(&b, L("Max depth          : %d ply\n", "Max. Tiefe         : %d ply\n", "Max diepte         : %d ply\n", "Profondeur max.     : %d ply\n", "Profundidad máx.    : %d ply\n", "最大深度            ：%d ply\n", "Макс. глубина       : %d ply\n"), cfg.MaxPly)
	fmt.Fprintf(&b, L("Minimum game       : %d ply\n", "Mindestpartie      : %d ply\n", "Minimum partij     : %d ply\n", "Partie minimale     : %d ply\n", "Partida mínima      : %d ply\n", "最短对局            ：%d ply\n", "Минимальная партия  : %d ply\n"), cfg.MinPly)
	fmt.Fprintf(&b, L("Elo filter         : %d\n", "Elo-Filter         : %d\n", "Elo-filter         : %d\n", "Filtre Elo          : %d\n", "Filtro Elo          : %d\n", "Elo 过滤器          ：%d\n", "Фильтр Elo          : %d\n"), cfg.MinElo)
	fmt.Fprintf(&b, L("Max Elo difference : %d\n", "Max. Elo-Differenz : %d\n", "Max Elo-verschil   : %d\n", "Écart Elo max.      : %d\n", "Diferencia Elo máx. : %d\n", "最大 Elo 差         ：%d\n", "Макс. разница Elo   : %d\n"), cfg.MaxEloGap)
	fmt.Fprintf(&b, L("Duration           : %s\n\n", "Dauer              : %s\n\n", "Duur               : %s\n\n", "Durée               : %s\n\n", "Duración            : %s\n\n", "耗时                ：%s\n\n", "Длительность        : %s\n\n"), c.Elapsed)

	fmt.Fprintln(&b, L("PROCESS FLOW", "PROZESSKETTE", "PROCESKETEN", "CHAÎNE DE TRAITEMENT", "FLUJO DEL PROCESO", "处理流程", "ЦЕПОЧКА ОБРАБОТКИ"))
	fmt.Fprintln(&b, "------------")
	fmt.Fprintln(&b, L(`PGN -------------------------------> per-source GAME RAW ----\`, `PGN -------------------------------> GAME RAW pro Quelle ----\`, `PGN -------------------------------> per-bron GAME RAW ----\`, `PGN -------------------------------> GAME RAW par source ----\`, `PGN -------------------------------> GAME RAW por fuente ----\`, `PGN -------------------------------> 每源 GAME RAW ----\`, `PGN -------------------------------> GAME RAW по источнику ----\`))
	fmt.Fprintln(&b, L("CBH  -> temporary adapter ----------> per-source GAME RAW -----+--> merged GAME RAW", "CBH  -> temporärer Adapter ----------> GAME RAW pro Quelle -----+--> zusammengeführtes GAME RAW", "CBH  -> tijdelijke adapter ----------> per-bron GAME RAW -----+--> samengevoegde GAME RAW", "CBH  -> adaptateur temporaire -------> GAME RAW par source -----+--> GAME RAW fusionné", "CBH  -> adaptador temporal -----------> GAME RAW por fuente -----+--> GAME RAW combinado", "CBH  -> 临时适配器 --------------------> 每源 GAME RAW ----------+--> 合并 GAME RAW", "CBH  -> временный адаптер -----------> GAME RAW по источнику ----+--> объединённый GAME RAW"))
	fmt.Fprintln(&b, L("2CBH -> temporary adapter ----------> per-source GAME RAW ----/", "2CBH -> temporärer Adapter ----------> GAME RAW pro Quelle ----/", "2CBH -> tijdelijke adapter ----------> per-bron GAME RAW ----/", "2CBH -> adaptateur temporaire -------> GAME RAW par source ----/", "2CBH -> adaptador temporal -----------> GAME RAW por fuente ----/", "2CBH -> 临时适配器 --------------------> 每源 GAME RAW ---------/", "2CBH -> временный адаптер -----------> GAME RAW по источнику ---/"))
	fmt.Fprintln(&b, L("PGN/CBH/2CBH -> legal position analysis -> combined GAME evidence -> real model games -> GAME METAL", "PGN/CBH/2CBH -> legale Positionsanalyse -> gemeinsame GAME-Evidenz -> echte Modellpartien -> GAME METAL", "PGN/CBH/2CBH -> legale positie-analyse -> gezamenlijke GAME evidence -> echte modelpartijen -> GAME METAL", "PGN/CBH/2CBH -> analyse de positions légales -> preuve GAME combinée -> vraies parties modèles -> GAME METAL", "PGN/CBH/2CBH -> análisis legal de posiciones -> evidencia GAME combinada -> partidas modelo reales -> GAME METAL", "PGN/CBH/2CBH -> 合法局面分析 -> 合并 GAME 证据 -> 真实模型对局 -> GAME METAL", "PGN/CBH/2CBH -> анализ легальных позиций -> объединённые данные GAME -> реальные модельные партии -> GAME METAL"))
	fmt.Fprintln(&b, L("CTG/CTO/CTB -> CTG RAW extractor -------------------------------------------> per-source CTG RAW", "CTG/CTO/CTB -> CTG-RAW-Extraktor ------------------------------------------> CTG RAW pro Quelle", "CTG/CTO/CTB -> CTG RAW-extractor -------------------------------------------> per-bron CTG RAW", "CTG/CTO/CTB -> extracteur CTG RAW -----------------------------------------> CTG RAW par source", "CTG/CTO/CTB -> extractor CTG RAW ------------------------------------------> CTG RAW por fuente", "CTG/CTO/CTB -> CTG RAW 提取器 ---------------------------------------------> 每源 CTG RAW", "CTG/CTO/CTB -> экстрактор CTG RAW ----------------------------------------> CTG RAW по источнику"))
	fmt.Fprintln(&b, "CTG RAW -> neutral CTG METAL (*) -> METAL/Metal.pgn")
	fmt.Fprintln(&b, L("GAME RAW + BOOK RAW -> merged ALL Sources RAW (without cross-class dedup)", "GAME RAW + BOOK RAW -> zusammengeführtes ALL Sources RAW (ohne klassenübergreifende Deduplizierung)", "GAME RAW + BOOK RAW -> samengevoegde ALLE Sources RAW (zonder cross-class dedup)", "GAME RAW + BOOK RAW -> ALL Sources RAW fusionné (sans déduplication interclasse)", "GAME RAW + BOOK RAW -> ALL Sources RAW combinado (sin deduplicación entre clases)", "GAME RAW + BOOK RAW -> 合并 ALL Sources RAW（不进行跨类别去重）", "GAME RAW + BOOK RAW -> объединённый ALL Sources RAW (без межклассовой дедупликации)"))

	fmt.Fprintln(&b, "\n"+L("SUMMARY RESULTS", "ZUSAMMENGEFASSTE ERGEBNISSE", "SAMENGEVATTE RESULTATEN", "RÉSULTATS RÉSUMÉS", "RESULTADOS RESUMIDOS", "结果摘要", "СВОДНЫЕ РЕЗУЛЬТАТЫ"))
	fmt.Fprintln(&b, "----------------------")
	fmt.Fprintf(&b, L("GAME RAW          : %s records | verified %s | %s\n", "GAME RAW          : %s Datensätze | verifiziert %s | %s\n", "GAME RAW          : %s records | geverifieerd %s | %s\n", "GAME RAW          : %s enregistrements | vérifiés %s | %s\n", "GAME RAW          : %s registros | verificados %s | %s\n", "GAME RAW          ：%s 记录 | 已验证 %s | %s\n", "GAME RAW          : %s записей | проверено %s | %s\n"), fmtInt(c.GamesAccepted), fmtInt(c.RawGamesVerified), relOutput(layout, layout.RawGamesFile))
	fmt.Fprint(&b, binSummary(c), bookSummary(c))
	fmt.Fprintln(&b, binScopeText())
	fmt.Fprintln(&b, bookPolicyText())
	fmt.Fprintf(&b, L("CTG RAW           : %s built | %s skipped | %s failed | %s records | verified %s\n", "CTG RAW           : %s erstellt | %s übersprungen | %s fehlgeschlagen | %s Datensätze | verifiziert %s\n", "CTG RAW           : %s gebouwd | %s overgeslagen | %s mislukt | %s records | geverifieerd %s\n", "CTG RAW           : %s créés | %s ignorés | %s échecs | %s enregistrements | vérifiés %s\n", "CTG RAW           : %s creados | %s omitidos | %s fallidos | %s registros | verificados %s\n", "CTG RAW           ：%s 已生成 | %s 已跳过 | %s 失败 | %s 记录 | 已验证 %s\n", "CTG RAW           : %s создано | %s пропущено | %s ошибок | %s записей | проверено %s\n"), fmtInt(c.CTGRawBuilt), fmtInt(c.CTGRawSkipped), fmtInt(c.CTGRawFailed), fmtInt(c.CTGRawGames), fmtInt(c.RawBooksVerified))
	if c.CombinedGames > 0 {
		fmt.Fprintf(&b, L("ALL SOURCES RAW   : %s records | verified %s\n", "ALL SOURCES RAW   : %s Datensätze | verifiziert %s\n", "ALLE SOURCES RAW  : %s records | geverifieerd %s\n", "ALL SOURCES RAW   : %s enregistrements | vérifiés %s\n", "ALL SOURCES RAW   : %s registros | verificados %s\n", "ALL SOURCES RAW   ：%s 记录 | 已验证 %s\n", "ALL SOURCES RAW   : %s записей | проверено %s\n"), fmtInt(c.CombinedGames), fmtInt(c.CombinedVerified))
	}
	fmt.Fprintf(&b, L("GAME METAL        : %s built | %s skipped | %s failed | %s model games | verified %s | %s\n", "GAME METAL        : %s erstellt | %s übersprungen | %s fehlgeschlagen | %s Modellpartien | verifiziert %s | %s\n", "GAME METAL        : %s gebouwd | %s overgeslagen | %s mislukt | %s modelpartijen | geverifieerd %s | %s\n", "GAME METAL        : %s créés | %s ignorés | %s échecs | %s parties modèles | vérifiés %s | %s\n", "GAME METAL        : %s creados | %s omitidos | %s fallidos | %s partidas modelo | verificados %s | %s\n", "GAME METAL        ：%s 已生成 | %s 已跳过 | %s 失败 | %s 模型对局 | 已验证 %s | %s\n", "GAME METAL        : %s создано | %s пропущено | %s ошибок | %s модельных партий | проверено %s | %s\n"), fmtInt(c.GameMetalBuilt), fmtInt(c.GameMetalSkipped), fmtInt(c.GameMetalFailed), fmtInt(c.GameMetalSelected), fmtInt(c.GameMetalVerified), relOutput(layout, layout.GameMetalFile))
	fmt.Fprintf(&b, L("CTG METAL         : %s built | %s skipped | %s failed\n", "CTG METAL         : %s erstellt | %s übersprungen | %s fehlgeschlagen\n", "CTG METAL         : %s gebouwd | %s overgeslagen | %s mislukt\n", "CTG METAL         : %s créés | %s ignorés | %s échecs\n", "CTG METAL         : %s creados | %s omitidos | %s fallidos\n", "CTG METAL         ：%s 已生成 | %s 已跳过 | %s 失败\n", "CTG METAL         : %s создано | %s пропущено | %s ошибок\n"), fmtInt(c.CTGMetalBuilt), fmtInt(c.CTGMetalSkipped), fmtInt(c.CTGMetalFailed))

	fmt.Fprintln(&b, "\n"+L("TRACEABILITY PER SOURCE", "RÜCKVERFOLGBARKEIT PRO QUELLE", "HERLEIDBAARHEID PER BRON", "TRAÇABILITÉ PAR SOURCE", "TRAZABILIDAD POR FUENTE", "每源可追溯性", "ТРАССИРУЕМОСТЬ ПО ИСТОЧНИКАМ"))
	fmt.Fprintln(&b, "-------------------------")
	traces := mergedSourceTraces(c.SourceTraces)
	sort.SliceStable(traces, func(i, j int) bool {
		a := strings.ToLower(string(traces[i].Kind) + "|" + traces[i].SourcePath)
		bb := strings.ToLower(string(traces[j].Kind) + "|" + traces[j].SourcePath)
		return a < bb
	})
	if len(traces) == 0 {
		fmt.Fprintln(&b, L("No product trace is available in this mode.", "In diesem Modus ist keine Produktverfolgung verfügbar.", "Geen producttrace beschikbaar in deze modus.", "Aucune trace de produit n’est disponible dans ce mode.", "No hay trazabilidad de producto disponible en este modo.", "此模式下没有可用的结果追踪。", "В этом режиме трассировка результата недоступна."))
	}
	for i, t := range traces {
		fmt.Fprintf(&b, "\n%02d. %s [%s]\n", i+1, t.Base, t.Kind)
		fmt.Fprintf(&b, L("    Source            : %s\n", "    Quelle            : %s\n", "    Bron              : %s\n", "    Source            : %s\n", "    Fuente            : %s\n", "    源                ：%s\n", "    Источник          : %s\n"), t.SourcePath)
		fmt.Fprintf(&b, L("    RAW result        : %s\n", "    RAW-Ergebnis      : %s\n", "    RAW resultaat     : %s\n", "    Résultat RAW      : %s\n", "    Resultado RAW     : %s\n", "    RAW 结果          ：%s\n", "    Результат RAW     : %s\n"), relOutput(layout, t.RawPath))
		if t.Kind == KindCTG || t.Kind == KindBIN {
			fmt.Fprintf(&b, L("    RAW produced      : %s | verified %s\n", "    RAW erzeugt       : %s | verifiziert %s\n", "    RAW geproduceerd  : %s | geverifieerd %s\n", "    RAW produit       : %s | vérifié %s\n", "    RAW producido     : %s | verificado %s\n", "    已生成 RAW        ：%s | 已验证 %s\n", "    RAW создано       : %s | проверено %s\n"), fmtInt(t.RawAccepted), fmtInt(t.RawVerified))
		} else {
			fmt.Fprintf(&b, L("    RAW seen          : %s\n", "    RAW gesehen       : %s\n", "    RAW gezien        : %s\n", "    RAW vus           : %s\n", "    RAW vistos        : %s\n", "    已查看 RAW        ：%s\n", "    RAW просмотрено   : %s\n"), fmtInt(t.RawSeen))
			fmt.Fprintf(&b, L("    RAW per source    : %s | local dup %s | cross-source dup %s | rejected %s | verified %s\n", "    RAW pro Quelle    : %s | lokale Dup %s | quellenübergreifende Dup %s | abgelehnt %s | verifiziert %s\n", "    RAW per bron      : %s | lokale dup %s | cross-source dup %s | afgewezen %s | geverifieerd %s\n", "    RAW par source    : %s | doublon local %s | doublon inter-source %s | rejeté %s | vérifié %s\n", "    RAW por fuente    : %s | duplicado local %s | duplicado entre fuentes %s | rechazado %s | verificado %s\n", "    每源 RAW          ：%s | 本地重复 %s | 跨源重复 %s | 已拒绝 %s | 已验证 %s\n", "    RAW по источнику  : %s | лок. дубликаты %s | межисточн. дубликаты %s | отклонено %s | проверено %s\n"), fmtInt(t.RawAccepted), fmtInt(t.RawLocalDuplicate), fmtInt(t.RawCrossDuplicate), fmtInt(t.RawRejected), fmtInt(t.RawVerified))
		}
		if t.Kind == KindBIN {
			fmt.Fprintln(&b, binScopeText())
			continue
		}
		fmt.Fprintf(&b, L("    METAL result      : %s\n", "    METAL-Ergebnis    : %s\n", "    METAL resultaat   : %s\n", "    Résultat METAL    : %s\n", "    Resultado METAL   : %s\n", "    METAL 结果        ：%s\n", "    Результат METAL   : %s\n"), relOutput(layout, t.MetalPath))
		if t.Kind == KindCTG {
			fmt.Fprintf(&b, L("    METAL selected    : %s\n", "    METAL ausgewählt  : %s\n", "    METAL geselecteerd: %s\n", "    METAL sélectionné : %s\n", "    METAL seleccionado: %s\n", "    已选择 METAL      ：%s\n", "    METAL отобрано    : %s\n"), fmtInt(t.MetalSelected))
		} else {
			fmt.Fprintf(&b, L("    METAL seen        : %s | model candidates %s | selected %s | invalid %s\n", "    METAL gesehen     : %s | Modellkandidaten %s | ausgewählt %s | ungültig %s\n", "    METAL gezien      : %s | modelkandidaten %s | geselecteerd %s | ongeldig %s\n", "    METAL vus         : %s | candidats modèles %s | sélectionnés %s | invalides %s\n", "    METAL vistos      : %s | candidatos modelo %s | seleccionados %s | no válidos %s\n", "    已查看 METAL      ：%s | 模型候选 %s | 已选择 %s | 无效 %s\n", "    METAL просмотрено : %s | кандидаты моделей %s | отобрано %s | недопустимо %s\n"), fmtInt(t.MetalSeen), fmtInt(t.MetalEligible), fmtInt(t.MetalSelected), fmtInt(t.MetalInvalid))
		}
	}

	fmt.Fprintln(&b, "\n"+L("CHECKS AND GUARDRAILS", "PRÜFUNGEN UND SCHUTZREGELN", "CONTROLES EN VANGRAILS", "CONTRÔLES ET GARDE-FOUS", "CONTROLES Y SALVAGUARDAS", "检查与保护措施", "ПРОВЕРКИ И ОГРАНИЧИТЕЛИ"))
	fmt.Fprintln(&b, "-----------------------")
	checks := []string{
		L("- Sources are read-only; original source files are never modified.", "- Quellen werden nur gelesen; Originaldateien werden nie verändert.", "- Bronnen worden alleen gelezen; originele bronbestanden worden niet gewijzigd.", "- Les sources sont en lecture seule ; les fichiers d’origine ne sont jamais modifiés.", "- Las fuentes son de solo lectura; los archivos originales nunca se modifican.", "- 源文件只读；原始源文件绝不会被修改。", "- Источники только читаются; исходные файлы никогда не изменяются."),
		L("- Source discovery only descends from the selected root; output and temporary folders are skipped.", "- Die Quellensuche steigt nur vom gewählten Stammordner ab; Ausgabe- und TEMP-Ordner werden übersprungen.", "- Bronzoektocht gaat alleen vanuit de gekozen root naar beneden; output- en tempmappen worden overgeslagen.", "- La recherche descend uniquement depuis la racine choisie ; les dossiers de sortie et temporaires sont ignorés.", "- La búsqueda solo desciende desde la raíz elegida; se omiten las carpetas de salida y temporales.", "- 源搜索只从所选根目录向下进行；输出目录和临时目录会被跳过。", "- Поиск идёт только вниз от выбранного корня; папки вывода и временные папки пропускаются."),
		L("- ZIP/RAR/7z archives are not opened as sources.", "- ZIP/RAR/7z-Archive werden nicht als Quellen geöffnet.", "- ZIP/RAR/7z-archieven worden niet als bron geopend.", "- Les archives ZIP/RAR/7z ne sont pas ouvertes comme sources.", "- Los archivos ZIP/RAR/7z no se abren como fuentes.", "- ZIP/RAR/7z 压缩包不会作为源打开。", "- Архивы ZIP/RAR/7z не открываются как источники."),
		L("- RAW and METAL PGNs receive structural record checks where applicable; GAME-METAL also receives legal real-game validation.", "- RAW- und METAL-PGNs erhalten soweit anwendbar strukturelle Record-Prüfungen; GAME-METAL zusätzlich eine Prüfung echter legaler Partien.", "- RAW- en METAL-PGN's krijgen waar toepasbaar structurele recordcontrole; GAME-METAL krijgt bovendien legale echte-partijcontrole.", "- Les PGN RAW et METAL reçoivent des contrôles structurels quand cela s’applique ; GAME-METAL reçoit aussi une validation de parties réelles légales.", "- Los PGN RAW y METAL reciben controles estructurales cuando corresponde; GAME-METAL además valida partidas reales legales.", "- RAW 和 METAL PGN 在适用时进行结构记录检查；GAME-METAL 还进行真实合法对局验证。", "- RAW- и METAL-PGN проходят структурные проверки там, где это применимо; GAME-METAL также проверяет реальные легальные партии."),
		L("- Temporary segments are excluded from the user checksum list.", "- Temporäre Segmente stehen nicht in der Benutzer-Prüfsummenliste.", "- Tijdelijke segmenten worden niet opgenomen in de gebruikerschecksumlijst.", "- Les segments temporaires sont exclus de la liste de sommes de contrôle utilisateur.", "- Los segmentos temporales se excluyen de la lista de sumas de verificación del usuario.", "- 临时分段不会列入用户校验和列表。", "- Временные сегменты исключены из пользовательского списка контрольных сумм."),
	}
	for _, x := range checks {
		fmt.Fprintln(&b, x)
	}
	return atomicWriteFile(path, []byte(b.String()), 0644)
}

func mergedSourceTraces(in []SourceTrace) []SourceTrace {
	var out []SourceTrace
	for _, t := range in {
		found := -1
		for i := range out {
			if out[i].Kind == t.Kind && strings.EqualFold(out[i].Base, t.Base) && strings.EqualFold(out[i].SourcePath, t.SourcePath) {
				found = i
				break
			}
		}
		if found < 0 {
			out = append(out, t)
			continue
		}
		d := &out[found]
		if t.RawPath != "" {
			d.RawPath = t.RawPath
		}
		if t.MetalPath != "" {
			d.MetalPath = t.MetalPath
		}
		d.RawSeen += t.RawSeen
		d.RawAccepted += t.RawAccepted
		d.RawLocalDuplicate += t.RawLocalDuplicate
		d.RawCrossDuplicate += t.RawCrossDuplicate
		d.RawRejected += t.RawRejected
		d.RawVerified += t.RawVerified
		d.MetalSeen += t.MetalSeen
		d.MetalEligible += t.MetalEligible
		d.MetalSelected += t.MetalSelected
		d.MetalInvalid += t.MetalInvalid
	}
	return out
}
