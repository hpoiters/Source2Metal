SyzygyCheck v2.0.9 V9F — Alle talen
================================

Doel
----
SyzygyCheck controleert de integriteit van Syzygy .rtbw/.rtbz-bestanden in dezelfde map als de EXE.
Alleen die map wordt gecontroleerd: geen bovenliggende map en geen submappen.

Verificatiekern en credits
--------------------------
Gebaseerd op de oorspronkelijke Syzygy-tablebase-verificatiecode van Ronald de Man.
De AL-laag implementeert de checksumlogica niet opnieuw. De meertalige interface, voortgangsweergave,
zelftest, checkpointing/hervatten, threadkeuze en rapportage zijn daaromheen gebouwd.
Upstream: https://github.com/syzygy1/tb

Threadkeuze
-----------
SyzygyCheck detecteert bij iedere start stil hoeveel logische processors Windows aan het proces beschikbaar stelt.
Daarna kunt u kiezen:

  1 = Alle gedetecteerde threads gebruiken
  2 = Helft gebruiken [standaard; Enter kiest dit]
  3 = Zelf een aantal kiezen van 1 t/m het gedetecteerde maximum

De werkelijk gekozen waarde wordt in het overzicht getoond als `Werkers (threads): <aantal>` en rechtstreeks
als `--threads n` aan de oorspronkelijke tbcheck doorgegeven. De standaardhelft is bedoeld als evenwichtige
keuze; voor snelheidstests kan alle threads of een eigen aantal worden gekozen. Esc brengt u vanuit de
threadkeuze terug; ook vanuit de handmatige aantal-invoer gaat Esc één niveau terug.

Pagefile / virtueel geheugen
---------------------------
Zeer grote Syzygy-controles kunnen veel virtueel geheugen vereisen en vele uren duren. Voor de zelftest
verschijnt daarom een korte waarschuwing. Enter = doorgaan; A/Esc = terug om eerst de Windows-instellingen
te controleren of wijzigen. De uitgebreide stap-voor-stap handleidingen worden in alle zeven ondersteunde
talen meegeleverd als PAGEFILE_GUIDE_XX.txt.

Voor de meeste systemen is Windows automatisch beheer van het wisselbestand/pagefile de eenvoudigste
veilige uitgangspositie. SyzygyCheck verandert deze instelling nooit zelf.

Automatische onboard zelftest
-----------------------------
Bij programmastart wordt een eventueel achtergebleven gereserveerd zelftestbestand stil verwijderd.
Daarna wordt tijdelijk een piepkleine onboard testfile geplaatst met een opzettelijk verkeerde checksum.
De tbcheck van Ronald de Man moet deze als FAIL herkennen. De testfile wordt onmiddellijk verwijderd vóór de
echte .rtbw/.rtbz-inventarisatie. Uw eigen Syzygy-bestanden worden voor deze test niet gebruikt of gewijzigd.
Na een geslaagde zelftest wacht het programma op Enter voordat de echte controle begint; Esc gaat terug naar
het hoofdmenu. Als de zelftest niet slaagt, wordt de echte controle niet gestart.

Voortgang
---------
De live status is een compact, resizebestendig blok van drie regels:

  Bezig met bestand: n/totaal        Leessnelheid: xx.x MB/s
  [adaptieve volumebalk]
  percentage   verwerkt/totaal       Resttijd: ...

`Leessnelheid` en `Resttijd` beginnen waar de consolebreedte dat toelaat in dezelfde rechterkolom.
Na een zichtbare FAIL/ERROR komt precies één lege regel vóór het live voortgangsblok. Bij veel fouten blijft
de actuele voortgang onderaan zichtbaar; oudere foutregels mogen naar boven uit beeld schuiven en blijven via
de console-scrollback terug te bekijken.

De getoonde leessnelheid is de effectieve controlesnelheid van de huidige sessie, niet per definitie de ruwe
fysieke schijf- of bussnelheid. Tijdens een lopende tbcheck-batch gebruikt de live weergave de Windows
proces-I/O-teller als voortgangsschatting. Na bevestigde bestandsuitkomsten wordt de snelheid op definitief
gecontroleerde bytes gebaseerd. Het eindrapport gebruikt de definitief gecontroleerde bytes van de huidige
sessie gedeeld door de verstreken controletijd. Bij hervatten worden bytes uit een eerdere sessie niet ten
onrechte in dat sessiegemiddelde meegerekend.

Datavolume is de hoofdmaat voor de voortgang; het bestandsnummer is aanvullende informatie. De balk gebruikt
zoveel mogelijk van de actuele consolebreedte en laat één consolekolom vrij om automatische wrap/linefeed te
vermijden.

FAIL en ERROR
-------------
FAIL betekent: de ingebedde checksum is aanwezig, maar komt niet overeen met de opnieuw berekende checksum.
ERROR betekent: het bestand kon niet betrouwbaar worden gecontroleerd, bijvoorbeeld door een lees-, structuur-
of procesprobleem. Eén problematisch bestand mag de uitkomst van latere bestanden niet verbergen.

