define setup_env
	$(eval ENV_FILE := $(1))
	$(eval include $(1))
	$(eval export)
endef

build-cli:
	go build -o ./cli cmd/*.go

build-push-cli:
	$(call setup_env, server/.env)
	CGO_ENABLED=0 GOOS=linux go build -o ./cli cmd/*.go
	docker build -f Dockerfile -t ${CLI_IMG_TAG} .
	docker push ${CLI_IMG_TAG}

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

deploy-server:
	$(call setup_env, server/.env)
	@$(MAKE) build-push-cli
	kustomize build --load-restrictor=LoadRestrictionsNone server/k8s | \
	sed -e "s;{{DOCKER_REPO}};$(DOCKER_REPO);g" | \
	sed -e "s;{{CLI_IMG_TAG}};$(CLI_IMG_TAG);g" | \
	kubectl apply -f -
	kubectl rollout restart deployment gredfin-backend

deploy-search-worker:
	$(call setup_env, worker/.env)
	@$(MAKE) build-push-cli
	kustomize build --load-restrictor=LoadRestrictionsNone worker/k8s/search | \
	sed -e "s;{{DOCKER_REPO}};$(DOCKER_REPO);g" | \
	sed -e "s;{{CLI_IMG_TAG}};$(CLI_IMG_TAG);g" | \
	kubectl apply -f -
	kubectl rollout restart deployment gredfin-search-worker

deploy-property-worker:
	$(call setup_env, worker/.env)
	@$(MAKE) build-push-cli
	kustomize build --load-restrictor=LoadRestrictionsNone worker/k8s/property | \
	sed -e "s;{{DOCKER_REPO}};$(DOCKER_REPO);g" | \
	sed -e "s;{{CLI_IMG_TAG}};$(CLI_IMG_TAG);g" | \
	kubectl apply -f -
	kubectl rollout restart deployment gredfin-property-worker

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
