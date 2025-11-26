🚀 Multi-Tenant PDF Summary Ingestion Service

A scalable, AI-powered microservice that dynamically provisions tenant-specific databases, extracts text from PDFs, generates intelligent summaries, and stores all data in a fully isolated multi-tenant architecture.

This project demonstrates strong backend engineering, distributed systems thinking, DevOps automation, and AI integration — all bundled into a production-grade service.

🔥 Key Features
🏷️ Multi-Tenant Architecture

Each tenant is completely isolated using dynamically created tenant-specific databases.

Master DB stores tenant metadata.

Tenant DBs are created on-the-fly during the upload process.

📄 PDF Ingestion & AI Summary

Upload a PDF via REST API (/upload).

Extracts full text using a PDF parser.

Summarizes content using an AI LLM (OpenAI / Gemini / Llama / etc.).

Stores all results in tenant DB.

🛢 Dual Database Design
Component	Database Type
Master DB	Relational (PostgreSQL / CockroachDB)
Tenant DB	NoSQL (MongoDB / Scylla / Cassandra / Elastic etc.)
📦 Storage

Original PDFs stored in local/cloud bucket.

Reference saved to tenant DB.

⚙️ Cloud-Native Infrastructure

Fully containerized with Docker

Deployable on Kubernetes (Minikube/kind/Cloud)

Tenant DB provisioning via Terraform / IaC

Works in local and cloud environments

🧠 Architecture Overview
                ┌──────────────┐
 Request        │   /upload    │
──────────────► │   API       │
                └──────┬───────┘
                       │
                       ▼
             ┌─────────────────────┐
             │ Check tenant exists │
             └─────────┬──────────┘
                       │ No
                       ▼
         ┌──────────────────────────────┐
         │ Dynamically create tenant DB │
         │   via Terraform / IaC       │
         └───────────┬─────────────────┘
                     │ Yes
                     ▼
          ┌─────────────────────────┐
          │ Extract PDF text        │
          └───────────┬────────────┘
                      │
                      ▼
          ┌─────────────────────────┐
          │ Generate AI Summary     │
          └───────────┬────────────┘
                      │
                      ▼
        ┌──────────────────────────────────┐
        │ Store (Text, Summary, Metadata, │
        │       FileRef) in tenant DB     │
        └──────────────────────────────────┘

🧩 Tech Stack
Backend

Golang (High-performance microservice)

REST API with clean architecture + modular design

Databases

PostgreSQL / CockroachDB (Master DB)

MongoDB / OpenSearch / Scylla / Cassandra (Tenant DB)

AI

OpenAI / Gemini / Llama (configurable)

Infrastructure

Docker

Kubernetes

Terraform (dynamic DB provisioning)

Minikube / Kind / Cloud deployment ready

📌 REST API: /upload
Request

multipart/form-data

tenantName

file (PDF)

Response
{
  "tenant": "alphaTech",
  "summary": "AI generated summary...",
  "status": "stored successfully"
}

🗂 Project Structure (Sample)
.
├── cmd
│   └── server
├── internal
│   ├── ai
│   ├── pdf
│   ├── storage
│   ├── tenants
│   ├── config
│   └── db
├── deployments
│   ├── docker
│   ├── k8s
│   └── terraform
└── README.md

🚀 How to Run
1️⃣ Clone Repo
git clone https://github.com/<your-repo>
cd project

2️⃣ Build Docker image
docker build -t pdf-summarizer .

3️⃣ Run Locally
docker compose up

4️⃣ Hit Upload API
POST http://localhost:8080/upload

🎯 Future Enhancements

Tenant deletion API

Auth / API key validation

Helm charts

Full multi-cloud support

Asynchronous processing via event queues (Kafka/RabbitMQ)

🏆 Author

Nikita — Backend Developer (Golang)
Working on distributed systems, cloud, AI integrations & microservices.
