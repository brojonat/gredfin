define setup_env
	$(eval ENV_FILE := $(1))
	$(eval include $(1))
	$(eval export)
endef

build-cli:
	go build -o ./cli cmd/*.go

build-push-server:
	$(call setup_env, server/.env)
	docker build -f Dockerfile.server -t ${SERVER_IMG_TAG} .
	docker push ${SERVER_IMG_TAG}

migrate:
	$(call setup_env, server/.env)
	atlas schema apply \
	--url "${DATABASE_URL}" \
	--dev-url "postgres://postgres:postgres@localhost:5432?sslmode=disable" \
	--to "file://server/sqlc/schema.sql"

run-http-server:
	$(call setup_env, server/.env)
	./cli run http-server -d ${DATABASE_URL} --log-level -4

run-search-worker:
	$(call setup_env, worker/.env)
	./cli run search-worker \
	--server-endpoint ${SERVER_ENDPOINT} \
	--auth-token ${AUTH_TOKEN} \
	--user-agent ${REDFIN_USER_AGENT} \
	-i 5s --log-level -4

run-property-worker:
	$(call setup_env, worker/.env)
	./cli run property-worker \
	--server-endpoint ${SERVER_ENDPOINT} \
	--auth-token ${AUTH_TOKEN} \
	--user-agent ${REDFIN_USER_AGENT} \
	-i 5s --log-level -4

deployment-server:
	$(call setup_env, server/.env)
	@sed -e "s;{{DOCKER_REPO}};$(DOCKER_REPO);g" server/k8s/server.yaml | \
	sed -e "s;{{SERVER_IMG_TAG}};$(SERVER_IMG_TAG);g" | \
	sed -e "s;{{SERVER_PORT}};$(SERVER_PORT);g" | \
	sed -e "s;{{DATABASE_URL}};$(DATABASE_URL);g" | \
	sed -e "s;{{SERVER_SECRET_KEY}};$(SERVER_SECRET_KEY);g" | \
	sed -e "s;{{AWS_REGION}};$(AWS_REGION);g" | \
	sed -e "s;{{AWS_ACCESS_KEY_ID}};$(AWS_ACCESS_KEY_ID);g" | \
	sed -e "s;{{AWS_SECRET_ACCESS_KEY}};$(AWS_SECRET_ACCESS_KEY);g" | \
	sed -e "s;{{S3_PROPERTY_BUCKET}};$(S3_PROPERTY_BUCKET);g" | \
	sed -e "s;{{CORS_ORIGINS}};$(CORS_ORIGINS);g" | \
	sed -e "s;{{CORS_METHODS}};$(CORS_METHODS);g" | \
	sed -e "s;{{CORS_HEADERS}};$(CORS_HEADERS);g" | \
	sed -e "s;{{REDFIN_USER_AGENT}};$(REDFIN_USER_AGENT);g" | \
	sed -e "s;{{FIREBASE_CONFIG}};$(FIREBASE_CONFIG);g"

backup-db:
	$(call setup_env, server/.env)
	docker run -it --rm \
	-v ./.pgdump:/.pgdump \
	postgres pg_dump -d ${DATABASE_URL} -Fc -b -v -f .pgdump/pgdump.sql
	mv .pgdump/pgdump.sql .
	rmdir .pgdump

restore-db:
	$(call setup_env, server/.env.restore-db)
	docker run -it --rm \
	-v ./pgdump.sql:/pgdump.sql \
	postgres pg_restore -d ${DATABASE_URL} -v -j 2 --no-owner pgdump.sql
