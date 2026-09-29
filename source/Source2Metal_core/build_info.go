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
  Source2Metal_v3.4.1-RC3_SOURCE.zip is included in this trial package; not published.

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
  Source2Metal_v3.4.1-RC3_SOURCE.zip ist im Testpaket enthalten; nicht veröffentlicht.

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
  Source2Metal_v3.4.1-RC3_SOURCE.zip zit in dit proefpakket; niet gepubliceerd.

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
  Source2Metal_v3.4.1-RC3_SOURCE.zip est inclus dans ce paquet de test ; non publié.

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
  Source2Metal_v3.4.1-RC3_SOURCE.zip está incluido en este paquete de prueba; no publicado.

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
  本测试包包含 Source2Metal_v3.4.1-RC3_SOURCE.zip；尚未发布。

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
  Source2Metal_v3.4.1-RC3_SOURCE.zip включён в тестовый пакет; не опубликован.

Включённая утилита:
  SyzygyCheck v%s можно извлечь через пункт 2 главного меню.
`)
	return fmt.Sprintf(tpl, version, buildDateUTC, syzygyCheckDisplay) + "\n" + binScopeText()
}
