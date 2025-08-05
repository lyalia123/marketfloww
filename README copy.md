# 🚀 MarketFlow: Real-Time Crypto Market Data Processor

![MarketFlow Demo](https://babypips-discourse-media-production.s3.amazonaws.com/original/3X/e/3/e39ae94f422443a667a190678196b03af5c56cc2.gif)

## 📌 Overview

MarketFlow is a high-performance system for processing real-time cryptocurrency market data from multiple exchanges. It efficiently handles concurrent data streams, processes updates in real-time, and provides a REST API for querying price information.

✨ **Key Features**:
- Real-time data ingestion from multiple exchanges
- Intelligent caching with Redis
- PostgreSQL for reliable data storage
- Concurrent processing with worker pools
- REST API for price queries
- Support for both live and test modes

## 🏗️ Architecture

<!-- ![Architecture Diagram](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExNnR1Z2c4d2hxN3VnZ3R5ZzR4eGJ6Y3BqZzZ1bWZ6YiZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/3ohzdIuqJoi8Q8kYAg/giphy.gif) -->

The system follows **Hexagonal Architecture** with clear separation of concerns:
- Domain Layer → Business logic and models
- Application Layer → Use cases and workflows
- Adapters Layer → External integrations (Web, DB, Cache)


## 🛠️ Technologies

<!-- ![Tech Stack](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExNnR1Z2c4d2hxN3VnZ3R5ZzR4eGJ6Y3BqZzZ1bWZ6YiZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/3ohzdIuqJoi8Q8kYAg/giphy.gif) -->

- **Go** (with goroutines for concurrency)
- **PostgreSQL** (data storage)
- **Redis** (caching)
- **Docker** (containerization)
- **slog** (structured logging)

## 🚀 Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.22+
- Redis

### Installation
```bash
# Clone the repository
git git@git.platform.alem.school:lalpieva/marketflow.git
cd marketflow

# Build and start containers
docker-compose up --build
