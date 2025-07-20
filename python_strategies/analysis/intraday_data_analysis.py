import pandas as pd
import mysql.connector
from datetime import datetime
from typing import Dict, List, Optional
import logging
import numpy as np
import os

# Optional SQLAlchemy import for better pandas compatibility
try:
    from sqlalchemy import create_engine
    SQLALCHEMY_AVAILABLE = True
except ImportError:
    SQLALCHEMY_AVAILABLE = False

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
            
    def get_sqlalchemy_engine(self):
        """Create SQLAlchemy engine for pandas compatibility"""
        if SQLALCHEMY_AVAILABLE:
            # Create connection string for SQLAlchemy
            connection_string = f"mysql+mysqlconnector://{self.db_config['user']}:{self.db_config['password']}@{self.db_config['host']}:{self.db_config['port']}/{self.db_config['database']}"
            return create_engine(connection_string)
        else:
            self.logger.warning("SQLAlchemy not available, using direct connection")
            return None
            
    def execute_query(self, query: str) -> pd.DataFrame:
        """Execute SQL query and return DataFrame, using SQLAlchemy if available"""
        # Use SQLAlchemy engine if available, otherwise use direct connection
        engine = self.get_sqlalchemy_engine()
        if engine:
            return pd.read_sql(query, engine)
        else:
            return pd.read_sql(query, self.conn)
            
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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        return self.execute_query(query)
    
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
        return self.execute_query(query)
    
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
        return self.execute_query(query)

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
        return self.execute_query(query)

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
        df_1st['PS_TYPE'] = np.where(df_1st['win_rate'] > 70, 'FIXED', 'DYNAMIC')
        df_1st['TREND'] = df_1st['trend']  # No trend info in this query, so leave blank
        df_1st.rename(columns={'name': 'SYMBOL', 'direction': 'DIRECTION'}, inplace=True)

        # 2_30_entry: All with total_pnl > 500
        df_2_30 = df_2_30[df_2_30['total_pnl'] > 500].copy()
        df_2_30['STRATEGY'] = '2_30'
        df_2_30['ENTRY_TYPE'] = 'EVENING'
        df_2_30['ENTRY_TIME'] = '1PM'
        df_2_30['SL%'] = 0.3
        df_2_30['PS_TYPE'] = np.where(df_2_30['win_rate'] > 70, 'FIXED', 'DYNAMIC')
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

    def get_comprehensive_stock_analysis(self) -> Dict:
        """Get comprehensive stock performance analysis similar to analyze_daily_trades.py"""
        query = """
        SELECT 
            name,
            COUNT(*) as trade_count,
            SUM(pnl) as total_pnl,
            AVG(pnl) as avg_pnl,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as profitable_trades,
            ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 1) as win_rate
        FROM trades
        GROUP BY name
        ORDER BY total_pnl DESC
        """
        
        df = self.execute_query(query)
        return df

    def get_top_performers_analysis(self) -> Dict:
        """Get top performers in different categories"""
        stock_stats = self.get_comprehensive_stock_analysis()
        
        results = {}
        
        # 1. Most profitable stocks by total profit amount
        top_profit = stock_stats.nlargest(5, 'total_pnl')[['name', 'total_pnl', 'trade_count', 'win_rate']]
        results['top_profit_amount'] = top_profit
        
        # 2. Most profitable stocks by number of profitable trades
        top_winning_trades = stock_stats.nlargest(5, 'profitable_trades')[['name', 'profitable_trades', 'trade_count', 'total_pnl', 'win_rate']]
        results['top_winning_trades'] = top_winning_trades
        
        # 3. Most traded stocks by number of trades
        top_traded = stock_stats.nlargest(5, 'trade_count')[['name', 'trade_count', 'total_pnl', 'profitable_trades', 'win_rate']]
        results['top_traded'] = top_traded
        
        return results

    def generate_comprehensive_report(self) -> str:
        """Generate comprehensive analysis report"""
        # Get overall statistics
        overall_query = """
        SELECT 
            COUNT(*) as total_trades,
            SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as profitable_trades,
            SUM(CASE WHEN status = 'LOSS' THEN 1 ELSE 0 END) as loss_trades,
            SUM(CASE WHEN status = 'FLAT' THEN 1 ELSE 0 END) as flat_trades,
            SUM(pnl) as total_pnl,
            AVG(pnl) as avg_pnl
        FROM trades
        """
        
        overall_stats = self.execute_query(overall_query).iloc[0]
        
        # Get top performers
        top_performers = self.get_top_performers_analysis()
        
        # Generate report
        report = []
        report.append("=" * 80)
        report.append("BB WIDTH BACKTESTING RESULTS ANALYSIS")
        report.append("=" * 80)
        report.append("")
        
        # Overall Performance Summary
        total_trades = overall_stats['total_trades']
        profitable_trades = overall_stats['profitable_trades']
        win_rate = (profitable_trades / total_trades * 100) if total_trades > 0 else 0
        
        report.append("OVERALL PERFORMANCE SUMMARY")
        report.append("-" * 40)
        report.append(f"Total Trades: {total_trades}")
        report.append(f"Profitable Trades: {profitable_trades}")
        report.append(f"Loss Trades: {overall_stats['loss_trades']}")
        report.append(f"Flat Trades: {overall_stats['flat_trades']}")
        report.append(f"Win Rate: {win_rate:.1f}%")
        report.append(f"Total PnL: ₹{overall_stats['total_pnl']:,.2f}")
        report.append(f"Average PnL per Trade: ₹{overall_stats['avg_pnl']:,.2f}")
        report.append("")
        
        # 1. Most Profitable Stocks by Profit Amount
        report.append("1. MOST PROFITABLE STOCKS (by Total Profit Amount)")
        report.append("-" * 50)
        for idx, row in top_performers['top_profit_amount'].iterrows():
            report.append(f"{idx+1}. {row['name']:<15} ₹{row['total_pnl']:>10,.2f} ({row['trade_count']} trades, {row['win_rate']}% win rate)")
        report.append("")
        
        # 2. Most Profitable Stocks by Number of Winning Trades
        report.append("2. MOST PROFITABLE STOCKS (by Number of Winning Trades)")
        report.append("-" * 55)
        for idx, row in top_performers['top_winning_trades'].iterrows():
            report.append(f"{idx+1}. {row['name']:<15} {row['profitable_trades']:>2} winning trades ({row['trade_count']} total, ₹{row['total_pnl']:>8,.0f} profit)")
        report.append("")
        
        # 3. Most Traded Stocks
        report.append("3. MOST TRADED STOCKS (by Number of Trades)")
        report.append("-" * 45)
        for idx, row in top_performers['top_traded'].iterrows():
            report.append(f"{idx+1}. {row['name']:<15} {row['trade_count']:>2} trades (₹{row['total_pnl']:>8,.0f} profit, {row['win_rate']}% win rate)")
        report.append("")
        
        # Additional Insights
        report.append("ADDITIONAL INSIGHTS")
        report.append("-" * 20)
        
        # Best performing stock overall
        best_stock = top_performers['top_profit_amount'].iloc[0]
        report.append(f"Best Overall Performer: {best_stock['name']} (₹{best_stock['total_pnl']:,.2f} profit)")
        
        # Stock with highest win rate (minimum 3 trades)
        stock_stats = self.get_comprehensive_stock_analysis()
        high_volume_stocks = stock_stats[stock_stats['trade_count'] >= 3]
        if len(high_volume_stocks) > 0:
            best_win_rate = high_volume_stocks.loc[high_volume_stocks['win_rate'].idxmax()]
            report.append(f"Highest Win Rate (3+ trades): {best_win_rate['name']} ({best_win_rate['win_rate']}% win rate)")
        
        # Risk analysis
        risk_query = """
        SELECT 
            AVG(CASE WHEN pnl > 0 THEN pnl ELSE NULL END) as avg_profit,
            AVG(CASE WHEN pnl < 0 THEN pnl ELSE NULL END) as avg_loss
        FROM trades
        """
        risk_stats = self.execute_query(risk_query).iloc[0]
        
        if not pd.isna(risk_stats['avg_profit']) and not pd.isna(risk_stats['avg_loss']) and risk_stats['avg_loss'] != 0:
            risk_reward_ratio = abs(risk_stats['avg_profit'] / risk_stats['avg_loss'])
            report.append(f"Risk-Reward Ratio: {risk_reward_ratio:.2f} (Avg Profit: ₹{risk_stats['avg_profit']:.0f}, Avg Loss: ₹{risk_stats['avg_loss']:.0f})")
        
        report.append("")
        report.append("=" * 80)
        
        return "\n".join(report)

    def save_comprehensive_report(self, output_path: str = "kb/7.2.1_bb_width_backtesting_results.txt"):
        """Save comprehensive analysis report to file"""
        report = self.generate_comprehensive_report()
        
        # Ensure directory exists
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        
        with open(output_path, 'w') as f:
            f.write(report)
        
        self.logger.info(f"Comprehensive analysis report saved to {output_path}")

    def close(self):
        """Close database connections"""
        if self.cursor:
            self.cursor.close()
        if self.conn:
            self.conn.close()
            self.logger.info("Database connections closed")

    def get_entry_time_top_stocks(self, min_trades: int = 5) -> dict:
        """
        For each EntryTimeString, return a DataFrame of top-performing stocks (by avg_pnl, win_rate, total_trades, etc.), filtered by min_trades.
        Returns a dict: {EntryTimeString: DataFrame}
        """
        # Try to get EntryTimeString from DB; if not present, fallback to CSV
        try:
            # Check if EntryTimeString column exists in trades table
            self.cursor.execute("SHOW COLUMNS FROM trades LIKE 'trade_type'")
            result = self.cursor.fetchone()
            if result:
                # EntryTimeString is in DB
                query = f"""
                SELECT 
                    name,
                    trade_type,
                    direction,
                    trend,
                    COUNT(*) as total_trades,
                    SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) as winning_trades,
                    ROUND(AVG(pnl), 2) as avg_pnl,
                    ROUND(SUM(CASE WHEN status = 'PROFIT' THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) as win_rate,
                    SUM(pnl) as total_pnl
                FROM trades
                GROUP BY name, trade_type, direction, trend
                HAVING total_trades >= {min_trades}
                ORDER BY trade_type, avg_pnl DESC, win_rate DESC, total_trades DESC;
                """
                df = self.execute_query(query)
                
                # Convert to the expected format for consistency
                result = {}
                for entry_time in sorted(df['trade_type'].unique()):
                    df_time = df[df['trade_type'] == entry_time].copy()
                    result[entry_time] = df_time
                return result
            else:
                # Fallback: load from CSV
                raise Exception('trade_type not in DB')
        except Exception:
            # Fallback: load from CSV (assume last loaded CSV is available)
            # This is a fallback for dev/test, not prod
            csv_path = os.path.join('/Users/gaurav/setbull_projects/setbull_trader/python_strategies/backtest_results', 'daily_trades.csv')
            df = pd.read_csv(csv_path)

        # If loaded from CSV, ensure EntryTimeString exists
        if 'trade_type' not in df.columns:
            raise ValueError('trade_type column not found in data')

        # Group by EntryTimeString and Name, aggregate metrics
        grouped = df.groupby(['trade_type', 'name', 'direction', 'trend'])
        summary = grouped.agg(
            total_trades=('pnl', 'count'),
            winning_trades=('status', lambda x: (x == 'PROFIT').sum()),
            avg_pnl=('pnl', 'mean'),
            win_rate=('status', lambda x: 100 * (x == 'PROFIT').sum() / len(x)),
            total_pnl=('pnl', 'sum')
        ).reset_index()

        # Filter by min_trades
        summary = summary[summary['total_trades'] >= min_trades]

        # For each EntryTimeString, get top stocks by avg_pnl, win_rate, total_trades
        result = {}
        for entry_time in sorted(summary['trade_type'].unique()):
            df_time = summary[summary['trade_type'] == entry_time].copy()
            df_time = df_time.sort_values(['avg_pnl', 'win_rate', 'total_trades', 'total_pnl'], ascending=[False, False, False, False])
            result[entry_time] = df_time 