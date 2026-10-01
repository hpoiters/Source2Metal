using System.Security.Cryptography;
using System.Text;

namespace GamesCleaner;

internal static class SelfTest
{
    public static int Run()
    {
        string root = Path.Combine(Path.GetTempPath(), "GamesCleanerSelfTest_" + Guid.NewGuid().ToString("N"));

        try
        {
            Directory.CreateDirectory(root);
            string nested = Path.Combine(root, "Bronnen", "Submap");
            Directory.CreateDirectory(nested);

            string source = Path.Combine(nested, "testgames.pgn");
            File.WriteAllText(source, BuildSample(), Encoding.ASCII);

            string source2 = Path.Combine(nested, "testgames_extra.pgn");
            string edgeCases =
                BuildGame("Extra strong", 2750, 2720, "1-0", "900+5", 30) +
                BuildGame("Missing Event tag", 2720, 2710, "1-0", "900+5", 30, includeEvent: false) +
                BuildGame("Event not first", 2710, 2700, "1-0", "900+5", 25, eventFirst: false) +
                BuildGame("Bulletin Open", 2680, 2670, "1-0", "600+0", 25) +
                BuildGame("Zero base", 2680, 2670, "1-0", "0+1", 25);

            // Bewust: UTF-8 BOM + lege regels vóór de eerste header.
            File.WriteAllText(
                source2,
                Environment.NewLine + Environment.NewLine + edgeCases,
                new UTF8Encoding(encoderShouldEmitUTF8Identifier: true));

            string sourceHash1 = Sha256(source);
            string sourceHash2 = Sha256(source2);

            // Bewijs dat een oude resultaatmap niet opnieuw als bron wordt gevonden.
            string oldResult = Path.Combine(root, CleanerEngine.ResultFolderName, "oud");
            Directory.CreateDirectory(oldResult);
            File.WriteAllText(Path.Combine(oldResult, "StrongGames.pgn"), BuildGame(
                "MOET WORDEN OVERGESLAGEN", 2700, 2700, "1-0", "600+0", 25), Encoding.ASCII);

            var discovered = CleanerEngine.DiscoverPgnFiles(root);
            Require(discovered.Count == 2, $"Verwacht 2 bron-PGN's, gevonden: {discovered.Count}");
            Require(discovered.Any(x => x.Path.Equals(source, StringComparison.OrdinalIgnoreCase)),
                "Het eerste verwachte bron-PGN ontbreekt.");
            Require(discovered.Any(x => x.Path.Equals(source2, StringComparison.OrdinalIgnoreCase)),
                "Het tweede verwachte bron-PGN ontbreekt.");

            CleanerSettings MakeSettings(int workers) => new()
            {
                MinimumElo = 2400,
                MinimumFullMoves = 20,
                RejectBullet = true,
                RejectVeryFast = true,
                VeryFastBaseSeconds = 120,
                WorkerThreads = workers
            };

            string customOutputRoot = Path.Combine(root, "ZelfGekozenUitvoer_1thread");
            var result = CleanerEngine.Run(
                customOutputRoot,
                discovered,
                MakeSettings(1),
                progress: null,
                CancellationToken.None);

            Require(
                result.OutputDirectory.StartsWith(
                    Path.GetFullPath(customOutputRoot) + Path.DirectorySeparatorChar,
                    StringComparison.OrdinalIgnoreCase),
                "De uitvoer staat niet onder de gekozen uitvoermap.");

            var c = result.Counters;
            Require(!result.Cancelled, "Self-test werd onverwacht als geannuleerd gemarkeerd.");
            Require(c.TotalGames == 12, $"Totaal moet 12 zijn, is {c.TotalGames}.");
            Require(c.StrongGames == 6, $"StrongGames moet 6 zijn, is {c.StrongGames}.");
            Require(c.RejectedGames == 6, $"Afgekeurd moet 6 zijn, is {c.RejectedGames}.");
            Require(c.Bullet == 1, $"Bullet moet 1 zijn, is {c.Bullet}.");
            Require(c.VeryFast == 2, $"Zeer snel moet 2 zijn, is {c.VeryFast}.");
            Require(c.TooShort == 1, $"Te kort moet 1 zijn, is {c.TooShort}.");
            Require(c.LowElo == 1, $"Lage Elo moet 1 zijn, is {c.LowElo}.");
            Require(c.InvalidResult == 1, $"Ongeldige uitslag moet 1 zijn, is {c.InvalidResult}.");
            Require(c.Malformed == 0, $"Beschadigd moet 0 zijn, is {c.Malformed}.");
            Require(c.WithoutEventTag == 1,
                $"Partijen zonder Event-tag moet 1 zijn, is {c.WithoutEventTag}.");

            string strong = File.ReadAllText(result.StrongPath, Encoding.Latin1);
            string rejected = File.ReadAllText(result.RejectedPath, Encoding.Latin1);

            Require(CountGcids(strong) == 6, "StrongGames.pgn bevat niet precies 6 testpartijen.");
            Require(CountGcids(rejected) == 6, "Afgekeurd.pgn bevat niet precies 6 testpartijen.");
            Require(CountGcids(strong) + CountGcids(rejected) == 12,
                "StrongGames + Afgekeurd vormen niet de complete self-testverzameling.");

            Require(strong.Contains("[Event \"Event not first\"]", StringComparison.Ordinal),
                "Partij met Event-tag op afwijkende positie is niet als Strong behouden.");
            Require(strong.Contains("[Event \"Bulletin Open\"]", StringComparison.Ordinal),
                "Bulletin Open is ten onrechte als Bullet afgekeurd.");

            string customOutputRoot2 = Path.Combine(root, "ZelfGekozenUitvoer_2threads");
            var result2 = CleanerEngine.Run(
                customOutputRoot2,
                discovered,
                MakeSettings(2),
                progress: null,
                CancellationToken.None);

            Require(CountersEqual(result.Counters, result2.Counters),
                "1-thread en 2-thread run geven verschillende tellingen.");

            string strong2 = File.ReadAllText(result2.StrongPath, Encoding.Latin1);
            string rejected2 = File.ReadAllText(result2.RejectedPath, Encoding.Latin1);
            Require(GcidSet(strong).SetEquals(GcidSet(strong2)),
                "1-thread en 2-thread StrongGames bevatten verschillende GCID's.");
            Require(GcidSet(rejected).SetEquals(GcidSet(rejected2)),
                "1-thread en 2-thread Afgekeurd bevatten verschillende GCID's.");

            Require(Sha256(source) == sourceHash1,
                "Eerste bron-PGN is tijdens de test gewijzigd.");
            Require(Sha256(source2) == sourceHash2,
                "Tweede bron-PGN is tijdens de test gewijzigd.");

            Directory.Delete(root, recursive: true);
            return 0;
        }
        catch (Exception ex)
        {
            try
            {
                File.WriteAllText(
                    Path.Combine(Path.GetTempPath(), "GamesCleanerSelfTest_FAILED.txt"),
                    ex.ToString(),
                    Encoding.UTF8);
            }
            catch
            {
                // Geen tweede fout laten maskeren.
            }

            try
            {
                if (Directory.Exists(root))
                    Directory.Delete(root, recursive: true);
            }
            catch
            {
                // Diagnosebestand is belangrijker dan opschonen.
            }

            return 1;
        }
    }

