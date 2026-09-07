SyzygyCheck v2.0.7 - BRONCODE
================================

Dit pakket bewaart de bron van de door de gebruiker op Windows praktisch
gevalideerde SyzygyCheck v2.0.7.

BELANGRIJKE VERANKERDE UX-EIGENSCHAP
De actieve voortgang gebruikt ÉÉN adaptieve fysieke consoleregel. De regel wordt
bij elke update passend gemaakt voor de actuele consolebreedte. Als Windows
Terminal tijdens de controle wordt vergroot of verkleind, wordt het scherm schoon
opgebouwd zodat resize/reflow geen historische voortgangsregels laat stapelen.

Dit gedrag is bewust een regressieregel. Vervang het niet door twee vaste
voortgangsregels met verticale cursorbeweging zonder opnieuw een echte
Windows-Terminal resize-test te doen.

VASTE SCANREGEL
Alleen .rtbw en .rtbz in exact dezelfde map als SyzygyCheck_v2.0.7.exe worden
gecontroleerd. Niet recursief, niet in submappen en niet in bovenliggende mappen.

CHECKSUM-ENGINE
`tbcheck.exe` is de bewezen checksum-engine en wordt als embedded dependency
ongewijzigd gebruikt. Wijzig deze binary niet zonder afzonderlijke validatie.

BOUWEN
Vereist: Go 1.23 of nieuwer.

Windows/amd64:
  set GOOS=windows
  set GOARCH=amd64
  set CGO_ENABLED=0
  go test ./...
  go vet ./...
  go build -trimpath -buildvcs=false -ldflags="-s -w" -o SyzygyCheck_v2.0.7.exe .

PRAKTIJKTEST VOOR EEN NIEUWE RELEASE
- gebruik een map met meerdere .rtbw/.rtbz-bestanden;
- gebruik minstens één groot bestand zodat de voortgang zichtbaar blijft;
- wijzig tijdens de run herhaald de breedte van Windows Terminal;
- controleer dat slechts één actuele voortgangsregel zichtbaar blijft;
- gebruik een bewust beschadigd bestand en controleer dat het definitieve
  foutresultaat onder `Klaar:`/`Done:` staat;
- controleer de taal-ontsnappingsroute Engels/Russisch/Chinees.

Deze controles horen bij de releasekwaliteit en mogen niet stil worden verwijderd.
