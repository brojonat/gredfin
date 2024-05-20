define setup_env
	$(eval ENV_FILE := $(1))
	$(eval include $(1))
	$(eval export)
endef

build-cli:
	go build -o ./cli cmd/*.go

run-server:
	$(call setup_env, server/.env)
	./cli run-http-server -d ${DATABASE_URL}

migrate:
	$(call setup_env, server/.env)
	atlas schema apply \
	--url "${DATABASE_URL}" \
	--dev-url "docker://postgres" \
	--to "file://server/sqlc/schema.sql"
