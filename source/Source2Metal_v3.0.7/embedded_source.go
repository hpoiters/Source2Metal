package main

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

//go:embed embedded_source.zip
var embeddedSourceZip []byte

func embeddedSourceSHA256() string {
	h := sha256.Sum256(embeddedSourceZip)
	return fmt.Sprintf("%x", h[:])
}

func extractEmbeddedSource() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	out := filepath.Join(dir, "Source2Metal_v"+version+"_SOURCE.zip")
	if data, err := os.ReadFile(out); err == nil {
		h := sha256.Sum256(data)
		if fmt.Sprintf("%x", h[:]) == embeddedSourceSHA256() {
			return out, nil
		}
		out = filepath.Join(dir, "Source2Metal_v"+version+"_SOURCE_"+time.Now().Format("20060102_150405")+".zip")
	}
	if err := atomicWriteFile(out, embeddedSourceZip, 0644); err != nil {
		return "", err
	}
	return out, nil
}

func buildInfoText() string {
	tpl := L(`Source2Metal v%s
Build date UTC        : %s
Embedded source bytes : %d
Embedded source SHA256: %s

CREDITS
Main development, architecture and technical implementation:
  OpenAI ChatGPT (GPT-5.6 Sol)
Practical direction and testing:
  Nocompany, Netherlands - computer-chess player

Source2Metal is a first open effort to make self-built opening books more
technically grounded and practically resilient in the NNUE era, while increasing
the enjoyment of computer chess. Users and developers are encouraged to improve
the project further.

Extract source package:
  choose Utilities > Extract Source2Metal source package
or use:
  Source2Metal_v%s.exe -extract-source

The EXE writes the source package next to itself. An existing identical package
is reused; a different file is never silently overwritten.

SyzygyCheck v2.0.7 source can be extracted separately via Utilities > Extract
SyzygyCheck source-code package (ZIP), or with -extract-syzygy-source.
`, `Source2Metal v%s
Build-Datum UTC        : %s
Eingebettete Quelle    : %d Bytes
Quellcode SHA256       : %s

CREDITS
Hauptentwicklung, Architektur und technische Umsetzung:
  OpenAI ChatGPT (GPT-5.6 Sol)
Praxisrichtung und Tests:
  Nocompany, Niederlande - Computerschachspieler

Source2Metal ist ein erster offener Ansatz, selbst erstellte Eröffnungsbücher im
NNUE-Zeitalter technisch fundierter und praktisch widerstandsfähiger zu machen
und den Spaß am Computerschach zu erhöhen. Weitere Verbesserungen durch Anwender
und Entwickler sind willkommen.

Quellcodepaket entpacken:
  Hilfsprogramme > Source2Metal-Quellcodepaket entpacken wählen
oder:
  Source2Metal_v%s.exe -extract-source

Die EXE schreibt das Quellcodepaket neben sich selbst. Ein identisches vorhandenes
Paket wird wiederverwendet; ein abweichendes wird nie still überschrieben.

Der Quellcode von SyzygyCheck v2.0.7 kann separat über Hilfsprogramme >
SyzygyCheck-Quellcodepaket (ZIP) entpacken oder mit -extract-syzygy-source
extrahiert werden.
`, `Source2Metal v%s
Build-datum UTC        : %s
Ingebedde bron bytes   : %d
Ingebedde bron SHA256  : %s

CREDITS
Hoofdontwikkeling, architectuur en technische uitwerking:
  OpenAI ChatGPT (GPT-5.6 Sol)
Praktijkrichting en testen:
  Nocompany, Nederland - computerschaker

Source2Metal is een eerste open aanzet om zelfgemaakte openingsboeken in het
NNUE-tijdperk technisch beter onderbouwd en praktisch weerbaarder te maken en
het plezier in computerschaak te vergroten. Verdere verbetering door gebruikers
en ontwikkelaars wordt aangemoedigd.

Broncodepakket uitpakken:
  kies Hulpprogramma's > Source2Metal-broncodepakket uitpakken
of gebruik:
  Source2Metal_v%s.exe -extract-source

De EXE schrijft het broncodepakket naast zichzelf. Een bestaand identiek pakket
wordt hergebruikt; een afwijkend bestand wordt nooit stil overschreven.

De bron van SyzygyCheck v2.0.7 kan afzonderlijk worden uitgepakt via
Hulpprogramma's > SyzygyCheck-broncodepakket uitpakken (ZIP), of met
-extract-syzygy-source.
`, `Source2Metal v%s
Date de build UTC      : %s
Source intégrée        : %d octets
SHA256 source intégrée : %s

CRÉDITS
Développement principal, architecture et réalisation technique :
  OpenAI ChatGPT (GPT-5.6 Sol)
Orientation pratique et tests :
  Nocompany, Pays-Bas - joueur d'échecs informatiques

Source2Metal est une première initiative ouverte visant à rendre les livres
d'ouvertures créés par l'utilisateur techniquement mieux fondés et plus robustes
à l'ère NNUE, tout en augmentant le plaisir des échecs informatiques. Les
améliorations par les utilisateurs et développeurs sont encouragées.

Extraire le paquet source :
  choisissez Utilitaires > Extraire le paquet source Source2Metal
ou utilisez :
  Source2Metal_v%s.exe -extract-source

L'EXE écrit le paquet source à côté de lui-même. Un paquet identique existant est
réutilisé ; un fichier différent n'est jamais écrasé silencieusement.

Le source de SyzygyCheck v2.0.7 peut être extrait séparément via Utilitaires >
Extraire le paquet source SyzygyCheck (ZIP), ou avec -extract-syzygy-source.
`, `Source2Metal v%s
Fecha de build UTC     : %s
Fuente integrada       : %d bytes
SHA256 fuente integrada: %s

CRÉDITOS
Desarrollo principal, arquitectura e implementación técnica:
  OpenAI ChatGPT (GPT-5.6 Sol)
Orientación práctica y pruebas:
  Nocompany, Países Bajos - jugador de ajedrez informático

Source2Metal es un primer esfuerzo abierto para hacer que los libros de aperturas
creados por el usuario estén mejor fundamentados técnicamente y sean más robustos
en la era NNUE, aumentando a la vez el disfrute del ajedrez informático. Se anima
a usuarios y desarrolladores a seguir mejorando el proyecto.

Extraer paquete fuente:
  elija Utilidades > Extraer paquete de código fuente Source2Metal
o use:
  Source2Metal_v%s.exe -extract-source

El EXE escribe el paquete fuente junto a sí mismo. Un paquete idéntico ya existente
se reutiliza; un archivo diferente nunca se sobrescribe silenciosamente.

El código fuente de SyzygyCheck v2.0.7 puede extraerse por separado mediante
Utilidades > Extraer paquete fuente SyzygyCheck (ZIP), o con -extract-syzygy-source.
`, `Source2Metal v%s
构建日期 UTC           ：%s
内嵌源代码字节         ：%d
内嵌源代码 SHA256      ：%s

致谢
主要开发、架构与技术实现：
  OpenAI ChatGPT (GPT-5.6 Sol)
实战方向与测试：
  Nocompany，荷兰 - 计算机国际象棋棋手

Source2Metal 是一个开放的初步尝试，目标是在 NNUE 时代让用户自己制作的开局库
在技术依据和实战稳定性方面更完善，同时增加计算机国际象棋的乐趣。欢迎用户和
开发者继续改进本项目。

解压源代码包：
  选择 实用工具 > 解压 Source2Metal 源代码包
或使用：
  Source2Metal_v%s.exe -extract-source

EXE 会把源代码包写到自身旁边。已有完全相同的包会直接复用；不同文件绝不会被
静默覆盖。

SyzygyCheck v2.0.7 的源代码可通过“实用工具 > 解压 SyzygyCheck 源代码包（ZIP）”
单独解压，也可使用 -extract-syzygy-source。
`, `Source2Metal v%s
Дата сборки UTC        : %s
Встроенный исходный код: %d байт
SHA256 исходного кода  : %s

БЛАГОДАРНОСТИ
Основная разработка, архитектура и техническая реализация:
  OpenAI ChatGPT (GPT-5.6 Sol)
Практическое направление и тестирование:
  Nocompany, Нидерланды - компьютерный шахматист

Source2Metal — первая открытая попытка сделать самостоятельно создаваемые
дебютные книги технически более обоснованными и практически более устойчивыми
в эпоху NNUE, одновременно повышая удовольствие от компьютерных шахмат.
Пользователи и разработчики могут продолжать улучшать проект.

Извлечь пакет исходного кода:
  выберите Утилиты > Извлечь пакет исходного кода Source2Metal
или используйте:
  Source2Metal_v%s.exe -extract-source

EXE записывает пакет исходного кода рядом с собой. Идентичный существующий пакет
используется повторно; отличающийся файл никогда не перезаписывается скрытно.

Исходный код SyzygyCheck v2.0.7 можно отдельно извлечь через «Утилиты >
Извлечь пакет исходного кода SyzygyCheck (ZIP)» или с помощью
-extract-syzygy-source.
`)
	return fmt.Sprintf(tpl, version, buildDateUTC, len(embeddedSourceZip), embeddedSourceSHA256(), version)
}
