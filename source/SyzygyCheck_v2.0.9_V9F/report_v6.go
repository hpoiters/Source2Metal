package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type reportText struct {
	basedOn, around                               string
	language, folder, started, finished, duration string
	filesFound, filesKnown, totalData, workers    string
	checksumFails, processErrors, resumed         string
	verificationCore, batching                    string
	resultReal                                    string
	diagnosticHeading, diagnosticPassed           string
	diagnosticTemporary, diagnosticNotPassed      string
	diagnosticError                               string
	failureHeading, errorHeading, fatalHeading    string
	technicalHeading                              string
	recoveryHeading, recoveryReference            string
	yes, no                                       string
	pass, fail, errStatus, incomplete             string
}

func reportTexts(lang string) reportText {
	switch lang {
	case "nl":
		return reportText{
			basedOn:  "Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man.",
			around:   "AL-interface, voortgangsweergave, zelftest, checkpointing en rapportage zijn daaromheen gebouwd.",
			language: "Taal", folder: "Map", started: "Gestart", finished: "Beëindigd", duration: "Duur van deze run",
			filesFound: "Bestanden gevonden", filesKnown: "Bestanden gecontroleerd/bekend", totalData: "Totale data", workers: "Werkers (threads)",
			checksumFails: "Checksumfouten (FAIL)", processErrors: "Lees-/procesproblemen (ERROR)", resumed: "Hervat vanaf checkpoint",
			verificationCore:    "Verificatiekern: Ronald de Man tbcheck; checksumlogica is niet opnieuw geïmplementeerd door AL.",
			batching:            "AL-batching: maximaal %d bestanden en doel %s per tbcheck-proces.",
			resultReal:          "RESULTAAT (ECHTE SYZYGY-BESTANDEN)",
			diagnosticHeading:   "DIAGNOSTISCHE ONBOARD TEST-FAIL",
			diagnosticPassed:    "TEST-FAIL: zoals verwacht gedetecteerd — diagnostische zelftest GESLAAGD.",
			diagnosticTemporary: "De kleine onboard testfile was tijdelijk, opzettelijk en vóór de echte tablebase-inventarisatie verwijderd.",
			diagnosticNotPassed: "DIAGNOSTISCHE ZELFTEST: NIET GESLAAGD. De echte controle had niet mogen starten.",
			diagnosticError:     "Diagnostische fout",
			failureHeading:      "CHECKSUMFOUTEN", errorHeading: "OPERATIONELE FOUTEN (geen checksum-FAILs)", fatalHeading: "FATALE ENGINEFOUT",
			technicalHeading:  "TECHNISCHE OPMERKINGEN (alleen uitzonderingen)",
			recoveryHeading:   "HERSTEL / VERVANGENDE SYZYGY-BESTANDEN",
			recoveryReference: "Voor herstel en betrouwbare downloadbronnen: zie %s in de programmamap.",
			yes:               "ja", no: "nee", pass: "GESLAAGD", fail: "FOUT", errStatus: "ERROR", incomplete: "ONVOLTOOID",
		}
	case "de":
		return reportText{
			basedOn:  "Basierend auf dem ursprünglichen Syzygy-Tablebase-Prüfcode von Ronald de Man.",
			around:   "AL-Oberfläche, Fortschrittsanzeige, Selbsttest, Checkpointing und Bericht wurden darum herum ergänzt.",
			language: "Sprache", folder: "Ordner", started: "Gestartet", finished: "Beendet", duration: "Dauer dieses Laufs",
			filesFound: "Gefundene Dateien", filesKnown: "Geprüfte/bekannte Dateien", totalData: "Gesamtdaten", workers: "Worker (Threads)",
			checksumFails: "Prüfsummenfehler (FAIL)", processErrors: "Lese-/Prozessprobleme (ERROR)", resumed: "Vom Checkpoint fortgesetzt",
			verificationCore:    "Prüfkern: Ronald de Man tbcheck; die Prüfsummenlogik wurde von AL nicht neu implementiert.",
			batching:            "AL-Batching: maximal %d Dateien und Ziel %s pro tbcheck-Prozess.",
			resultReal:          "ERGEBNIS (ECHTE SYZYGY-DATEIEN)",
			diagnosticHeading:   "DIAGNOSTISCHER INTERNER TEST-FEHLER",
			diagnosticPassed:    "TEST-FEHLER: wie erwartet erkannt — diagnostischer Selbsttest BESTANDEN.",
			diagnosticTemporary: "Die kleine interne Testdatei war temporär, absichtlich fehlerhaft und wurde vor der echten Tablebase-Inventur entfernt.",
			diagnosticNotPassed: "DIAGNOSTISCHER SELBSTTEST: NICHT BESTANDEN. Die echte Prüfung hätte nicht starten dürfen.",
			diagnosticError:     "Diagnostischer Fehler",
			failureHeading:      "PRÜFSUMMENFEHLER", errorHeading: "BETRIEBSFEHLER (keine Prüfsummen-FAILs)", fatalHeading: "FATALER ENGINE-FEHLER",
			technicalHeading:  "TECHNISCHE HINWEISE (nur Ausnahmen)",
			recoveryHeading:   "WIEDERHERSTELLUNG / ERSATZDATEIEN",
			recoveryReference: "Hinweise zur Wiederherstellung und verlässliche Downloadquellen: siehe %s im Programmordner.",
			yes:               "ja", no: "nein", pass: "BESTANDEN", fail: "FEHLER", errStatus: "ERROR", incomplete: "UNVOLLSTÄNDIG",
		}
	case "fr":
		return reportText{
			basedOn:  "Basé sur le code de vérification original des tablebases Syzygy de Ronald de Man.",
			around:   "L'interface AL, l'affichage de progression, l'auto-test, les points de reprise et le rapport sont construits autour de ce noyau.",
			language: "Langue", folder: "Dossier", started: "Démarré", finished: "Terminé", duration: "Durée de cette exécution",
			filesFound: "Fichiers trouvés", filesKnown: "Fichiers vérifiés/connus", totalData: "Données totales", workers: "Travailleurs (threads)",
			checksumFails: "Erreurs de somme de contrôle (FAIL)", processErrors: "Problèmes de lecture/processus (ERROR)", resumed: "Repris depuis un point de contrôle",
			verificationCore:    "Noyau de vérification : tbcheck de Ronald de Man ; la logique de somme de contrôle n'a pas été réimplémentée par AL.",
			batching:            "Traitement AL : au maximum %d fichiers et cible de %s par processus tbcheck.",
			resultReal:          "RÉSULTAT (FICHIERS SYZYGY RÉELS)",
			diagnosticHeading:   "ÉCHEC DE TEST INTERNE DE DIAGNOSTIC",
			diagnosticPassed:    "ÉCHEC DE TEST : détecté comme prévu — auto-test de diagnostic RÉUSSI.",
			diagnosticTemporary: "Le petit fichier de test interne était temporaire, volontairement erroné et a été supprimé avant l'inventaire réel des tablebases.",
			diagnosticNotPassed: "AUTO-TEST DE DIAGNOSTIC : ÉCHEC. Le contrôle réel n'aurait pas dû démarrer.",
			diagnosticError:     "Erreur de diagnostic",
			failureHeading:      "ERREURS DE SOMME DE CONTRÔLE", errorHeading: "ERREURS OPÉRATIONNELLES (pas des FAIL de somme de contrôle)", fatalHeading: "ERREUR FATALE DU MOTEUR",
			technicalHeading:  "REMARQUES TECHNIQUES (exceptions uniquement)",
			recoveryHeading:   "RÉCUPÉRATION / FICHIERS SYZYGY DE REMPLACEMENT",
			recoveryReference: "Pour la récupération et des sources de téléchargement fiables : voir %s dans le dossier du programme.",
			yes:               "oui", no: "non", pass: "RÉUSSI", fail: "ÉCHEC", errStatus: "ERREUR", incomplete: "INCOMPLET",
		}
	case "es":
		return reportText{
			basedOn:  "Basado en el código original de verificación de tablebases Syzygy de Ronald de Man.",
			around:   "La interfaz AL, el progreso, la autoprueba, los puntos de reanudación y el informe están construidos alrededor de ese núcleo.",
			language: "Idioma", folder: "Carpeta", started: "Iniciado", finished: "Finalizado", duration: "Duración de esta ejecución",
			filesFound: "Archivos encontrados", filesKnown: "Archivos comprobados/conocidos", totalData: "Datos totales", workers: "Trabajadores (threads)",
			checksumFails: "Errores de suma de comprobación (FAIL)", processErrors: "Problemas de lectura/proceso (ERROR)", resumed: "Reanudado desde checkpoint",
			verificationCore:    "Núcleo de verificación: tbcheck de Ronald de Man; AL no reimplementa la lógica de suma de comprobación.",
			batching:            "Lotes AL: máximo %d archivos y objetivo de %s por proceso tbcheck.",
			resultReal:          "RESULTADO (ARCHIVOS SYZYGY REALES)",
			diagnosticHeading:   "FALLO DE PRUEBA INTERNO DE DIAGNÓSTICO",
			diagnosticPassed:    "FALLO DE PRUEBA: detectado como se esperaba — autoprueba de diagnóstico SUPERADA.",
			diagnosticTemporary: "El pequeño archivo interno de prueba fue temporal, intencional y se eliminó antes del inventario real de tablebases.",
			diagnosticNotPassed: "AUTOPRUEBA DE DIAGNÓSTICO: NO SUPERADA. La comprobación real no debería haber empezado.",
			diagnosticError:     "Error de diagnóstico",
			failureHeading:      "ERRORES DE SUMA DE COMPROBACIÓN", errorHeading: "ERRORES OPERATIVOS (no son FAIL de suma de comprobación)", fatalHeading: "ERROR FATAL DEL MOTOR",
			technicalHeading:  "NOTAS TÉCNICAS (solo excepciones)",
			recoveryHeading:   "RECUPERACIÓN / ARCHIVOS SYZYGY DE REEMPLAZO",
			recoveryReference: "Para recuperación y fuentes de descarga fiables: consulte %s en la carpeta del programa.",
			yes:               "sí", no: "no", pass: "SUPERADO", fail: "FALLO", errStatus: "ERROR", incomplete: "INCOMPLETO",
		}
	case "zh":
		return reportText{
			basedOn:  "基于 Ronald de Man 原始的 Syzygy 表库验证代码。",
			around:   "AL 多语言界面、进度显示、自检、检查点和报告功能构建在该验证核心外围。",
			language: "语言", folder: "文件夹", started: "开始时间", finished: "结束时间", duration: "本次运行时长",
			filesFound: "找到的文件", filesKnown: "已检查/已知文件", totalData: "总数据量", workers: "工作线程 (threads)",
			checksumFails: "校验和不匹配 (FAIL)", processErrors: "读取/进程问题 (ERROR)", resumed: "从检查点继续",
			verificationCore:    "验证核心：Ronald de Man 的 tbcheck；AL 未重新实现校验和逻辑。",
			batching:            "AL 批处理：每个 tbcheck 进程最多 %d 个文件，目标数据量 %s。",
			resultReal:          "结果（真实 SYZYGY 文件）",
			diagnosticHeading:   "内置诊断测试失败",
			diagnosticPassed:    "测试失败：按预期检测到 — 诊断自检通过。",
			diagnosticTemporary: "内置小型测试文件是临时且故意损坏的，并在真实 tablebase 清单生成前删除。",
			diagnosticNotPassed: "诊断自检：未通过。真实检查不应启动。",
			diagnosticError:     "诊断错误",
			failureHeading:      "校验和错误", errorHeading: "运行错误（不是校验和 FAIL）", fatalHeading: "致命引擎错误",
			technicalHeading:  "技术说明（仅异常情况）",
			recoveryHeading:   "恢复 / 替换 SYZYGY 文件",
			recoveryReference: "有关恢复方法和可靠下载来源，请查看程序文件夹中的 %s。",
			yes:               "是", no: "否", pass: "通过", fail: "失败", errStatus: "错误", incomplete: "未完成",
		}
	case "ru":
		return reportText{
			basedOn:  "Основано на оригинальном коде проверки таблиц Syzygy Ronald de Man.",
			around:   "Интерфейс AL, индикация прогресса, самотест, checkpoint и отчётность построены вокруг этого проверочного ядра.",
			language: "Язык", folder: "Папка", started: "Начало", finished: "Окончание", duration: "Длительность этого запуска",
			filesFound: "Найдено файлов", filesKnown: "Проверено/известно файлов", totalData: "Объём данных", workers: "Рабочие потоки (threads)",
			checksumFails: "Несовпадения контрольной суммы (FAIL)", processErrors: "Проблемы чтения/процесса (ERROR)", resumed: "Продолжено с checkpoint",
			verificationCore:    "Ядро проверки: tbcheck Ronald de Man; AL не реализует заново логику контрольной суммы.",
			batching:            "Пакеты AL: до %d файлов и целевой объём %s на процесс tbcheck.",
			resultReal:          "РЕЗУЛЬТАТ (РЕАЛЬНЫЕ ФАЙЛЫ SYZYGY)",
			diagnosticHeading:   "ДИАГНОСТИЧЕСКИЙ ВСТРОЕННЫЙ ТЕСТОВЫЙ СБОЙ",
			diagnosticPassed:    "ТЕСТОВЫЙ СБОЙ: обнаружен как ожидалось — диагностический самотест ПРОЙДЕН.",
			diagnosticTemporary: "Маленький встроенный тестовый файл был временным, намеренно повреждённым и удалён до инвентаризации настоящих tablebase.",
			diagnosticNotPassed: "ДИАГНОСТИЧЕСКИЙ САМОТЕСТ: НЕ ПРОЙДЕН. Настоящая проверка не должна была запускаться.",
			diagnosticError:     "Диагностическая ошибка",
			failureHeading:      "ОШИБКИ КОНТРОЛЬНОЙ СУММЫ", errorHeading: "ОПЕРАЦИОННЫЕ ОШИБКИ (не checksum FAIL)", fatalHeading: "ФАТАЛЬНАЯ ОШИБКА ДВИЖКА",
			technicalHeading:  "ТЕХНИЧЕСКИЕ ПРИМЕЧАНИЯ (только исключения)",
			recoveryHeading:   "ВОССТАНОВЛЕНИЕ / ЗАМЕНА ФАЙЛОВ SYZYGY",
			recoveryReference: "Для восстановления и надёжных источников загрузки см. %s в папке программы.",
			yes:               "да", no: "нет", pass: "ПРОЙДЕНО", fail: "СБОЙ", errStatus: "ОШИБКА", incomplete: "НЕ ЗАВЕРШЕНО",
		}
	default:
		return reportText{
			basedOn:  "Based on the original Syzygy tablebase verification code by Ronald de Man.",
			around:   "AL interface, progress display, self-test, checkpointing and reporting are built around that verification core.",
			language: "Language", folder: "Folder", started: "Started", finished: "Finished", duration: "Duration this run",
			filesFound: "Files found", filesKnown: "Files processed/known", totalData: "Total data", workers: "Workers (threads)",
			checksumFails: "Checksum mismatches (FAIL)", processErrors: "Read/process problems (ERROR)", resumed: "Resumed from checkpoint",
			verificationCore:    "Verification core: Ronald de Man tbcheck; checksum logic not reimplemented by AL.",
			batching:            "AL batching: up to %d files and target %s per tbcheck process.",
			resultReal:          "RESULT (REAL SYZYGY FILES)",
			diagnosticHeading:   "DIAGNOSTIC ONBOARD TEST-FAIL",
			diagnosticPassed:    "TEST-FAIL: detected as expected — diagnostic self-test PASSED.",
			diagnosticTemporary: "The tiny onboard test file was temporary, intentional and removed before the real tablebase inventory.",
			diagnosticNotPassed: "DIAGNOSTIC SELF-TEST: NOT PASSED. Real checking should not have started.",
			diagnosticError:     "Diagnostic error",
			failureHeading:      "CHECKSUM FAILURES", errorHeading: "OPERATIONAL ERRORS (not checksum FAILs)", fatalHeading: "FATAL ENGINE ERROR",
			technicalHeading:  "TECHNICAL NOTES (exceptions only)",
			recoveryHeading:   "RECOVERY / REPLACEMENT SYZYGY FILES",
			recoveryReference: "For recovery steps and reliable download sources, see %s in the program folder.",
			yes:               "yes", no: "no", pass: "PASS", fail: "FAIL", errStatus: "ERROR", incomplete: "INCOMPLETE",
		}
	}
}

