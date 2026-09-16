package main

import (
	"fmt"
	"time"
)

// Display-only historical heuristic: the user's approximately 1.4 GB book
// took approximately two hours. File size is NOT a measure of reachable work.
// The embedded converter and its output protocol remain unchanged.
type roughEstimate struct {
	referenceSeconds float64
	started          bool
	start            time.Duration
	lastUpdate       time.Duration
	roundedMinutes   int
	exceeded         bool
}

func (e *roughEstimate) update(elapsed time.Duration, records int64, loc locale) (string, string) {
	if !e.started {
		e.started, e.start = true, elapsed
	}
	active := elapsed - e.start
	if active < 5*time.Minute || records <= 0 {
		return "", ""
	}
	// Recompute once per minute; preserve the displayed rounded value in between.
	// No completed-work counter is supplied after scanning: do not invent one.
	if e.roundedMinutes == 0 || active-e.lastUpdate >= time.Minute {
		e.lastUpdate = active
		// Work in floating point before multiplication to avoid integer overflow.
		totalMinutes := float64(records) * 16 / 1.4e9 * 120
		if e.referenceSeconds > 0 {
			totalMinutes = e.referenceSeconds / 60
		}
		minutes := totalMinutes - active.Minutes()
		e.exceeded = minutes <= 0
		step := 10.0
		if minutes < 30 {
			step = 5
		}
		if minutes >= 90 {
			step = 30
		}
		if minutes >= 180 {
			step = 60
		}
		e.roundedMinutes = int(minutes/step+0.5) * int(step)
		if e.roundedMinutes < 5 {
			e.roundedMinutes = 5
		}
	}
	explanations := []string{
		"Based on file size and a prior run; may differ greatly.",
		"Nach Dateigröße und früherem Lauf; starke Abweichung möglich.",
		"Op basis van bestandsgrootte en eerdere run; kan sterk afwijken.",
		"Taille du fichier et essai précédent ; forte variation possible.",
		"Según tamaño y ejecución previa; puede variar mucho.",
		"根据文件大小和以往运行估算；可能相差很大。",
		"По размеру и прошлому запуску; возможны большие отклонения.",
	}
	if e.exceeded {
		exceeded := []string{
			"Reference time exceeded; remaining duration unknown.",
			"Richtzeit überschritten; verbleibende Dauer unbekannt.",
			"Richttijd overschreden; resterende duur onbekend.",
			"Durée indicative dépassée ; temps restant inconnu.",
			"Tiempo orientativo superado; tiempo restante desconocido.",
			"已超过参考时间；剩余时间未知。",
			"Ориентир превышен; оставшееся время неизвестно.",
		}
		return "", exceeded[loc]
	}
	return coarseDuration(e.roundedMinutes, loc), explanations[loc]
}

func coarseDuration(minutes int, loc locale) string {
	if minutes < 60 || minutes%30 != 0 {
		units := []string{"minutes", "Minuten", "minuten", "minutes", "minutos", "分钟", "мин"}
		return fmt.Sprintf("%d %s", minutes, units[loc])
	}
	number := fmt.Sprint(minutes / 60)
	if minutes%60 == 30 {
		number += "½"
	}
	units := []string{"hours", "Stunden", "uur", "heures", "horas", "小时", "ч"}
	if minutes == 60 {
		units = []string{"hour", "Stunde", "uur", "heure", "hora", "小时", "ч"}
	}
	return number + " " + units[loc]
}
