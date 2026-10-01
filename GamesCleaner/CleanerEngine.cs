using System.Diagnostics;
using System.Globalization;
using System.Text;

namespace GamesCleaner;

internal sealed record InputPgn(string Path, long SizeBytes)
{
    public string DisplayPath(string root) =>
        System.IO.Path.GetRelativePath(root, Path);

    public string DisplaySize =>
        SizeBytes >= 1024L * 1024 * 1024
            ? $"{SizeBytes / (1024d * 1024 * 1024):0.00} GB"
            : $"{SizeBytes / (1024d * 1024):0.0} MB";
}

internal sealed class CleanerSettings
{
    public int MinimumElo { get; init; } = 2400;
    public int MinimumFullMoves { get; init; } = 20;
    public bool RejectBullet { get; init; } = true;
    public bool RejectVeryFast { get; init; } = true;
    public int VeryFastBaseSeconds { get; init; } = 120;
    public int WorkerThreads { get; init; } = Math.Max(1, Environment.ProcessorCount / 2);
}

internal enum RejectReason
{
    None,
    Bullet,
    VeryFast,
    TooShort,
    MissingElo,
    LowElo,
    InvalidResult,
    Malformed
}

internal sealed class CleanerCounters
{
    public long TotalGames;
    public long StrongGames;
    public long RejectedGames;
    public long Bullet;
    public long VeryFast;
    public long TooShort;
    public long MissingElo;
    public long LowElo;
    public long InvalidResult;
    public long Malformed;
    public long WithoutEventTag;

    public void Count(RejectReason reason, bool hasEventTag)
    {
        TotalGames++;
        if (!hasEventTag)
            WithoutEventTag++;

        if (reason == RejectReason.None)
        {
            StrongGames++;
            return;
        }

        RejectedGames++;
        switch (reason)
        {
            case RejectReason.Bullet: Bullet++; break;
            case RejectReason.VeryFast: VeryFast++; break;
            case RejectReason.TooShort: TooShort++; break;
            case RejectReason.MissingElo: MissingElo++; break;
            case RejectReason.LowElo: LowElo++; break;
            case RejectReason.InvalidResult: InvalidResult++; break;
            case RejectReason.Malformed: Malformed++; break;
        }
    }
}

internal sealed class CleanerProgress
{
    public required string CurrentFile { get; init; }
    public required long ProcessedBytes { get; init; }
    public required long TotalBytes { get; init; }
    public required CleanerCounters Counters { get; init; }
    public required TimeSpan Elapsed { get; init; }
    public TimeSpan? Eta { get; init; }

    public double Percent => TotalBytes <= 0 ? 0 : Math.Clamp(ProcessedBytes * 100d / TotalBytes, 0, 100);
}

internal sealed class CleanerResult
{
    public required string OutputDirectory { get; init; }
    public required string StrongPath { get; init; }
    public required string RejectedPath { get; init; }
    public required string ReportPath { get; init; }
    public required CleanerCounters Counters { get; init; }
    public required TimeSpan Elapsed { get; init; }
    public required bool Cancelled { get; init; }
}

internal static class CleanerEngine
{
    internal const string ResultFolderName = "GamesCleaner_Resultaat";

    private static readonly string[] ExcludedDirectoryNames =
    {
        ResultFolderName, "Result", "Results", "Resultaat", "Resultaten",
        "Output", "Source2Metal_Output"
    };