func averageReadSpeedLabel(lang string) string {
	switch lang {
	case "nl":
		return "Gemiddelde leessnelheid"
	case "de":
		return "Durchschnittliche Lesegeschwindigkeit"
	case "fr":
		return "Vitesse moyenne de lecture"
	case "es":
		return "Velocidad media de lectura"
	case "zh":
		return "平均读取速度"
	case "ru":
		return "Средняя скорость чтения"
	default:
		return "Average read speed"
	}
}

func recoveryGuideFilename(lang string) string {
	switch lang {
	case "nl":
		return "SYZYGY_HERSTEL_NL.txt"
	case "de":
		return "SYZYGY_WIEDERHERSTELLUNG_DE.txt"
	case "fr":
		return "SYZYGY_RECUPERATION_FR.txt"
	case "es":
		return "SYZYGY_RECUPERACION_ES.txt"
	case "zh":
		return "SYZYGY_RECOVERY_ZH.txt"
	case "ru":
		return "SYZYGY_RECOVERY_RU.txt"
	default:
		return "SYZYGY_RECOVERY_EN.txt"
	}
}

func localizedBool(v bool, t reportText) string {
	if v {
		return t.yes
	}
	return t.no
}

func reportStatus(t reportText, files []item, scan realScanResult) string {
	if scan.FatalErr != nil || scan.CompletedFiles != len(files) {
		return t.incomplete
	}
	if len(scan.Errors) > 0 {
		return t.errStatus
	}
	if len(scan.Failures) > 0 {
		return t.fail
	}
	return t.pass
}

