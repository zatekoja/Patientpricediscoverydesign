.PHONY: help vault-init

help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

vault-init: ## Start Vault and seed dev secrets (root compose)
	@echo "Starting Vault + seeding dev secrets..."
	docker compose up -d vault vault-init

.DEFAULT_GOAL := help