    private static string BuildSample()
    {
        var sb = new StringBuilder();
        sb.Append(BuildGame("Strong classical", 2650, 2580, "1-0", "600+0", 25));
        sb.Append(BuildGame("Low Elo", 2399, 2600, "0-1", "600+0", 25));
        sb.Append(BuildGame("Rated Bullet game", 2700, 2680, "1-0", "60+0", 25));
        sb.Append(BuildGame("Very fast game", 2700, 2680, "1-0", "120+1", 25));
        sb.Append(BuildGame("Too short", 2700, 2680, "1-0", "600+0", 10));
        sb.Append(BuildGame("Unfinished", 2700, 2680, "*", "600+0", 25));
        sb.Append(BuildGame("Classical game", 2700, 2680, "1-0", "600+0", 25, "BulletMaster", "NormalPlayer"));
        return sb.ToString();
    }

    private static string BuildGame(
        string eventName,
        int whiteElo,
        int blackElo,
        string result,
        string timeControl,
        int moves,
        string white = "White",
        string black = "Black",
        bool includeEvent = true,
        bool eventFirst = true)
    {
        var sb = new StringBuilder();
        if (includeEvent && eventFirst)
            sb.AppendLine($"[Event \"{eventName}\"]");
        sb.AppendLine($"[GCID \"{eventName}\"]");
        sb.AppendLine("[Site \"SelfTest\"]");
        if (includeEvent && !eventFirst)
            sb.AppendLine($"[Event \"{eventName}\"]");
        sb.AppendLine("[Date \"2026.10.01\"]");
        sb.AppendLine("[Round \"1\"]");
        sb.AppendLine($"[White \"{white}\"]");
        sb.AppendLine($"[Black \"{black}\"]");
        sb.AppendLine($"[WhiteElo \"{whiteElo}\"]");
        sb.AppendLine($"[BlackElo \"{blackElo}\"]");
        sb.AppendLine($"[Result \"{result}\"]");
        sb.AppendLine($"[TimeControl \"{timeControl}\"]");
        sb.AppendLine();

        for (int move = 1; move <= moves; move++)
            sb.Append($"{move}. Nf3 Nf6 ");

        sb.Append(result);
        sb.AppendLine();
        sb.AppendLine();
        return sb.ToString();
    }

    private static int CountGcids(string text) => GcidSet(text).Count;

    private static HashSet<string> GcidSet(string text)
    {
        var result = new HashSet<string>(StringComparer.Ordinal);
        const string prefix = "[GCID \"";
        int index = 0;

        while ((index = text.IndexOf(prefix, index, StringComparison.Ordinal)) >= 0)
        {
            int start = index + prefix.Length;
            int end = text.IndexOf("\"]", start, StringComparison.Ordinal);
            if (end < 0)
                break;

            result.Add(text[start..end]);
            index = end + 2;
        }

        return result;
    }

    private static bool CountersEqual(CleanerCounters a, CleanerCounters b) =>
        a.TotalGames == b.TotalGames &&
        a.StrongGames == b.StrongGames &&
        a.RejectedGames == b.RejectedGames &&
        a.Bullet == b.Bullet &&
        a.VeryFast == b.VeryFast &&
        a.TooShort == b.TooShort &&
        a.MissingElo == b.MissingElo &&
        a.LowElo == b.LowElo &&
        a.InvalidResult == b.InvalidResult &&
        a.Malformed == b.Malformed &&
        a.WithoutEventTag == b.WithoutEventTag;

    private static string Sha256(string path)
    {
        using var stream = File.OpenRead(path);
        return Convert.ToHexString(SHA256.HashData(stream));
    }

    private static void Require(bool condition, string message)
    {
        if (!condition)
            throw new InvalidOperationException(message);
    }
}
