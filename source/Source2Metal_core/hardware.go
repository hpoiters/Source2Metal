package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func defaultWorkers(threads int) int {
	if threads <= 1 {
		return 1
	}
	w := (threads + 1) / 2
	if w < 1 {
		w = 1
	}
	return w
}

func detectHardware(root string) HardwareInfo {
	h := HardwareInfo{LogicalThreads: runtime.NumCPU()}
	if h.LogicalThreads < 1 {
		h.LogicalThreads = 1
	}
	if runtime.GOOS == "windows" {
		detectWindowsHardware(root, &h)
	} else {
		detectUnixLikeHardware(root, &h)
	}
	return h
}

func detectWindowsHardware(root string, h *HardwareInfo) {
	drive := filepath.VolumeName(root)
	if drive == "" {
		drive = "C:"
	}
	letter := strings.TrimSuffix(drive, ":")
	ps := fmt.Sprintf(`$ErrorActionPreference='SilentlyContinue';
$cpu=(Get-CimInstance Win32_Processor | Select-Object -First 1 -ExpandProperty Name);
$ram=[int64](Get-CimInstance Win32_ComputerSystem | Select-Object -ExpandProperty TotalPhysicalMemory);
$pf=Get-CimInstance Win32_PageFileUsage;
$pfmb=[int64](($pf | Measure-Object -Property AllocatedBaseSize -Sum).Sum);
$pfpaths=($pf | ForEach-Object {$_.Name}) -join ';';
$drv=Get-PSDrive -Name '%s';
$free=[int64]$drv.Free;
'CPU='+$cpu; 'RAM_BYTES='+$ram; 'PAGEFILE_MB='+$pfmb; 'PAGEFILE_PATHS='+$pfpaths; 'FREE_BYTES='+$free;`, letter)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key, val := parts[0], strings.TrimSpace(parts[1])
		switch key {
		case "CPU":
			h.CPUName = val
		case "RAM_BYTES":
			if n, e := strconv.ParseInt(val, 10, 64); e == nil && n > 0 {
				h.RAMBytes, h.RAMKnown = n, true
			}
		case "PAGEFILE_MB":
			if n, e := strconv.ParseInt(val, 10, 64); e == nil {
				h.PagefileBytes, h.PagefileKnown = n*1024*1024, true
			}
		case "PAGEFILE_PATHS":
			h.PagefilePaths = val
		case "FREE_BYTES":
			if n, e := strconv.ParseInt(val, 10, 64); e == nil && n >= 0 {
				h.FreeBytes, h.FreeKnown = n, true
			}
		}
	}
}

func detectUnixLikeHardware(root string, h *HardwareInfo) {
	if f, err := os.Open("/proc/cpuinfo"); err == nil {
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := s.Text()
			if strings.HasPrefix(line, "model name") {
				if p := strings.SplitN(line, ":", 2); len(p) == 2 {
					h.CPUName = strings.TrimSpace(p[1])
					break
				}
			}
		}
		_ = f.Close()
	}
	if f, err := os.Open("/proc/meminfo"); err == nil {
		s := bufio.NewScanner(f)
		for s.Scan() {
			fields := strings.Fields(s.Text())
			if len(fields) < 2 {
				continue
			}
			n, _ := strconv.ParseInt(fields[1], 10, 64)
			switch strings.TrimSuffix(fields[0], ":") {
			case "MemTotal":
				h.RAMBytes, h.RAMKnown = n*1024, true
			case "SwapTotal":
				h.PagefileBytes, h.PagefileKnown = n*1024, true
			}
		}
		_ = f.Close()
	}
}

