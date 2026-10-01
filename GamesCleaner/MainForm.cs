using System.Diagnostics;
using System.Globalization;

namespace GamesCleaner;

internal sealed class MainForm : Form
{
    private readonly string _baseDirectory = AppContext.BaseDirectory.TrimEnd(Path.DirectorySeparatorChar);

    private readonly DataGridView _filesGrid = new();
    private readonly NumericUpDown _minimumElo = new();
    private readonly NumericUpDown _minimumMoves = new();
    private readonly NumericUpDown _fastSeconds = new();
    private readonly CheckBox _rejectBullet = new();
    private readonly CheckBox _rejectFast = new();
    private readonly ProgressBar _progressBar = new();
    private readonly Label _progressText = new();
    private readonly Label _currentFile = new();
    private readonly Label _counts = new();
    private readonly Label _fileSummary = new();
    private readonly Button _refreshButton = new();
    private readonly Button _allOnButton = new();
    private readonly Button _allOffButton = new();
    private readonly Button _startButton = new();
    private readonly Button _cancelButton = new();
    private readonly Button _openResultButton = new();

    private CancellationTokenSource? _cts;
    private string? _lastOutputDirectory;
    private bool _running;

    public MainForm()
    {
        Text = "GamesCleaner";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(860, 600);
        Size = new Size(1120, 760);

        BuildUi();
        Shown += (_, _) => RefreshFiles();
        FormClosing += OnFormClosing;
    }

    private void BuildUi()
    {
        var root = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            Padding = new Padding(12),
            ColumnCount = 1,
            RowCount = 6
        };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        Controls.Add(root);

        var titlePanel = new TableLayoutPanel
        {
            Dock = DockStyle.Top,
            AutoSize = true,
            ColumnCount = 1,
            Margin = new Padding(0, 0, 0, 8)
        };

        var title = new Label
        {
            AutoSize = true,
            Font = new Font(Font, FontStyle.Bold),
            Text = "GamesCleaner — sterke partijen bewaren, afgekeurde partijen apart houden"
        };
        titlePanel.Controls.Add(title);

        var location = new Label
        {
            AutoSize = true,
            MaximumSize = new Size(1000, 0),
            Text = $"Hoofdmap: {_baseDirectory}\r\nEr wordt alleen in deze map en submappen gezocht. Bovenliggende mappen en resultaatmappen worden niet gebruikt."
        };
        titlePanel.Controls.Add(location);
        root.Controls.Add(titlePanel, 0, 0);

        var settingsBox = new GroupBox
        {
            Text = "Selectieregels",
            Dock = DockStyle.Top,
            AutoSize = true,
            Padding = new Padding(10),
            Margin = new Padding(0, 0, 0, 8)
        };

        var settingsFlow = new FlowLayoutPanel
        {
            Dock = DockStyle.Fill,
            AutoSize = true,
            WrapContents = true
        };

        settingsFlow.Controls.Add(MakeLabel("Minimum Elo beide:"));
        _minimumElo.Minimum = 0;
        _minimumElo.Maximum = 4000;
        _minimumElo.Value = 2400;
        _minimumElo.Width = 80;
        settingsFlow.Controls.Add(_minimumElo);

        settingsFlow.Controls.Add(MakeSpacer());
        settingsFlow.Controls.Add(MakeLabel("Minimum zetten:"));
        _minimumMoves.Minimum = 0;
        _minimumMoves.Maximum = 500;
        _minimumMoves.Value = 20;
        _minimumMoves.Width = 70;
        settingsFlow.Controls.Add(_minimumMoves);

        settingsFlow.Controls.Add(MakeSpacer());
        _rejectBullet.Text = "Bullet afkeuren";
        _rejectBullet.Checked = true;
        _rejectBullet.AutoSize = true;
        settingsFlow.Controls.Add(_rejectBullet);

        settingsFlow.Controls.Add(MakeSpacer());
        _rejectFast.Text = "Zeer snelle tijdcontrole afkeuren";
        _rejectFast.Checked = true;
        _rejectFast.AutoSize = true;
        settingsFlow.Controls.Add(_rejectFast);

        settingsFlow.Controls.Add(MakeLabel("t/m basis-seconden:"));
        _fastSeconds.Minimum = 1;
        _fastSeconds.Maximum = 1800;
        _fastSeconds.Value = 120;
        _fastSeconds.Width = 70;
        settingsFlow.Controls.Add(_fastSeconds);

        settingsBox.Controls.Add(settingsFlow);
        root.Controls.Add(settingsBox, 0, 1);

