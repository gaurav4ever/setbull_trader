package cache

import (
	"sync"

	"github.com/go-gota/gota/dataframe"
)

// CandleDataFrame wraps a DataFrame with lifecycle management for pooling
type CandleDataFrame struct {
	DataFrame   dataframe.DataFrame
	InUse       bool
	ReleaseFunc func()
}

// Acquire marks the DataFrame as in use
func (cdf *CandleDataFrame) Acquire() {
	cdf.InUse = true
}

// Release marks the DataFrame as available and calls the release function if set
func (cdf *CandleDataFrame) Release() {
	if cdf.InUse && cdf.ReleaseFunc != nil {
		cdf.ReleaseFunc()
	}
	cdf.InUse = false
}

// Reset clears the state for reuse in the pool
func (cdf *CandleDataFrame) Reset() {
	cdf.InUse = false
	cdf.ReleaseFunc = nil
	// Keep the DataFrame but reset to empty state
	cdf.DataFrame = dataframe.New()
}

// DataFramePool provides object pooling for CandleDataFrames to reduce GC pressure
type DataFramePool struct {
	pool sync.Pool
}

// NewDataFramePool creates a new DataFrame pool
func NewDataFramePool() *DataFramePool {
	return &DataFramePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &CandleDataFrame{
					DataFrame: dataframe.New(),
					InUse:     false,
				}
			},
		},
	}
}

// Get retrieves a CandleDataFrame from the pool
func (p *DataFramePool) Get() *CandleDataFrame {
	cdf := p.pool.Get().(*CandleDataFrame)
	cdf.Reset() // Ensure clean state
	return cdf
}

// Put returns a CandleDataFrame to the pool after resetting it
func (p *DataFramePool) Put(cdf *CandleDataFrame) {
	if cdf != nil {
		cdf.Reset()
		p.pool.Put(cdf)
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

// GetPooledCandleDataFrame gets a managed CandleDataFrame from the pool
func (pp *ProcessingPool) GetPooledCandleDataFrame() *CandleDataFrame {
	candleDF := pp.dataFramePool.Get()

	// Set up release function to return to pool
	candleDF.ReleaseFunc = func() {
		pp.dataFramePool.Put(candleDF)
	}

	candleDF.Acquire()
	return candleDF
}

// Stats returns pool statistics
func (pp *ProcessingPool) Stats() map[string]interface{} {
	pp.mutex.RLock()
	defer pp.mutex.RUnlock()

	return map[string]interface{}{
		"dataframe_pool_active": 0, // Could be tracked if needed
		"pool_type":             "memory_pool",
	}
}