func printHardware(h HardwareInfo) {
	fmt.Println("\n" + U("HARDWAREDETECTIE"))
	if h.CPUName != "" {
		fmt.Println(U("CPU              :"), h.CPUName)
	}
	fmt.Printf(U("Logische threads : %d\n"), h.LogicalThreads)
	if h.RAMKnown {
		fmt.Printf("RAM              : %.1f GB\n", gib(h.RAMBytes))
	} else {
		fmt.Println(U("RAM              : niet betrouwbaar uitgelezen"))
	}
	if h.PagefileKnown {
		fmt.Printf("Pagefile         : %.1f GB", gib(h.PagefileBytes))
		if h.PagefilePaths != "" {
			fmt.Printf(" (%s)", h.PagefilePaths)
		}
		fmt.Println()
		if h.PagefileBytes == 0 {
			fmt.Println(L("WARNING          : the pagefile appears disabled; for very large runs, a pagefile on a fast SSD/NVMe is recommended.", "WARNUNG          : Die Auslagerungsdatei scheint deaktiviert; für sehr große Läufe wird eine Auslagerungsdatei auf einer schnellen SSD/NVMe empfohlen.", "WAARSCHUWING     : pagefile lijkt uitgeschakeld; voor zeer grote runs wordt een pagefile op een snelle SSD/NVMe aanbevolen.", "AVERTISSEMENT    : le fichier d’échange semble désactivé ; pour les très gros traitements, un fichier d’échange sur SSD/NVMe rapide est recommandé.", "ADVERTENCIA      : el archivo de paginación parece desactivado; para ejecuciones muy grandes se recomienda uno en un SSD/NVMe rápido.", "警告             ：页面文件似乎已禁用；对于超大型运行，建议在高速 SSD/NVMe 上启用页面文件。", "ПРЕДУПРЕЖДЕНИЕ   : файл подкачки, похоже, отключён; для очень больших запусков рекомендуется файл подкачки на быстром SSD/NVMe."))
		}
	} else {
		fmt.Println(U("Pagefile         : niet betrouwbaar uitgelezen"))
	}
	if h.FreeKnown {
		fmt.Printf(U("Vrije uitvoerruimte: %.1f GB\n"), gib(h.FreeBytes))
	}
	fmt.Printf(L("Default workers  : %d (50%% of %d threads)\n", "Standard-Worker  : %d (50%% von %d Threads)\n", "Standaard workers: %d (50%% van %d threads)\n", "Workers par défaut : %d (50%% de %d threads)\n", "Workers predeterminados: %d (50%% de %d hilos)\n", "默认 worker 数   ：%d（%d 个线程的 50%%）\n", "Рабочие потоки по умолчанию: %d (50%% от %d потоков)\n"), defaultWorkers(h.LogicalThreads), h.LogicalThreads)
}

func gib(n int64) float64 { return float64(n) / (1024 * 1024 * 1024) }

