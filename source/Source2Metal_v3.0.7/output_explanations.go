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
		rawTemplate := L(
			`Source2Metal v%s - RAW RESULTS EXPLAINED

GAME RAW
PGN, CBH and 2CBH are treated as GAME sources. CBH/2CBH are temporarily
translated to PGN by the integrated adapter and then pass through the same RAW
validation as PGN. Per-source and merged results remain visible.

CTG RAW
Each complete CTG/CTO/CTB set receives its own RAW PGN. Output size follows the
book content and source structure that can actually be decoded.

ALL SOURCES RAW
When both GAME RAW and CTG RAW exist, Source2Metal also creates
"Source2Metal - Merged ALL Sources - RAW.pgn". This is a streaming merge of
already validated products. No additional cross-class deduplication is performed
between GAME and CTG because their origin and representation differ.

ACTIVE SETTINGS
Maximum RAW depth : %d ply
Minimum game      : %d ply
Elo filter        : %d (0 = keep source selection)
Max Elo difference: %d (0 = no limit)
`,
			`Source2Metal v%s - ERKLÄRUNG DER RAW-ERGEBNISSE

GAME RAW
PGN, CBH und 2CBH werden als GAME-Quellen behandelt. CBH/2CBH werden durch den
integrierten Adapter vorübergehend in PGN übersetzt und danach wie PGN durch
dieselbe RAW-Prüfung geführt. Ergebnisse pro Quelle und zusammengeführt bleiben
sichtbar.

CTG RAW
Jedes vollständige CTG/CTO/CTB-Set erhält eine eigene RAW-PGN. Die Ausgabegröße
folgt dem tatsächlich dekodierbaren Buchinhalt und der Quellenstruktur.

ALL SOURCES RAW
Wenn GAME RAW und CTG RAW vorhanden sind, erstellt Source2Metal zusätzlich
"Source2Metal - Merged ALL Sources - RAW.pgn". Bereits geprüfte Produkte werden
gestreamt zusammengeführt. Zwischen GAME und CTG findet bewusst keine zusätzliche
klassenübergreifende Deduplizierung statt.

AKTIVE EINSTELLUNGEN
Maximale RAW-Tiefe : %d ply
Mindestpartie      : %d ply
Elo-Filter         : %d (0 = Quellenauswahl beibehalten)
Max. Elo-Differenz : %d (0 = keine Grenze)
`,
			`Source2Metal v%s - UITLEG RAW RESULTATEN

GAME RAW
PGN, CBH en 2CBH worden als GAME-bronnen behandeld. CBH/2CBH worden via de
geïntegreerde adapter tijdelijk naar PGN vertaald en daarna door dezelfde RAW-
controle verwerkt. Per bron en samengevoegd blijven de resultaten zichtbaar.

CTG RAW
Iedere complete CTG/CTO/CTB-set krijgt een eigen RAW-PGN. De uitvoergrootte
volgt de werkelijk decodeerbare boekinhoud en bronstructuur.

ALLE SOURCES RAW
Als zowel GAME RAW als CTG RAW bestaan, maakt Source2Metal daarnaast
"Source2Metal - Merged ALL Sources - RAW.pgn". Dit is een streaming samenvoeging
van reeds gecontroleerde producten. Tussen GAME en CTG vindt bewust geen extra
cross-class deduplicatie plaats omdat herkomst en representatie verschillen.

ACTIEVE INSTELLINGEN
Maximale RAW-diepte : %d ply
Minimum partij      : %d ply
Elo-filter          : %d (0 = bronselectie behouden)
Max Elo-verschil    : %d (0 = geen grens)
`,
			`Source2Metal v%s - EXPLICATION DES RÉSULTATS RAW

GAME RAW
PGN, CBH et 2CBH sont traités comme sources GAME. CBH/2CBH sont temporairement
convertis en PGN par l’adaptateur intégré puis soumis à la même validation RAW
que PGN. Les résultats par source et fusionnés restent visibles.

CTG RAW
Chaque ensemble CTG/CTO/CTB complet reçoit son propre PGN RAW. La taille suit le
contenu du livre et la structure réellement décodables.

ALL SOURCES RAW
Lorsque GAME RAW et CTG RAW existent, Source2Metal crée aussi
"Source2Metal - Merged ALL Sources - RAW.pgn". Les produits déjà validés sont
fusionnés en flux, sans déduplication supplémentaire entre GAME et CTG.

PARAMÈTRES ACTIFS
Profondeur RAW max. : %d ply
Partie minimale     : %d ply
Filtre Elo          : %d (0 = conserver la sélection source)
Écart Elo max.      : %d (0 = aucune limite)
`,
			`Source2Metal v%s - EXPLICACIÓN DE RESULTADOS RAW

GAME RAW
PGN, CBH y 2CBH se tratan como fuentes GAME. CBH/2CBH se convierten temporalmente
a PGN mediante el adaptador integrado y después pasan la misma validación RAW
que PGN. Los resultados por fuente y combinados siguen siendo visibles.

CTG RAW
Cada conjunto CTG/CTO/CTB completo obtiene su propio PGN RAW. El tamaño sigue el
contenido y la estructura que realmente pueden decodificarse.

ALL SOURCES RAW
Cuando existen GAME RAW y CTG RAW, Source2Metal crea también
"Source2Metal - Merged ALL Sources - RAW.pgn". Es una combinación por streaming
de productos ya validados, sin deduplicación adicional entre GAME y CTG.

AJUSTES ACTIVOS
Profundidad RAW máx.: %d ply
Partida mínima      : %d ply
Filtro Elo          : %d (0 = mantener selección de origen)
Diferencia Elo máx. : %d (0 = sin límite)
`,
			`Source2Metal v%s - RAW 结果说明

GAME RAW
PGN、CBH 和 2CBH 作为 GAME 源处理。CBH/2CBH 先由集成适配器临时转换为 PGN，
然后使用与 PGN 相同的 RAW 验证。每个源以及合并后的结果都会保留可见。

CTG RAW
每个完整 CTG/CTO/CTB 文件组都会生成自己的 RAW PGN。输出大小由实际可以解码的
开局库内容和源结构决定。

ALL SOURCES RAW
当 GAME RAW 和 CTG RAW 都存在时，Source2Metal 还会创建
"Source2Metal - Merged ALL Sources - RAW.pgn"。这是对已经验证结果的流式合并；
GAME 与 CTG 之间不额外进行跨类别去重，因为二者来源和表示方式不同。

当前设置
最大 RAW 深度 : %d ply
最短对局      : %d ply
Elo 过滤器    : %d（0 = 保留源选择）
最大 Elo 差   : %d（0 = 无限制）
`,
			`Source2Metal v%s - ПОЯСНЕНИЕ РЕЗУЛЬТАТОВ RAW

GAME RAW
PGN, CBH и 2CBH обрабатываются как GAME-источники. CBH/2CBH временно
преобразуются в PGN встроенным адаптером и затем проходят ту же RAW-проверку,
что и PGN. Результаты по источникам и объединённые результаты остаются видимыми.

CTG RAW
Каждый полный набор CTG/CTO/CTB получает собственный RAW PGN. Размер вывода
определяется реально декодируемым содержимым книги и структурой источника.

ALL SOURCES RAW
Если существуют GAME RAW и CTG RAW, Source2Metal также создаёт
"Source2Metal - Merged ALL Sources - RAW.pgn". Уже проверенные продукты
объединяются потоково, без дополнительной межклассовой дедупликации GAME и CTG.

АКТИВНЫЕ НАСТРОЙКИ
Макс. глубина RAW : %d ply
Минимальная партия: %d ply
Фильтр Elo        : %d (0 = сохранить исходный отбор)
Макс. разница Elo : %d (0 = без ограничения)
`)
		rawText := fmt.Sprintf(rawTemplate, version, cfg.MaxPly, cfg.MinPly, cfg.MinElo, cfg.MaxEloGap)
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
CTG uses book-oriented selection. Strict selection is tried first. With zero
records, concrete rejection reasons are counted. Only when diagnostics show
statistical potential is a visible conservative Small/Broad fallback attempted.
The evidence requirement is never lowered silently.

