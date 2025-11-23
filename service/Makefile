GOCMD := go
DOCKER_COMPOSE := deployment/docker-compose.yml
DOCKER_COMPOSE_TEST := deployment/docker-compose.test.yml

all: setup coverage up

setup:
	go mod tidy

up:
	sudo docker-compose  -f $(DOCKER_COMPOSE) up --build

down:
	sudo docker-compose -f $(DOCKER_COMPOSE) down -v

test:
	go clean -testcache
	sudo docker-compose -f $(DOCKER_COMPOSE_TEST) down -v
	sudo docker system prune -f
	sudo docker-compose -f $(DOCKER_COMPOSE_TEST) build --no-cache
	sudo docker-compose -f $(DOCKER_COMPOSE_TEST)  up -d test-db
	sleep 5
	sudo docker-compose -f $(DOCKER_COMPOSE_TEST)  run --rm test-app \
		go test -v -coverprofile=coverage/coverage.out ./internal/...
	sudo docker-compose -f $(DOCKER_COMPOSE_TEST) down -v

coverage: test
	sudo docker-compose -f $(DOCKER_COMPOSE_TEST)  run --rm test-app \
		go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	echo "Отчет покрытия: coverage/coverage.html"
	open ./deployment/coverage/coverage.html

clean:
	sudo rm -f ./deployment/coverage/coverage.out ./deployment/coverage/coverage.html 
	sudo docker stop $$(sudo docker ps -a -q) 
	sudo docker rm $$(sudo docker ps -a -q) 
