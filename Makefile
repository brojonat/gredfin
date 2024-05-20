define setup_env
	$(eval ENV_FILE := $(1))
	$(eval include $(1))
	$(eval export)
endef

build-cli:
	go build -o ./cli cmd/*.go

migrate:
	$(call setup_env, server/.env)
	atlas schema apply \
	--url "${DATABASE_URL}" \
	--dev-url "docker://postgres" \
	--to "file://server/sqlc/schema.sql"

run-http-server:
	$(call setup_env, server/.env)
	./cli run-http-server -d ${DATABASE_URL}

run-search-worker:
	$(call setup_env, worker/.env)
	./cli run-search-worker --server-endpoint ${SERVER_ENDPOINT} --auth-token ${AUTH_TOKEN} -i 5s

run-property-worker:
	$(call setup_env, worker/.env)
	./cli run-property-worker --server-endpoint ${SERVER_ENDPOINT} --auth-token ${AUTH_TOKEN} -i 5s


