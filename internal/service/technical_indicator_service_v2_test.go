package service

import (
	"math"
	"testing"
	"time"

	"setbull_trader/internal/domain"
)

// Test data for indicator calculations
func createTestCandles() []domain.Candle {
	baseTime := time.Date(2024, 1, 1, 9, 15, 0, 0, time.UTC)
	candles := make([]domain.Candle, 50)

	for i := 0; i < 50; i++ {
		// Create realistic OHLC data with some volatility
		base := 100.0 + float64(i)*0.5 + math.Sin(float64(i)*0.1)*2.0

		candles[i] = domain.Candle{
			InstrumentKey: "TEST_INSTRUMENT",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          base - 0.5,
			High:          base + 1.0 + math.Abs(math.Sin(float64(i)*0.2)),
			Low:           base - 1.0 - math.Abs(math.Cos(float64(i)*0.2)),
			Close:         base + math.Sin(float64(i)*0.15)*0.5,
			Volume:        1000 + int64(math.Abs(math.Sin(float64(i)*0.3))*500),
		}
	}

	return candles
}

func TestTechnicalIndicatorServiceV2_EMA(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	t.Run("EMA9_Calculation", func(t *testing.T) {
		ema9 := service.emaCalculator.CalculateEMA(candles, 9)

		// Validate basic properties
		if len(ema9) != len(candles) {
			t.Errorf("EMA length mismatch: expected %d, got %d", len(candles), len(ema9))
		}

		// First 8 values should be NaN
		for i := 0; i < 8; i++ {
			if !math.IsNaN(ema9[i].Value) {
				t.Errorf("Expected NaN for index %d, got %f", i, ema9[i].Value)
			}
		}

		// Check that we have valid values after the warmup period
		validCount := 0
		for i := 8; i < len(ema9); i++ {
			if !math.IsNaN(ema9[i].Value) {
				validCount++
			}
		}

		if validCount == 0 {
			t.Error("No valid EMA values found after warmup period")
		}

		t.Logf("EMA9 calculation successful: %d valid values out of %d total", validCount, len(ema9))
	})

	t.Run("Multiple_EMA_Periods", func(t *testing.T) {
		periods := []int{5, 9, 20, 50}
		emaResults := service.emaCalculator.CalculateMultipleEMAs(candles, periods)

		for _, period := range periods {
			ema, exists := emaResults[period]
			if !exists {
				t.Errorf("EMA%d not calculated", period)
				continue
			}

			if len(ema) != len(candles) {
				t.Errorf("EMA%d length mismatch: expected %d, got %d", period, len(candles), len(ema))
			}

			// Check warmup period
			for i := 0; i < period-1; i++ {
				if !math.IsNaN(ema[i].Value) {
					t.Errorf("EMA%d: Expected NaN for index %d, got %f", period, i, ema[i].Value)
				}
			}
		}

		t.Logf("Multiple EMA calculation successful for periods: %v", periods)
	})
}

func TestTechnicalIndicatorServiceV2_RSI(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	t.Run("RSI14_Calculation", func(t *testing.T) {
		rsi14 := service.rsiCalculator.CalculateRSI(candles, 14)

		// Validate basic properties
		if len(rsi14) != len(candles) {
			t.Errorf("RSI length mismatch: expected %d, got %d", len(candles), len(rsi14))
		}

		// First 14 values should be NaN
		for i := 0; i < 14; i++ {
			if !math.IsNaN(rsi14[i].Value) {
				t.Errorf("Expected NaN for index %d, got %f", i, rsi14[i].Value)
			}
		}

		// Check that RSI values are in valid range (0-100)
		validCount := 0
		for i := 14; i < len(rsi14); i++ {
			if !math.IsNaN(rsi14[i].Value) {
				if rsi14[i].Value < 0 || rsi14[i].Value > 100 {
					t.Errorf("RSI value out of range at index %d: %f", i, rsi14[i].Value)
				}
				validCount++
			}
		}

		if validCount == 0 {
			t.Error("No valid RSI values found after warmup period")
		}

		t.Logf("RSI14 calculation successful: %d valid values out of %d total", validCount, len(rsi14))
	})
}

func TestTechnicalIndicatorServiceV2_BollingerBands(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	t.Run("BollingerBands_Calculation", func(t *testing.T) {
		bbResult := service.bollingerCalculator.CalculateBollingerBands(candles, 20, 2.0)

		// Validate basic properties
		if len(bbResult.Upper) != len(candles) ||
			len(bbResult.Middle) != len(candles) ||
			len(bbResult.Lower) != len(candles) ||
			len(bbResult.Width) != len(candles) {
			t.Error("Bollinger Bands length mismatch with candles")
		}

		// First 19 values should be NaN
		for i := 0; i < 19; i++ {
			if !math.IsNaN(bbResult.Upper[i].Value) ||
				!math.IsNaN(bbResult.Middle[i].Value) ||
				!math.IsNaN(bbResult.Lower[i].Value) {
				t.Errorf("Expected NaN for Bollinger Bands at index %d", i)
			}
		}

		// Check that upper > middle > lower for valid values
		validCount := 0
		for i := 19; i < len(bbResult.Upper); i++ {
			if !math.IsNaN(bbResult.Upper[i].Value) &&
				!math.IsNaN(bbResult.Middle[i].Value) &&
				!math.IsNaN(bbResult.Lower[i].Value) {

				if bbResult.Upper[i].Value <= bbResult.Middle[i].Value {
					t.Errorf("Upper band should be > middle band at index %d", i)
				}
				if bbResult.Middle[i].Value <= bbResult.Lower[i].Value {
					t.Errorf("Middle band should be > lower band at index %d", i)
				}
				validCount++
			}
		}

		if validCount == 0 {
			t.Error("No valid Bollinger Bands values found after warmup period")
		}

		t.Logf("Bollinger Bands calculation successful: %d valid values", validCount)
	})
}

