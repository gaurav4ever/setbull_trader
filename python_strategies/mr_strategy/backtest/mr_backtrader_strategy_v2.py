import backtrader as bt
import pandas as pd
import datetime as dt
import numpy as np

class MorningRangeStrategy(bt.Strategy):
    params = (
        ('mr_type', '5MR'),          # 5MR or 15MR
        ('risk_amount', 30.0),       # Risk amount in currency
        ('sl_percent', 0.75),        # Stop loss as percentage of entry
        ('tp1_rr', 3.0),             # First target R:R ratio
        ('tp2_rr', 5.0),             # Second target R:R ratio
        ('tp3_rr', 7.0),             # Third target R:R ratio
        ('tp1_size', 10.0),          # First target size (percentage)
        ('tp2_size', 40.0),          # Second target size (percentage)
        ('tick_buffer', 5),          # Buffer ticks for entry
        ('tick_size', 0.01),         # Tick size
        ('respect_trend', True),     # Whether to respect daily trend
        ('atr_mr_ratio', 3.0),       # Minimum ATR to MR ratio for valid range
    )

    def log(self, txt, dt=None):
        """Logging function for strategy"""
        dt = dt or self.datas[0].datetime.datetime(0)
        print(f'{dt.isoformat()} {txt}')

    def __init__(self):
        # Store reference to OHLC data
        self.dataclose = self.datas[0].close
        self.datahigh = self.datas[0].high
        self.datalow = self.datas[0].low
        self.dataopen = self.datas[0].open
        
        # Indicators for strategy
        self.daily_50ema = bt.indicators.ExponentialMovingAverage(self.datas[0].close, period=50)
        self.daily_9ema = bt.indicators.ExponentialMovingAverage(self.datas[0].close, period=9)
        self.daily_5ema = bt.indicators.ExponentialMovingAverage(self.datas[0].close, period=5)
        self.rsi = bt.indicators.RSI(self.datas[0].close, period=14)
        self.atr = bt.indicators.ATR(self.datas[0], period=14)
        
        # Morning Range variables
        self.mr_high = None
        self.mr_low = None
        self.mr_calculated = False
        self.mr_valid = False
        
        # Entry variables
        self.long_entry_price = None
        self.short_entry_price = None
        
        # Trade management
        self.intraday_orders = {}     # Track orders by ID
        self.long_entry_order = None
        self.short_entry_order = None
        self.long_sl_order = None
        self.short_sl_order = None
        self.long_tp_orders = []      # Track take profit orders
        self.short_tp_orders = []     # Track take profit orders
        
        # Position tracking
        self.in_long_trade = False
        self.in_short_trade = False
        self.traded_today = False
        
        # Risk management
        self.entry_price = None
        self.stop_price = None
        self.risk_points = None
        self.quantity = None
        self.tp1_price = None
        self.tp2_price = None
        self.tp3_price = None
        
        # State tracking
        self.current_day = None
        self.order_execution_time = {}  # Track when orders were executed
        self.day_done = False
        
        # Time variables
        self.market_open = dt.time(9, 15)
        self.market_close = dt.time(15, 30)
        self.mr5_end = dt.time(9, 20)
        self.mr15_end = dt.time(9, 30)
        self.last_bar_time = dt.time(15, 25)  # Last bar to close positions

    def prenext(self):
        self.next()

    def next(self):
        # Get current datetime
        current_date = self.datas[0].datetime.date(0)
        current_time = self.datas[0].datetime.time(0)
        
        # New day check
        if self.current_day != current_date:
            self.on_new_day(current_date)
        
        # Morning Range calculation phase
        if not self.mr_calculated:
            self.calculate_morning_range(current_time)
        
        # Entry orders phase - only if MR is valid and we haven't traded today
        if self.mr_calculated and self.mr_valid and not self.traded_today:
            self.manage_entry_orders(current_time)
        
        # Manage existing trades
        if self.in_long_trade or self.in_short_trade:
            self.manage_active_trades()
        
        # End of day clean-up
        if current_time >= self.last_bar_time and not self.day_done:
            self.end_of_day_cleanup()
            self.day_done = True

    def on_new_day(self, current_date):
        """Handle logic for a new trading day"""
        self.log(f"New trading day: {current_date}")
        
        # Reset daily variables
        self.mr_high = None
        self.mr_low = None
        self.mr_calculated = False
        self.mr_valid = False
        self.long_entry_price = None
        self.short_entry_price = None
        self.traded_today = False
        self.day_done = False
        
        # Cancel any open orders from previous day
        self.cancel_all_pending_orders()
        
        # Close any open positions from previous day
        if self.in_long_trade or self.in_short_trade:
            self.close()
            self.in_long_trade = False
            self.in_short_trade = False
        
        # Store current day
        self.current_day = current_date

    def calculate_morning_range(self, current_time):
        """Calculate the morning range based on strategy parameters"""
        
        # 5MR calculation
        if self.p.mr_type == '5MR' and self.market_open <= current_time < self.mr5_end:
            if not self.mr_calculated and current_time.minute == 15:
                # First 5-minute candle
                self.mr_high = self.datahigh[0]
                self.mr_low = self.datalow[0]
                self.log(f"5MR Initial - High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}")
                
            elif current_time >= self.mr5_end and not self.mr_calculated:
                # Range calculation completed
                self.mr_calculated = True
                mr_size = self.mr_high - self.mr_low
                
                # Calculate entry prices with tick buffer
                self.long_entry_price = self.mr_high + (self.p.tick_size * self.p.tick_buffer)
                self.short_entry_price = self.mr_low - (self.p.tick_size * self.p.tick_buffer)
                
                # Validate MR using ATR ratio
                atr_value = self.atr[0]
                atr_to_mr_ratio = atr_value / mr_size if mr_size > 0 else 0
                self.mr_valid = atr_to_mr_ratio > self.p.atr_mr_ratio
                
                self.log(f"5MR Calculated - High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}, Size: {mr_size:.2f}")
                self.log(f"Entry Prices - Long: {self.long_entry_price:.2f}, Short: {self.short_entry_price:.2f}")
                self.log(f"MR Valid: {self.mr_valid}, ATR/MR Ratio: {atr_to_mr_ratio:.2f}")
                
        # 15MR calculation
        elif self.p.mr_type == '15MR' and self.market_open <= current_time < self.mr15_end:
            if not self.mr_high or self.datahigh[0] > self.mr_high:
                self.mr_high = self.datahigh[0]
                
            if not self.mr_low or self.datalow[0] < self.mr_low:
                self.mr_low = self.datalow[0]
                
            if current_time >= self.mr15_end and not self.mr_calculated:
                # Range calculation completed
                self.mr_calculated = True
                mr_size = self.mr_high - self.mr_low
                
                # Calculate entry prices with tick buffer
                self.long_entry_price = self.mr_high + (self.p.tick_size * self.p.tick_buffer)
                self.short_entry_price = self.mr_low - (self.p.tick_size * self.p.tick_buffer)
                
                # Validate MR using ATR ratio
                atr_value = self.atr[0]
                atr_to_mr_ratio = atr_value / mr_size if mr_size > 0 else 0
                self.mr_valid = atr_to_mr_ratio > self.p.atr_mr_ratio
                
                self.log(f"15MR Calculated - High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}, Size: {mr_size:.2f}")
                self.log(f"Entry Prices - Long: {self.long_entry_price:.2f}, Short: {self.short_entry_price:.2f}")
                self.log(f"MR Valid: {self.mr_valid}, ATR/MR Ratio: {atr_to_mr_ratio:.2f}")

    def manage_entry_orders(self, current_time):
        """Place entry orders when MR is valid"""
        # Check if we're past the MR calculation time and haven't traded today
        mr_end_time = self.mr5_end if self.p.mr_type == '5MR' else self.mr15_end
        
        if current_time >= mr_end_time and not self.traded_today and not self.long_entry_order and not self.short_entry_order:
            # Get trend for the day
            daily_trend_bullish = self.dataclose[0] > self.daily_50ema[0]
            
            # Place long entry order if valid
            if not self.p.respect_trend or daily_trend_bullish:
                self.log(f"Placing LONG entry order at {self.long_entry_price:.2f}")
                self.long_entry_order = self.buy(
                    exectype=bt.Order.StopLimit,
                    price=self.long_entry_price,
                    plimit=self.long_entry_price + (self.p.tick_size * 2),  # Limit price slightly above stop price
                    transmit=True
                )
                self.intraday_orders[self.long_entry_order.ref] = {
                    'type': 'long_entry',
                    'price': self.long_entry_price
                }
            
            # Place short entry order if valid
            if not self.p.respect_trend or not daily_trend_bullish:
                self.log(f"Placing SHORT entry order at {self.short_entry_price:.2f}")
                self.short_entry_order = self.sell(
                    exectype=bt.Order.StopLimit,
                    price=self.short_entry_price,
                    plimit=self.short_entry_price - (self.p.tick_size * 2),  # Limit price slightly below stop price
                    transmit=True
                )
                self.intraday_orders[self.short_entry_order.ref] = {
                    'type': 'short_entry',
                    'price': self.short_entry_price
                }

    def manage_active_trades(self):
        """Manage active trades - handle trailing stops and profit targets"""
        if self.in_long_trade:
            # Check for trailing stop adjustment at TP2
            if self.datahigh[0] >= self.tp2_price and self.stop_price < self.tp1_price:
                new_stop = self.entry_price + ((self.tp1_price - self.entry_price) * 0.5)
                self.update_stop_loss(new_stop, 'long')
                self.log(f"LONG trade - Moving stop to breakeven plus: {new_stop:.2f}")
            
            # Check for trailing stop adjustment at TP1
            elif self.datahigh[0] >= self.tp1_price and self.stop_price < self.entry_price:
                new_stop = self.entry_price
                self.update_stop_loss(new_stop, 'long')
                self.log(f"LONG trade - Moving stop to breakeven: {new_stop:.2f}")
        
        elif self.in_short_trade:
            # Check for trailing stop adjustment at TP2
            if self.datalow[0] <= self.tp2_price and self.stop_price > self.tp1_price:
                new_stop = self.entry_price - ((self.entry_price - self.tp1_price) * 0.5)
                self.update_stop_loss(new_stop, 'short')
                self.log(f"SHORT trade - Moving stop to breakeven plus: {new_stop:.2f}")
            
            # Check for trailing stop adjustment at TP1
            elif self.datalow[0] <= self.tp1_price and self.stop_price > self.entry_price:
                new_stop = self.entry_price
                self.update_stop_loss(new_stop, 'short')
                self.log(f"SHORT trade - Moving stop to breakeven: {new_stop:.2f}")

    def end_of_day_cleanup(self):
        """Close positions and cancel orders at end of day"""
        self.log("End of day cleanup")
        
        # Cancel all pending orders
        self.cancel_all_pending_orders()
        
        # Close all open positions
        if self.in_long_trade or self.in_short_trade:
            self.close()
            self.log("Closing all positions at end of day")
            self.in_long_trade = False
            self.in_short_trade = False

    def notify_order(self, order):
        """Handle order notifications"""
        # Extract order information
        order_ref = order.ref
        order_info = self.intraday_orders.get(order_ref, {})
        order_type = order_info.get('type', 'unknown')
        
        if order.status in [order.Submitted, order.Accepted]:
            # Order submitted/accepted - nothing to do
            return
        
        # Check if order was completed
        if order.status in [order.Completed]:
            if order_type == 'long_entry':
                self.handle_long_entry_execution(order)
            elif order_type == 'short_entry':
                self.handle_short_entry_execution(order)
            elif order_type == 'long_sl':
                self.handle_long_sl_execution(order)
            elif order_type == 'short_sl':
                self.handle_short_sl_execution(order)
            elif order_type == 'long_tp1':
                self.log(f"LONG TP1 executed at {order.executed.price:.2f}")
            elif order_type == 'long_tp2':
                self.log(f"LONG TP2 executed at {order.executed.price:.2f}")
            elif order_type == 'long_tp3':
                self.log(f"LONG TP3 executed at {order.executed.price:.2f}")
                self.in_long_trade = False
            elif order_type == 'short_tp1':
                self.log(f"SHORT TP1 executed at {order.executed.price:.2f}")
            elif order_type == 'short_tp2':
                self.log(f"SHORT TP2 executed at {order.executed.price:.2f}")
            elif order_type == 'short_tp3':
                self.log(f"SHORT TP3 executed at {order.executed.price:.2f}")
                self.in_short_trade = False
            else:
                self.log(f"Order Completed - Type: {order_type}, Price: {order.executed.price:.2f}")
        
        elif order.status in [order.Canceled, order.Margin, order.Rejected]:
            self.log(f"Order Canceled/Margin/Rejected - Type: {order_type}")
            
            # Clear references to canceled orders
            if order_ref == getattr(self.long_entry_order, 'ref', None):
                self.long_entry_order = None
            elif order_ref == getattr(self.short_entry_order, 'ref', None):
                self.short_entry_order = None
            elif order_ref == getattr(self.long_sl_order, 'ref', None):
                self.long_sl_order = None
            elif order_ref == getattr(self.short_sl_order, 'ref', None):
                self.short_sl_order = None
            
            # Remove from order tracking
            if order_ref in self.intraday_orders:
                del self.intraday_orders[order_ref]

    def handle_long_entry_execution(self, order):
        """Handle execution of a long entry order"""
        self.log(f"LONG ENTRY executed at {order.executed.price:.2f}")
        
        # Cancel the opposite entry order
        if self.short_entry_order:
            self.cancel(self.short_entry_order)
            self.short_entry_order = None
        
        # Set trade variables
        self.entry_price = order.executed.price
        self.in_long_trade = True
        self.traded_today = True
        
        # Calculate stop loss
        self.stop_price = self.entry_price * (1 - self.p.sl_percent/100)
        self.risk_points = self.entry_price - self.stop_price
        
        # Calculate quantity based on risk
        self.quantity = self.p.risk_amount / self.risk_points
        
        # Calculate take profit levels
        self.tp1_price = self.entry_price + (self.risk_points * self.p.tp1_rr)
        self.tp2_price = self.entry_price + (self.risk_points * self.p.tp2_rr)
        self.tp3_price = self.entry_price + (self.risk_points * self.p.tp3_rr)
        
        # Place stop loss order
        self.place_long_stop_loss()
        
        # Place take profit orders
        self.place_long_take_profits()
        
        # Log trade details
        self.log(f"LONG trade details - Entry: {self.entry_price:.2f}, Stop: {self.stop_price:.2f}, " + 
                f"TP1: {self.tp1_price:.2f}, TP2: {self.tp2_price:.2f}, TP3: {self.tp3_price:.2f}")

    def handle_short_entry_execution(self, order):
        """Handle execution of a short entry order"""
        self.log(f"SHORT ENTRY executed at {order.executed.price:.2f}")
        
        # Cancel the opposite entry order
        if self.long_entry_order:
            self.cancel(self.long_entry_order)
            self.long_entry_order = None
        
        # Set trade variables
        self.entry_price = order.executed.price
        self.in_short_trade = True
        self.traded_today = True
        
        # Calculate stop loss
        self.stop_price = self.entry_price * (1 + self.p.sl_percent/100)
        self.risk_points = self.stop_price - self.entry_price
        
        # Calculate quantity based on risk
        self.quantity = self.p.risk_amount / self.risk_points
        
        # Calculate take profit levels
        self.tp1_price = self.entry_price - (self.risk_points * self.p.tp1_rr)
        self.tp2_price = self.entry_price - (self.risk_points * self.p.tp2_rr)
        self.tp3_price = self.entry_price - (self.risk_points * self.p.tp3_rr)
        
        # Place stop loss order
        self.place_short_stop_loss()
        
        # Place take profit orders
        self.place_short_take_profits()
        
        # Log trade details
        self.log(f"SHORT trade details - Entry: {self.entry_price:.2f}, Stop: {self.stop_price:.2f}, " + 
                f"TP1: {self.tp1_price:.2f}, TP2: {self.tp2_price:.2f}, TP3: {self.tp3_price:.2f}")

    def handle_long_sl_execution(self, order):
        """Handle execution of a long stop loss order"""
        self.log(f"LONG STOP LOSS executed at {order.executed.price:.2f}")
        
        # Cancel any remaining take profit orders
        for tp_order in self.long_tp_orders:
            if tp_order:
                self.cancel(tp_order)
        
        self.long_tp_orders = []
        self.in_long_trade = False

    def handle_short_sl_execution(self, order):
        """Handle execution of a short stop loss order"""
        self.log(f"SHORT STOP LOSS executed at {order.executed.price:.2f}")
        
        # Cancel any remaining take profit orders
        for tp_order in self.short_tp_orders:
            if tp_order:
                self.cancel(tp_order)
        
        self.short_tp_orders = []
        self.in_short_trade = False

    def place_long_stop_loss(self):
        """Place a stop-limit order for long stop loss"""
        # Cancel any existing stop loss order
        if self.long_sl_order:
            self.cancel(self.long_sl_order)
        
        # Place new stop loss order
        self.long_sl_order = self.sell(
            exectype=bt.Order.StopLimit,
            price=self.stop_price,
            plimit=self.stop_price - (self.p.tick_size * 2),  # Limit price slightly below stop price
            size=self.quantity,
            transmit=True
        )
        
        # Track order
        self.intraday_orders[self.long_sl_order.ref] = {
            'type': 'long_sl',
            'price': self.stop_price
        }

    def place_short_stop_loss(self):
        """Place a stop-limit order for short stop loss"""
        # Cancel any existing stop loss order
        if self.short_sl_order:
            self.cancel(self.short_sl_order)
        
        # Place new stop loss order
        self.short_sl_order = self.buy(
            exectype=bt.Order.StopLimit,
            price=self.stop_price,
            plimit=self.stop_price + (self.p.tick_size * 2),  # Limit price slightly above stop price
            size=self.quantity,
            transmit=True
        )
        
        # Track order
        self.intraday_orders[self.short_sl_order.ref] = {
            'type': 'short_sl',
            'price': self.stop_price
        }

    def place_long_take_profits(self):
        """Place take profit orders for long position"""
        # Calculate sizes for each take profit level
        tp1_size = self.quantity * (self.p.tp1_size / 100)
        tp2_size = self.quantity * (self.p.tp2_size / 100)
        tp3_size = self.quantity - tp1_size - tp2_size
        
        # Place TP1 order
        tp1_order = self.sell(
            exectype=bt.Order.Limit,
            price=self.tp1_price,
            size=tp1_size,
            transmit=True
        )
        self.intraday_orders[tp1_order.ref] = {
            'type': 'long_tp1',
            'price': self.tp1_price
        }
        
        # Place TP2 order
        tp2_order = self.sell(
            exectype=bt.Order.Limit,
            price=self.tp2_price,
            size=tp2_size,
            transmit=True
        )
        self.intraday_orders[tp2_order.ref] = {
            'type': 'long_tp2',
            'price': self.tp2_price
        }
        
        # Place TP3 order
        tp3_order = self.sell(
            exectype=bt.Order.Limit,
            price=self.tp3_price,
            size=tp3_size,
            transmit=True
        )
        self.intraday_orders[tp3_order.ref] = {
            'type': 'long_tp3',
            'price': self.tp3_price
        }
        
        # Store take profit orders
        self.long_tp_orders = [tp1_order, tp2_order, tp3_order]

    def place_short_take_profits(self):
        """Place take profit orders for short position"""
        # Calculate sizes for each take profit level
        tp1_size = self.quantity * (self.p.tp1_size / 100)
        tp2_size = self.quantity * (self.p.tp2_size / 100)
        tp3_size = self.quantity - tp1_size - tp2_size
        
        # Place TP1 order
        tp1_order = self.buy(
            exectype=bt.Order.Limit,
            price=self.tp1_price,
            size=tp1_size,
            transmit=True
        )
        self.intraday_orders[tp1_order.ref] = {
            'type': 'short_tp1',
            'price': self.tp1_price
        }
        
        # Place TP2 order
        tp2_order = self.buy(
            exectype=bt.Order.Limit,
            price=self.tp2_price,
            size=tp2_size,
            transmit=True
        )
        self.intraday_orders[tp2_order.ref] = {
            'type': 'short_tp2',
            'price': self.tp2_price
        }
        
        # Place TP3 order
        tp3_order = self.buy(
            exectype=bt.Order.Limit,
            price=self.tp3_price,
            size=tp3_size,
            transmit=True
        )
        self.intraday_orders[tp3_order.ref] = {
            'type': 'short_tp3',
            'price': self.tp3_price
        }
        
        # Store take profit orders
        self.short_tp_orders = [tp1_order, tp2_order, tp3_order]

    def update_stop_loss(self, new_stop, trade_type):
        """Update stop loss price for a trade"""
        if trade_type == 'long':
            self.stop_price = new_stop
            self.place_long_stop_loss()
        else:
            self.stop_price = new_stop
            self.place_short_stop_loss()

    def cancel_all_pending_orders(self):
        """Cancel all pending orders"""
        for order_ref, order_info in list(self.intraday_orders.items()):
            for order in self.broker.get_orders_open():
                if order.ref == order_ref:
                    self.cancel(order)
                    self.log(f"Canceled pending {order_info.get('type', 'unknown')} order")
        
        # Reset order tracking
        self.long_entry_order = None
        self.short_entry_order = None
        self.long_sl_order = None
        self.short_sl_order = None
        self.long_tp_orders = []
        self.short_tp_orders = []

    def notify_trade(self, trade):
        """Handle trade notifications"""
        if not trade.isclosed:
            return
        
        # Log trade result
        self.log(f"TRADE CLOSED - P&L: {trade.pnl:.2f}, P&L%: {trade.pnlcomm:.2f}%")