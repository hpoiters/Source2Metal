# Source2Metal V2 — beta-specificatie

Versie: **2.2.0-beta4**

## Ontwerpprincipe
Bron → decoder/adapter → filtering/selectie → per-bron resultaat → samengevoegd
resultaat moet achteraf volledig herleidbaar zijn in Verkenner, console en
rapport. Een test wordt niet 'geslaagd gemaakt' door verborgen drempelverlaging.

## Broninventaris
Recursief vanaf EXE/root, nooit erboven. Ondersteund: PGN, complete CBH,
complete 2CBH en complete CTG. Eigen output, `_output`, TEMP/tijdelijk worden
overgeslagen. Archieven worden genegeerd.

## RAW
### GAME RAW
PGN/CBH/2CBH. CBH/2CBH gaan tijdelijk via CB2PGN v0.1.3. Per bron wordt RAW
bewaard. Daarna volgt één gezamenlijke GAME-RAW met exacte deduplicatie binnen
en tussen GAME-bronnen.

### CTG RAW
Per complete CTG-set een neutrale representatieve PGN. De oude 100.000-game
minimumvloer is in beta4 verwijderd; output volgt de werkelijke bronstatistiek
(met alleen een veiligheidsmaximum).

### ALLE Sources RAW
Alleen als GAME-RAW én CTG-RAW bestaan. Streaming concatenatie, bewust zonder
cross-class deduplicatie.

## GAME METAL
PGN/CBH/2CBH delen één transpositie-bewuste positiedatabase. De legaliteits- en
SAN-parser gebruikt hetzelfde compacte schaakbord als de CTG-route.

Beta4-policy (Gebalanceerd):
- min 32 waarnemingen per zet;
- min 4 beslissende uitslagen;
- score vanuit kleur aan zet: W=1, D=0.5, L=0;
- beste venster 0.035;
- max 2 voorkeuren per positie;
- modelpartij min 3 treffers en 80% dekking.

Uitvoerrecords zijn volledige echte bronpartijen met oorspronkelijke uitslag.
Geen AnchorPly-pseudopartijen, fictieve uitslagen of duplicatie als leergewicht.
Per-bron bijdragen worden bewaard plus exact samengevoegde GAME-METAL.

## CTG METAL
Eerst strikte Ctg2Metal-selectie. Zero-record krijgt concrete afwijsdiagnostiek.
Alleen als de diagnostiek `SmallBroadBasePotential > 0` én bruikbare W/D/L toont,
wordt zichtbaar de conservatieve Kleine/Brede-policy geprobeerd. Als ook die
nul geeft: veilig overslaan, geen lege PGN.

## METAL samenvoegen
GAME-METAL en CTG-METAL blijven in beta4 herkenbaar gescheiden omdat hun
leerrecord-semantiek verschilt. Geen blind alles-METAL-bestand totdat een
geharmoniseerde position-centric Evidence Engine dosis en semantiek gelijk kan
behandelen.

## Uitvoerstructuur
```
!Source2Metal_Output/<timestamp>/
  RAW/
    1 - Separate Sources - RAW PGNs/
    2 - Samengevoegde Sources - RAW PGNs/
    UITLEG - RAW RESULTATEN.txt
  METAL/
    1 - Separate Sources - METAL PGNs/
    2 - Samengevoegde Sources - METAL PGNs/
    UITLEG - METAL RESULTATEN.txt
    UITLEG - METAL ROUTES.txt
  RAPPORT/
    Source2Metal - Volledig Procesrapport.txt
    ...
  TEMP/  (na normale run verwijderd)
```

## Validatie
- RAW PGN structurele recordtelling.
- GAME-METAL: recordtelling + originele uitslag + verbod op AnchorPly/Weight +
  volledige legale hoofdvariant.
- CTG-METAL: geïntegreerde structurele verificatie + SHA-256.
- Checksums over eindproducten en rapporten.

## Console
Actieve progressregel past zich aan consolebreedte aan. Na voltooiing wordt de
lange dynamische regel gewist en alleen een korte eindregel afgedrukt, zodat
Windows Terminal na resize minder historische progressregels reflowt.

## Acceptatieprocedure
1. Kleine diagnostische bronnen.
2. Grotere kleine CTG om grenzen te zien.
3. **Perfect2023** als belangrijke CTG-acceptatietest/omslagpunt.
4. Daarna pas zeer grote bronnen zoals Fritz20 en grote PGN/CBH/2CBH-mixen.

Bij een mislukking eerst concrete diagnostiek verklaren; nooit blind filters
verruimen. Alleen aantoonbaar gecontroleerde verbeteringen promoveren.
