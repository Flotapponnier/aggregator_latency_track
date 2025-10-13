# ============================================================================
# Aggregator Latency Monitor with Grafana Dashboard
# ============================================================================

BINARY_NAME = latency_monitor
GO_FILES = ./cmd/script

.PHONY: help
help:
	@echo "Aggregator Latency Monitor - Grafana Dashboard"
	@echo "=============================================="
	@echo ""
	@echo "Commands:"
	@echo "  make run      - Start everything (Grafana + Monitor in background)"
	@echo "  make stop     - Stop all services"
	@echo "  make logs     - Follow monitor logs in real-time"
	@echo "  make status   - Show status of all services"
	@echo "  make build    - Build the Go binary only"
	@echo "  make clean    - Stop services and remove binary"
	@echo "  make destroy  - Remove everything including volumes (asks confirmation)"
	@echo ""
	@echo "Dashboard Access:"
	@echo "  Grafana:    http://localhost:3000 (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Metrics:    http://localhost:2112/metrics"
	@echo ""

.PHONY: deps
deps:
	@echo "📦 Downloading dependencies..."
	@go mod download
	@echo "✓ Dependencies ready"
	@echo ""

.PHONY: build
build: deps
	@echo "🔨 Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(GO_FILES)
	@echo "✓ Build complete: $(BINARY_NAME)"
	@echo ""

.PHONY: start-grafana
start-grafana:
	@echo "📊 Starting Grafana + Prometheus stack..."
	@docker-compose up -d
	@echo "✓ Grafana stack running"
	@echo "  → Grafana:    http://localhost:3000 (admin/admin)"
	@echo "  → Prometheus: http://localhost:9090"
	@echo ""

.PHONY: run
run: build start-grafana
	@echo "🚀 Starting Aggregator Latency Monitor in background..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./$(BINARY_NAME) > monitor.log 2>&1 & echo $$! > monitor.pid
	@sleep 2
	@if [ -f monitor.pid ]; then \
		echo "✓ Monitor started (PID: $$(cat monitor.pid))"; \
		echo "✓ Logs: tail -f monitor.log"; \
		echo "✓ Stop: make stop"; \
		echo ""; \
	else \
		echo "❌ Failed to start monitor"; \
	fi

.PHONY: down
down:
	@echo "🛑 Stopping services..."
	@if [ -f monitor.pid ]; then \
		echo "  → Stopping monitor (PID: $$(cat monitor.pid))..."; \
		kill $$(cat monitor.pid) 2>/dev/null || true; \
		rm -f monitor.pid; \
	fi
	@echo "  → Killing all latency_monitor processes..."
	@pkill -9 latency_monitor 2>/dev/null || true
	@echo "  → Stopping Docker containers..."
	@docker-compose down 2>/dev/null || true
	@docker stop prometheus grafana 2>/dev/null || true
	@docker rm prometheus grafana 2>/dev/null || true
	@echo "✓ All services stopped"

.PHONY: stop
stop: down

.PHONY: clean
clean: down
	@echo "🧹 Cleaning binary..."
	@rm -f $(BINARY_NAME)
	@echo "✓ Clean complete"

.PHONY: logs
logs:
	@if [ -f monitor.log ]; then \
		tail -f monitor.log; \
	else \
		echo "❌ No log file found. Is the monitor running?"; \
		echo "   Run 'make run' first"; \
	fi

.PHONY: status
status:
	@echo "📊 Service Status:"
	@echo ""
	@if [ -f monitor.pid ] && kill -0 $$(cat monitor.pid) 2>/dev/null; then \
		echo "  ✓ Monitor:    Running (PID: $$(cat monitor.pid))"; \
	else \
		echo "  ✗ Monitor:    Stopped"; \
	fi
	@if docker-compose ps | grep -q "Up"; then \
		echo "  ✓ Grafana:    Running (http://localhost:3000)"; \
		echo "  ✓ Prometheus: Running (http://localhost:9090)"; \
	else \
		echo "  ✗ Grafana:    Stopped"; \
		echo "  ✗ Prometheus: Stopped"; \
	fi
	@echo ""

.PHONY: destroy
destroy:
	@echo "⚠️  WARNING: This will remove all containers, volumes, and the binary!"
	@echo "⚠️  All Grafana dashboards and Prometheus data will be lost!"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "🗑️  Destroying everything..."; \
		if [ -f monitor.pid ]; then kill $$(cat monitor.pid) 2>/dev/null || true; rm -f monitor.pid; fi; \
		pkill -9 latency_monitor 2>/dev/null || true; \
		docker-compose down -v 2>/dev/null || true; \
		rm -f $(BINARY_NAME) monitor.log monitor.pid; \
		echo "✓ Everything destroyed"; \
	else \
		echo "❌ Cancelled"; \
	fi

.DEFAULT_GOAL := help
