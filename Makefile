.PHONY: run
run: clean build up pg-schema pg-populate

.PHONY: build
build:
	docker-compose build --no-cache

.PHONY: up
up:
	docker-compose up -d --force-recreate

.PHONY: clean
clean:
	docker-compose down --rmi all -v && docker volume prune -f && docker image prune -f

.PHONY: pg-schema
pg-schema:
	psql --host=localhost --port=5432 --dbname=db --user=root -f ./sql/schema.sql

.PHONY: pg-populate
pg-populate:
	psql --host=localhost --port=5432 --dbname=db --user=root -f ./sql/populate.sql

.PHONY: integration-test
integration-test:
	./tests/integration-test.sh
