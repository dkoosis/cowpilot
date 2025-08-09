# Add to Makefile
.PHONY: rtm-predeploy-check
rtm-predeploy-check:
	@echo "Pre-deploy OAuth validation..."
	@go test ./internal/core -run TestCriticalOAuthEndpoints
	@echo "✓ OAuth endpoints validated"

rtm-deploy: rtm-predeploy-check
	flyctl deploy --config fly-rtm.toml --app rtm
	@sleep 5
	@bash scripts/diagnostics/check_rtm_oauth.sh
