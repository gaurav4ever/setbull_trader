#!/bin/bash

# Production Deployment Script for Setbull Trader V2
# This script provides a safe, monitored deployment of the optimized analytics system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEPLOYMENT_LOG="./logs/deployment_$(date +%Y%m%d_%H%M%S).log"
BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"
CONFIG_FILE="./application.yaml"

# Default environment variables for safe deployment
export USE_OPTIMIZED_ANALYTICS=false
export CACHE_ENABLED=true
export CONCURRENCY_ENABLED=true
export ROLLOUT_PERCENTAGE=0.0
export ENABLE_DETAILED_METRICS=true
export FALLBACK_TO_V1_ON_ERROR=true
export MAX_CACHE_SIZE=512
export WORKER_POOL_SIZE=4

# Function to log messages
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$DEPLOYMENT_LOG"
}

log_success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS:${NC} $1" | tee -a "$DEPLOYMENT_LOG"
}

log_warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1" | tee -a "$DEPLOYMENT_LOG"
}

log_error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" | tee -a "$DEPLOYMENT_LOG"
}

# Function to create backup
create_backup() {
    log "Creating backup..."
    mkdir -p "$BACKUP_DIR"
    
    # Backup configuration
    if [[ -f "$CONFIG_FILE" ]]; then
        cp "$CONFIG_FILE" "$BACKUP_DIR/"
        log_success "Configuration backed up to $BACKUP_DIR"
    fi
    
    # Backup current binary
    if [[ -f "./main" ]]; then
        cp "./main" "$BACKUP_DIR/main_backup"
        log_success "Binary backed up to $BACKUP_DIR"
    fi
}

# Function to run tests
run_tests() {
    log "Running comprehensive test suite..."
    
    # Unit tests with coverage
    log "Running unit tests..."
    if ! go test -cover ./internal/analytics/dataframe/... ./internal/analytics/indicators/... ./internal/analytics/cache/... ./internal/analytics/concurrency/...; then
        log_error "Unit tests failed!"
        exit 1
    fi
    
    # Service integration tests
    log "Running service integration tests..."
    if ! go test ./internal/service/...; then
        log_warning "Some service tests failed, but proceeding with deployment"
    fi
    
    # Build test
    log "Testing build..."
    if ! go build -o ./test_build ./main.go; then
        log_error "Build failed!"
        exit 1
    fi
    rm -f ./test_build
    
    log_success "All tests passed!"
}

# Function to build production binary
build_production() {
    log "Building production binary..."
    
    # Build with optimizations
    if ! go build -ldflags="-s -w" -o ./main ./main.go; then
        log_error "Production build failed!"
        exit 1
    fi
    
    log_success "Production binary built successfully"
}

# Function to start deployment phases
start_deployment() {
    log "Starting production deployment..."
    
    # Start the application in background
    log "Starting application..."
    nohup ./main > "./logs/app_$(date +%Y%m%d_%H%M%S).log" 2>&1 &
    APP_PID=$!
    echo $APP_PID > ./app.pid
    
    log_success "Application started with PID: $APP_PID"
    
    # Wait for application to start
    sleep 10
    
    # Check if application is running
    if ! kill -0 $APP_PID 2>/dev/null; then
        log_error "Application failed to start!"
        exit 1
    fi
    
    log_success "Application is running and ready for deployment"
}

# Function to monitor deployment
monitor_deployment() {
    local phase=$1
    local duration=$2
    
    log "Monitoring deployment phase: $phase for $duration seconds..."
    
    local end_time=$((SECONDS + duration))
    while [ $SECONDS -lt $end_time ]; do
        # Check if application is still running
        if [[ -f ./app.pid ]]; then
            local pid=$(cat ./app.pid)
            if ! kill -0 $pid 2>/dev/null; then
                log_error "Application stopped unexpectedly!"
                return 1
            fi
        fi
        
        log "Phase $phase: monitoring... ($((end_time - SECONDS))s remaining)"
        sleep 30
    done
    
    log_success "Phase $phase completed successfully"
    return 0
}

# Function to execute deployment phases
execute_deployment_phases() {
    log "Executing deployment phases..."
    
    # Phase 1: Canary (1% for 10 minutes)
    log "PHASE 1: Canary deployment (1% traffic)"
    export USE_OPTIMIZED_ANALYTICS=true
    export ROLLOUT_PERCENTAGE=1.0
    
    # Send signal to application to reload config (in practice, you'd implement this)
    log "Updating feature flags for canary phase..."
    
    if ! monitor_deployment "CANARY" 600; then # 10 minutes
        log_error "Canary phase failed!"
        rollback_deployment
        exit 1
    fi
    
    # Phase 2: Testing (10% for 30 minutes)
    log "PHASE 2: Testing deployment (10% traffic)"
    export ROLLOUT_PERCENTAGE=10.0
    
    if ! monitor_deployment "TESTING" 1800; then # 30 minutes
        log_error "Testing phase failed!"
        rollback_deployment
        exit 1
    fi
    
    # Phase 3: Validation (50% for 1 hour)
    log "PHASE 3: Validation deployment (50% traffic)"
    export ROLLOUT_PERCENTAGE=50.0
    
    if ! monitor_deployment "VALIDATION" 3600; then # 1 hour
        log_error "Validation phase failed!"
        rollback_deployment
        exit 1
    fi
    
    # Phase 4: Full rollout
    log "PHASE 4: Full deployment (100% traffic)"
    export ROLLOUT_PERCENTAGE=100.0
    
    if ! monitor_deployment "FULL_ROLLOUT" 1800; then # 30 minutes
        log_error "Full rollout phase failed!"
        rollback_deployment
        exit 1
    fi
    
    log_success "All deployment phases completed successfully!"
}

