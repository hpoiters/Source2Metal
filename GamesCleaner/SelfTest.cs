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

            // Bewijs dat een oude resultaatmap niet opnieuw als bron wordt gevonden.
            string oldResult = Path.Combine(root, CleanerEngine.ResultFolderName, "oud");
            Directory.CreateDirectory(oldResult);
            File.WriteAllText(Path.Combine(oldResult, "StrongGames.pgn"), BuildGame(
                "MOET WORDEN OVERGESLAGEN", 2700, 2700, "1-0", "600+0", 25), Encoding.ASCII);

            var discovered = CleanerEngine.DiscoverPgnFiles(root);
            Require(discovered.Count == 1, $"Verwacht 1 bron-PGN, gevonden: {discovered.Count}");
            Require(discovered[0].Path.Equals(source, StringComparison.OrdinalIgnoreCase),
                "De gevonden bron-PGN is niet het verwachte bestand.");

            var settings = new CleanerSettings
            {
                MinimumElo = 2400,
                MinimumFullMoves = 20,
                RejectBullet = true,
                RejectVeryFast = true,
                VeryFastBaseSeconds = 120
            };

            var result = CleanerEngine.Run(
                root,
                discovered,
                settings,
                progress: null,
                CancellationToken.None);

            var c = result.Counters;
            Require(!result.Cancelled, "Self-test werd onverwacht als geannuleerd gemarkeerd.");
            Require(c.TotalGames == 6, $"Totaal moet 6 zijn, is {c.TotalGames}.");
            Require(c.StrongGames == 1, $"StrongGames moet 1 zijn, is {c.StrongGames}.");
            Require(c.RejectedGames == 5, $"Afgekeurd moet 5 zijn, is {c.RejectedGames}.");
            Require(c.Bullet == 1, $"Bullet moet 1 zijn, is {c.Bullet}.");
            Require(c.VeryFast == 1, $"Zeer snel moet 1 zijn, is {c.VeryFast}.");
            Require(c.TooShort == 1, $"Te kort moet 1 zijn, is {c.TooShort}.");
            Require(c.LowElo == 1, $"Lage Elo moet 1 zijn, is {c.LowElo}.");
            Require(c.InvalidResult == 1, $"Ongeldige uitslag moet 1 zijn, is {c.InvalidResult}.");

            string strong = File.ReadAllText(result.StrongPath, Encoding.Latin1);
            string rejected = File.ReadAllText(result.RejectedPath, Encoding.Latin1);

            Require(CountEvents(strong) == 1, "StrongGames.pgn bevat niet precies 1 partij.");
            Require(CountEvents(rejected) == 5, "Afgekeurd.pgn bevat niet precies 5 partijen.");
            Require(CountEvents(strong) + CountEvents(rejected) == 6,
                "StrongGames + Afgekeurd vormen niet de complete self-testverzameling.");

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
        sb.Append(BuildGame("Fast without Bullet word", 2700, 2680, "1-0", "120+1", 25));
        sb.Append(BuildGame("Too short", 2700, 2680, "1-0", "600+0", 10));
        sb.Append(BuildGame("Unfinished", 2700, 2680, "*", "600+0", 25));
        return sb.ToString();
    }

    private static string BuildGame(
        string eventName,
        int whiteElo,
        int blackElo,
        string result,
        string timeControl,
        int moves)
    {
        var sb = new StringBuilder();
        sb.AppendLine($"[Event \"{eventName}\"]");
        sb.AppendLine("[Site \"SelfTest\"]");
        sb.AppendLine("[Date \"2026.10.01\"]");
        sb.AppendLine("[Round \"1\"]");
        sb.AppendLine("[White \"White\"]");
        sb.AppendLine("[Black \"Black\"]");
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

    private static int CountEvents(string text)
    {
        int count = 0;
        int index = 0;
        const string marker = "[Event \"";

        while ((index = text.IndexOf(marker, index, StringComparison.Ordinal)) >= 0)
        {
            count++;
            index += marker.Length;
        }

        return count;
    }

    private static void Require(bool condition, string message)
    {
        if (!condition)
            throw new InvalidOperationException(message);
    }
}
