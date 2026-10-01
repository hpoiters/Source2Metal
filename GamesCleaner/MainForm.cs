using System.Diagnostics;

namespace GamesCleaner;

internal sealed class MainForm : Form
{
    private readonly string _baseDirectory =
        AppContext.BaseDirectory.TrimEnd(Path.DirectorySeparatorChar);

    private readonly HashSet<string> _manualPgnPaths =
        new(StringComparer.OrdinalIgnoreCase);

    private string _outputRootDirectory;

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
    private readonly TextBox _outputPathBox = new();

    private readonly Button _refreshButton = new();
    private readonly Button _addPgnButton = new();
    private readonly Button _allOnButton = new();
    private readonly Button _allOffButton = new();
    private readonly Button _chooseOutputButton = new();
    private readonly Button _defaultOutputButton = new();
    private readonly Button _startButton = new();
    private readonly Button _cancelButton = new();
    private readonly Button _openResultButton = new();

    private CancellationTokenSource? _cts;
    private string? _lastOutputDirectory;
    private bool _running;

    private string DefaultOutputRoot =>
        Path.Combine(_baseDirectory, CleanerEngine.ResultFolderName);

    public MainForm()
    {
        _outputRootDirectory = DefaultOutputRoot;

        Text = "GamesCleaner v2";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(900, 640);
        Size = new Size(1180, 800);

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
            RowCount = 7
        };

        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
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

        titlePanel.Controls.Add(new Label
        {
            AutoSize = true,
            Font = new Font(Font, FontStyle.Bold),
            Text = "GamesCleaner v2 — sterke partijen bewaren, afgekeurde partijen apart houden"
        });

