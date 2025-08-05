#!/bin/bash

echo "Загрузка Docker-образов из папки tar_files..."

# Ждем пока docker daemon будет доступен
while ! docker ps > /dev/null 2>&1; do
    echo "Ожидание docker daemon..."
    sleep 1
done

# Проверяем существование сети
if ! docker network inspect marketflow >/dev/null 2>&1; then
    echo "Сеть marketflow не найдена, создаем..."
    docker network create marketflow
fi

# Загружаем образы
for tar_file in tar_files/*.tar; do
    echo "Загрузка $tar_file..."
    docker load -i "$tar_file"
done

echo "Запуск контейнеров бирж..."

# Запускаем контейнеры с биржами
docker run --rm -d \
    --name exchange1 \
    --network marketflow \
    -p 40101:40101 \
    exchange1

docker run --rm -d \
    --name exchange2 \
    --network marketflow \
    -p 40102:40102 \
    exchange2

docker run --rm -d \
    --name exchange3 \
    --network marketflow \
    -p 40103:40103 \
    exchange3

echo "Проверка запущенных контейнеров:"
docker ps | grep exchange

while true; do
    sleep 3600
done