    public static List<InputPgn> DiscoverPgnFiles(string root)
    {
        var result = new List<InputPgn>();
        var stack = new Stack<string>();
        stack.Push(root);

        while (stack.Count > 0)
        {
            var dir = stack.Pop();

            IEnumerable<string> subDirs;
            try { subDirs = Directory.EnumerateDirectories(dir); }
            catch { continue; }

            foreach (var sub in subDirs)
            {
                try
                {
                    var name = Path.GetFileName(sub);
                    if (ShouldSkipDirectory(name))
                        continue;

                    var attributes = File.GetAttributes(sub);
                    if ((attributes & FileAttributes.ReparsePoint) != 0)
                        continue;

                    stack.Push(sub);
                }
                catch
                {
                    // Een ontoegankelijke submap mag het zoeken niet blokkeren.
                }
            }

            IEnumerable<string> files;
            try { files = Directory.EnumerateFiles(dir, "*.pgn", SearchOption.TopDirectoryOnly); }
            catch { continue; }

            foreach (var file in files)
            {
                try
                {
                    var name = Path.GetFileName(file);
                    if (name.Equals("StrongGames.pgn", StringComparison.OrdinalIgnoreCase) ||
                        name.Equals("Afgekeurd.pgn", StringComparison.OrdinalIgnoreCase) ||
                        name.Contains("_ONVOLLEDIG", StringComparison.OrdinalIgnoreCase))
                        continue;

                    var fi = new FileInfo(file);
                    if (fi.Length > 0)
                        result.Add(new InputPgn(fi.FullName, fi.Length));
                }
                catch
                {
                    // Overslaan in de zoekfase; openen wordt later strikt gecontroleerd.
                }
            }
        }

        return result
            .OrderBy(x => x.Path, StringComparer.OrdinalIgnoreCase)
            .ToList();
    }

    private static bool ShouldSkipDirectory(string name)
    {
        if (name.StartsWith("GamesCleaner_", StringComparison.OrdinalIgnoreCase))
            return true;

        return ExcludedDirectoryNames.Any(x =>
            name.Equals(x, StringComparison.OrdinalIgnoreCase));
    }

    public static CleanerResult Run(
        string outputRootDirectory,
        IReadOnlyList<InputPgn> inputs,
        CleanerSettings settings,
        IProgress<CleanerProgress>? progress,
        CancellationToken cancellationToken)
    {
        if (inputs.Count == 0)
            throw new InvalidOperationException("Er zijn geen PGN-bestanden geselecteerd.");

        long totalBytes = inputs.Sum(x => x.SizeBytes);
        long processedBytes = 0;
        var counters = new CleanerCounters();
        var stopwatch = Stopwatch.StartNew();

        if (string.IsNullOrWhiteSpace(outputRootDirectory))
            throw new ArgumentException("De uitvoermap ontbreekt.", nameof(outputRootDirectory));

        var resultRoot = Path.GetFullPath(outputRootDirectory);
        Directory.CreateDirectory(resultRoot);

        var runDirectory = CreateUniqueRunDirectory(resultRoot);
        Directory.CreateDirectory(runDirectory);

        var strongTmp = Path.Combine(runDirectory, "StrongGames.tmp");
        var rejectedTmp = Path.Combine(runDirectory, "Afgekeurd.tmp");
        var strongFinal = Path.Combine(runDirectory, "StrongGames.pgn");
        var rejectedFinal = Path.Combine(runDirectory, "Afgekeurd.pgn");
        var reportPath = Path.Combine(runDirectory, "GamesCleaner_Report.txt");

        try
        {
            // Writers zitten in een eigen scope zodat de .tmp-bestanden zeker dicht zijn
            // vóór ze naar de definitieve PGN-namen worden verplaatst.
            using (var strongStream = new FileStream(
                strongTmp, FileMode.CreateNew, FileAccess.Write, FileShare.Read,
                1024 * 1024, FileOptions.SequentialScan))
            using (var rejectedStream = new FileStream(
                rejectedTmp, FileMode.CreateNew, FileAccess.Write, FileShare.Read,
                1024 * 1024, FileOptions.SequentialScan))
            using (var strongWriter = new StreamWriter(
                strongStream, Encoding.Latin1, 1024 * 1024, leaveOpen: false))
            using (var rejectedWriter = new StreamWriter(
                rejectedStream, Encoding.Latin1, 1024 * 1024, leaveOpen: false))
            {
                var outputLock = new object();
                int maxWorkers = Math.Max(1, settings.WorkerThreads);

                if (maxWorkers == 1 || inputs.Count == 1)
                {
                    foreach (var input in inputs)
                    {
                        cancellationToken.ThrowIfCancellationRequested();
                        ProcessOneFile(
                            input,
                            settings,
                            strongWriter,
                            rejectedWriter,
                            counters,
                            outputLock,
                            ref processedBytes,
                            totalBytes,
                            stopwatch,
                            progress,
                            cancellationToken);
                    }
                }
                else
                {
                    var options = new ParallelOptions
                    {
                        CancellationToken = cancellationToken,
                        MaxDegreeOfParallelism = Math.Min(maxWorkers, inputs.Count)
                    };

                    Parallel.ForEach(inputs, options, input =>
                    {
                        ProcessOneFile(
                            input,
                            settings,
                            strongWriter,
                            rejectedWriter,
                            counters,
                            outputLock,
                            ref processedBytes,
                            totalBytes,
                            stopwatch,
                            progress,
                            options.CancellationToken);
                    });
                }

                strongWriter.Flush();
                rejectedWriter.Flush();
            }

            File.Move(strongTmp, strongFinal);
            File.Move(rejectedTmp, rejectedFinal);

            stopwatch.Stop();
            WriteReport(
                reportPath, inputs, settings, counters, stopwatch.Elapsed,
                cancelled: false, strongFinal, rejectedFinal);

            ReportProgress(progress, inputs[^1].Path, totalBytes, totalBytes, counters, stopwatch.Elapsed);

            return new CleanerResult
            {
                OutputDirectory = runDirectory,
                StrongPath = strongFinal,
                RejectedPath = rejectedFinal,
                ReportPath = reportPath,
                Counters = counters,
                Elapsed = stopwatch.Elapsed,
                Cancelled = false
            };
        }
        catch (OperationCanceledException)
        {
            stopwatch.Stop();

            TryRenameIncomplete(strongTmp, Path.Combine(runDirectory, "StrongGames_ONVOLLEDIG.pgn"));
            TryRenameIncomplete(rejectedTmp, Path.Combine(runDirectory, "Afgekeurd_ONVOLLEDIG.pgn"));

            WriteReport(
                reportPath, inputs, settings, counters, stopwatch.Elapsed,
                cancelled: true,
                Path.Combine(runDirectory, "StrongGames_ONVOLLEDIG.pgn"),
                Path.Combine(runDirectory, "Afgekeurd_ONVOLLEDIG.pgn"));

            return new CleanerResult
            {
                OutputDirectory = runDirectory,
                StrongPath = Path.Combine(runDirectory, "StrongGames_ONVOLLEDIG.pgn"),
                RejectedPath = Path.Combine(runDirectory, "Afgekeurd_ONVOLLEDIG.pgn"),
                ReportPath = reportPath,
                Counters = counters,
                Elapsed = stopwatch.Elapsed,
                Cancelled = true
            };
        }
        catch
        {
            stopwatch.Stop();
            TryRenameIncomplete(strongTmp, Path.Combine(runDirectory, "StrongGames_ONVOLLEDIG.pgn"));
            TryRenameIncomplete(rejectedTmp, Path.Combine(runDirectory, "Afgekeurd_ONVOLLEDIG.pgn"));
            throw;
        }
    }