WHY NO SINGLE ALL-METAL FILE?
GAME-METAL consists of complete real source games. CTG-METAL encodes book-oriented
learning signals from CTG statistics. These different evidence semantics remain
recognizably separate.

FRITZ - LEARN FROM DATABASE
Wins             ON
Losses           ON
White            OFF
Black            OFF
Player           OFF
Player name      EMPTY
Games            ALL
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
CTG verwendet buchorientierte Auswahl. Zuerst gilt die strikte Auswahl. Bei null
Records werden konkrete Ablehnungsgründe gezählt. Nur bei statistischem Potenzial
wird sichtbar ein vorsichtiger Small/Broad-Fallback versucht. Die Beweisanforderung
wird nie still abgesenkt.

WARUM KEINE EINZIGE ALL-METAL-DATEI?
GAME-METAL besteht aus vollständigen echten Quellpartien. CTG-METAL kodiert
buchorientierte Lernsignale aus CTG-Statistik. Diese unterschiedlichen Evidenzarten
bleiben erkennbar getrennt.

FRITZ - AUS DATENBANK LERNEN
Siege            EIN
Niederlagen      EIN
Weiß             AUS
Schwarz          AUS
Spieler          AUS
Spielername      LEER
Partien          ALLE
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
CTG gebruikt de boekgerichte selectie. Eerst geldt de strikte selectie. Bij nul
records worden concrete afwijsredenen geteld. Alleen wanneer de diagnostiek
statistisch potentieel toont, wordt zichtbaar een voorzichtige Kleine/Brede-
fallback geprobeerd. De bewijslast wordt niet stilzwijgend verlaagd.

