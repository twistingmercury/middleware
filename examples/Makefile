default: run

.PHONY:server
server:
	docker build --file server/Dockerfile ./server -t example/server:latest

.PHONY: client
client:
	docker build --file client/Dockerfile ./client -t example/client:latest

.PHONY: prune
prune:
	docker system prune -f

.PHONY:run
run: server client prune
	docker compose up


