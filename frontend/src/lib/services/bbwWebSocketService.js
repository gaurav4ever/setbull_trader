// BBW WebSocket Service for real-time dashboard updates
class BBWWebSocketService {
    constructor() {
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000; // Start with 1 second
        this.listeners = new Map();
        this.heartbeatInterval = null;
        this.connectionTimeout = null;
    }

    // Connect to WebSocket
    connect() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            console.log('WebSocket already connected');
            return;
        }

        try {
            // Use relative WebSocket URL to leverage Vite proxy
            const wsUrl = `ws://${window.location.host}/api/v1/bbw/live`;
            console.log('Connecting to BBW WebSocket:', wsUrl);
            
            this.ws = new WebSocket(wsUrl);
            this.setupEventHandlers();
            
            // Set connection timeout
            this.connectionTimeout = setTimeout(() => {
                if (this.ws && this.ws.readyState !== WebSocket.OPEN) {
                    console.error('WebSocket connection timeout');
                    this.handleConnectionError();
                }
            }, 5000);
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.handleConnectionError();
        }
    }

    // Setup WebSocket event handlers
    setupEventHandlers() {
        this.ws.onopen = () => {
            console.log('BBW WebSocket connected');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;
            clearTimeout(this.connectionTimeout);
            this.startHeartbeat();
            this.notifyListeners('connected', { connected: true });
        };

        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                console.log('BBW WebSocket message received:', data);
                this.handleMessage(data);
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };

        this.ws.onclose = (event) => {
            console.log('BBW WebSocket disconnected:', event.code, event.reason);
            this.isConnected = false;
            this.stopHeartbeat();
            this.notifyListeners('disconnected', { connected: false });
            
            // Attempt to reconnect if not a clean close
            if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (error) => {
            console.error('BBW WebSocket error:', error);
            this.handleConnectionError();
        };
    }

    // Handle incoming messages
    handleMessage(data) {
        switch (data.type) {
            case 'bbw_update':
                this.notifyListeners('bbw_update', data.data);
                break;
            case 'alert_triggered':
                this.notifyListeners('alert_triggered', data.data);
                break;
            case 'market_status':
                this.notifyListeners('market_status', data.data);
                break;
            case 'pong':
                // Heartbeat response
                break;
            default:
                console.log('Unknown message type:', data.type);
        }
    }

    // Send heartbeat
    startHeartbeat() {
        this.heartbeatInterval = setInterval(() => {
            if (this.isConnected) {
                this.send({ type: 'ping' });
            }
        }, 30000); // Send ping every 30 seconds
    }

    // Stop heartbeat
    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }

    // Send message to WebSocket
    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        } else {
            console.warn('WebSocket not connected, cannot send message');
        }
    }

    // Schedule reconnection
    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), 30000);
        
        console.log(`Scheduling WebSocket reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
        
        setTimeout(() => {
            if (!this.isConnected) {
                this.connect();
            }
        }, delay);
    }

    // Handle connection errors
    handleConnectionError() {
        this.isConnected = false;
        this.stopHeartbeat();
        clearTimeout(this.connectionTimeout);
        this.notifyListeners('error', { error: 'Connection failed' });
    }

    // Add event listener
    addEventListener(event, callback) {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, []);
        }
        this.listeners.get(event).push(callback);
    }

    // Remove event listener
    removeEventListener(event, callback) {
        if (this.listeners.has(event)) {
            const callbacks = this.listeners.get(event);
            const index = callbacks.indexOf(callback);
            if (index > -1) {
                callbacks.splice(index, 1);
            }
        }
    }

    // Notify all listeners for an event
    notifyListeners(event, data) {
        if (this.listeners.has(event)) {
            this.listeners.get(event).forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error('Error in WebSocket event listener:', error);
                }
            });
        }
    }

    // Disconnect WebSocket
    disconnect() {
        if (this.ws) {
            this.ws.close(1000, 'User initiated disconnect');
        }
        this.isConnected = false;
        this.stopHeartbeat();
        clearTimeout(this.connectionTimeout);
    }

    // Get connection status
    getConnectionStatus() {
        return {
            connected: this.isConnected,
            readyState: this.ws ? this.ws.readyState : WebSocket.CLOSED
        };
    }
}

// Create singleton instance
const bbwWebSocketService = new BBWWebSocketService();

export default bbwWebSocketService; 