WAAROM GEEN ENKEL ALLES-METAL-BESTAND?
GAME-METAL bestaat uit echte complete bronpartijen. CTG-METAL encodeert
boekgerichte leersignalen uit CTG-statistiek. Deze verschillende evidence-
semantieken blijven daarom herkenbaar gescheiden.

FRITZ - BIJLEREN UIT DATABASE
Overwinningen    AAN
Verliespartijen  AAN
Wit              UIT
Zwart            UIT
Speler           UIT
Spelernaam       LEEG
Partijen         ALLE
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
CTG utilise une sélection orientée livre. La sélection stricte est essayée d’abord.
Avec zéro enregistrement, les raisons de rejet sont comptées. Un repli Small/Broad
prudent et visible n’est tenté que si le diagnostic montre un potentiel statistique.

POURQUOI PAS UN FICHIER ALL-METAL UNIQUE ?
GAME-METAL contient des parties source réelles complètes. CTG-METAL encode des
signaux d’apprentissage issus des statistiques CTG. Ils restent donc séparés.

FRITZ - APPRENDRE DEPUIS UNE BASE
Victoires        ACTIVÉES
Défaites         ACTIVÉES
Blancs           DÉSACTIVÉS
Noirs            DÉSACTIVÉS
Joueur           DÉSACTIVÉ
Nom du joueur    VIDE
Parties          TOUTES
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
CTG utiliza selección orientada al libro. Primero se prueba la selección estricta.
Con cero registros se cuentan los motivos de rechazo. Solo si el diagnóstico
muestra potencial estadístico se intenta una alternativa Small/Broad prudente y visible.

¿POR QUÉ NO UN ÚNICO ARCHIVO ALL-METAL?
GAME-METAL contiene partidas fuente reales completas. CTG-METAL codifica señales
de aprendizaje basadas en estadísticas CTG. Se mantienen separados.

FRITZ - APRENDER DE BASE DE DATOS
Victorias        ACTIVADAS
Derrotas         ACTIVADAS
Blancas          DESACTIVADAS
Negras           DESACTIVADAS
Jugador          DESACTIVADO
Nombre           VACÍO
Partidas         TODAS
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
CTG 使用面向开局库的选择。先执行严格选择。若结果为 0，则统计具体拒绝原因。
只有诊断显示存在统计潜力时，才会明确显示并尝试谨慎的 Small/Broad 后备模式。
证据要求不会在后台悄悄降低。