    private static void ProcessOneFile(
        InputPgn input,
        CleanerSettings settings,
        StreamWriter strongWriter,
        StreamWriter rejectedWriter,
        CleanerCounters counters,
        object outputLock,
        ref long processedBytes,
        long totalBytes,
        Stopwatch stopwatch,
        IProgress<CleanerProgress>? progress,
        CancellationToken cancellationToken)
    {
        int newlineBytes = DetectNewlineBytes(input.Path);
        string newline = newlineBytes == 1 ? "\n" : "\r\n";
        bool wholeFileBullet =
            Path.GetFileName(input.Path).Contains("bullet", StringComparison.OrdinalIgnoreCase);

        using var fs = new FileStream(
            input.Path, FileMode.Open, FileAccess.Read, FileShare.Read,
            1024 * 1024, FileOptions.SequentialScan);
        using var reader = new StreamReader(
            fs, Encoding.Latin1, detectEncodingFromByteOrderMarks: false,
            bufferSize: 1024 * 1024, leaveOpen: false);

        var game = new StringBuilder(4096);
        var meta = new GameMeta();
        bool firstLine = true;
        string? line;
        long localBytes = 0;
        TimeSpan lastProgress = stopwatch.Elapsed;

        while ((line = reader.ReadLine()) is not null)
        {
            cancellationToken.ThrowIfCancellationRequested();

            long lineBytes = line.Length + newlineBytes;
            localBytes += lineBytes;
            Interlocked.Add(ref processedBytes, lineBytes);

            bool headerLine = IsHeaderLine(line, firstLine);
            bool eventLine = IsEventLine(line, firstLine);

            // Normale PGN: [Event ...] begint een nieuwe partij.
            // Afwijkende PGN zonder Event-tag: zodra na reeds gelezen zetten weer
            // een headerregel begint, behandelen we dat eveneens als nieuwe partij.
            bool newGame =
                game.Length > 0 &&
                ((eventLine && meta.HasAnyHeader) ||
                 (headerLine && meta.SawMovetext));

            if (newGame)
            {
                FinalizeGame(game, meta, wholeFileBullet, settings, strongWriter, rejectedWriter, counters, outputLock);
                game.Clear();
                meta.Reset();
            }

            game.Append(line);
            game.Append(newline);
            meta.ObserveLine(line);

            firstLine = false;

            if (stopwatch.Elapsed - lastProgress >= TimeSpan.FromMilliseconds(250))
            {
                lastProgress = stopwatch.Elapsed;
                long nowBytes = Math.Min(totalBytes, Interlocked.Read(ref processedBytes));
                lock (outputLock)
                {
                    ReportProgress(progress, input.Path, nowBytes, totalBytes, counters, stopwatch.Elapsed);
                }
            }

            if (game.Length > 64 * 1024 * 1024)
            {
                // Beschermt tegen een zwaar beschadigd bestand zonder herkenbare partijgrenzen.
                meta.ForceMalformed = true;
                FinalizeGame(game, meta, wholeFileBullet, settings, strongWriter, rejectedWriter, counters, outputLock);
                game.Clear();
                meta.Reset();
            }
        }

        if (game.Length > 0)
            FinalizeGame(game, meta, wholeFileBullet, settings, strongWriter, rejectedWriter, counters, outputLock);

        // Maak de bytevoortgang per bronbestand exact, ook bij een ontbrekende slot-newline.
        long correction = input.SizeBytes - localBytes;
        if (correction != 0)
            Interlocked.Add(ref processedBytes, correction);

        long doneBytes = Math.Min(totalBytes, Interlocked.Read(ref processedBytes));
        lock (outputLock)
        {
            ReportProgress(progress, input.Path, doneBytes, totalBytes, counters, stopwatch.Elapsed);
        }
    }

