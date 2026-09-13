package main

import "fmt"

func buildInfoText() string {
	tpl := L(`Source2Metal v%s
Build date UTC : %s

CREDITS
Main development, architecture and technical implementation:
  OpenAI ChatGPT (GPT-5.6 Sol)
Practical direction and testing:
  Nocompany, Netherlands - computer-chess player

Source code:
  Download the separate source archive from the Source2Metal GitHub release.

Included utility:
  SyzygyCheck v%s can be unpacked via main-menu option 2.
`, `Source2Metal v%s
Build-Datum UTC: %s

CREDITS
Hauptentwicklung, Architektur und technische Umsetzung:
  OpenAI ChatGPT (GPT-5.6 Sol)
Praxisrichtung und Tests:
  Nocompany, Niederlande - Computerschachspieler

Quellcode:
  Das separate Quellcodearchiv kann von der Source2Metal-GitHub-Version heruntergeladen werden.

Enthaltenes Hilfsprogramm:
  SyzygyCheck v%s kann über Hauptmenü-Option 2 entpackt werden.
`, `Source2Metal v%s
Build-datum UTC: %s

CREDITS
Hoofdontwikkeling, architectuur en technische uitwerking:
  OpenAI ChatGPT (GPT-5.6 Sol)
Praktijkrichting en testen:
  Nocompany, Nederland - computerschaker

Broncode:
  Download het afzonderlijke bronarchief bij de Source2Metal-release op GitHub.

Meegeleverd hulpprogramma:
  SyzygyCheck v%s kan worden uitgepakt via optie 2 in het hoofdmenu.
`, `Source2Metal v%s
Date de build UTC : %s

CRÉDITS
Développement principal, architecture et réalisation technique :
  OpenAI ChatGPT (GPT-5.6 Sol)
Orientation pratique et tests :
  Nocompany, Pays-Bas - joueur d'échecs informatiques

Code source :
  Téléchargez l'archive source séparée depuis la version Source2Metal sur GitHub.

Utilitaire inclus :
  SyzygyCheck v%s peut être extrait via l’option 2 du menu principal.
`, `Source2Metal v%s
Fecha de build UTC: %s

CRÉDITOS
Desarrollo principal, arquitectura e implementación técnica:
  OpenAI ChatGPT (GPT-5.6 Sol)
Orientación práctica y pruebas:
  Nocompany, Países Bajos - jugador de ajedrez informático

Código fuente:
  Descargue el archivo fuente independiente desde la versión de Source2Metal en GitHub.

Utilidad incluida:
  SyzygyCheck v%s se puede extraer mediante la opción 2 del menú principal.
`, `Source2Metal v%s
构建日期 UTC：%s

致谢
主要开发、架构与技术实现：
  OpenAI ChatGPT (GPT-5.6 Sol)
实战方向与测试：
  Nocompany，荷兰 - 计算机国际象棋棋手

源代码：
  请从 GitHub 的 Source2Metal 发布页下载单独的源代码归档。

随附实用工具：
  可通过主菜单选项 2 解压 SyzygyCheck v%s。
`, `Source2Metal v%s
Дата сборки UTC: %s

БЛАГОДАРНОСТИ
Основная разработка, архитектура и техническая реализация:
  OpenAI ChatGPT (GPT-5.6 Sol)
Практическое направление и тестирование:
  Nocompany, Нидерланды - компьютерный шахматист

Исходный код:
  Загрузите отдельный архив исходного кода со страницы выпуска Source2Metal на GitHub.

Включённая утилита:
  SyzygyCheck v%s можно извлечь через пункт 2 главного меню.
`)
	return fmt.Sprintf(tpl, version, buildDateUTC, syzygyCheckDisplay) + "\n" + binScopeText()
}
