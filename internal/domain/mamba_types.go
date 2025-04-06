package domain

import "time"

// MoveType represents the type of price movement
type MoveType int
type SequenceType string

const (
	MambaSequence    SequenceType = "MAMBA"
	NonMambaSequence SequenceType = "NON_MAMBA"
)

const (
	NonMamba MoveType = iota
	Mamba
	BullishMamba
	BearishMamba
)

func (m MoveType) String() string {
	switch m {
	case BullishMamba:
		return "Bullish Mamba"
	case BearishMamba:
		return "Bearish Mamba"
	default:
		return "Non-Mamba"
	}
}

// DailyMove represents a single day's movement analysis
type DailyMove struct {
	Date           time.Time
	PriceChange    float64
	HighPrice      float64
	LowPrice       float64
	OpenPrice      float64
	ClosePrice     float64
	Volume         float64
	IsMamba        bool
	IsBullishMamba bool
	IsBearishMamba bool
	IsBullish      bool
	MoveStrength   float64
}

// MoveSeries represents a collection of daily moves for analysis
type MoveSeries struct {
	Stock         StockUniverse
	Moves         []DailyMove
	MambaCount    int     // Total Mamba moves (both bullish and bearish)
	BullishCount  int     // Bullish Mamba moves
	BearishCount  int     // Bearish Mamba moves
	NonMambaCount int     // Non-Mamba moves
	MaxMambaMove  float64 // Largest Mamba move percentage
	LastMambaMove time.Time
	LastMoveType  MoveType
}

// MoveSequence represents a consecutive series of similar moves
type MoveSequence struct {
	Type      SequenceType
	Length    int
	StartDate time.Time
	EndDate   time.Time
	Moves     []DailyMove
	Strength  float64
}

type TrendType string

const (
	BullishTrend TrendType = "BULLISH"
	BearishTrend TrendType = "BEARISH"
	NeutralTrend TrendType = "NEUTRAL"
)

type TrendAnalysis struct {
	Type            TrendType
	BelowEMACount   int
	AboveEMACount   int
	TrendPercentage float64
	EMAValues       []IndicatorValue
	StartDate       time.Time
	EndDate         time.Time
}

// SequenceAnalysis contains analysis of move sequences
type SequenceAnalysis struct {
	Stock              StockUniverse
	Trend              TrendAnalysis
	MambaSequences     []int
	NonMambaSequences  []int
	AverageMambaLen    float64
	AverageNonMambaLen float64
	CurrentSequence    MoveSequence
	TotalMambaDays     int
	TotalNonMambaDays  int
	Sequences          []MoveSequence
}