    private static bool IsHeaderLine(string line, bool firstLine)
    {
        if (line.StartsWith("[", StringComparison.Ordinal))
            return true;

        if (firstLine && line.StartsWith("ï»¿[", StringComparison.Ordinal))
            return true;

        return false;
    }

    private static bool IsEventLine(string line, bool firstLine)
    {
        if (line.StartsWith("[Event ", StringComparison.Ordinal))
            return true;

        if (firstLine && line.Length > 10)
        {
            // UTF-8 BOM gelezen als Latin1: EF BB BF -> ï»¿
            if (line.StartsWith("ï»¿[Event ", StringComparison.Ordinal))
                return true;
        }

        return false;
    }

    private static void FinalizeGame(
        StringBuilder game,
        GameMeta meta,
        bool wholeFileBullet,
        CleanerSettings settings,
        StreamWriter strongWriter,
        StreamWriter rejectedWriter,
        CleanerCounters counters,
        object outputLock)
    {
        var reason = Classify(meta, wholeFileBullet, settings);

        lock (outputLock)
        {
            counters.Count(reason, meta.SawEvent);

            var writer = reason == RejectReason.None ? strongWriter : rejectedWriter;
            foreach (var chunk in game.GetChunks())
                writer.Write(chunk.Span);
            if (game.Length > 0 && game[^1] != '\n')
                writer.WriteLine();
        }
    }

    private static RejectReason Classify(GameMeta meta, bool wholeFileBullet, CleanerSettings settings)
    {
        if (meta.ForceMalformed || !meta.HasAnyHeader || !meta.SawMovetext)
            return RejectReason.Malformed;

        if (settings.RejectBullet && (wholeFileBullet || meta.BulletInHeaders))
            return RejectReason.Bullet;

        if (!IsValidResult(meta.Result))
            return RejectReason.InvalidResult;

        if (!meta.WhiteElo.HasValue || !meta.BlackElo.HasValue)
            return RejectReason.MissingElo;

        if (meta.WhiteElo.Value < settings.MinimumElo || meta.BlackElo.Value < settings.MinimumElo)
            return RejectReason.LowElo;

        if (settings.RejectVeryFast &&
            IsVeryFastTimeControl(meta.TimeControl, settings.VeryFastBaseSeconds))
            return RejectReason.VeryFast;

        if (meta.MaxMoveNumber < settings.MinimumFullMoves)
            return RejectReason.TooShort;

        return RejectReason.None;
    }

