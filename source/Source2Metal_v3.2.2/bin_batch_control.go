package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

const binNightRestDuration = 10 * time.Hour

type binCancelChoice int

const (
	binCancelSkip binCancelChoice = iota + 1
	binCancelNightRest
	binCancelStopBatch
)

func askBINCancelChoice(source string) binCancelChoice {
	fmt.Println()
	fmt.Println(L(
		"Current BIN source stopped safely:",
		"Aktuelle BIN-Quelle sicher gestoppt:",
		"Huidige BIN-bron veilig gestopt:",
		"Source BIN actuelle arrêtée en sécurité :",
		"Fuente BIN actual detenida de forma segura:",
		"当前 BIN 源已安全停止：",
		"Текущий BIN-источник безопасно остановлен:"), source)
	fmt.Println(L(
		"  1 = skip this source and continue with the rest",
		"  1 = diese Quelle überspringen und mit dem Rest fortfahren",
		"  1 = deze bron overslaan en doorgaan met de rest",
		"  1 = ignorer cette source et continuer avec le reste",
		"  1 = omitir esta fuente y continuar con el resto",
		"  1 = 跳过此源并继续处理其余源",
		"  1 = пропустить этот источник и продолжить с остальными"))
	fmt.Println(L(
		"  2 = night-rest pause for 10 hours, then retry this source automatically",
		"  2 = Nachtruhe-Pause für 10 Stunden, danach diese Quelle automatisch erneut starten",
		"  2 = nachtrustpauze van 10 uur, daarna deze bron automatisch opnieuw starten",
		"  2 = pause nocturne de 10 heures, puis relancer automatiquement cette source",
		"  2 = pausa nocturna de 10 horas y reintentar automáticamente esta fuente",
		"  2 = 休息 10 小时，然后自动重新处理此源",
		"  2 = пауза на ночной отдых 10 часов, затем автоматически повторить этот источник"))
	fmt.Println(L(
		"  3 = stop the complete batch",
		"  3 = kompletten Batch stoppen",
		"  3 = de hele batch stoppen",
		"  3 = arrêter tout le lot",
		"  3 = detener todo el lote",
		"  3 = 停止整个批处理",
		"  3 = остановить весь пакет"))
	for {
		fmt.Print(L("Choice [1-3] (Enter = 1): ", "Auswahl [1-3] (Enter = 1): ", "Keuze [1-3] (Enter = 1): ", "Choix [1-3] (Entrée = 1) : ", "Elección [1-3] (Enter = 1): ", "选择 [1-3]（回车 = 1）：", "Выбор [1-3] (Enter = 1): "))
		line, _ := stdin.ReadString('\n')
		switch strings.TrimSpace(line) {
		case "", "1":
			return binCancelSkip
		case "2":
			return binCancelNightRest
		case "3":
			return binCancelStopBatch
		}
	}
}

func waitBINNightRest(d time.Duration, source string) bool {
	resumeAt := time.Now().Add(d)
	fmt.Println()
	fmt.Printf(L(
		"Night-rest pause: %s. This source will restart automatically at about %s.\n",
		"Nachtruhe-Pause: %s. Diese Quelle startet ungefähr um %s automatisch neu.\n",
		"Nachtrustpauze: %s. Deze bron start rond %s automatisch opnieuw.\n",
		"Pause nocturne : %s. Cette source redémarrera automatiquement vers %s.\n",
		"Pausa nocturna: %s. Esta fuente se reiniciará automáticamente hacia las %s.\n",
		"休息暂停：%s。此源将在约 %s 自动重新开始。\n",
		"Пауза на ночной отдых: %s. Этот источник автоматически запустится снова примерно в %s.\n"), formatBINETA(d), resumeAt.Format("15:04"))
	fmt.Println(L(
		"Press Ctrl+C during the pause to stop the complete batch.",
		"Drücken Sie während der Pause Ctrl+C, um den kompletten Batch zu stoppen.",
		"Druk tijdens de pauze op Ctrl+C om de hele batch te stoppen.",
		"Appuyez sur Ctrl+C pendant la pause pour arrêter tout le lot.",
		"Pulse Ctrl+C durante la pausa para detener todo el lote.",
		"暂停期间按 Ctrl+C 可停止整个批处理。",
		"Нажмите Ctrl+C во время паузы, чтобы остановить весь пакет."))

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	deadline := time.NewTimer(d)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			fmt.Println(L("Night-rest pause finished; retrying source:", "Nachtruhe-Pause beendet; Quelle wird erneut gestartet:", "Nachtrustpauze voorbij; bron wordt opnieuw gestart:", "Pause nocturne terminée ; nouvelle tentative de la source :", "Pausa nocturna terminada; reintentando fuente:", "休息暂停结束；正在重新处理源：", "Пауза на ночной отдых завершена; повтор источника:"), source)
			return true
		case <-interrupts:
			fmt.Println("\n" + L("Batch stop requested during night-rest pause.", "Batch-Stopp während der Nachtruhe-Pause angefordert.", "Batchstop gevraagd tijdens de nachtrustpauze.", "Arrêt du lot demandé pendant la pause nocturne.", "Parada del lote solicitada durante la pausa nocturna.", "休息暂停期间已请求停止批处理。", "Во время паузы на ночной отдых запрошена остановка пакета."))
			return false
		case <-ticker.C:
			remaining := time.Until(resumeAt)
			if remaining < 0 {
				remaining = 0
			}
			fmt.Printf("\r%s", L("Nachtrust: automatisch verder over ", "Nachtruhe: automatisch weiter in ", "Nachtrust: automatisch verder over ", "Repos nocturne : reprise automatique dans ", "Descanso nocturno: continuación automática en ", "休息：自动继续还需 ", "Ночной отдых: автоматическое продолжение через ")+formatBINETA(remaining)+"   ")
		}
	}
}