        titlePanel.Controls.Add(new Label
        {
            AutoSize = true,
            MaximumSize = new Size(1120, 0),
            Text =
                $"Automatisch zoeken vanaf: {_baseDirectory}\r\n" +
                "Daarnaast kunt u met ‘PGN's toevoegen…’ bestanden van elke andere map of schijf kiezen."
        });

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
            ColumnCount = 6,
            Margin = new Padding(0, 0, 0, 4)
        };

        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        fileToolbar.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));

        _refreshButton.Text = "PGN's opnieuw zoeken";
        _refreshButton.AutoSize = true;
        _refreshButton.Click += (_, _) => RefreshFiles();
        fileToolbar.Controls.Add(_refreshButton, 0, 0);

        _addPgnButton.Text = "PGN's toevoegen…";
        _addPgnButton.AutoSize = true;
        _addPgnButton.Margin = new Padding(8, 3, 0, 3);
        _addPgnButton.Click += (_, _) => AddPgnFiles();
        fileToolbar.Controls.Add(_addPgnButton, 1, 0);

        _allOnButton.Text = "Alles aan";
        _allOnButton.AutoSize = true;
        _allOnButton.Margin = new Padding(8, 3, 0, 3);
        _allOnButton.Click += (_, _) => SetAllChecked(true);
        fileToolbar.Controls.Add(_allOnButton, 2, 0);

        _allOffButton.Text = "Alles uit";
        _allOffButton.AutoSize = true;
        _allOffButton.Margin = new Padding(8, 3, 0, 3);
        _allOffButton.Click += (_, _) => SetAllChecked(false);
        fileToolbar.Controls.Add(_allOffButton, 3, 0);

        _fileSummary.AutoSize = true;
        _fileSummary.Anchor = AnchorStyles.Right;
        _fileSummary.TextAlign = ContentAlignment.MiddleRight;
        fileToolbar.Controls.Add(_fileSummary, 5, 0);

        root.Controls.Add(fileToolbar, 0, 2);

        ConfigureGrid();
        root.Controls.Add(_filesGrid, 0, 3);

        var outputBox = new GroupBox
        {
            Text = "Uitvoer",
            Dock = DockStyle.Top,
            AutoSize = true,
            Padding = new Padding(10),
            Margin = new Padding(0, 8, 0, 0)
        };

        var outputLayout = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            AutoSize = true,
            ColumnCount = 4,
            RowCount = 1
        };

        outputLayout.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        outputLayout.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        outputLayout.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        outputLayout.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));

        outputLayout.Controls.Add(MakeLabel("Uitvoermap:"), 0, 0);

        _outputPathBox.ReadOnly = true;
        _outputPathBox.Dock = DockStyle.Fill;
        _outputPathBox.Text = _outputRootDirectory;
        _outputPathBox.Margin = new Padding(6, 3, 6, 3);
        outputLayout.Controls.Add(_outputPathBox, 1, 0);

        _chooseOutputButton.Text = "Uitvoermap kiezen…";
        _chooseOutputButton.AutoSize = true;
        _chooseOutputButton.Click += (_, _) => ChooseOutputDirectory();
        outputLayout.Controls.Add(_chooseOutputButton, 2, 0);

        _defaultOutputButton.Text = "Standaard";
        _defaultOutputButton.AutoSize = true;
        _defaultOutputButton.Margin = new Padding(8, 3, 0, 3);
        _defaultOutputButton.Click += (_, _) => ResetOutputDirectory();
        outputLayout.Controls.Add(_defaultOutputButton, 3, 0);

        outputBox.Controls.Add(outputLayout);
        root.Controls.Add(outputBox, 0, 4);

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
        root.Controls.Add(statusBox, 0, 5);

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

        root.Controls.Add(buttons, 0, 6);
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

        _filesGrid.CurrentCellDirtyStateChanged += (_, _) =>
        {
            if (_filesGrid.IsCurrentCellDirty)
                _filesGrid.CommitEdit(DataGridViewDataErrorContexts.Commit);
        };
        _filesGrid.CellValueChanged += (_, e) =>
        {
            if (e.ColumnIndex == 0 && e.RowIndex >= 0)
                UpdateFileSummary();
        };

        _filesGrid.Columns.Add(new DataGridViewCheckBoxColumn
        {
            HeaderText = "Gebruik",
            Width = 58,
            Frozen = true
        });

        _filesGrid.Columns.Add(new DataGridViewTextBoxColumn
        {
            HeaderText = "PGN-bestand",
            AutoSizeMode = DataGridViewAutoSizeColumnMode.Fill,
            MinimumWidth = 320,
            ReadOnly = true
        });

        _filesGrid.Columns.Add(new DataGridViewTextBoxColumn
        {
            HeaderText = "Grootte",
            Width = 105,
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

        var previousChecks = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase);
        foreach (DataGridViewRow row in _filesGrid.Rows)
        {
            if (row.Tag is InputPgn input)
                previousChecks[input.Path] = row.Cells[0].Value is bool b && b;
        }

        Cursor = Cursors.WaitCursor;
        try
        {
            var byPath = new Dictionary<string, InputPgn>(StringComparer.OrdinalIgnoreCase);

            foreach (var file in CleanerEngine.DiscoverPgnFiles(_baseDirectory))
                byPath[file.Path] = file;

            foreach (string path in _manualPgnPaths.ToArray())
            {
                try
                {
                    if (!File.Exists(path))
                    {
                        _manualPgnPaths.Remove(path);
                        continue;
                    }

                    var fi = new FileInfo(path);
                    if (fi.Length > 0)
                        byPath[fi.FullName] = new InputPgn(fi.FullName, fi.Length);
                }
                catch
                {
                    _manualPgnPaths.Remove(path);
                }
            }

            var files = byPath.Values
                .OrderBy(f => f.Path, StringComparer.OrdinalIgnoreCase)
                .ToList();

            _filesGrid.Rows.Clear();

            bool hasMergedGame = files.Any(f =>
                Path.GetFileName(f.Path).Contains(
                    "Merged GAME Sources",
                    StringComparison.OrdinalIgnoreCase));

            foreach (var file in files)
            {
                bool selected;
                if (previousChecks.TryGetValue(file.Path, out bool wasChecked))
                {
                    selected = wasChecked;
                }
                else if (_manualPgnPaths.Contains(file.Path))
                {
                    selected = true;
                }
                else
                {
                    selected = !hasMergedGame ||
                        Path.GetFileName(file.Path).Contains(
                            "Merged GAME Sources",
                            StringComparison.OrdinalIgnoreCase);
                }

                int rowIndex = _filesGrid.Rows.Add(
                    selected,
                    DisplaySourcePath(file.Path),
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

    private void AddPgnFiles()
    {
        if (_running)
            return;

        using var dialog = new OpenFileDialog
        {
            Title = "PGN-bestanden toevoegen",
            Filter = "PGN-bestanden (*.pgn)|*.pgn|Alle bestanden (*.*)|*.*",
            Multiselect = true,
            CheckFileExists = true,
            CheckPathExists = true,
            InitialDirectory = Directory.Exists(_baseDirectory) ? _baseDirectory : null
        };

        if (dialog.ShowDialog(this) != DialogResult.OK)
            return;

        foreach (string file in dialog.FileNames)
        {
            try
            {
                string full = Path.GetFullPath(file);
                if (File.Exists(full))
                    _manualPgnPaths.Add(full);
            }
            catch
            {
                // Eén onbruikbaar gekozen pad mag de overige keuzes niet blokkeren.
            }
        }

        RefreshFiles();
    }

    private void ChooseOutputDirectory()
    {
        if (_running)
            return;

        using var dialog = new FolderBrowserDialog
        {
            Description =
                "Kies de map waaronder GamesCleaner voor deze run een eigen datum-tijdmap mag maken.",
            UseDescriptionForTitle = true,
            ShowNewFolderButton = true,
            SelectedPath = Directory.Exists(_outputRootDirectory)
                ? _outputRootDirectory
                : _baseDirectory
        };

        if (dialog.ShowDialog(this) != DialogResult.OK)
            return;

        try
        {
            _outputRootDirectory = Path.GetFullPath(dialog.SelectedPath);
            _outputPathBox.Text = _outputRootDirectory;
            RefreshFiles();
        }
        catch (Exception ex)
        {
            MessageBox.Show(
                this,
                "Deze uitvoermap kan niet worden gebruikt.\r\n\r\n" + ex.Message,
                "GamesCleaner",
                MessageBoxButtons.OK,
                MessageBoxIcon.Warning);
        }
    }

    private void ResetOutputDirectory()
    {
        if (_running)
            return;

        _outputRootDirectory = DefaultOutputRoot;
        _outputPathBox.Text = _outputRootDirectory;
        RefreshFiles();
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

        try
        {
            Directory.CreateDirectory(_outputRootDirectory);
        }
        catch (Exception ex)
        {
            MessageBox.Show(
                this,
                "De gekozen uitvoermap kan niet worden aangemaakt of geopend.\r\n\r\n" + ex.Message,
                "GamesCleaner — uitvoermap",
                MessageBoxButtons.OK,
                MessageBoxIcon.Error);
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
                    _outputRootDirectory,
                    inputs,
                    settings,
                    progress,
                    _cts.Token));

            _lastOutputDirectory = result.OutputDirectory;
            _openResultButton.Enabled = true;

            if (result.Cancelled)
            {
                _currentFile.Text =
                    "Afgebroken. De gedeeltelijke bestanden zijn als ONVOLLEDIG gemarkeerd.";

                MessageBox.Show(
                    this,
                    "De verwerking is afgebroken.\r\n\r\n" +
                    "Gedeeltelijke uitvoer is bewaard met _ONVOLLEDIG in de bestandsnaam.",
                    "GamesCleaner",
                    MessageBoxButtons.OK,
                    MessageBoxIcon.Information);
            }
            else
            {
                _progressBar.Value = _progressBar.Maximum;
                _progressText.Text =
                    $"100,0% — klaar in {CleanerEngine.FormatDuration(result.Elapsed)}";
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
                    $"Uitvoer:\r\n{result.OutputDirectory}\r\n\r\n" +
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
                "GamesCleaner is gestopt omdat een fout optrad.\r\n" +
                "De bronbestanden zijn niet gewijzigd.\r\n\r\n" + ex.Message,
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
        _progressBar.Value =
            Math.Clamp(value, _progressBar.Minimum, _progressBar.Maximum);

        string eta = p.Eta.HasValue
            ? $" — resterend ca. {CleanerEngine.FormatDuration(p.Eta.Value)}"
            : string.Empty;

        _progressText.Text =
            $"{p.Percent:0.0}% — {FormatBytes(p.ProcessedBytes)} / {FormatBytes(p.TotalBytes)}" +
            $" — verstreken {CleanerEngine.FormatDuration(p.Elapsed)}{eta}";

        _currentFile.Text = "Nu: " + DisplaySourcePath(p.CurrentFile);
        _counts.Text =
            $"Partijen: {p.Counters.TotalGames:N0}   " +
            $"StrongGames: {p.Counters.StrongGames:N0}   " +
            $"Afgekeurd: {p.Counters.RejectedGames:N0}";
    }

    private void SetRunning(bool running)
    {
        _running = running;

        _refreshButton.Enabled = !running;
        _addPgnButton.Enabled = !running;
        _allOnButton.Enabled = !running;
        _allOffButton.Enabled = !running;
        _chooseOutputButton.Enabled = !running;
        _defaultOutputButton.Enabled = !running;
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

    private string DisplaySourcePath(string path)
    {
        try
        {
            if (IsUnderDirectory(path, _baseDirectory))
                return Path.GetRelativePath(_baseDirectory, path);
        }
        catch
        {
            // Toon bij twijfel gewoon het volledige pad.
        }

        return path;
    }

    private static bool IsUnderDirectory(string filePath, string directoryPath)
    {
        try
        {
            string fullFile = Path.GetFullPath(filePath);
            string fullDir = Path.GetFullPath(directoryPath)
                .TrimEnd(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar)
                + Path.DirectorySeparatorChar;

            return fullFile.StartsWith(fullDir, StringComparison.OrdinalIgnoreCase);
        }
        catch
        {
            return false;
        }
    }

    private void OnFormClosing(object? sender, FormClosingEventArgs e)
    {
        if (!_running)
            return;

        var answer = MessageBox.Show(
            this,
            "GamesCleaner is nog bezig. Wilt u de verwerking annuleren?",
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
        _currentFile.Text =
            "Annuleren… wacht tot de huidige partij veilig is weggeschreven.";
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
