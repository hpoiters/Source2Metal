CB2PGN Builder v0.1.3
=====================

Doel
----
Eén Windows-programma voor een neutrale ChessBase -> RAW-PGN tussenlaag in
Source2Metal.

Ondersteund
-----------
* Oud CBH: .cbh + .cbg + .cbp + .cbt
* Nieuw 2CBH: .2cbh + .2cbg + .2lid
* Automatische formaatherkenning
* Hoofdvariant-only
* Legale zetcontrole en eigen SAN-opbouw
* Tijdgestempelde uitvoermap
* Rapport met aantallen, overgeslagen records en SHA-256
* --selftest

Belangrijkste wijziging t.o.v. v0.1.0
-------------------------------------
De eerste echte CBH-proef met StrongGames2017 vond 128.653 fysieke records,
maar v0.1.0 converteerde 0 partijen. Alle actieve records strandden op de
melding "CBH geometrie buiten bord".

v0.1.3 gooit die oude interpretatie weg en volgt nu rechtstreeks de door
de geintegreerde cbh2pgn-0.1 decoder van Dominik Klein
(2022, MIT). De kernvolgorde is letterlijk:

    token = (ruwe_CBG_byte - processed_moves) mod 256
      -> originele 234 één-byte zet-tabellen
      -> of 0x29 + twee gecodeerde bytes voor een multibyte-zet
      -> originele ChessBase-stuknummering / capture-verschuiving
      -> eigen legaal bord
      -> eigen SAN

Ook 0xDC/0x0C-variatiestack, 0x9F filler, null-moves, promoties en
niet-standaard beginstellingen volgen de referentiecode. De originele
cbh2pgn-0.1 bron staat ongewijzigd in de actuele bronboom, zodat de Go-port
later regel voor regel controleerbaar blijft.

Deze reparatie is intern/selftest-gevalideerd, maar de oude CBH-route blijft
EXP ERIMENTEEL totdat dezelfde StrongGames2017.cbh in de praktijk partijen en
ply correct oplevert en liefst tegen een ChessBase-PGN-export is vergeleken.
De 2CBH-route blijft ongewijzigd en bouwt voort op de eerdere 43/43 partijen,
3119/3119 ply, 0-overgeslagen validatie.

RAW blijft RAW
--------------
CB2PGN doet geen Metal-selectie, weging, winstfiltering of boekleren. De
bedoelde keten blijft voor deze tussenversie:

    CBH / 2CBH -> CB2PGN -> *_Raw.pgn -> Pgn2Metal -> Metal.pgn

De latere volwassen Builder krijgt daarnaast de gevraagde Raw/ en Metal/
uitvoertakken; v0.1.3 is eerst bedoeld om de binaire CBH-decoder betrouwbaar
te maken.

Gebruik
-------
Zet de EXE bij de ChessBase-bestandenset en dubbelklik. Bij meerdere complete
databases verschijnt een keuzemenu. De uitvoer staat onder:

    CB2PGN_output\YYYY-MM-DD_HH-MM-SS\

U kunt ook een .cbh/.cbg/.cbp/.cbt of .2cbh/.2cbg/.2lid op de EXE slepen.

WIJZIGING v0.1.3
----------------
- Legacy CBP/CBT tekstvelden worden nu, net als in de oorspronkelijke
  cbh2pgn-0.1 referentie, bij de EERSTE NUL-byte afgekapt.
- PGN tagwaarden worden bovendien strikt éénregelig en control-character-vrij
  geschreven. Dit richt zich op de parse-fouten die Scid/ChessBase nog zagen
  bij de verder vrijwel volledig geconverteerde StrongGames2017-test.
