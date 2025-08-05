# Makefile для управления docker-контейнерами

CONTAINERS = exchange1 exchange2 exchange3

.PHONY: start stop clean

start:
	@echo "🛑 Остановка всех запущенных контейнеров (если есть)..."
	@if [ -n "$$(docker ps -q)" ]; then docker stop $$(docker ps -q); fi
	@echo "🗑 Удаление всех контейнеров (если есть)..."
	@if [ -n "$$(docker ps -aq)" ]; then docker rm $$(docker ps -aq); fi
	@echo "🚀 Сборка и запуск docker-compose..."
	docker-compose up --build

stop:
	@echo "🛑 Остановка контейнеров: $(CONTAINERS)..."
	@docker ps -a --format '{{.Names}}' | grep -qE '(^| )($(CONTAINERS))($$| )' && docker stop $(CONTAINERS) || echo "Контейнеры не найдены"
	@echo "🗑 Удаление контейнеров: $(CONTAINERS)..."
	@docker ps -a --format '{{.Names}}' | grep -qE '(^| )($(CONTAINERS))($$| )' && docker rm $(CONTAINERS) || echo "Контейнеры уже удалены"
	@echo "🧹 Завершение docker-compose..."
	docker-compose down

clean:
	@echo "🧹 Полная очистка docker: контейнеры, тома, сети..."
	@if [ -n "$$(docker ps -aq)" ]; then docker stop $$(docker ps -aq); fi
	@if [ -n "$$(docker ps -aq)" ]; then docker rm $$(docker ps -aq); fi
	@docker volume prune -f
	@docker network prune -f
	@docker-compose down -v