// compactTechnicalNotes removes normal per-file OK!/FAIL! output from the
// user-facing report. Those results remain in checkpoint state when needed for
// resume, but a 3022-file tablebase should not produce a 3022-line report.
func compactTechnicalNotes(lang string, messages []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, block := range messages {
		for _, raw := range strings.Split(strings.ReplaceAll(block, "\r\n", "\n"), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasSuffix(line, ": OK!") || strings.HasSuffix(line, ": FAIL!") {
				continue
			}
			if strings.HasPrefix(line, "tbcheck origin: ") {
				continue // SHA-256 + verification-core line already identify the engine.
			}
			switch {
			case strings.HasPrefix(line, "Checkpoint disabled: "):
				detail := strings.TrimPrefix(line, "Checkpoint disabled: ")
				switch lang {
				case "nl":
					line = "Checkpoint uitgeschakeld: " + detail
				case "de":
					line = "Checkpoint deaktiviert: " + detail
				case "fr":
					line = "Point de reprise désactivé : " + detail
				case "es":
					line = "Checkpoint desactivado: " + detail
				case "zh":
					line = "检查点已禁用：" + detail
				case "ru":
					line = "Checkpoint отключён: " + detail
				}
			case strings.HasPrefix(line, "Checkpoint write warning: "):
				detail := strings.TrimPrefix(line, "Checkpoint write warning: ")
				switch lang {
				case "nl":
					line = "Waarschuwing bij schrijven checkpoint: " + detail
				case "de":
					line = "Warnung beim Schreiben des Checkpoints: " + detail
				case "fr":
					line = "Avertissement lors de l'écriture du point de reprise : " + detail
				case "es":
					line = "Advertencia al escribir el checkpoint: " + detail
				case "zh":
					line = "写入检查点警告：" + detail
				case "ru":
					line = "Предупреждение записи checkpoint: " + detail
				}
			}
			if !seen[line] {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	return out
}

func uniqueReportPath(folder string, ended time.Time) (string, *os.File, error) {
	base := "SyzygyCheck_Result_" + ended.Format("20060102_1504")
	for n := 1; n < 10000; n++ {
		name := base + ".txt"
		if n > 1 {
			name = fmt.Sprintf("%s_%d.txt", base, n)
		}
		p := filepath.Join(folder, name)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			return p, f, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return "", nil, err
	}
	return "", nil, fmt.Errorf("could not create a unique result report name")
}

func writeReportData(folder string, ended time.Time, data []byte) (string, error) {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", err
	}
	p, f, err := uniqueReportPath(folder, ended)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(p)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	if err := f.Sync(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	ok = true
	return p, nil
}

func writeReportV6(lang, dir string, files []item, totalBytes int64, scan realScanResult, testResult intentionalTestResult) (string, error) {
	t := reportTexts(lang)
	var b strings.Builder
	fmt.Fprintf(&b, "SyzygyCheck v%s\r\n", version)
	fmt.Fprintf(&b, "%s\r\n", t.basedOn)
	fmt.Fprintf(&b, "%s\r\n\r\n", t.around)
	fmt.Fprintf(&b, "%s: %s\r\n", t.language, displayFor(lang))
	fmt.Fprintf(&b, "%s: %s\r\n", t.folder, dir)
	fmt.Fprintf(&b, "%s: %s\r\n", t.started, scan.Started.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "%s: %s\r\n", t.finished, scan.Ended.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "%s: %s    %s: %s\r\n", t.duration, fmtDuration(scan.Ended.Sub(scan.Started)), averageReadSpeedLabel(lang), formatReadSpeedValue(scan.AverageReadBps))
	fmt.Fprintf(&b, "%s: %d\r\n", t.filesFound, len(files))
	fmt.Fprintf(&b, "%s: %d\r\n", t.filesKnown, scan.CompletedFiles)
	fmt.Fprintf(&b, "%s: %s\r\n", t.totalData, humanSize(totalBytes))
	if scan.Workers > 0 {
		fmt.Fprintf(&b, "%s: %d\r\n", t.workers, scan.Workers)
	}
	fmt.Fprintf(&b, "%s: %d\r\n", t.checksumFails, len(scan.Failures))
	fmt.Fprintf(&b, "%s: %d\r\n", t.processErrors, len(scan.Errors))
	fmt.Fprintf(&b, "%s: %s\r\n", t.resumed, localizedBool(scan.Resumed, t))
	fmt.Fprintf(&b, "%s\r\n", t.verificationCore)
	fmt.Fprintf(&b, "tbcheck.exe SHA-256: %s\r\n", expectedTBCheckSHA256)
	fmt.Fprintf(&b, t.batching+"\r\n\r\n", maxBatchFiles, humanSize(targetBatchBytes))
	fmt.Fprintf(&b, "%s: %s\r\n", t.resultReal, reportStatus(t, files, scan))

	fmt.Fprintf(&b, "\r\n%s:\r\n", t.diagnosticHeading)
	if testResult.ExpectedFailDetected {
		fmt.Fprintf(&b, "%s\r\n", t.diagnosticPassed)
		fmt.Fprintf(&b, "%s\r\n", t.diagnosticTemporary)
		fmt.Fprintf(&b, "%s\r\n", selfTestFilesUntouchedLine(lang))
	} else {
		fmt.Fprintf(&b, "%s\r\n", t.diagnosticNotPassed)
		if testResult.Err != nil {
			fmt.Fprintf(&b, "%s: %v\r\n", t.diagnosticError, testResult.Err)
		}
	}

	if len(scan.Failures) > 0 {
		fmt.Fprintf(&b, "\r\n%s:\r\n", t.failureHeading)
		for _, f := range scan.Failures {
			fmt.Fprintf(&b, "- %s\r\n", f)
		}
	}
	if len(scan.Errors) > 0 {
		fmt.Fprintf(&b, "\r\n%s:\r\n", t.errorHeading)
		names := make([]string, 0, len(scan.Errors))
		for n := range scan.Errors {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "- %s: %s\r\n", n, scan.Errors[n])
		}
	}
	if scan.FatalErr != nil {
		fmt.Fprintf(&b, "\r\n%s:\r\n%v\r\n", t.fatalHeading, scan.FatalErr)
	}
	if len(scan.Failures) > 0 || len(scan.Errors) > 0 {
		fmt.Fprintf(&b, "\r\n%s:\r\n", t.recoveryHeading)
		fmt.Fprintf(&b, t.recoveryReference+"\r\n", recoveryGuideFilename(lang))
	}
	if notes := compactTechnicalNotes(lang, scan.EngineMessages); len(notes) > 0 {
		fmt.Fprintf(&b, "\r\n%s:\r\n", t.technicalHeading)
		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\r\n", n)
		}
	}

	data := []byte(b.String())
	if desktop, err := desktopPath(); err == nil {
		if p, err := writeReportData(desktop, scan.Ended, data); err == nil {
			return p, nil
		}
	}
	// Known Folder resolution or Desktop writing can fail on redirected or
	// locked-down Windows profiles. Never lose the report: fall back to EXE dir.
	return writeReportData(dir, scan.Ended, data)
}

func writeSelfTestFailureReportV6(lang, dir, helperOrigin string, testResult intentionalTestResult) (string, error) {
	now := time.Now()
	scan := realScanResult{
		Started: now, Ended: now, Errors: map[string]string{},
		FatalErr:       fmt.Errorf("diagnostic self-test failed; real Syzygy check was not started"),
		EngineMessages: []string{"tbcheck origin: " + helperOrigin},
	}
	return writeReportV6(lang, dir, nil, 0, scan, testResult)
}