        var fileToolbar = new TableLayoutPanel
        {
            Dock = DockStyle.Top,
            AutoSize = true,
            ColumnCount = 5,
            Margin = new Padding(0, 0, 0, 4)
        };
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));

        _refreshButton.Text = "PGN's opnieuw zoeken";
        _refreshButton.AutoSize = true;
        _refreshButton.Click += (_, _) => RefreshFiles();
        fileToolbar.Controls.Add(_refreshButton, 0, 0);

        _allOnButton.Text = "Alles aan";
        _allOnButton.AutoSize = true;
        _allOnButton.Margin = new Padding(8, 3, 0, 3);
        _allOnButton.Click += (_, _) => SetAllChecked(true);
        fileToolbar.Controls.Add(_allOnButton, 1, 0);

        _allOffButton.Text = "Alles uit";
        _allOffButton.AutoSize = true;
        _allOffButton.Margin = new Padding(8, 3, 0, 3);
        _allOffButton.Click += (_, _) => SetAllChecked(false);
        fileToolbar.Controls.Add(_allOffButton, 2, 0);

        _fileSummary.AutoSize = true;
        _fileSummary.Anchor = AnchorStyles.Right;
        _fileSummary.TextAlign = ContentAlignment.MiddleRight;
        fileToolbar.Controls.Add(_fileSummary, 4, 0);

        root.Controls.Add(fileToolbar, 0, 2);

        ConfigureGrid();
        root.Controls.Add(_filesGrid, 0, 3);

        var statusBox = new GroupBox
        {
            Text = "Voortgang",
            Dock = DockStyle.Top,
            AutoSize = true,
            Padding = new Padding(10),
            Margin = new Padding(0, 8, 0, 8)
        };

        var statusLayout = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            AutoSize = true,
            ColumnCount = 1,
            RowCount = 4
        };

        _currentFile.AutoEllipsis = true;
        _currentFile.Dock = DockStyle.Fill;
        _currentFile.Text = "Gereed.";
        statusLayout.Controls.Add(_currentFile, 0, 0);

        _progressBar.Dock = DockStyle.Top;
        _progressBar.Minimum = 0;
        _progressBar.Maximum = 1000;
        _progressBar.Height = 22;
        _progressBar.Margin = new Padding(0, 5, 0, 2);
        statusLayout.Controls.Add(_progressBar, 0, 1);

        _progressText.AutoSize = true;
        _progressText.Text = "0,0%";
        statusLayout.Controls.Add(_progressText, 0, 2);

        _counts.AutoSize = true;
        _counts.Text = "Partijen: 0   StrongGames: 0   Afgekeurd: 0";
        statusLayout.Controls.Add(_counts, 0, 3);

        statusBox.Controls.Add(statusLayout);
        root.Controls.Add(statusBox, 0, 4);

        var buttons = new FlowLayoutPanel
        {
            Dock = DockStyle.Fill,
            AutoSize = true,
            FlowDirection = FlowDirection.RightToLeft,
            WrapContents = false
        };

        _startButton.Text = "Start GamesCleaner";
        _startButton.AutoSize = true;
        _startButton.Padding = new Padding(10, 4, 10, 4);
        _startButton.Click += async (_, _) => await StartCleaningAsync();
        buttons.Controls.Add(_startButton);

        _cancelButton.Text = "Annuleren";
        _cancelButton.AutoSize = true;
        _cancelButton.Padding = new Padding(10, 4, 10, 4);
        _cancelButton.Enabled = false;
        _cancelButton.Click += (_, _) => _cts?.Cancel();
        buttons.Controls.Add(_cancelButton);

        _openResultButton.Text = "Open resultaatmap";
        _openResultButton.AutoSize = true;
        _openResultButton.Padding = new Padding(10, 4, 10, 4);
        _openResultButton.Enabled = false;
        _openResultButton.Click += (_, _) => OpenResultDirectory();
        buttons.Controls.Add(_openResultButton);

        root.Controls.Add(buttons, 0, 5);
    }

    private void ConfigureGrid()
    {
        _filesGrid.Dock = DockStyle.Fill;
        _filesGrid.AllowUserToAddRows = false;
        _filesGrid.AllowUserToDeleteRows = false;
        _filesGrid.AllowUserToResizeRows = false;
        _filesGrid.MultiSelect = false;
        _filesGrid.RowHeadersVisible = false;
        _filesGrid.SelectionMode = DataGridViewSelectionMode.FullRowSelect;
        _filesGrid.AutoGenerateColumns = false;
        _filesGrid.BackgroundColor = SystemColors.Window;
        _filesGrid.BorderStyle = BorderStyle.Fixed3D;

        var useColumn = new DataGridViewCheckBoxColumn
        {
            HeaderText = "Gebruik",
            Width = 58,
            Frozen = true
        };
        _filesGrid.Columns.Add(useColumn);

        _filesGrid.Columns.Add(new DataGridViewTextBoxColumn
        {
            HeaderText = "PGN-bestand",
            AutoSizeMode = DataGridViewAutoSizeColumnMode.Fill,
            MinimumWidth = 280,
            ReadOnly = true
        });

        _filesGrid.Columns.Add(new DataGridViewTextBoxColumn
        {
            HeaderText = "Grootte",
            Width = 100,
            ReadOnly = true
        });
    }

    private static Label MakeLabel(string text) => new()
    {
        AutoSize = true,
        Text = text,
        Margin = new Padding(3, 7, 3, 3)
    };

    private static Control MakeSpacer() => new Panel
    {
        Width = 14,
        Height = 1,
        Margin = Padding.Empty
    };

    private void RefreshFiles()
    {
        if (_running)
            return;

        Cursor = Cursors.WaitCursor;
        try
        {
            var files = CleanerEngine.DiscoverPgnFiles(_baseDirectory);

            _filesGrid.Rows.Clear();

            // Als er een samengevoegde GAME-bron aanwezig is, selecteer die standaard.
            // Zo worden de onderliggende bron-PGN's niet per ongeluk nogmaals meegeteld.
            bool hasMergedGame = files.Any(f =>
                Path.GetFileName(f.Path).Contains("Merged GAME Sources", StringComparison.OrdinalIgnoreCase));

            foreach (var file in files)
            {
                bool selected = !hasMergedGame ||
                    Path.GetFileName(file.Path).Contains("Merged GAME Sources", StringComparison.OrdinalIgnoreCase);

                int rowIndex = _filesGrid.Rows.Add(
                    selected,
                    file.DisplayPath(_baseDirectory),
                    file.DisplaySize);
                _filesGrid.Rows[rowIndex].Tag = file;
            }

            UpdateFileSummary();
        }
        finally
        {
            Cursor = Cursors.Default;
        }
    }

    private void SetAllChecked(bool value)
    {
        foreach (DataGridViewRow row in _filesGrid.Rows)
            row.Cells[0].Value = value;
        UpdateFileSummary();
    }

    private void UpdateFileSummary()
    {
        var selected = GetSelectedFiles();
        long bytes = selected.Sum(x => x.SizeBytes);
        _fileSummary.Text = $"{selected.Count:N0} geselecteerd — {FormatBytes(bytes)}";
    }

    private List<InputPgn> GetSelectedFiles()
    {
        _filesGrid.EndEdit();
        var result = new List<InputPgn>();

        foreach (DataGridViewRow row in _filesGrid.Rows)
        {
            bool selected = row.Cells[0].Value is bool b && b;
            if (selected && row.Tag is InputPgn input)
                result.Add(input);
        }

        return result;
    }

    private async Task StartCleaningAsync()
    {
        if (_running)
            return;

        var inputs = GetSelectedFiles();
        if (inputs.Count == 0)
        {
            MessageBox.Show(
                this,
                "Selecteer ten minste één PGN-bestand.",
                "GamesCleaner",
                MessageBoxButtons.OK,
                MessageBoxIcon.Information);
            return;
        }

        var settings = new CleanerSettings
        {
            MinimumElo = (int)_minimumElo.Value,
            MinimumFullMoves = (int)_minimumMoves.Value,
            RejectBullet = _rejectBullet.Checked,
            RejectVeryFast = _rejectFast.Checked,
            VeryFastBaseSeconds = (int)_fastSeconds.Value
        };

        _cts = new CancellationTokenSource();
        SetRunning(true);
        _lastOutputDirectory = null;
        _openResultButton.Enabled = false;
        _progressBar.Value = 0;
        _progressText.Text = "0,0%";
        _counts.Text = "Partijen: 0   StrongGames: 0   Afgekeurd: 0";

        var progress = new Progress<CleanerProgress>(UpdateProgress);

        try
        {
            CleanerResult result = await Task.Run(() =>
                CleanerEngine.Run(
                    _baseDirectory,
                    inputs,
                    settings,
                    progress,
                    _cts.Token));

            _lastOutputDirectory = result.OutputDirectory;
            _openResultButton.Enabled = true;

            if (result.Cancelled)
            {
                _currentFile.Text = "Afgebroken. De gedeeltelijke bestanden zijn als ONVOLLEDIG gemarkeerd.";
                MessageBox.Show(
                    this,
                    "De verwerking is afgebroken.\r\n\r\nGedeeltelijke uitvoer is bewaard met _ONVOLLEDIG in de bestandsnaam.",
                    "GamesCleaner",
                    MessageBoxButtons.OK,
                    MessageBoxIcon.Information);
            }
            else
            {
                _progressBar.Value = _progressBar.Maximum;
                _progressText.Text = $"100,0% — klaar in {CleanerEngine.FormatDuration(result.Elapsed)}";
                _currentFile.Text = $"Klaar. Resultaat: {result.OutputDirectory}";
                _counts.Text =
                    $"Partijen: {result.Counters.TotalGames:N0}   " +
                    $"StrongGames: {result.Counters.StrongGames:N0}   " +
                    $"Afgekeurd: {result.Counters.RejectedGames:N0}";

                MessageBox.Show(
                    this,
                    $"Klaar.\r\n\r\n" +
                    $"Totaal: {result.Counters.TotalGames:N0}\r\n" +
                    $"StrongGames: {result.Counters.StrongGames:N0}\r\n" +
                    $"Afgekeurd: {result.Counters.RejectedGames:N0}\r\n\r\n" +
                    "StrongGames.pgn en Afgekeurd.pgn vormen samen de complete verwerkte verzameling.",
                    "GamesCleaner",
                    MessageBoxButtons.OK,
                    MessageBoxIcon.Information);
            }
        }
        catch (Exception ex)
        {
            _currentFile.Text = "Fout — verwerking gestopt.";
            MessageBox.Show(
                this,
                "GamesCleaner is gestopt omdat een fout optrad.\r\nDe bronbestanden zijn niet gewijzigd.\r\n\r\n" + ex.Message,
                "GamesCleaner — fout",
                MessageBoxButtons.OK,
                MessageBoxIcon.Error);
        }
        finally
        {
            _cts.Dispose();
            _cts = null;
            SetRunning(false);
        }
    }

    private void UpdateProgress(CleanerProgress p)
    {
        int value = (int)Math.Round(p.Percent * 10);
        _progressBar.Value = Math.Clamp(value, _progressBar.Minimum, _progressBar.Maximum);

        string eta = p.Eta.HasValue
            ? $" — resterend ca. {CleanerEngine.FormatDuration(p.Eta.Value)}"
            : string.Empty;

        _progressText.Text =
            $"{p.Percent:0.0}% — {FormatBytes(p.ProcessedBytes)} / {FormatBytes(p.TotalBytes)}" +
            $" — verstreken {CleanerEngine.FormatDuration(p.Elapsed)}{eta}";

        _currentFile.Text = "Nu: " + Path.GetRelativePath(_baseDirectory, p.CurrentFile);
        _counts.Text =
            $"Partijen: {p.Counters.TotalGames:N0}   " +
            $"StrongGames: {p.Counters.StrongGames:N0}   " +
            $"Afgekeurd: {p.Counters.RejectedGames:N0}";
    }

    private void SetRunning(bool running)
    {
        _running = running;

        _refreshButton.Enabled = !running;
        _allOnButton.Enabled = !running;
        _allOffButton.Enabled = !running;
        _startButton.Enabled = !running;
        _cancelButton.Enabled = running;

        _filesGrid.Enabled = !running;
        _minimumElo.Enabled = !running;
        _minimumMoves.Enabled = !running;
        _rejectBullet.Enabled = !running;
        _rejectFast.Enabled = !running;
        _fastSeconds.Enabled = !running;
    }

    private void OpenResultDirectory()
    {
        if (string.IsNullOrWhiteSpace(_lastOutputDirectory) ||
            !Directory.Exists(_lastOutputDirectory))
            return;

        Process.Start(new ProcessStartInfo
        {
            FileName = _lastOutputDirectory,
            UseShellExecute = true
        });
    }

    private void OnFormClosing(object? sender, FormClosingEventArgs e)
    {
        if (!_running)
            return;

        var answer = MessageBox.Show(
            this,
            "GamesCleaner is nog bezig. Wilt u de verwerking annuleren en afsluiten?",
            "GamesCleaner",
            MessageBoxButtons.YesNo,
            MessageBoxIcon.Warning);

        if (answer == DialogResult.No)
        {
            e.Cancel = true;
            return;
        }

        _cts?.Cancel();
        e.Cancel = true;
        _currentFile.Text = "Annuleren… wacht tot de huidige partij veilig is weggeschreven.";
    }

    private static string FormatBytes(long bytes)
    {
        if (bytes >= 1024L * 1024 * 1024)
            return $"{bytes / (1024d * 1024 * 1024):0.00} GB";
        if (bytes >= 1024L * 1024)
            return $"{bytes / (1024d * 1024):0.0} MB";
        if (bytes >= 1024)
            return $"{bytes / 1024d:0.0} kB";
        return $"{bytes:N0} bytes";
    }
}