    private static bool IsValidResult(string? result) =>
        result is "1-0" or "0-1" or "1/2-1/2";

    private static bool IsVeryFastTimeControl(string? value, int maxBaseSeconds)
    {
        if (string.IsNullOrWhiteSpace(value))
            return false;

        var s = value.Trim();
        if (s is "-" or "?")
            return false;

        // Meerstaps tijdcontroles (bijv. 40/7200:3600) zijn geen bullet.
        if (s.Contains('/') || s.Contains(':'))
            return false;

        var plus = s.IndexOf('+');
        var basePart = plus >= 0 ? s[..plus] : s;

        if (!int.TryParse(basePart, NumberStyles.Integer, CultureInfo.InvariantCulture, out int seconds))
            return false;

        return seconds > 0 && seconds <= maxBaseSeconds;
    }

    private static int DetectNewlineBytes(string path)
    {
        using var fs = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.ReadWrite);
        var buffer = new byte[Math.Min(1024 * 1024, (int)Math.Min(int.MaxValue, fs.Length))];
        int read = fs.Read(buffer, 0, buffer.Length);
        int lf = 0;
        int crlf = 0;

        for (int i = 0; i < read; i++)
        {
            if (buffer[i] == (byte)'\n')
            {
                lf++;
                if (i > 0 && buffer[i - 1] == (byte)'\r')
                    crlf++;
            }
        }