Vensterbeveiliging tijdens de echte controle
------------------------------------------------
Tijdens een actieve echte Syzygy-controle probeert SyzygyCheck als best-effort maatregel SC_CLOSE van een klassiek
Windows-consolevenster uit te schakelen. In Windows Terminal behoort het buitenste kruisje rechtsboven echter aan
Windows Terminal zelf en kan het venster nog steeds worden gesloten. Verkleinen, vergroten/maximaliseren,
vensterformaat wijzigen en scrollback worden niet bewust geblokkeerd. Checkpoint/hervatten blijft de betrouwbare
bescherming tegen verlies van reeds bevestigd werk na een onbedoelde afsluiting.

Checkpoint / hervatten
----------------------
Na bevestigde bestandsuitkomsten wordt een klein checkpoint bijgewerkt. Bij een onderbroken grote controle kan
een passende run worden hervat zolang naam, grootte en wijzigingstijd van de bestanden nog overeenkomen.
Reeds definitief vastgelegde FAIL-resultaten blijven in het checkpoint bewaard en worden bij hervatten weer
meegenomen. Een nog niet definitief afgeronde batch kan bij hervatten opnieuw worden gecontroleerd.

Een volledig afgeronde controle zonder operationele ERROR verwijdert het checkpoint. Tijdens de echte controle
vraagt SyzygyCheck Windows om systeem-slaapstand tijdelijk te voorkomen; die instelling wordt na de run weer
vrijgegeven.

Resultaatrapport
----------------
Het gebruikersrapport wordt bij voorkeur op het Windows-bureaublad (Desktop) geplaatst; het eindscherm noemt
zowel die gewone locatie als het volledige pad. Normale OK-bestanden worden niet afzonderlijk opgesomd. Alleen
samenvatting, echte checksum-FAILs en operationele ERRORs worden vermeld.

Op dezelfde regel als de duur van de run staat de gemiddelde effectieve leessnelheid van de huidige sessie.
De rapportnaam gebruikt datum en uur/minuut:
SyzygyCheck_Result_YYYYMMDD_HHMM.txt; bij een uitzonderlijk naamconflict volgt _2, _3, enz.

Herstel bij FAIL/ERROR
---------------------
Bij een echte checksum-FAIL of operationele ERROR verwijst het desktoprapport naar de meegeleverde
SYZYGY_HERSTEL_NL.txt. De release bevat dezelfde herstelhandleiding in alle zeven talen. Daarin staan een korte
vervang-/hercontroleprocedure en twee downloadbronnen: de Lichess Syzygy mirror en de Sesse mirror.

Talen
-----
Ondersteund: Engels, Duits, Nederlands, Frans, Spaans, Chinees en Russisch. De gekozen taal geldt ook voor het
resultaatrapport. In ieder hoofdmenu blijft de optie om van taal te wisselen herkenbaar via Engelse/Chinese/
Russische bruglabels, zodat een gebruiker altijd uit een per ongeluk gekozen onbekend schrift kan terugkeren.
De fysieke Esc-toets is bovendien een echte terugfunctie in de keuzeschermen: in het hoofdmenu opent Esc de
taalkeuze; in taal-, pagefile-, thread- en hervatkeuzes gaat Esc terug zonder een controle te starten.

Ronald-helper
-------------
De bekende tbcheck.exe uit de v2.0.7-referentie heeft SHA-256:
9cb2f0ba2343f343bab29f9b9cd6aad6e4a2612702a93e06aa84a417f8eabbcc
Als exact die helper naast SyzygyCheck staat, wordt hij gebruikt. Anders haalt v2.0.9 tijdens de run tijdelijk
de vaste v3.0.7-kopie op, controleert de SHA-256 en verwijdert de tijdelijke helper na afloop. Voor de gebruiker
blijft SyzygyCheck zelf één EXE.

Praktijkvalidatie / status
--------------------------
V9F bouwt voort op de succesvol geteste V9C/V9D/V9E-lijn. De verificatiekern en threadwerking zijn niet gewijzigd.
De hoofdprompt is in alle zeven talen rustiger gemaakt; bij uren gebruikt Russisch `ч (h)` en Chinees `小时 (h)`.
De V9E-maatregel tegen SC_CLOSE blijft alleen een best-effort bescherming voor een klassiek consolevenster. Het
zichtbare kruisje van Windows Terminal behoort aan Terminal zelf en kan het venster nog steeds sluiten.
Checkpoint/hervatten blijft daarom de betrouwbare bescherming tegen verlies van reeds bevestigd werk.

Praktijkresultaten van V9C waarop V9F is gebaseerd:
- i7-6800K: 12 werkers, 57 bestanden / 20,22 GB, na herstart 27 s en gemiddeld 805,0 MB/s; de twee bedoelde
  checksumfouten werden gevonden en er waren 0 lees-/procesfouten.
- Drobo: controle werkt ook daar goed; met 4 threads is het mechanische geluid weer redelijk normaal en de
  snelheid praktisch acceptabel.
- De grote X299-proef op H:\W1 met 1511 RTBW-bestanden op 5× NVMe RAID0 blijft een nuttige schaaltest,
  maar de reeds behaalde functionele resultaten blokkeren het veilig bewaren/publiceren van deze stand niet.