func pagefileHelpText() string {
	return L(`Recommended system setting for large databases

For very large Source2Metal runs, a generous Windows pagefile on a fast SSD/NVMe
can provide extra stability. A pagefile does not replace RAM, but increases the
Windows commit reserve. Prefer not to disable the pagefile completely.

Windows 11:
1. Press Win+R
2. type: sysdm.cpl
3. Advanced -> Performance: Settings
4. Advanced -> Virtual memory: Change
5. optionally clear "Automatically manage paging file size"
6. choose a fast SSD/NVMe
7. choose "System managed size" or a custom size
8. click Set and restart Windows

With 32 GB RAM, 32-64 GB is a practical generous manual reserve for very large
databases, provided enough free disk space is available. Source2Metal never
changes Windows settings itself.`,
		`Empfohlene Systemeinstellung für große Datenbanken

Für sehr große Source2Metal-Läufe kann eine großzügige Windows-Auslagerungsdatei
auf einer schnellen SSD/NVMe zusätzliche Stabilität bieten. Eine Auslagerungsdatei
ersetzt keinen RAM, vergrößert aber die Commit-Reserve von Windows. Deaktivieren
Sie die Auslagerungsdatei möglichst nicht vollständig.

Windows 11:
1. Win+R drücken
2. eingeben: sysdm.cpl
3. Erweitert -> Leistung: Einstellungen
4. Erweitert -> Virtueller Arbeitsspeicher: Ändern
5. optional "Auslagerungsdateigröße für alle Laufwerke automatisch verwalten" deaktivieren
6. eine schnelle SSD/NVMe wählen
7. "Vom System verwaltete Größe" oder eine benutzerdefinierte Größe wählen
8. Festlegen anklicken und Windows neu starten

Bei 32 GB RAM sind 32-64 GB als großzügige manuelle Reserve für sehr große
Datenbanken praktikabel, sofern genügend freier Speicher vorhanden ist.
Source2Metal ändert Windows-Einstellungen niemals selbst.`,
		`Aanbevolen systeeminstelling voor grote databases

Voor zeer grote Source2Metal-runs kan een ruime Windows-pagefile op een snelle
SSD/NVMe extra stabiliteit geven. Een pagefile vervangt geen RAM, maar vergroot
de beschikbare commit-reserve van Windows. Schakel de pagefile bij voorkeur
niet volledig uit.

Windows 11:
1. Win+R
2. typ: sysdm.cpl
3. Geavanceerd -> Prestaties: Instellingen
4. Geavanceerd -> Virtueel geheugen: Wijzigen
5. desgewenst "Wisselbestandsgrootte automatisch beheren" uitvinken
6. kies een snelle SSD/NVMe
7. kies "Systeem beheerde grootte" of een aangepaste grootte
8. klik Instellen en herstart Windows

Bij 32 GB RAM is 32-64 GB als ruime handmatige reserve een praktische keuze
voor zeer grote databases, mits voldoende vrije schijfruimte beschikbaar is.
Source2Metal wijzigt Windows-instellingen nooit zelf.`,
		`Réglage système recommandé pour les grandes bases de données

Pour de très gros traitements Source2Metal, un fichier d’échange Windows généreux
sur un SSD/NVMe rapide peut apporter une stabilité supplémentaire. Il ne remplace
pas la RAM, mais augmente la réserve d’engagement de Windows. Évitez de désactiver
complètement le fichier d’échange.

Windows 11 :
1. Appuyez sur Win+R
2. tapez : sysdm.cpl
3. Avancé -> Performances : Paramètres
4. Avancé -> Mémoire virtuelle : Modifier
5. décochez si besoin "Gérer automatiquement le fichier d’échange"
6. choisissez un SSD/NVMe rapide
7. choisissez "Taille gérée par le système" ou une taille personnalisée
8. cliquez sur Définir puis redémarrez Windows

Avec 32 Go de RAM, 32-64 Go constituent une réserve manuelle généreuse et
pratique pour de très grandes bases, si l’espace disque libre est suffisant.
Source2Metal ne modifie jamais lui-même les paramètres Windows.`,
		`Configuración del sistema recomendada para bases de datos grandes

Para ejecuciones muy grandes de Source2Metal, un archivo de paginación de Windows
amplio en un SSD/NVMe rápido puede aportar estabilidad adicional. No sustituye a
la RAM, pero aumenta la reserva de compromiso de Windows. Es preferible no
desactivar por completo el archivo de paginación.

Windows 11:
1. Pulse Win+R
2. escriba: sysdm.cpl
3. Opciones avanzadas -> Rendimiento: Configuración
4. Opciones avanzadas -> Memoria virtual: Cambiar
5. opcionalmente desmarque "Administrar automáticamente el tamaño"
6. elija un SSD/NVMe rápido
7. elija "Tamaño administrado por el sistema" o un tamaño personalizado
8. pulse Establecer y reinicie Windows

Con 32 GB de RAM, 32-64 GB es una reserva manual amplia y práctica para bases
muy grandes, siempre que haya suficiente espacio libre. Source2Metal nunca
modifica por sí mismo la configuración de Windows.`,
		`大型数据库的推荐系统设置

对于非常大的 Source2Metal 运行，在高速 SSD/NVMe 上配置较大的 Windows 页面文件
可以提高稳定性。页面文件不能替代 RAM，但可以增加 Windows 的提交内存余量。
建议不要完全关闭页面文件。

Windows 11：
1. 按 Win+R
2. 输入：sysdm.cpl
3. 高级 -> 性能：设置
4. 高级 -> 虚拟内存：更改
5. 如需要，取消“自动管理所有驱动器的分页文件大小”
6. 选择高速 SSD/NVMe
7. 选择“系统管理的大小”或自定义大小
8. 点击“设置”，然后重新启动 Windows

对于 32 GB RAM，在磁盘空间充足时，32-64 GB 是处理超大型数据库时较实用的
手动预留范围。Source2Metal 从不自行更改 Windows 设置。`,
		`Рекомендуемая системная настройка для больших баз данных

Для очень крупных запусков Source2Metal большой файл подкачки Windows на быстром
SSD/NVMe может повысить стабильность. Файл подкачки не заменяет оперативную память,
но увеличивает резерв виртуальной памяти Windows. Лучше не отключать его полностью.

Windows 11:
1. Нажмите Win+R
2. введите: sysdm.cpl
3. Дополнительно -> Быстродействие: Параметры
4. Дополнительно -> Виртуальная память: Изменить
5. при необходимости снимите автоматическое управление размером файла подкачки
6. выберите быстрый SSD/NVMe
7. выберите "Размер по выбору системы" или задайте размер вручную
8. нажмите Задать и перезапустите Windows

При 32 ГБ RAM резерв 32-64 ГБ является практичным вариантом для очень больших
баз данных, если на диске достаточно свободного места. Source2Metal никогда не
изменяет настройки Windows самостоятельно.`)
}
