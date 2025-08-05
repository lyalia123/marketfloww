```text
marketflow/
├── cmd/
│   ├── datagen/
│   │   ├── main.go 
│   │   └── Dockerfile 
│   ├── docker/
│   │   ├── tar_files/
│   │   │   ├── exchange1_amd64.tar 
│   │   │   ├── exchange1_arm64.tar 
│   │   │   ├── exchange2_amd64.tar 
│   │   │   ├── exchange2_arm64.tar 
│   │   │   ├── exchange3_amd64.tar 
│   │   │   ├── exchange3_arm64.tar 
│   │   ├──start_exchanges.sh 
│   │   └──Dockerfile 
│   └── marketflow/
│   │   ├── config.json 
│   │   ├── main.go 
│   │   └── Dockerfile    
├── internal/
│   ├── adapters/
│   │   ├── cache/
│   │   │   └── redis.go 
│   │   ├── exchange/
│   │   │   ├── listener.go 
│   │   ├── storage/
│   │   │   ├── batch.go 
│   │   │   ├── init.go 
│   │   │   ├── init.sql
│   │   │   └── postgres.go 
│   │   └── web/
│   │       ├── handler.go 
│   │       └── router.go 
│   ├── config/
│   │   ├── loadConfig.go
│   │   └── config.go
│   ├── domain/
│   │   ├── state.go
│   │   └── priceUpdate.go 
│   └── worker/
│       ├── fanin.go 
│       ├── fanout.go 
├── logs/
│   └──...
├── pkg/
│   └── logger/
│       └── logger.go
├── docs/
│   └──...
├── .env
├── docker-compose.yml
├── Dockerfile
├── config.json
├── go.mod
├── go.sum
├── README.md
├── Makefile
├── marketflow.postman_collection.json
├── marketflow
├── architecture.md
├── task.md
└── taskCommon.md
```