package main

import "time"

const (
	version                    = "3.0.9"
	syzygyCheckVersion         = "2.0.9"
	syzygyCheckVariant         = "V9F"
	syzygyCheckDisplay         = "2.0.9 V9F"
	syzygyCheckExeFilename     = "!SyzygyCheck_v2.0.9_V9F.exe"
	syzygyCheckPackageFilename = "SyzygyCheck_v2.0.9_V9F_FINAL_GitHub_Release_Bundle.zip"
	syzygyCheckSourceFilename  = "SyzygyCheck_v2.0.9_V9F_FINAL_SOURCE.zip"
	syzygyCheckSourceRoot      = "SyzygyCheck_v2.0.9_V9F_SOURCE/"
	defaultMaxPly              = 100
	defaultMinPly              = 24
	maxAllowedPly              = 120
	buildDateUTC               = "2026-09-11T22:46:20Z"
)

type Config struct {
	Input               string
	Mode                string
	MaxPly              int
	MinPly              int
	MinElo              int
	MaxEloGap           int
	Workers             int
	NoPause             bool
	SelfTest            bool
	Interactive         bool
	ExtractSource       bool
	ExtractSyzygySource bool
	BuildInfo           bool
	PagefileHelp        bool
}

type HardwareInfo struct {
	CPUName        string
	LogicalThreads int
	RAMBytes       int64
	PagefileBytes  int64
	PagefilePaths  string
	FreeBytes      int64
	RAMKnown       bool
	PagefileKnown  bool
	FreeKnown      bool
}

type SourceKind string

const (
	KindPGN  SourceKind = "PGN"
	KindCTG  SourceKind = "CTG"
	KindCBH  SourceKind = "CBH"
	Kind2CBH SourceKind = "2CBH"
)

type Source struct {
	Kind     SourceKind
	Path     string
	Base     string
	Bytes    int64
	Complete bool
	Aux      []string
	// Origin is the original source shown in RAW provenance when Path points
	// at a temporary adapter PGN. Empty means Path itself.
	Origin string
	// OriginKind preserves CBH/2CBH provenance when Path is a temporary adapter PGN.
	OriginKind SourceKind
}

type Inventory struct {
	Root    string
	Sources []Source
}

type SourceTrace struct {
	Kind                                                                                 SourceKind
	Base                                                                                 string
	SourcePath                                                                           string
	RawPath                                                                              string
	RawSeen, RawAccepted, RawLocalDuplicate, RawCrossDuplicate, RawRejected, RawVerified int64
	MetalPath                                                                            string
	MetalSeen, MetalEligible, MetalSelected, MetalInvalid                                int64
}

type Counters struct {
	FilesPGN, SetsCTG, FilesCBH, Files2CBH                        int64
	GamesSeen, GamesAccepted, GamesDuplicate                      int64
	GamesNoResult, GamesSetup, GamesShort                         int64
	GamesElo, GamesEloGap, GamesMalformed                         int64
	VariationsStripped, CommentsStripped                          int64
	PlyWritten                                                    int64
	OutputBytes                                                   int64
	OutputSHA256                                                  string
	CTGSetsProcessed, CTGRawBuilt, CTGRawSkipped, CTGRawFailed    int64
	CTGRawGames, CTGRawPlies, CTGRawBytes, CTGDecodeErrors        int64
	RawGamesVerified, RawBooksVerified, CombinedVerified          int64
	CTGMetalBuilt, CTGMetalSkipped, CTGMetalFailed, CTGMetalBytes int64
	CBHSetsProcessed, CBHSetsBuilt, CBHSetsFailed                 int64
	CBHRecords, CBHConverted, CBHSkipped, CBHPlies                int64
	TwoCBHSetsProcessed, TwoCBHSetsBuilt, TwoCBHSetsFailed        int64
	TwoCBHRecords, TwoCBHConverted, TwoCBHSkipped, TwoCBHPlies    int64
	CombinedBytes                                                 int64
	CombinedGames                                                 int64
	CombinedSHA256                                                string
	Started                                                       time.Time
	Elapsed                                                       time.Duration
	SourceTraces                                                  []SourceTrace
	GameMetalBuilt, GameMetalSkipped, GameMetalFailed             int64
	GameMetalGamesSeen, GameMetalRejected, GameMetalDuplicates    int64
	GameMetalGamesAccepted, GameMetalObservations                 int64
	GameMetalPositions, GameMetalCandidates, GameMetalQualified   int64
	GameMetalAnchorPositions, GameMetalAnchorSignals              int64
	GameMetalModelCandidates, GameMetalSelected                   int64
	GameMetalVerified, GameMetalInvalid                           int64
	GameMetalBytes                                                int64
	GameMetalSHA256                                               string
}

type PGNGame struct {
	Tags     map[string]string
	TagOrder []string
	MoveText string
	Source   string
}

type OutputLayout struct {
	RunDir           string
	RawDir           string
	RawSeparateDir   string
	RawMergedDir     string
	MetalDir         string
	MetalSeparateDir string
	MetalMergedDir   string
	ReportDir        string
	TempDir          string
	RawGamesFile     string
	GameMetalFile    string

	// Compatibility aliases used by the legacy integration code.
	RawGamesDir    string
	RawBooksDir    string
	RawCombinedDir string
}
