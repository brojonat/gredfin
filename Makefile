define setup_env
	$(eval ENV_FILE := $(1))
	$(eval include $(1))
	$(eval export)
endef

build-cli:
	go build -o cli cli/*.go

migrate:
	$(call setup_env, server/.env)
	atlas schema apply \
	--url "${DATABASE_URL}" \
	--dev-url "docker://postgres" \
	--to "file://server/sqlc/schema.sql"
