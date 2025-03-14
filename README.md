# SetBull Trader

SetBull Trader is a comprehensive trading platform for managing and executing trades on the Dhan trading API. This full-stack application features a Go backend API and a Svelte frontend interface.

## Features

- **Order Management**: Place, modify, and cancel orders
- **Trade Tracking**: View today's trades and historical trade data
- **User-Friendly Interface**: Clean, responsive UI built with Svelte and Tailwind CSS
- **Real-Time Data**: Quick access to your trading information

## Architecture

### Backend (Go)

The backend is built with Go, featuring a clean, modular architecture:

- **Transport Layer**: HTTP handlers using Gin framework
- **Service Layer**: Business logic implementation
- **Adapter Layer**: Client interfaces for external services (Dhan API)
- **Model Layer**: Data structures for the application

### Frontend (Svelte)

The frontend is built with SvelteKit and includes:

- **Dashboard**: Overview of trading activity
- **Order Pages**: Forms for placing, modifying, and canceling orders
- **Trade History**: View and filter historical trades
- **Responsive Design**: Mobile-friendly interface using Tailwind CSS

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Node.js 16 or higher
- npm or yarn
- Dhan trading account and API credentials

### Installation

1. Clone the repository
   ```bash
   git clone https://github.com/yourusername/setbull_trader.git
   cd setbull_trader
   ```

2. Set up the backend
   ```bash
   # Install Go dependencies
   go mod download
   ```

3. Set up the frontend
   ```bash
   # Navigate to the frontend directory
   cd frontend
   
   # Install dependencies
   npm install
   ```

4. Configure your environment
   ```bash
   # Copy the example application configuration
   cp application.example.yaml application.dev.yaml
   
   # Edit the file with your Dhan API credentials
   # Replace YOUR_API_TOKEN and YOUR_CLIENT_ID with your actual credentials
   ```

### Running the Application

#### Option 1: Using VS Code

1. Open the project in VS Code
2. Install recommended extensions if prompted
3. Press `F5` or select "Launch Full Stack Application" from the Run and Debug panel

#### Option 2: Using Command Line

1. Start the Go backend:
   ```bash
   go run main.go
   ```

2. In a separate terminal, start the Svelte frontend:
   ```bash
   cd frontend
   npm run dev -- --open
   ```

## Development

### Project Structure

```
.
├── cmd/
│   └── trading/              # Command-line applications
│       ├── app/              # Application setup
│       └── transport/        # HTTP handlers
├── dhan/                     # Dhan API documentation
├── frontend/                 # Svelte frontend
├── internal/
│   ├── core/                 # Core business logic
│   │   ├── adapters/         # External service clients
│   │   ├── constant/         # Constants
│   │   ├── dto/              # Data transfer objects
│   │   └── service/          # Business services
│   └── trading/              # Trading specific implementation
├── pkg/                      # Shared packages
│   ├── apperrors/            # Error handling
│   ├── database/             # Database utilities
│   └── log/                  # Logging utilities
└── main.go                   # Application entry point
```

### Backend Development

- The API uses the Gin web framework
- Error handling is centralized in the `pkg/apperrors` package
- Logging is configured in the `pkg/log` package

### Frontend Development

- Located in the `frontend` directory
- Uses SvelteKit for routing and SSR capabilities
- Tailwind CSS for styling
- API services are in `src/lib/services`

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/orders` | POST | Place a new order |
| `/api/v1/orders/:orderID` | PUT | Modify an existing order |
| `/api/v1/orders/:orderID` | DELETE | Cancel an order |
| `/api/v1/trades` | GET | Get all trades for the current day |
| `/api/v1/trades/history` | GET | Get historical trade data with filtering |
| `/api/v1/health` | GET | Health check endpoint |

## Troubleshooting

### Common Issues

1. **API Connection Errors**:
   - Ensure the Go backend is running on port 8080
   - Check that your Dhan API credentials are correct in `application.dev.yaml`
   - Verify that CORS is properly configured

2. **Frontend Build Issues**:
   - Make sure all dependencies are installed: `npm install`
   - Check for compile errors in the Svelte components

3. **Order Placement Failures**:
   - Verify that your Dhan account has sufficient funds
   - Check the order parameters (quantity, price, etc.) are valid

### Logs

- Backend logs can be found in the terminal where you started the Go server
- Frontend development logs appear in the browser console

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add new feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [Dhan API](https://api.dhan.co) for providing the trading API
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [SvelteKit](https://kit.svelte.dev)
- [Tailwind CSS](https://tailwindcss.com)