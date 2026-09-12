package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type realScanResult struct {
	Started        time.Time
	Ended          time.Time
	CompletedFiles int
	CompletedBytes int64
	Failures       []string
	Errors         map[string]string
	EngineMessages []string
	FatalErr       error
	Resumed        bool
	CheckpointUsed bool
	AverageReadBps float64
	Workers        int
	Recursive      bool
	Cancelled      bool
}

func resumeCheckpointPrompt(lang string, done, total int, bytesDone, totalBytes int64) string {
	switch lang {
	case "nl":
		return fmt.Sprintf("Vorige onvoltooide controle gevonden: %d/%d bestanden, %s/%s.\nEnter = hervatten, N = opnieuw beginnen, Esc = terug: ", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "de":
		return fmt.Sprintf("Unvollständige vorherige Prüfung gefunden: %d/%d Dateien, %s/%s.\nEnter = fortsetzen, N = neu beginnen, Esc = zurück: ", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "fr":
		return fmt.Sprintf("Contrôle précédent inachevé trouvé : %d/%d fichiers, %s/%s.\nEntrée = reprendre, N = recommencer, Échap = retour : ", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "es":
		return fmt.Sprintf("Se encontró una comprobación anterior incompleta: %d/%d archivos, %s/%s.\nEnter = continuar, N = empezar de nuevo, Esc = volver: ", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "zh":
		return fmt.Sprintf("发现上次未完成的检查：%d/%d 个文件，%s/%s。\nEnter = 继续，N = 重新开始，Esc = 返回：", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "ru":
		return fmt.Sprintf("Найдена незавершённая предыдущая проверка: %d/%d файлов, %s/%s.\nEnter = продолжить, N = начать заново, Esc = назад: ", done, total, humanSize(bytesDone), humanSize(totalBytes))
	default:
		return fmt.Sprintf("Previous incomplete check found: %d/%d files, %s/%s.\nEnter = resume, N = start again, Esc = back: ", done, total, humanSize(bytesDone), humanSize(totalBytes))
	}
}

func checkpointMismatchLine(lang string) string {
	switch lang {
	case "nl":
		return "Oud checkpoint past niet bij de huidige bestanden en wordt genegeerd."
	case "de":
		return "Alter Prüfpunkt passt nicht zu den aktuellen Dateien und wird ignoriert."
	case "fr":
		return "L'ancien point de reprise ne correspond pas aux fichiers actuels et sera ignoré."
	case "es":
		return "El punto de reanudación anterior no coincide con los archivos actuales y se ignora."
	case "zh":
		return "旧检查点与当前文件不匹配，已忽略。"
	case "ru":
		return "Старый checkpoint не соответствует текущим файлам и будет проигнорирован."
	default:
		return "Old checkpoint does not match the current files and is ignored."
	}
}

func perFileErrorLine(lang, name, detail string) string {
	switch lang {
	case "nl":
		return fmt.Sprintf("FOUT: bestand kon niet betrouwbaar worden gecontroleerd - %s (%s)", name, detail)
	case "de":
		return fmt.Sprintf("FEHLER: Datei konnte nicht zuverlässig geprüft werden - %s (%s)", name, detail)
	case "fr":
		return fmt.Sprintf("ERREUR : le fichier n'a pas pu être vérifié de façon fiable - %s (%s)", name, detail)
	case "es":
		return fmt.Sprintf("ERROR: el archivo no pudo comprobarse de forma fiable - %s (%s)", name, detail)
	case "zh":
		return fmt.Sprintf("错误：文件无法可靠校验 - %s (%s)", name, detail)
	case "ru":
		return fmt.Sprintf("ОШИБКА: файл не удалось надёжно проверить - %s (%s)", name, detail)
	default:
		return fmt.Sprintf("ERROR: file could not be reliably checked - %s (%s)", name, detail)
	}
}

func runRealScan(r *bufio.Reader, lang, helper, helperOrigin, dir string, files []item, totalBytes int64, workers int, recursive bool) realScanResult {
	res := realScanResult{Started: time.Now(), Errors: map[string]string{}, EngineMessages: []string{"tbcheck origin: " + helperOrigin}, Workers: workers, Recursive: recursive}
	restoreSleep := preventSystemSleep()
	defer restoreSleep()

	hash, hashErr := inventoryHash(dir, files)
	cp := newCheckpoint("", len(files), totalBytes)
	cpEnabled := hashErr == nil
	if cpEnabled {
		cp.InventoryHash = hash
		if old, ok := loadMatchingCheckpoint(dir, hash, len(files), totalBytes); ok {
			done, bytesDone, _, _ := checkpointSummary(old, files)
			if done > 0 {
				fmt.Print(resumeCheckpointPrompt(lang, done, len(files), bytesDone, totalBytes))
				choice, escaped := readMenuInput(r)
				if escaped {
					res.Cancelled = true
					res.Ended = time.Now()
					return res
				}
				choice = strings.ToLower(choice)
				if choice != "n" {
					cp = old
					res.Resumed = true
					res.CheckpointUsed = true
				} else {
					removeCheckpoint(dir)
				}
			}
		} else {
			// If a checkpoint file exists but does not match this inventory,
			// do not silently trust it. Remove it and make the mismatch visible.
			if _, err := osStat(checkpointPath(dir)); err == nil {
				fmt.Println(checkpointMismatchLine(lang))
				removeCheckpoint(dir)
			}
		}
	} else {
		res.EngineMessages = append(res.EngineMessages, "Checkpoint disabled: "+hashErr.Error())
	}

	completedFiles, completedBytes, failures, _ := checkpointSummary(cp, files)
	res.CompletedFiles, res.CompletedBytes, res.Failures = completedFiles, completedBytes, append([]string(nil), failures...)
	pending := checkpointPending(cp, files)
	initialSessionBytes := completedBytes

	eventLines := []string{}
	if res.Resumed {
		eventLines = append(eventLines, resumedLine(lang, completedFiles, len(files), completedBytes, totalBytes))
	}
	redrawScanScreen(lang, dir, len(files), workers, recursive, eventLines)

	if len(pending) == 0 {
		res.Ended = time.Now()
		return res
	}

	// While the long real scan is active, disable only the console window's
	// Close command. This prevents an accidental click on X (and the equivalent
	// system Close command) from killing a good run. Normal window controls such
	// as minimize, maximize, resize and scroll remain available.
	restoreConsoleClose := protectConsoleCloseDuringScan()
	defer restoreConsoleClose()

	batches := makeBatchesBySize(pending, maxBatchFiles, targetBatchBytes)
	width := visibleConsoleWidth()
	spinner := 0
	renderer := &progressBlockRenderer{}
	ordinalByName := make(map[string]int, len(files))
	for i, f := range files {
		ordinalByName[strings.ToLower(f.name)] = i + 1
	}
	currentOrdinal := ordinalByName[strings.ToLower(batches[0][0].name)]
	throughputStarted := time.Now()
	renderer.render(formatProgressBlockV11(lang, width, currentOrdinal, len(files), res.CompletedBytes, res.CompletedBytes, totalBytes, throughputStarted, initialSessionBytes, spinner, false))
	spinner++
	cpWarned := false

	record := func(f item, status, detail string) {
		switch status {
		case "FAIL":
			res.Failures = append(res.Failures, f.name)
		case "ERROR":
			res.Errors[f.name] = detail
		}
		res.CompletedFiles++
		res.CompletedBytes += f.size
		cp.Results[f.name] = checkpointEntry{Status: status, Size: f.size, Detail: detail}
		if cpEnabled {
			if err := saveCheckpoint(dir, cp); err != nil && !cpWarned {
				res.EngineMessages = append(res.EngineMessages, "Checkpoint write warning: "+err.Error())
				cpWarned = true
				cpEnabled = false
			}
		}
	}

	for bi, batch := range batches {
		batchBytes := sumItemBytes(batch)
		names := make([]string, 0, len(batch))
		for _, f := range batch {
			names = append(names, f.name)
		}
		args := tbcheckArgs(workers, names...)
		cmd := exec.Command(helper, args...)
		cmd.Dir = dir
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Start(); err != nil {
			res.FatalErr = fmt.Errorf("tbcheck could not start: %w", err)
			break
		}

		baseRead, ioCounterOK := processReadBytes(cmd.Process.Pid)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		ticker := time.NewTicker(250 * time.Millisecond)
		running := true
		var runErr error
		liveBytes := res.CompletedBytes
		liveFromIO := false
		for running {
			select {
			case runErr = <-done:
				running = false
			case <-ticker.C:
				if nowRead, ok := processReadBytes(cmd.Process.Pid); ok {
					if !ioCounterOK {
						baseRead, ioCounterOK = nowRead, true
					} else if nowRead >= baseRead {
						delta := int64(nowRead - baseRead)
						if delta > batchBytes {
							delta = batchBytes
						}
						if delta > 0 {
							liveFromIO = true
						}
						liveBytes = res.CompletedBytes + delta
					}
				}
				nw := visibleConsoleWidth()
				if nw != width {
					width = nw
					redrawScanScreen(lang, dir, len(files), workers, recursive, eventLines)
					renderer.resetAfterFullRedraw()
				}
				idx := estimateCurrentBatchIndex(batch, liveBytes-res.CompletedBytes)
				if idx >= 0 && idx < len(batch) {
					if ord := ordinalByName[strings.ToLower(batch[idx].name)]; ord > 0 {
						currentOrdinal = ord
					}
				}
				renderer.render(formatProgressBlockV11(lang, width, currentOrdinal, len(files), liveBytes, res.CompletedBytes, totalBytes, throughputStarted, initialSessionBytes, spinner, liveFromIO))
				spinner++
			}
		}
		ticker.Stop()

		checked, failed := parseTBCheckOutput(out.String())
		okSet, failSet := map[string]bool{}, map[string]bool{}
		for _, n := range checked {
			okSet[strings.ToLower(n)] = true
		}
		for _, n := range failed {
			failSet[strings.ToLower(n)] = true
		}
		if runErr != nil && strings.TrimSpace(out.String()) != "" {
			res.EngineMessages = append(res.EngineMessages, strings.TrimSpace(out.String()))
		}

		var unresolved []item
		var newEvents []string
		for _, f := range batch {
			k := strings.ToLower(f.name)
			switch {
			case failSet[k]:
				newEvents = append(newEvents, fmt.Sprintf(texts[lang].mismatch, f.name))
				record(f, "FAIL", "checksum mismatch")
			case okSet[k]:
				record(f, "OK", "")
			default:
				unresolved = append(unresolved, f)
			}
		}

		// A malformed/unreadable file can make Ronald's process exit before
		// later arguments are reached. Recheck only unresolved members one at
		// a time, so one bad file never hides the state of the rest.
		for _, f := range unresolved {
			status, detail, startErr := checkSingle(helper, dir, f.name, workers)
			if startErr != nil {
				res.FatalErr = startErr
				break
			}
			if status == "FAIL" {
				newEvents = append(newEvents, fmt.Sprintf(texts[lang].mismatch, f.name))
			} else if status == "ERROR" {
				newEvents = append(newEvents, perFileErrorLine(lang, f.name, detail))
			}
			record(f, status, detail)
		}
		if res.FatalErr != nil {
			break
		}

		// Real exceptions are placed above the fixed live zone and remain
		// visible for the rest of the run. Normal OK results never create lines.
		if len(newEvents) > 0 {
			renderer.clear()
			for _, line := range newEvents {
				fmt.Println(line)
				eventLines = append(eventLines, line)
			}
			// Keep exactly one blank separator between a visible exception and
			// the compact live progress block.
			fmt.Println()
		}
		if res.CompletedFiles >= len(files) {
			currentOrdinal = len(files)
		} else if bi+1 < len(batches) && len(batches[bi+1]) > 0 {
			currentOrdinal = ordinalByName[strings.ToLower(batches[bi+1][0].name)]
		}
		// Capture the same session-average throughput used by the live read-speed
		// line, so the final report can show a directly comparable value.
		res.AverageReadBps = averageReadRate(res.CompletedBytes, initialSessionBytes, time.Since(throughputStarted))
		renderer.render(formatProgressBlockV11(lang, width, currentOrdinal, len(files), res.CompletedBytes, res.CompletedBytes, totalBytes, throughputStarted, initialSessionBytes, spinner, false))
		spinner++

		// If an exception is first known in the last batch, the final screen
		// would otherwise replace it almost instantaneously. Keep that live event
		// visible briefly for human eyes. This happens only after verification is
		// complete and therefore cannot reduce checksum throughput.
		if bi == len(batches)-1 && len(newEvents) > 0 {
			time.Sleep(1200 * time.Millisecond)
		}
	}
	// The live renderer deliberately leaves the cursor at the end of its third
	// line while updating in place. Finish that line here. The report-destination
	// menu prints its own leading newline, which then becomes exactly one blank
	// line between the two visually distinct sections.
	renderer.finish()

	sort.Strings(res.Failures)
	res.Ended = time.Now()
	// A fully verified run (OK and/or checksum FAIL) no longer needs resume
	// state. Operational ERRORs remain in the checkpoint so a later run can
	// retry only those files.
	if res.FatalErr == nil && res.CompletedFiles == len(files) && len(res.Errors) == 0 {
		removeCheckpoint(dir)
	}
	return res
}

func checkSingle(helper, dir, name string, workers int) (status, detail string, startErr error) {
	cmd := exec.Command(helper, tbcheckArgs(workers, name)...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("tbcheck could not start: %w", err)
	}
	runErr := cmd.Wait()
	checked, failed := parseTBCheckOutput(out.String())
	for _, n := range failed {
		if strings.EqualFold(n, name) {
			return "FAIL", "checksum mismatch", nil
		}
	}
	for _, n := range checked {
		if strings.EqualFold(n, name) {
			return "OK", "", nil
		}
	}
	detail = strings.TrimSpace(out.String())
	if detail == "" && runErr != nil {
		detail = runErr.Error()
	}
	if detail == "" {
		detail = "tbcheck returned no OK!/FAIL! result"
	}
	return "ERROR", detail, nil
}

func redrawScanScreen(lang, dir string, totalFiles, workers int, recursive bool, eventLines []string) {
	clearConsoleScreen()
	drawRunHeader(lang, dir, totalFiles, workers, recursive)
	for _, line := range eventLines {
		fmt.Println(line)
	}
	if len(eventLines) > 0 {
		fmt.Println()
	}
}

func resumedLine(lang string, done, total int, bytesDone, totalBytes int64) string {
	switch lang {
	case "nl":
		return fmt.Sprintf("Controle hervat vanaf bevestigd checkpoint: %d/%d bestanden, %s/%s.", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "de":
		return fmt.Sprintf("Prüfung ab bestätigtem Checkpoint fortgesetzt: %d/%d Dateien, %s/%s.", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "fr":
		return fmt.Sprintf("Contrôle repris depuis le point confirmé : %d/%d fichiers, %s/%s.", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "es":
		return fmt.Sprintf("Comprobación reanudada desde el punto confirmado: %d/%d archivos, %s/%s.", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "zh":
		return fmt.Sprintf("从已确认检查点继续：%d/%d 个文件，%s/%s。", done, total, humanSize(bytesDone), humanSize(totalBytes))
	case "ru":
		return fmt.Sprintf("Проверка продолжена с подтверждённого checkpoint: %d/%d файлов, %s/%s.", done, total, humanSize(bytesDone), humanSize(totalBytes))
	default:
		return fmt.Sprintf("Check resumed from confirmed checkpoint: %d/%d files, %s/%s.", done, total, humanSize(bytesDone), humanSize(totalBytes))
	}
}
