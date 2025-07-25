package cache

import (
	"sync"

	"github.com/go-gota/gota/dataframe"
)

// DataFramePool provides object pooling for DataFrames to reduce GC pressure
type DataFramePool struct {
	pool sync.Pool
}

// NewDataFramePool creates a new DataFrame pool
func NewDataFramePool() *DataFramePool {
	return &DataFramePool{
		pool: sync.Pool{
			New: func() interface{} {
				return dataframe.New()
			},
		},
	}
}

// Get retrieves a DataFrame from the pool
func (p *DataFramePool) Get() dataframe.DataFrame {
	return p.pool.Get().(dataframe.DataFrame)
}

// Put returns a DataFrame to the pool after clearing it
func (p *DataFramePool) Put(df dataframe.DataFrame) {
	// Clear the DataFrame before returning to pool
	// Note: Gota doesn't have a direct clear method, so we create a new empty one
	emptyDF := dataframe.New()
	p.pool.Put(emptyDF)
}

// CanclesesDataFrame represents a reusable wrapper for candle data processing
type CandleDataFrame struct {
	df   dataframe.DataFrame
	pool *DataFramePool
}

// NewCandleDataFrame creates a new candle DataFrame with pooling support
func NewCandleDataFrame(pool *DataFramePool) *CandleDataFrame {
	return &CandleDataFrame{
		df:   pool.Get(),
		pool: pool,
	}
}

// LoadCandles loads candle data into the DataFrame
func (cdf *CandleDataFrame) LoadCandles(candles interface{}) error {
	// Load data using LoadStructs or similar method
	df := dataframe.LoadStructs(candles)
	cdf.df = df
	return nil
}

// GetDataFrame returns the underlying DataFrame
func (cdf *CandleDataFrame) GetDataFrame() dataframe.DataFrame {
	return cdf.df
}

// Release returns the DataFrame to the pool
func (cdf *CandleDataFrame) Release() {
	if cdf.pool != nil {
		cdf.pool.Put(cdf.df)
		cdf.df = dataframe.New() // Reset to empty
	}
}

// ProcessingPool manages pools for different data processing components
type ProcessingPool struct {
	dataFramePool *DataFramePool
	mutex         sync.RWMutex
}

// NewProcessingPool creates a new processing pool manager
func NewProcessingPool() *ProcessingPool {
	return &ProcessingPool{
		dataFramePool: NewDataFramePool(),
	}
}

// GetDataFramePool returns the DataFrame pool
func (pp *ProcessingPool) GetDataFramePool() *DataFramePool {
	pp.mutex.RLock()
	defer pp.mutex.RUnlock()
	return pp.dataFramePool
}

// GetPooledCandleDataFrame gets a pooled candle DataFrame
func (pp *ProcessingPool) GetPooledCandleDataFrame() *CandleDataFrame {
	return NewCandleDataFrame(pp.GetDataFramePool())
}

// Stats returns pool statistics
func (pp *ProcessingPool) Stats() map[string]interface{} {
	pp.mutex.RLock()
	defer pp.mutex.RUnlock()
	
	return map[string]interface{}{
		"dataframe_pool_active": "tracking not available in sync.Pool",
		"pool_type":             "dataframe",
	}
}