为什么没有单一的 ALL-METAL 文件？
GAME-METAL 是完整真实的源对局；CTG-METAL 则编码来自 CTG 统计的开局库学习信号。
两种证据语义不同，因此保持清晰分离。

FRITZ - 从数据库学习
胜局            开启
负局            开启
白方            关闭
黑方            关闭
棋手            关闭
棋手名          留空
对局            全部
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
CTG использует книжный отбор. Сначала применяется строгий режим. При нуле записей
считаются конкретные причины отказа. Осторожный Small/Broad-резерв применяется
явно и только при наличии статистического потенциала. Требования не снижаются скрытно.

ПОЧЕМУ НЕТ ЕДИНОГО ALL-METAL-ФАЙЛА?
GAME-METAL состоит из полных реальных партий. CTG-METAL кодирует книжные сигналы
обучения из статистики CTG. Эти типы данных остаются явно разделёнными.

FRITZ - ОБУЧЕНИЕ ИЗ БАЗЫ
Победы           ВКЛ.
Поражения        ВКЛ.
Белые            ВЫКЛ.
Чёрные           ВЫКЛ.
Игрок            ВЫКЛ.
Имя игрока       ПУСТО
Партии           ВСЕ
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
	fmt.Fprintln(&b, L("CTG/CTO/CTB -> strict CTG Metal selection -> optional visible cautious fallback -> per-source CTG METAL", "CTG/CTO/CTB -> strikte CTG-Metal-Auswahl -> ggf. sichtbarer vorsichtiger Fallback -> CTG METAL pro Quelle", "CTG/CTO/CTB -> strikte CTG Metal-selectie -> evt. zichtbare voorzichtige fallback -> per-bron CTG METAL", "CTG/CTO/CTB -> sélection CTG Metal stricte -> repli prudent visible éventuel -> CTG METAL par source", "CTG/CTO/CTB -> selección CTG Metal estricta -> posible alternativa prudente visible -> CTG METAL por fuente", "CTG/CTO/CTB -> 严格 CTG Metal 选择 -> 可选且明确显示的谨慎后备方案 -> 每源 CTG METAL", "CTG/CTO/CTB -> строгий отбор CTG Metal -> при необходимости явно показанный осторожный резерв -> CTG METAL по источнику"))
	fmt.Fprintln(&b, L("GAME RAW + CTG RAW -> merged ALL Sources RAW (without cross-class dedup)", "GAME RAW + CTG RAW -> zusammengeführtes ALL Sources RAW (ohne klassenübergreifende Deduplizierung)", "GAME RAW + CTG RAW -> samengevoegde ALLE Sources RAW (zonder cross-class dedup)", "GAME RAW + CTG RAW -> ALL Sources RAW fusionné (sans déduplication interclasse)", "GAME RAW + CTG RAW -> ALL Sources RAW combinado (sin deduplicación entre clases)", "GAME RAW + CTG RAW -> 合并 ALL Sources RAW（不进行跨类别去重）", "GAME RAW + CTG RAW -> объединённый ALL Sources RAW (без межклассовой дедупликации)"))

	fmt.Fprintln(&b, "\n"+L("SUMMARY RESULTS", "ZUSAMMENGEFASSTE ERGEBNISSE", "SAMENGEVATTE RESULTATEN", "RÉSULTATS RÉSUMÉS", "RESULTADOS RESUMIDOS", "结果摘要", "СВОДНЫЕ РЕЗУЛЬТАТЫ"))
	fmt.Fprintln(&b, "----------------------")
	fmt.Fprintf(&b, L("GAME RAW          : %s records | verified %s | %s\n", "GAME RAW          : %s Datensätze | verifiziert %s | %s\n", "GAME RAW          : %s records | geverifieerd %s | %s\n", "GAME RAW          : %s enregistrements | vérifiés %s | %s\n", "GAME RAW          : %s registros | verificados %s | %s\n", "GAME RAW          ：%s 记录 | 已验证 %s | %s\n", "GAME RAW          : %s записей | проверено %s | %s\n"), fmtInt(c.GamesAccepted), fmtInt(c.RawGamesVerified), relOutput(layout, layout.RawGamesFile))
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
		if t.Kind == KindCTG {
			fmt.Fprintf(&b, L("    RAW produced      : %s | verified %s\n", "    RAW erzeugt       : %s | verifiziert %s\n", "    RAW geproduceerd  : %s | geverifieerd %s\n", "    RAW produit       : %s | vérifié %s\n", "    RAW producido     : %s | verificado %s\n", "    已生成 RAW        ：%s | 已验证 %s\n", "    RAW создано       : %s | проверено %s\n"), fmtInt(t.RawAccepted), fmtInt(t.RawVerified))
		} else {
			fmt.Fprintf(&b, L("    RAW seen          : %s\n", "    RAW gesehen       : %s\n", "    RAW gezien        : %s\n", "    RAW vus           : %s\n", "    RAW vistos        : %s\n", "    已查看 RAW        ：%s\n", "    RAW просмотрено   : %s\n"), fmtInt(t.RawSeen))
			fmt.Fprintf(&b, L("    RAW per source    : %s | local dup %s | cross-source dup %s | rejected %s | verified %s\n", "    RAW pro Quelle    : %s | lokale Dup %s | quellenübergreifende Dup %s | abgelehnt %s | verifiziert %s\n", "    RAW per bron      : %s | lokale dup %s | cross-source dup %s | afgewezen %s | geverifieerd %s\n", "    RAW par source    : %s | doublon local %s | doublon inter-source %s | rejeté %s | vérifié %s\n", "    RAW por fuente    : %s | duplicado local %s | duplicado entre fuentes %s | rechazado %s | verificado %s\n", "    每源 RAW          ：%s | 本地重复 %s | 跨源重复 %s | 已拒绝 %s | 已验证 %s\n", "    RAW по источнику  : %s | лок. дубликаты %s | межисточн. дубликаты %s | отклонено %s | проверено %s\n"), fmtInt(t.RawAccepted), fmtInt(t.RawLocalDuplicate), fmtInt(t.RawCrossDuplicate), fmtInt(t.RawRejected), fmtInt(t.RawVerified))
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
		L("- A technically readable but statistically insufficient CTG result is SKIPPED, not FAILED.", "- Ein technisch korrekt lesbares, aber statistisch unzureichendes CTG-Ergebnis wird ÜBERSPRUNGEN, nicht als FEHLGESCHLAGEN markiert.", "- Een technisch correct gelezen maar statistisch onvoldoende CTG-resultaat krijgt status OVERGESLAGEN, niet MISLUKT.", "- Un résultat CTG techniquement lisible mais statistiquement insuffisant est IGNORÉ, et non en ÉCHEC.", "- Un resultado CTG técnicamente legible pero estadísticamente insuficiente se marca OMITIDO, no FALLIDO.", "- 技术上可读取但统计不足的 CTG 结果会标记为“已跳过”，而不是“失败”。", "- Технически читаемый, но статистически недостаточный CTG-результат получает статус ПРОПУЩЕНО, а не ОШИБКА."),
		L("- The conservative CTG fallback runs only when strict diagnostics show real statistical potential and is reported visibly.", "- Der vorsichtige CTG-Fallback läuft nur, wenn die strikte Diagnose echtes statistisches Potenzial zeigt, und wird sichtbar gemeldet.", "- De voorzichtige CTG-fallback wordt alleen gestart als de strikte diagnostiek daarvoor reëel statistisch potentieel toont en wordt zichtbaar gemeld.", "- Le repli CTG prudent n’est lancé que si le diagnostic strict montre un potentiel statistique réel et il est signalé clairement.", "- La alternativa CTG prudente solo se ejecuta si el diagnóstico estricto muestra potencial estadístico real y se informa de forma visible.", "- 谨慎的 CTG 后备模式只有在严格诊断显示真实统计潜力时才运行，并会明确报告。", "- Осторожный резерв CTG запускается только при реальном статистическом потенциале по строгой диагностике и явно отображается."),
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