func TestTechnicalIndicatorServiceV2_VWAP(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	t.Run("VWAP_Calculation", func(t *testing.T) {
		vwap := service.vwapCalculator.CalculateVWAP(candles)

		// Validate basic properties
		if len(vwap) != len(candles) {
			t.Errorf("VWAP length mismatch: expected %d, got %d", len(candles), len(vwap))
		}

		// VWAP should have valid values for all periods (no warmup needed)
		validCount := 0
		for i, v := range vwap {
			if !math.IsNaN(v.Value) && v.Value > 0 {
				validCount++
			} else {
				t.Logf("Invalid VWAP at index %d: %f", i, v.Value)
			}
		}

		if validCount == 0 {
			t.Error("No valid VWAP values found")
		}

		t.Logf("VWAP calculation successful: %d valid values out of %d total", validCount, len(vwap))
	})
}

func TestTechnicalIndicatorServiceV2_ATR(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	t.Run("ATR14_Calculation", func(t *testing.T) {
		atr14 := service.atrCalculator.CalculateATR(candles, 14)

		// Validate basic properties
		if len(atr14) != len(candles) {
			t.Errorf("ATR length mismatch: expected %d, got %d", len(candles), len(atr14))
		}

		// First 14 values should be NaN
		for i := 0; i < 14; i++ {
			if !math.IsNaN(atr14[i].Value) {
				t.Errorf("Expected NaN for ATR at index %d, got %f", i, atr14[i].Value)
			}
		}

		// Check that ATR values are positive
		validCount := 0
		for i := 14; i < len(atr14); i++ {
			if !math.IsNaN(atr14[i].Value) {
				if atr14[i].Value <= 0 {
					t.Errorf("ATR should be positive at index %d: %f", i, atr14[i].Value)
				}
				validCount++
			}
		}

		if validCount == 0 {
			t.Error("No valid ATR values found after warmup period")
		}

		t.Logf("ATR14 calculation successful: %d valid values out of %d total", validCount, len(atr14))
	})
}

func TestTechnicalIndicatorServiceV2_AllIndicators(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	t.Run("Calculate_All_Indicators", func(t *testing.T) {
		indicatorSet, err := service.CalculateIndicatorsFromCandles(candles)
		if err != nil {
			t.Fatalf("Failed to calculate indicators: %v", err)
		}

		// Validate that all indicators are calculated
		if len(indicatorSet.MA9) != len(candles) {
			t.Error("MA9 length mismatch")
		}
		if len(indicatorSet.EMA5) != len(candles) {
			t.Error("EMA5 length mismatch")
		}
		if len(indicatorSet.EMA9) != len(candles) {
			t.Error("EMA9 length mismatch")
		}
		if len(indicatorSet.EMA50) != len(candles) {
			t.Error("EMA50 length mismatch")
		}
		if len(indicatorSet.BBUpper) != len(candles) {
			t.Error("BBUpper length mismatch")
		}
		if len(indicatorSet.BBMiddle) != len(candles) {
			t.Error("BBMiddle length mismatch")
		}
		if len(indicatorSet.BBLower) != len(candles) {
			t.Error("BBLower length mismatch")
		}
		if len(indicatorSet.BBWidth) != len(candles) {
			t.Error("BBWidth length mismatch")
		}
		if len(indicatorSet.VWAP) != len(candles) {
			t.Error("VWAP length mismatch")
		}
		if len(indicatorSet.ATR) != len(candles) {
			t.Error("ATR length mismatch")
		}
		if len(indicatorSet.RSI) != len(candles) {
			t.Error("RSI length mismatch")
		}
		if len(indicatorSet.Timestamps) != len(candles) {
			t.Error("Timestamps length mismatch")
		}

		t.Logf("All indicators calculated successfully for %d candles", len(candles))
	})
}

func BenchmarkTechnicalIndicatorServiceV2(b *testing.B) {
	service := NewTechnicalIndicatorServiceV2(nil)
	candles := createTestCandles()

	b.Run("EMA_GoNum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.emaCalculator.CalculateEMA(candles, 9)
		}
	})

	b.Run("RSI_GoNum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.rsiCalculator.CalculateRSI(candles, 14)
		}
	})

	b.Run("BollingerBands_GoNum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.bollingerCalculator.CalculateBollingerBands(candles, 20, 2.0)
		}
	})

	b.Run("VWAP_GoNum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.vwapCalculator.CalculateVWAP(candles)
		}
	})

	b.Run("ATR_GoNum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.atrCalculator.CalculateATR(candles, 14)
		}
	})

	b.Run("AllIndicators_GoNum", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			service.CalculateIndicatorsFromCandles(candles)
		}
	})
}
