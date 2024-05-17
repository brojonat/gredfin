define setup_env
	$(eval ENV_FILE := $(1))
	$(eval include $(1))
	$(eval export)
endef

run-local:
	$(call setup_env, .env.local)
	CGO_ENABLED=0 GOOS=darwin go build -o server .
	./server

migrate:
	$(call setup_env, .env.local)
	atlas schema apply \
	--url "${DATABASE_URL}" \
	--dev-url "docker://postgres" \
	--to "file://sqlc_schema.sql"

automigrate:
	$(call setup_env, .env.local)
	atlas schema apply \
	--url "{{DATABASE_URL}}" \
	--dev-url "docker://postgres" \
	--to "file://sqlc_schema.sql" \
	--auto-approve