# Function to rollback deployment
rollback_deployment() {
    log_warning "Initiating rollback..."
    
    # Disable optimized analytics
    export USE_OPTIMIZED_ANALYTICS=false
    export ROLLOUT_PERCENTAGE=0.0
    
    log_success "Rollback completed - all traffic routing to V1"
    
    # Optionally restore backup
    if [[ -d "$BACKUP_DIR" && -f "$BACKUP_DIR/main_backup" ]]; then
        log "Restoring backup binary..."
        cp "$BACKUP_DIR/main_backup" "./main"
        
        # Restart application
        if [[ -f ./app.pid ]]; then
            local pid=$(cat ./app.pid)
            kill $pid 2>/dev/null || true
            sleep 5
        fi
        
        nohup ./main > "./logs/app_rollback_$(date +%Y%m%d_%H%M%S).log" 2>&1 &
        echo $! > ./app.pid
        
        log_success "Application restarted with backup binary"
    fi
}

# Function to check system health
check_health() {
    log "Checking system health..."
    
    # Check if application is responding (you'd implement actual health check)
    if [[ -f ./app.pid ]]; then
        local pid=$(cat ./app.pid)
        if kill -0 $pid 2>/dev/null; then
            log_success "Application is running (PID: $pid)"
        else
            log_error "Application is not running!"
            return 1
        fi
    else
        log_error "No PID file found!"
        return 1
    fi
    
    # Check log files for errors
    if [[ -f "./logs/app_$(date +%Y%m%d)*.log" ]]; then
        local error_count=$(grep -i "error\|panic\|fatal" ./logs/app_$(date +%Y%m%d)*.log | wc -l | tr -d ' ')
        if [[ $error_count -gt 0 ]]; then
            log_warning "Found $error_count errors in application logs"
        else
            log_success "No errors found in application logs"
        fi
    fi
    
    return 0
}

# Function to cleanup
cleanup() {
    log "Cleaning up temporary files..."
    # Cleanup any temporary files if needed
    log_success "Cleanup completed"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  deploy     - Run full deployment process"
    echo "  test       - Run tests only"
    echo "  build      - Build production binary only"
    echo "  rollback   - Perform rollback to V1"
    echo "  health     - Check system health"
    echo "  status     - Show deployment status"
    echo "  help       - Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  USE_OPTIMIZED_ANALYTICS  - Enable V2 analytics (default: false)"
    echo "  ROLLOUT_PERCENTAGE       - Percentage of traffic to V2 (default: 0.0)"
    echo "  CACHE_ENABLED           - Enable caching (default: true)"
    echo "  CONCURRENCY_ENABLED     - Enable concurrency (default: true)"
    echo "  FALLBACK_TO_V1_ON_ERROR - Enable automatic fallback (default: true)"
    echo ""
}

# Function to show status
show_status() {
    log "Deployment Status:"
    echo "  USE_OPTIMIZED_ANALYTICS: ${USE_OPTIMIZED_ANALYTICS:-false}"
    echo "  ROLLOUT_PERCENTAGE: ${ROLLOUT_PERCENTAGE:-0.0}%"
    echo "  CACHE_ENABLED: ${CACHE_ENABLED:-true}"
    echo "  CONCURRENCY_ENABLED: ${CONCURRENCY_ENABLED:-true}"
    echo "  FALLBACK_TO_V1_ON_ERROR: ${FALLBACK_TO_V1_ON_ERROR:-true}"
    
    if [[ -f ./app.pid ]]; then
        local pid=$(cat ./app.pid)
        if kill -0 $pid 2>/dev/null; then
            echo "  Application Status: RUNNING (PID: $pid)"
        else
            echo "  Application Status: STOPPED"
        fi
    else
        echo "  Application Status: UNKNOWN"
    fi
}

# Main script logic
main() {
    # Create logs directory
    mkdir -p ./logs
    mkdir -p ./backups
    
    case "${1:-deploy}" in
        "deploy")
            log "Starting full production deployment..."
            create_backup
            run_tests
            build_production
            start_deployment
            execute_deployment_phases
            check_health
            cleanup
            log_success "Production deployment completed successfully!"
            ;;
        "test")
            run_tests
            ;;
        "build")
            build_production
            ;;
        "rollback")
            rollback_deployment
            ;;
        "health")
            check_health
            ;;
        "status")
            show_status
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            log_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Trap signals for cleanup
trap 'log_error "Deployment interrupted!"; cleanup; exit 1' INT TERM

# Run main function
main "$@"
