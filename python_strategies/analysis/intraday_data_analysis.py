import pandas as pd
import mysql.connector
from datetime import datetime
from typing import Dict, List, Optional
import logging
import numpy as np
class IntradayDataAnalysis:
    def __init__(self):
        self.db_config = {
            'host': '127.0.0.1',
            'port': 3306,
            'user': 'root',
            'password': 'root1234',
            'database': 'setbull_trader'
        } 
        self.conn = None
        self.cursor = None
        self.setup_logging()
        
    def setup_logging(self):
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        self.logger = logging.getLogger(__name__)
        
    def connect_db(self):
        try:
            self.conn = mysql.connector.connect(**self.db_config)
            self.cursor = self.conn.cursor()
            self.logger.info("Successfully connected to database")
        except Exception as e:
            self.logger.error(f"Database connection failed: {str(e)}")
            raise
            
    def create_tables(self):
        create_trades_table = """
        CREATE TABLE IF NOT EXISTS trades (
            id INT AUTO_INCREMENT PRIMARY KEY,
            date DATE NOT NULL,
            name VARCHAR(50) NOT NULL,
            pnl DECIMAL(10,2),
            status VARCHAR(10),
            direction VARCHAR(10),
            trade_type VARCHAR(20),
            max_r_multiple DECIMAL(10,2),
            cumulative_pnl DECIMAL(10,2),
            opening_type VARCHAR(10),
            trend VARCHAR(10),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            UNIQUE KEY unique_trade (date, name, direction)
        );
        """
        
        create_stock_analysis_table = """
        CREATE TABLE IF NOT EXISTS stock_analysis (
            id INT AUTO_INCREMENT PRIMARY KEY,
            name VARCHAR(50) NOT NULL,
            direction VARCHAR(10),
            oah_success_rate DECIMAL(5,2),
            oal_success_rate DECIMAL(5,2),
            oam_success_rate DECIMAL(5,2),
            avg_profit_oah DECIMAL(10,2),
            avg_profit_oal DECIMAL(10,2),
            avg_profit_oam DECIMAL(10,2),
            mamba_move_count INT,
            oah_trade_count INT DEFAULT 0,
            oal_trade_count INT DEFAULT 0,
            oam_trade_count INT DEFAULT 0,
            last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            UNIQUE KEY unique_stock (name, direction)
        );
        """
        
        try:
            # Execute each CREATE TABLE statement separately
            self.cursor.execute(create_trades_table)
            self.cursor.execute(create_stock_analysis_table)
            self.conn.commit()
            self.logger.info("Tables created successfully")
        except Exception as e:
            self.logger.error(f"Table creation failed: {str(e)}")
            raise
            
    def load_data_from_csv(self, file_path: str):
        df = pd.read_csv(file_path)
        df['date'] = pd.to_datetime(df['Date']).dt.date
        
        # Prepare data for insertion
        trades_data = []
        for _, row in df.iterrows():
            trade = (
                row['Date'],
                row['Name'],
                float(row['PnL']),
                row['Status'],
                row['Direction'],
                row['EntryType'],
                float(row['RMultiple']),
                float(row['Cumulative']),
                row['OpeningType'],
                row['Trend']
            )
            trades_data.append(trade)
            
        # Insert or update data
        insert_sql = """
        INSERT INTO trades (date, name, pnl, status, direction, trade_type, max_r_multiple, cumulative_pnl, opening_type, trend)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE
            pnl = VALUES(pnl),
            status = VALUES(status),
            trade_type = VALUES(trade_type),
            max_r_multiple = VALUES(max_r_multiple),
            cumulative_pnl = VALUES(cumulative_pnl),
            opening_type = VALUES(opening_type),
            trend = VALUES(trend)
        """
        
        try:
            self.cursor.executemany(insert_sql, trades_data)
            self.conn.commit()
            self.logger.info(f"Successfully loaded {len(trades_data)} trades")
        except Exception as e:
            self.logger.error(f"Data loading failed: {str(e)}")
            raise
            
    def analyze_stock_performance(self):
        # Update stock analysis table with latest metrics
        analysis_sql = """
        INSERT INTO stock_analysis (
            name, direction, oah_success_rate, oal_success_rate, oam_success_rate,
            avg_profit_oah, avg_profit_oal, avg_profit_oam, mamba_move_count,
            oah_trade_count, oal_trade_count, oam_trade_count
        )
        SELECT 
            name,
            direction,
            ROUND(SUM(CASE WHEN opening_type = 'OAH' AND status = 'PROFIT' THEN 1 ELSE 0 END) / 
                  NULLIF(SUM(CASE WHEN opening_type = 'OAH' THEN 1 ELSE 0 END), 0) * 100, 2) as oah_success_rate,
            ROUND(SUM(CASE WHEN opening_type = 'OAL' AND status = 'PROFIT' THEN 1 ELSE 0 END) / 
                  NULLIF(SUM(CASE WHEN opening_type = 'OAL' THEN 1 ELSE 0 END), 0) * 100, 2) as oal_success_rate,
            ROUND(SUM(CASE WHEN opening_type = 'OAM' AND status = 'PROFIT' THEN 1 ELSE 0 END) / 
                  NULLIF(SUM(CASE WHEN opening_type = 'OAM' THEN 1 ELSE 0 END), 0) * 100, 2) as oam_success_rate,
            ROUND(AVG(CASE WHEN opening_type = 'OAH' THEN pnl ELSE NULL END), 2) as avg_profit_oah,
            ROUND(AVG(CASE WHEN opening_type = 'OAL' THEN pnl ELSE NULL END), 2) as avg_profit_oal,
            ROUND(AVG(CASE WHEN opening_type = 'OAM' THEN pnl ELSE NULL END), 2) as avg_profit_oam,
            SUM(CASE WHEN ABS(max_r_multiple) >= 5 THEN 1 ELSE 0 END) as mamba_move_count,
            SUM(CASE WHEN opening_type = 'OAH' THEN 1 ELSE 0 END) as oah_trade_count,
            SUM(CASE WHEN opening_type = 'OAL' THEN 1 ELSE 0 END) as oal_trade_count,
            SUM(CASE WHEN opening_type = 'OAM' THEN 1 ELSE 0 END) as oam_trade_count
        FROM trades
        GROUP BY name, direction
        ON DUPLICATE KEY UPDATE
            oah_success_rate = VALUES(oah_success_rate),
            oal_success_rate = VALUES(oal_success_rate),
            oam_success_rate = VALUES(oam_success_rate),
            avg_profit_oah = VALUES(avg_profit_oah),
            avg_profit_oal = VALUES(avg_profit_oal),
            avg_profit_oam = VALUES(avg_profit_oam),
            mamba_move_count = VALUES(mamba_move_count),
            oah_trade_count = VALUES(oah_trade_count),
            oal_trade_count = VALUES(oal_trade_count),
            oam_trade_count = VALUES(oam_trade_count)
        """
        
        try:
            self.cursor.execute(analysis_sql)
            self.conn.commit()
            self.logger.info("Stock analysis updated successfully")
        except Exception as e:
            self.logger.error(f"Stock analysis update failed: {str(e)}")
            raise

    def get_opening_condition_analysis(self) -> pd.DataFrame:
        """Analyze OAH vs OAL trading effectiveness"""
        query = """
        SELECT 
            direction,
            opening_type,
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
            ROUND(AVG(pnl), 2) as avg_pnl,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate
        FROM trades
        WHERE opening_type IN ('OAH', 'OAL')
        GROUP BY direction, opening_type
        ORDER BY direction, opening_type;
        """
        return pd.read_sql(query, self.conn)

    def get_trend_analysis(self) -> pd.DataFrame:
        """Analyze trading performance based on trend conditions"""
        query = """
        SELECT 
            direction,
            trend,
            opening_type,
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
            ROUND(AVG(pnl), 2) as avg_pnl,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
            ROUND(AVG(max_r_multiple), 2) as avg_r_multiple
        FROM trades
        GROUP BY direction, trend, opening_type
        ORDER BY direction, trend, opening_type;
        """
        return pd.read_sql(query, self.conn)

    def get_market_reversal_analysis(self) -> pd.DataFrame:
        """Analyze lottery day conditions and market reversals with trend consideration"""
        query = """
        SELECT 
            t1.date,
            t1.name,
            t1.direction,
            t1.trend,
            t1.opening_type,
            t1.pnl,
            t1.max_r_multiple
        FROM trades t1
        WHERE t1.max_r_multiple >= 5
        AND t1.opening_type IN ('OAH', 'OAL')
        AND t1.direction != t1.trend  -- Added condition to identify reversals
        ORDER BY t1.date, t1.max_r_multiple DESC;
        """
        return pd.read_sql(query, self.conn)

    def get_pattern_recognition(self) -> pd.DataFrame:
        """Analyze patterns after 2-3 days of consolidation"""
        query = """
        WITH daily_trades AS (
            SELECT 
                date,
                name,
                direction,
                opening_type,
                pnl,
                max_r_multiple,
                LAG(date) OVER (PARTITION BY name ORDER BY date) as prev_date
            FROM trades
        )
        SELECT 
            date,
            name,
            direction,
            opening_type,
            pnl,
            max_r_multiple,
            DATEDIFF(date, prev_date) as days_since_last_trade
        FROM daily_trades
        WHERE DATEDIFF(date, prev_date) BETWEEN 2 AND 3
        ORDER BY date, name;
        """
        return pd.read_sql(query, self.conn)

    def get_mamba_move_analysis(self) -> pd.DataFrame:
        """Analyze mamba moves and their success rates"""
        query = """
        SELECT 
            name,
            direction,
            COUNT(*) as total_trades,
            SUM(CASE WHEN max_r_multiple >= 5 THEN 1 ELSE 0 END) as mamba_moves,
            ROUND(AVG(CASE WHEN max_r_multiple >= 5 THEN pnl ELSE NULL END), 2) as avg_mamba_pnl,
            ROUND(SUM(CASE WHEN max_r_multiple >= 5 AND status = 'PROFIT' THEN 1 ELSE 0 END) / 
                  NULLIF(SUM(CASE WHEN max_r_multiple >= 5 THEN 1 ELSE 0 END), 0) * 100, 2) as mamba_win_rate
        FROM trades
        GROUP BY name, direction
        HAVING mamba_moves > 0
        ORDER BY mamba_win_rate DESC;
        """
        return pd.read_sql(query, self.conn)

    def get_high_potential_stocks(self) -> pd.DataFrame:
        """Identify high-potential stocks based on performance metrics with trend consideration"""
        query = """
        SELECT 
            name,
            direction,
            trend,
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
            ROUND(AVG(pnl), 2) as avg_pnl,
            ROUND(MAX(max_r_multiple), 2) as max_r,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
            SUM(CASE WHEN direction != trend THEN 1 ELSE 0 END) as reversal_trades,
            ROUND(AVG(CASE WHEN direction != trend THEN pnl ELSE NULL END), 2) as avg_reversal_pnl
        FROM trades
        GROUP BY name, direction, trend
        HAVING win_rate >= 60 AND avg_pnl > 0
        ORDER BY avg_pnl DESC;
        """
        return pd.read_sql(query, self.conn)

    def get_stocks_performing_against_trend(self) -> pd.DataFrame:
        """Find stocks that perform better when trading against their trend"""
        query = """
        WITH trend_analysis AS (
            SELECT 
                name,
                direction,
                trend,
                COUNT(*) as total_trades,
                SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
                ROUND(AVG(pnl), 2) as avg_pnl,
                ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
                ROUND(AVG(max_r_multiple), 2) as avg_r_multiple
            FROM trades
            GROUP BY name, direction, trend
        )
        SELECT 
            name,
            direction,
            trend,
            total_trades,
            winning_trades,
            avg_pnl,
            win_rate,
            avg_r_multiple,
            CASE 
                WHEN direction != trend THEN 'AGAINST_TREND'
                ELSE 'WITH_TREND'
            END as trade_type
        FROM trend_analysis
        WHERE direction != trend
        AND total_trades >= 5  -- Minimum number of trades to consider
        AND win_rate >= 60     -- Minimum win rate
        AND avg_pnl > 0        -- Positive average PnL
        ORDER BY avg_pnl DESC, win_rate DESC;
        """
        return pd.read_sql(query, self.conn)

    def get_stocks_performing_with_trend(self) -> pd.DataFrame:
        """Find stocks that perform better when trading with their trend"""
        query = """
        WITH trend_analysis AS (
            SELECT 
                name,
                direction,
                trend,
                COUNT(*) as total_trades,
                SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
                ROUND(AVG(pnl), 2) as avg_pnl,
                ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
                ROUND(AVG(max_r_multiple), 2) as avg_r_multiple
            FROM trades
            GROUP BY name, direction, trend
        )
        SELECT 
            name,
            direction,
            trend,
            total_trades,
            winning_trades,
            avg_pnl,
            win_rate,
            avg_r_multiple,
            CASE 
                WHEN direction = trend THEN 'WITH_TREND'
                ELSE 'AGAINST_TREND'
            END as trade_type
        FROM trend_analysis
        WHERE direction = trend
        AND total_trades >= 5  -- Minimum number of trades to consider
        AND win_rate >= 60     -- Minimum win rate
        AND avg_pnl > 0        -- Positive average PnL
        ORDER BY avg_pnl DESC, win_rate DESC;
        """
        return pd.read_sql(query, self.conn)

    def get_1st_entry_trend_analysis(self) -> pd.DataFrame:
        """Analyze 1st_entry trades with/against trend, split by direction."""
        query = """
        SELECT 
            direction,
            CASE WHEN direction = trend THEN 'WITH_TREND' ELSE 'AGAINST_TREND' END as trend_relation,
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
            ROUND(AVG(pnl), 2) as avg_pnl,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
            ROUND(AVG(max_r_multiple), 2) as avg_r_multiple
        FROM trades
        WHERE trade_type = '1st_entry'
        GROUP BY direction, trend_relation
        ORDER BY direction, trend_relation;
        """
        return pd.read_sql(query, self.conn)

    def get_2_30_entry_top_stocks(self, min_trades: int = 5) -> pd.DataFrame:
        """Get top performing stocks for 2_30_entry by win rate (min_trades filter)."""
        query = f"""
        SELECT 
            name,
            direction,
            trend,
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
            ROUND(AVG(pnl), 2) as avg_pnl,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
            SUM(pnl) as total_pnl
        FROM trades
        WHERE trade_type = '2_30_entry'
        GROUP BY name, direction, trend
        HAVING total_trades >= {min_trades} AND win_rate > 50
        ORDER BY total_pnl DESC, avg_pnl DESC, win_rate DESC, winning_trades DESC;
        """
        return pd.read_sql(query, self.conn)
    
    def get_1st_entry_top_stocks(self, min_trades: int = 5) -> pd.DataFrame:
        """Get top performing stocks for 1st_entry by win rate (min_trades filter)."""
        query = f"""
        SELECT 
            name,
            direction,
            trend,
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
            ROUND(AVG(pnl), 2) as avg_pnl,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
            SUM(pnl) as total_pnl
        FROM trades
        WHERE trade_type = '1st_entry'
        GROUP BY name, direction, trend
        HAVING total_trades >= {min_trades} AND win_rate > 50
        ORDER BY total_pnl DESC, avg_pnl DESC, win_rate DESC, winning_trades DESC;
        """
        return pd.read_sql(query, self.conn)
    
    def get_monthly_1st_entry_top_stocks(self, min_trades: int = 5, top_n: int = 10) -> pd.DataFrame:
        """Get top N performing stocks for 1st_entry by month, using win rate and avg_pnl (min_trades filter)."""
        query = f'''
        WITH monthly_stats AS (
            SELECT 
                YEAR(date) AS year,
                MONTH(date) AS month,
                name,
                direction,
                COUNT(*) as total_trades,
                SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
                ROUND(AVG(pnl), 2) as avg_pnl,
                ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
                SUM(pnl) as total_pnl
            FROM trades
            WHERE trade_type = '1st_entry'
            GROUP BY year, month, name, direction
            HAVING total_trades >= {min_trades}
        ), ranked AS (
            SELECT *,
                ROW_NUMBER() OVER (PARTITION BY year, month ORDER BY avg_pnl DESC, win_rate DESC, winning_trades DESC, total_pnl DESC) as `rank`
            FROM monthly_stats
        )
        SELECT * FROM ranked WHERE `rank` <= {top_n}
        ORDER BY year DESC, month DESC, `rank` ASC;
        '''
        return pd.read_sql(query, self.conn)

    def get_monthly_2_30_entry_top_stocks(self, min_trades: int = 5, top_n: int = 10) -> pd.DataFrame:
        """Get top N performing stocks for 2_30_entry by month, using win rate and avg_pnl (min_trades filter)."""
        query = f'''
        WITH monthly_stats AS (
            SELECT 
                YEAR(date) AS year,
                MONTH(date) AS month,
                name,
                direction,
                COUNT(*) as total_trades,
                SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
                ROUND(AVG(pnl), 2) as avg_pnl,
                ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
                SUM(pnl) as total_pnl
            FROM trades
            WHERE trade_type = '2_30_entry'
            GROUP BY year, month, name, direction
            HAVING total_trades >= {min_trades}
        ), ranked AS (
            SELECT *,
                ROW_NUMBER() OVER (PARTITION BY year, month ORDER BY avg_pnl DESC, win_rate DESC, winning_trades DESC, total_pnl DESC) as `rank`
            FROM monthly_stats
        )
        SELECT * FROM ranked WHERE `rank` <= {top_n}
        ORDER BY year DESC, month DESC, `rank` ASC;
        '''
        return pd.read_sql(query, self.conn)

    def export_backtest_analysis_csv(self, output_path: str):
        """
        Export top 3 1st_entry stocks and all 2_30_entry stocks with total_pnl > 500 to a CSV with required columns and static values.
        Columns: SYMBOL, TREND, DIRECTION, STRATEGY, ENTRY_TYPE, ENTRY_TIME, SL%, PS_TYPE
        """
        # Get data
        df_1st = self.get_1st_entry_top_stocks()
        df_2_30 = self.get_2_30_entry_top_stocks()

        # 1st_entry: Top 3 only
        df_1st = df_1st.head(3).copy()
        df_1st['STRATEGY'] = 'MR'
        df_1st['ENTRY_TYPE'] = '1ST_ENTRY'
        df_1st['ENTRY_TIME'] = '9:20AM'
        df_1st['SL%'] = 0.5
        df_1st['PS_TYPE'] = np.where(df_1st['win_rate'] > 70, 'FIXED', 'DISTRIBUTED')
        df_1st['TREND'] = df_1st['trend']  # No trend info in this query, so leave blank
        df_1st.rename(columns={'name': 'SYMBOL', 'direction': 'DIRECTION'}, inplace=True)

        # 2_30_entry: All with total_pnl > 500
        df_2_30 = df_2_30[df_2_30['total_pnl'] > 500].copy()
        df_2_30['STRATEGY'] = '2_30'
        df_2_30['ENTRY_TYPE'] = 'EVENING'
        df_2_30['ENTRY_TIME'] = '1PM'
        df_2_30['SL%'] = 0.3
        df_2_30['PS_TYPE'] = np.where(df_2_30['win_rate'] > 70, 'FIXED', 'DISTRIBUTED')
        df_2_30['TREND'] = df_2_30['trend']  # No trend info in this query, so leave blank
        df_2_30.rename(columns={'name': 'SYMBOL', 'direction': 'DIRECTION'}, inplace=True)

        # Select and order columns
        columns = ['SYMBOL', 'TREND', 'DIRECTION', 'STRATEGY', 'ENTRY_TYPE', 'ENTRY_TIME', 'SL%', 'PS_TYPE']
        df_final = pd.concat([
            df_1st[columns],
            df_2_30[columns]
        ], ignore_index=True)

        # Save to CSV
        df_final.to_csv(output_path, index=False)
        self.logger.info(f"Backtest analysis exported to {output_path}")

    def close(self):
        """Close database connections"""
        if self.cursor:
            self.cursor.close()
        if self.conn:
            self.conn.close()
            self.logger.info("Database connections closed") 