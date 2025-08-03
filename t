.
├── build
├── BUILD_RESTORATION_COMPLETE.md
├── build_rtm.sh
├── build-check.sh
├── check-fmt.sh
├── cmd
│   ├── core
│   │   ├── main_test.go
│   │   └── main.go
│   ├── mcp_debug_proxy
│   │   └── main.go
│   ├── oauth_spec_test
│   │   ├── main.go
│   │   ├── oauth-spec-test
│   │   ├── README.md
│   │   └── run-test.sh
│   ├── rtm
│   │   └── main.go
│   ├── spektrix
│   │   └── main.go
│   └── test-examples-server
│       └── main.go
├── COMMIT_MSG.txt
├── Dockerfile
├── Dockerfile.demo
├── docs
│   ├── adr
│   │   ├── 009-mcp-sdk-selection.md
│   │   ├── 010-mcp-debug-system-architecture.md
│   │   ├── 011-conditional-compilation-lightweight-debug.md
│   │   ├── 012-runtime-debug-configuration.md
│   │   ├── 013-mcp-transport-selection.md
│   │   ├── 013-multi-server-architecture-shared-codebase.md
│   │   ├── adr-template.md
│   │   └── README.md
│   ├── assets
│   │   ├── cowgnition_logo.png
│   │   └── rtm-authentication.jpg
│   ├── AUTH_FLOWS.yaml
│   ├── claude_project_overview.md
│   ├── claude-integration
│   │   ├── deployment.md
│   │   ├── oauth-adapter-plan.md
│   │   ├── oauth-deployment.md
│   │   ├── oauth-implementation-plan.md
│   │   ├── oauth-june-2025-compliance.md
│   │   ├── oauth-research.md
│   │   ├── README.md
│   │   └── troubleshooting.md
│   ├── contributing.md
│   ├── debug
│   │   ├── mcp-conformance-plan.md
│   │   └── phase1-implementation-complete.md
│   ├── KNOWN-ISSUES.md
│   ├── llm_integration_guide.md
│   ├── LONGRUNNING_TASKS.md
│   ├── mcp_go_main_tags.txt
│   ├── oauth_implementation_log.md
│   ├── project_history.md
│   ├── project_tags.txt
│   ├── protocol-standards.md
│   ├── README.md
│   ├── reference
│   │   ├── oauth-protocol-requirements.md
│   │   ├── rtm-authentication-flow.md
│   │   ├── schema.json
│   │   └── schema.ts
│   ├── reviews
│   │   ├── code-smell-2025-07-25.md
│   │   ├── comments-docs-2025-07-25.md
│   │   ├── semantic-naming-2025-07-25.md
│   │   └── semantic-naming-2025-07-26.md
│   ├── ROADMAP.md
│   ├── RTM_ENHANCEMENTS_BACKLOG.yaml
│   ├── sessions
│   │   └── quick-start-next.md
│   ├── STATE.yaml
│   ├── TEST_STRATEGY.md
│   ├── test-formatting.md
│   ├── testing-guide.md
│   ├── testing-strategy.md
│   ├── TODO.md
│   ├── tree.txt
│   └── user-guide.md
├── fly-core-tmp.toml
├── fly-rtm.toml
├── GIT_GUIDE.md
├── go.mod
├── go.sum
├── HEALTH_CHECK_FIXED.md
├── INTEGRATION_TEST_FIX.md
├── internal
│   ├── auth
│   │   ├── csrf_token_test.go
│   │   ├── middleware.go
│   │   ├── oauth_adapter_test.go
│   │   ├── oauth_adapter.go
│   │   ├── oauth_callback_server.go
│   │   ├── oauth_callback_test.go
│   │   ├── token_store_factory.go
│   │   ├── token_store_sqlite.go.removed
│   │   ├── token_store.go
│   │   └── types.go
│   ├── core
│   │   └── infrastructure.go
│   ├── debug
│   │   ├── config.go
│   │   ├── interceptor.go
│   │   ├── storage.go
│   │   ├── validation_integration.go
│   │   └── validation_interceptor.go.disabled
│   ├── longrunning
│   │   ├── cancellation.go
│   │   ├── doc.go
│   │   ├── manager_test.go
│   │   ├── manager.go
│   │   ├── progress.go
│   │   └── task.go
│   ├── middleware
│   │   └── cors.go
│   ├── protocol
│   │   ├── specs
│   │   │   ├── middleware.go
│   │   │   ├── oauth_test_helpers.go
│   │   │   ├── oauth_test.go
│   │   │   └── oauth.go
│   │   └── validation
│   │       ├── jsonrpc.go
│   │       ├── mcp.go
│   │       ├── security.go
│   │       └── validation_engine.go
│   ├── rtm
│   │   ├── batch_handlers.go
│   │   ├── client.go
│   │   ├── credential_store_test.go
│   │   ├── credential_store.go
│   │   ├── enhanced_handlers_test.go
│   │   ├── enhanced_handlers.go
│   │   ├── handlers.go
│   │   ├── job_queue.go
│   │   ├── oauth_adapter_test_dir
│   │   ├── oauth_adapter_test.go
│   │   ├── oauth_adapter.go
│   │   ├── setup_test.go
│   │   └── setup.go
│   ├── spektrix
│   │   ├── auth.go
│   │   ├── client.go
│   │   ├── handler.go
│   │   ├── hmac.go
│   │   └── types.go
│   └── testutil
│       ├── http_test_helpers.go
│       └── test_assertions.go
├── LINTER_FIXES_COMPLETE.md
├── make-executable.sh
├── Makefile
├── makefile_help.mk
├── OAUTH_TEST_INSTRUCTIONS.md
├── prompts
│   ├── 01_semantic_naming_review.md
│   ├── 02_code_smell_analysis.md
│   ├── README.md
│   └── REVIEW-GUIDE.md
├── quick_build.sh
├── README.md
├── run-gofmt.sh
├── scripts
│   ├── cleanup-old-tests.sh
│   ├── debug-tests.sh
│   ├── deploy
│   │   ├── check-status.sh
│   │   ├── create-volume.sh
│   │   ├── debug-deployment.sh
│   │   ├── deploy-and-monitor.sh
│   │   ├── deploy-debug-to-fly.sh
│   │   ├── monitor-registration.sh
│   │   ├── README.md
│   │   └── setup.sh
│   ├── docs_cleanup.sh
│   ├── docs_reorganize_final.sh
│   ├── docs_reorganize.sh
│   ├── init_module.sh
│   ├── session_summary.sh
│   ├── setup-oauth-test.sh
│   ├── test
│   │   ├── check-oauth-build.sh
│   │   ├── debug-tools-integration-test.sh
│   │   ├── make-executable.sh
│   │   ├── mcp-inspector-integration-test.sh
│   │   ├── mcp-protocol-smoke-test.sh
│   │   ├── mcp-transport-diagnostics.sh
│   │   ├── oauth-test-suite.sh
│   │   ├── project-health-check.sh
│   │   ├── README.md
│   │   ├── run-tests.sh
│   │   ├── sse-transport-test.sh
│   │   ├── test-mcp-integration.sh
│   │   ├── test-oauth-callback.sh
│   │   ├── test-oauth-flow.sh
│   │   └── test-sse-transport.sh
│   ├── test_debug_initial.sh
│   ├── test-all-fixes.sh
│   ├── test-all-systems.sh
│   ├── test-build.sh
│   ├── test-final-fixes.sh
│   ├── test-integration-fix.sh
│   ├── test-linter-fixes.sh
│   ├── test-makefile-and-oauth.sh
│   ├── test-mcp.sh
│   ├── test-oauth-build.sh
│   ├── test-verbose.sh
│   ├── update_imports.sh
│   ├── utils
│   │   ├── parse-test-times.sh
│   │   └── track-performance.sh
│   └── validate_protocol.sh
├── t
├── test-build-rtm.sh
├── test-build.sh
├── test-longrunning.sh
├── test-rtm-build.sh
├── test-rtm-enhanced.sh
├── tests
│   ├── helpers
│   │   ├── test_debug_components.go
│   │   ├── test_fixtures.go
│   │   ├── test_phase2_validation.go
│   │   └── test_runtime_config.go
│   ├── integration
│   │   ├── debug_tools_test.go
│   │   ├── mcp_integration_test.go
│   │   ├── oauth_flow_test.go
│   │   ├── rtm_oauth_test.go
│   │   ├── rtm_server_test.go
│   │   └── server_test.go
│   ├── README.md
│   └── scenarios
│       ├── claude_connector_test.go
│       ├── DEBUG_GUIDE.md
│       ├── ENHANCED_SUMMARY.md
│       ├── FILE_INVENTORY.md
│       ├── IMPLEMENTATION_REVIEW.md
│       ├── IMPLEMENTATION_SUMMARY.md
│       ├── manual-test-examples.sh
│       ├── mcp_scenarios.sh
│       ├── quick-setup.sh
│       ├── raw_examples.sh
│       ├── raw_sse_test.sh
│       ├── README.md
│       ├── RTFM_CORRECTION.md
│       ├── scenario_test.go
│       ├── setup.sh
│       ├── TESTING_GUIDE.md
│       └── verify-inspector.sh
└── update-deps.sh

39 directories, 218 files