        return lf > 0 && crlf * 2 >= lf ? 2 : 1;
    }

    private static void ReportProgress(
        IProgress<CleanerProgress>? progress,
        string currentFile,
        long processedBytes,
        long totalBytes,
        CleanerCounters counters,
        TimeSpan elapsed)
    {
        if (progress is null)
            return;

        TimeSpan? eta = null;
        if (processedBytes > 0 && totalBytes > processedBytes && elapsed.TotalSeconds > 0.5)
        {
            double bytesPerSecond = processedBytes / elapsed.TotalSeconds;
            if (bytesPerSecond > 0)
                eta = TimeSpan.FromSeconds((totalBytes - processedBytes) / bytesPerSecond);
        }

        progress.Report(new CleanerProgress
        {
            CurrentFile = currentFile,
            ProcessedBytes = processedBytes,
            TotalBytes = totalBytes,
            Counters = Snapshot(counters),
            Elapsed = elapsed,
            Eta = eta
        });
    }

    private static CleanerCounters Snapshot(CleanerCounters c) => new()
    {
        TotalGames = c.TotalGames,
        StrongGames = c.StrongGames,
        RejectedGames = c.RejectedGames,
        Bullet = c.Bullet,
        VeryFast = c.VeryFast,
        TooShort = c.TooShort,
        MissingElo = c.MissingElo,
        LowElo = c.LowElo,
        InvalidResult = c.InvalidResult,
        Malformed = c.Malformed,
        WithoutEventTag = c.WithoutEventTag
    };

    private static string CreateUniqueRunDirectory(string root)
    {
        string stamp = DateTime.Now.ToString("yyyy-MM-dd_HH-mm-ss", CultureInfo.InvariantCulture);
        string folderName = "GamesCleaner_" + stamp;
        string candidate = Path.Combine(root, folderName);
        int n = 2;
        while (Directory.Exists(candidate))
            candidate = Path.Combine(root, $"{folderName}_{n++}");
        return candidate;
    }

    private static void TryRenameIncomplete(string source, string destination)
    {
        try
        {
            if (File.Exists(source))
                File.Move(source, destination, overwrite: true);
        }
        catch
        {
            // Bij een harde I/O-fout laten we het .tmp-bestand staan voor diagnose.
        }
    }

    private static void WriteReport(
        string path,
        IReadOnlyList<InputPgn> inputs,
        CleanerSettings settings,
        CleanerCounters c,
        TimeSpan elapsed,
        bool cancelled,
        string strongPath,
        string rejectedPath)
    {
        using var w = new StreamWriter(path, false, new UTF8Encoding(false));
        w.WriteLine("GamesCleaner Report");
        w.WriteLine("===================");
        w.WriteLine($"Status                 : {(cancelled ? "AFGEBROKEN - uitvoer is ONVOLLEDIG" : "GESLAAGD")}");
        w.WriteLine($"Datum                   : {DateTime.Now:yyyy-MM-dd HH:mm:ss}");
        w.WriteLine($"Duur                    : {FormatDuration(elapsed)}");
        w.WriteLine();
        w.WriteLine("Instellingen");
        w.WriteLine("------------");
        w.WriteLine($"Minimum Elo beide       : {settings.MinimumElo}");
        w.WriteLine($"Minimum volledige zetten: {settings.MinimumFullMoves}");
        w.WriteLine($"Bullet afkeuren         : {(settings.RejectBullet ? "JA" : "NEE")}");
        w.WriteLine($"Zeer snel afkeuren      : {(settings.RejectVeryFast ? "JA" : "NEE")}");
        w.WriteLine($"Zeer snel t/m basis     : {settings.VeryFastBaseSeconds} seconden");
        w.WriteLine($"Max. werkthreads        : {settings.WorkerThreads}");
        w.WriteLine();
        w.WriteLine("Uitkomst");
        w.WriteLine("--------");
        w.WriteLine($"Totaal verwerkt         : {c.TotalGames:N0}");
        w.WriteLine($"StrongGames             : {c.StrongGames:N0}");
        w.WriteLine($"Afgekeurd               : {c.RejectedGames:N0}");
        w.WriteLine();
        w.WriteLine("Primaire afkeurreden");
        w.WriteLine("-------------------");
        w.WriteLine($"Bullet                  : {c.Bullet:N0}");
        w.WriteLine($"Zeer snelle tijdcontrole: {c.VeryFast:N0}");
        w.WriteLine($"Te kort                 : {c.TooShort:N0}");
        w.WriteLine($"Elo ontbreekt           : {c.MissingElo:N0}");
        w.WriteLine($"Elo te laag             : {c.LowElo:N0}");
        w.WriteLine($"Uitslag ongeldig/onaf   : {c.InvalidResult:N0}");
        w.WriteLine($"PGN afwijkend/beschadigd: {c.Malformed:N0}");
        w.WriteLine();
        w.WriteLine("Diagnostiek partijgrenzen");
        w.WriteLine("------------------------");
        w.WriteLine($"Partijen zonder Event-tag: {c.WithoutEventTag:N0}");
        w.WriteLine();
        w.WriteLine($"Uitvoermap              : {Path.GetDirectoryName(strongPath)}");
        w.WriteLine($"StrongGames-bestand     : {strongPath}");
        w.WriteLine($"Afgekeurd-bestand       : {rejectedPath}");
        w.WriteLine();
        w.WriteLine("Bronbestanden");
        w.WriteLine("-------------");
        foreach (var input in inputs)
            w.WriteLine($"{input.SizeBytes,15:N0} bytes  {input.Path}");
        w.WriteLine();
        w.WriteLine("Controle");
        w.WriteLine("--------");
        w.WriteLine("Iedere herkende ingelezen partij wordt exact naar één van de twee PGN-uitvoerbestanden geschreven.");
        w.WriteLine("De originele bronbestanden worden uitsluitend gelezen en nooit gewijzigd.");
    }

    internal static string FormatDuration(TimeSpan t)
    {
        if (t.TotalDays >= 1)
            return $"{(int)t.TotalDays}d {t:hh\\:mm\\:ss}";
        return t.ToString(@"hh\:mm\:ss");
    }

    private sealed class GameMeta
    {
        public int? WhiteElo;
        public int? BlackElo;
        public string? Result;
        public string? TimeControl;
        public bool BulletInHeaders;
        public bool HasAnyHeader;
        public int MaxMoveNumber;
        public bool SawEvent;
        public bool SawMovetext;
        public bool ForceMalformed;

        private bool _inHeaders = true;
        private int _braceDepth;
        private int _parenDepth;

        public void Reset()
        {
            WhiteElo = null;
            BlackElo = null;
            Result = null;
            TimeControl = null;
            BulletInHeaders = false;
            HasAnyHeader = false;
            MaxMoveNumber = 0;
            SawEvent = false;
            SawMovetext = false;
            ForceMalformed = false;
            _inHeaders = true;
            _braceDepth = 0;
            _parenDepth = 0;
        }

        public void ObserveLine(string line)
        {
            string inspect = line;
            if (inspect.StartsWith("ï»¿", StringComparison.Ordinal))
                inspect = inspect[3..];

            if (_inHeaders)
            {
                if (inspect.Length == 0)
                {
                    _inHeaders = false;
                    return;
                }

                if (inspect.StartsWith("[", StringComparison.Ordinal))
                {
                    HasAnyHeader = true;
                    if (inspect.StartsWith("[Event ", StringComparison.Ordinal))
                        SawEvent = true;

                    // Alleen velden die het speeltempo kunnen beschrijven.
                    // Een speler met bijvoorbeeld "Bullet" in de gebruikersnaam mag
                    // niet om die reden worden afgekeurd.
                    if ((TryReadTag(inspect, "Event", out var eventValue) ||
                         TryReadTag(inspect, "Site", out eventValue) ||
                         TryReadTag(inspect, "Speed", out eventValue) ||
                         TryReadTag(inspect, "TimeClass", out eventValue)) &&
                        eventValue.Contains("bullet", StringComparison.OrdinalIgnoreCase))
                    {
                        BulletInHeaders = true;
                    }

                    if (TryReadTag(inspect, "WhiteElo", out var whiteElo))
                        WhiteElo = ParseElo(whiteElo);
                    else if (TryReadTag(inspect, "BlackElo", out var blackElo))
                        BlackElo = ParseElo(blackElo);
                    else if (TryReadTag(inspect, "Result", out var result))
                        Result = result;
                    else if (TryReadTag(inspect, "TimeControl", out var tc))
                        TimeControl = tc;

                    return;
                }

                // Sommige PGN's missen de lege regel tussen tags en zetten.
                _inHeaders = false;
            }

            if (inspect.Length > 0)
            {
                SawMovetext = true;
                ScanMoveNumbers(inspect);
            }
        }

        private void ScanMoveNumbers(string line)
        {
            bool semicolonComment = false;

            for (int i = 0; i < line.Length;)
            {
                char ch = line[i];

                if (semicolonComment)
                    break;

                if (_braceDepth > 0)
                {
                    if (ch == '}') _braceDepth--;
                    i++;
                    continue;
                }

                if (ch == '{')
                {
                    _braceDepth++;
                    i++;
                    continue;
                }

                if (ch == ';')
                {
                    semicolonComment = true;
                    break;
                }

                if (ch == '(')
                {
                    _parenDepth++;
                    i++;
                    continue;
                }

                if (ch == ')')
                {
                    if (_parenDepth > 0) _parenDepth--;
                    i++;
                    continue;
                }

                if (_parenDepth == 0 && char.IsDigit(ch) &&
                    (i == 0 || !char.IsLetterOrDigit(line[i - 1])))
                {
                    int j = i;
                    int number = 0;
                    while (j < line.Length && char.IsDigit(line[j]))
                    {
                        number = number * 10 + (line[j] - '0');
                        j++;
                        if (number > 10000) break;
                    }

                    if (j < line.Length && line[j] == '.' && number > MaxMoveNumber)
                        MaxMoveNumber = number;

                    i = Math.Max(j, i + 1);
                    continue;
                }

                i++;
            }
        }

        private static int? ParseElo(string value)
        {
            if (int.TryParse(value, NumberStyles.Integer, CultureInfo.InvariantCulture, out int elo) &&
                elo > 0 && elo < 10000)
                return elo;
            return null;
        }

        private static bool TryReadTag(string line, string tag, out string value)
        {
            value = string.Empty;
            if (!line.StartsWith("[" + tag + " ", StringComparison.Ordinal))
                return false;

            int firstQuote = line.IndexOf('"');
            if (firstQuote < 0)
                return false;

            int lastQuote = line.LastIndexOf('"');
            if (lastQuote <= firstQuote)
                return false;

            value = line.Substring(firstQuote + 1, lastQuote - firstQuote - 1);
            return true;
        }
    }
}
