# PulsePoll — NextGen Real-Time Live Polling Platform

[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![Redis](https://img.shields.io/badge/Redis-Pub%2FSub%20%2B%20Counters-DC382D?logo=redis)](https://redis.io)
[![MongoDB](https://img.shields.io/badge/MongoDB-Audit%20%26%20Persistence-47A248?logo=mongodb)](https://mongodb.com)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A production-grade, high-concurrency live polling platform built for real-world audience interaction. When voters submit their choice from their phones or laptops, every connected presenter and audience screen updates in real time with zero page refreshes.

Built for the **HCL GUVI Developer Task**.

---

## 🌟 Key Highlights & Extra Features

- **🖥️ Direct Terminal Runner & Interactive CLI Suite**: Run the entire stack with a single terminal command (`./pulsepoll start` or `./run.sh`). Includes a standalone **Terminal Live Stream Monitor** (`./pulsepoll watch <id>`) that renders real-time updating ASCII bar charts over WebSockets, plus an interactive poll creator, voter, and concurrent traffic simulator.
- **Sub-millisecond Real-Time Synchronization**: Backed by **Redis Pub/Sub** and **Gorilla WebSockets** with automatic reconnection and heartbeat keep-alive.
- **Race-Condition-Free Voting**: Uses Redis atomic `HINCRBY` operations for instant counter increments without database lock contention.
- **Duplicate Vote Prevention**: Redis `SADD` set membership checks enforce one vote per device/voter identity at \(O(1)\) time complexity.
- **Live Viewer Presence**: Real-time tracker showing active audience viewer counts (`INCR` / `DECR` on WebSocket connection lifecycle).
- **Durable Persistence & Audit Trail**: MongoDB stores user profiles, poll metadata, and an immutable log of every vote (timestamp, voter ID, IP, user-agent) for auditability.
- **QR Code & Instant Sharing**: One-click SVG QR code generator, copyable short links, and direct WhatsApp/X share links.
- **Presenter & Fullscreen Mode**: High-contrast, large-display layout ideal for stage projectors and live screen sharing.
- **Creator Management & Analytics**: Open/close toggle, poll deletion, live percentage breakdowns, and 1-click **CSV export** of audit logs.
- **Celebratory UI/UX Polish**: Glassmorphic dark aesthetic, animated gradient progress bars, winner leader badge, and confetti explosion on vote submission.

---

## 🏗️ Architecture & Stack Decisions

### Separation of Concerns
The project strictly separates concerns into independent services:
- `/backend`: Go service powered by the **Gin** HTTP framework. Handles JWT auth, input validation, WebSocket multiplexing, Redis caching/pub-sub, and MongoDB persistence.
- `/frontend`: React SPA built with **Vite**, **Tailwind CSS**, and **Lucide Icons**. Contains clean component isolation, custom hooks (`usePollWebSocket`), and responsive layout.

```
                     ┌────────────────────────┐
                     │    NextGen React UI    │
                     │  (Voter / Presenter)   │
                     └───────┬────────┬───────┘
                             │        │
               HTTP / REST   │        │ WebSocket Stream
             (Login / Vote)  │        │ (Live Events)
                             ▼        ▼
                     ┌────────────────────────┐
                     │    Go (Gin) Service    │
                     │   - Input Validation   │
                     │   - JWT Auth & Bcrypt  │
                     │   - WS Upgrader Hub    │
                     └───────┬────────┬───────┘
                             │        │
               Atomic Ops &  │        │ Durable Storage
               Pub/Sub       │        │ & Audit Logs
                             ▼        ▼
                ┌────────────────┐   ┌────────────────┐
                │  Redis 7       │   │  MongoDB 6     │
                │  (Real-Time)   │   │  (Persistence) │
                └────────────────┘   └────────────────┘
```

---

## 🧠 Why Redis and MongoDB Are Both Doing Meaningful Work

A common anti-pattern is using Redis only as a key-value passthrough or using MongoDB with heavy polling. In PulsePoll, each datastore plays a distinct, high-impact architectural role:

### 1. Redis: The Real-Time High-Throughput Engine
- **Atomic Counters (`HINCRBY poll:{id}:votes {opt} 1`)**: In a poll with thousands of concurrent audience voters, writing directly to a disk-based SQL/NoSQL document causes write-lock queues and race conditions. Redis executes vote tallies in-memory in microseconds.
- **Duplicate Prevention (`SADD poll:{id}:voters {voterId}`)**: Redis sets check and register voter participation in \(O(1)\) constant time, rejecting duplicate votes immediately before touching the database.
- **Pub/Sub Fan-Out (`PUBLISH poll:{id}:updates`)**: Enables horizontal scalability. If multiple Go backend instances are running behind a load balancer, any vote on instance A is immediately published across the Redis channel and delivered to WebSocket clients connected to instance B.
- **Presence Tracking (`INCR/DECR poll:{id}:viewers`)**: Tracks real-time active audience viewers with instant updates pushed to all participants.

### 2. MongoDB: The Durable Document & Audit Store
- **User & Credential Storage**: Secure salted bcrypt password hashes and creator profiles.
- **Poll Schema & State**: Stores poll questions, configured options, creator relationships, active/closed state, and expiration.
- **Asynchronous Audit History (`votes` collection)**: Every cast vote is written to a dedicated `votes` collection in the background. This records voter fingerprint, IP, user-agent, and RFC3339 timestamps for analytics, verification, and CSV export without slowing down the voter's real-time experience.

---

## 🚀 Quick Start & Local Run

### Prerequisites
- Go 1.22+ and Node.js 18+ *(or Docker & Docker Compose)*
- Redis *(optional, automatically started or falls back to in-memory engine)*
- MongoDB *(optional, falls back to in-memory store)*

---

### ⚡ Method 1: Direct Run Through Terminal (Zero Setup)

Simply run the master executable in the project root:

```bash
./pulsepoll start
# OR using the alias:
./run.sh
# OR using make:
make start
```

This single command:
1. Performs automatic environment & dependency checks.
2. Connects to or launches local Redis (or activates the zero-crash fallback engine).
3. Connects to MongoDB (or activates the in-memory persistence fallback).
4. Boots the **Go (Gin) Backend** on `http://localhost:8080`.
5. Boots the **React (Vite) Frontend** on `http://localhost:5173`.
6. Seeds an initial demo poll and demo account (`demo@pulsepoll.io` / `demopass123`) so you can test instantly.

---

### 💻 Method 2: The Interactive Terminal Suite

You can interact with live polls directly inside your terminal window:

```bash
# Launch interactive terminal UI (browse, create, vote, watch)
./pulsepoll cli

# List all active polls in terminal
./pulsepoll list

# Interactively create a poll from terminal
./pulsepoll create

# Cast a vote from terminal
./pulsepoll vote <poll-id>

# ⚡ LIVE STREAM: Watch real-time updating ASCII bar chart over WebSockets/Redis
./pulsepoll watch <poll-id>

# 🚀 SIMULATE: Fire 100 concurrent votes to test Redis atomic throughput
./pulsepoll simulate <poll-id> 100
```

---

### Method 3: 1-Click Run with Docker Compose

If you prefer multi-container Docker:

```bash
./pulsepoll docker
# OR: docker compose up --build
```
Services boot automatically:
- **Frontend App**: [http://localhost:3000](http://localhost:3000)
- **Backend API**: [http://localhost:8080](http://localhost:8080)
- **Health Check**: [http://localhost:8080/api/health](http://localhost:8080/api/health)
- **MongoDB**: `localhost:27017`
- **Redis**: `localhost:6379`

---

### Option B: Manual Local Development

#### 1. Start MongoDB and Redis
```bash
# Start Redis (port 6379)
redis-server

# Start MongoDB (port 27017)
mongod --dbpath ./data/db
```

#### 2. Start Go Backend
```bash
cd backend
go mod download
go run main.go
```
*The backend starts at `http://localhost:8080`.*

#### 3. Start React Frontend
```bash
cd frontend
npm install
npm run dev
```
*The frontend starts at `http://localhost:5173`.*

---

## 🌐 Generating the Live Public Host Link

The brief states: *"Actually deployed. Ship it to a live, working link, not just localhost."*

You have two simple ways to obtain a live working link:

### Method 1: Instant Live Tunnel (Zero Cloud Setup Needed)
While running the application locally:
```bash
# Using localtunnel (no installation required)
npx localtunnel --port 3000

# OR using Cloudflare Quick Tunnel
brew install cloudflared
cloudflared tunnel --url http://localhost:3000
```
This generates a live public HTTPS URL (e.g., `https://pulsepoll-live.loca.lt` or `https://xyz.trycloudflare.com`) accessible from any smartphone or browser anywhere in the world.

---

### Method 2: Permanent Cloud Hosting (Render / Railway)

1. **Database & Cache (Free Tier)**:
   - **MongoDB Atlas**: Create a free M0 cluster at [mongodb.com/atlas](https://www.mongodb.com/cloud/atlas) and copy the connection string.
   - **Upstash Redis**: Create a free serverless Redis database at [upstash.com](https://upstash.com) and copy the `rediss://...` URI.
2. **Deploy on Render**:
   - Push this repo to GitHub.
   - Go to [render.com](https://render.com) -> New -> **Blueprint**.
   - Connect the repo: Render reads [`render.yaml`](./render.yaml) and automatically provisions both the Go backend and React frontend!
   - Fill in your `MONGO_URI` and `REDIS_URI` in the dashboard.

---

## 🎬 3–5 Minute Submission Video Guide

Per submission instructions, record a short **3–5 minute video** (unlisted YouTube or Google Drive link). Below is a battle-tested walkthrough script tailored to the evaluation criteria:

### Part 1: Product Demo (1.5 minutes)
1. **Show Auth & Poll Creation**: Log in using the Demo Account button (`demo@pulsepoll.io`), create a new live poll with 3–4 options.
2. **Side-by-Side Multi-Window Test**:
   - Open the poll in Window A (Audience/Voter).
   - Open the poll in Window B (Presenter).
   - Cast a vote in Window A. Show that Window B updates **instantly** without refreshing.
   - Attempt to vote again in Window A to demonstrate duplicate vote protection.
3. **Show Features**: QR code sharing, active viewer count, CSV audit export, and closing the poll.

### Part 2: The Challenge That Gave the Most Trouble (1.5 minutes)
> **Script Suggestion**:
> *"The biggest architectural challenge was handling high-frequency real-time updates without overwhelming the persistent database or causing race conditions between concurrent voters.*
>
> *If two users vote at the exact same millisecond, standard database read-modify-write patterns suffer from write collisions. We solved this by using Redis as the front-line shock absorber: Redis atomic `HINCRBY` tallies votes in sub-milliseconds and checks duplicate voters via `SADD`. Redis Pub/Sub then broadcasts the state to all connected WebSockets immediately. MongoDB persistence is handled asynchronously in a background goroutine for audit logging, ensuring the voter never experiences latency and the database is never overwhelmed."*

### Part 3: Did You Use AI Tools? (1 minute)
> **Script Suggestion**:
> *"Yes, I utilized AI coding assistance (Antigravity with Gemini) during development. The AI was particularly helpful in scaffolding the initial Go Gin boilerplate, generating the Tailwind glassmorphic styling system, and setting up the multi-container Docker configuration.*
>
> *However, the core distributed state architecture—specifically designing the Redis hash keys, implementing the WebSocket hub lifecycle with automatic viewer presence tracking, and ensuring asynchronous MongoDB audit writes—required careful human oversight, manual validation, and testing to ensure race condition safety and robust edge-case handling."*

---

## 📬 Submission Checklist

Before emailing **dev@hclguvi.com**:
- [ ] Public GitHub repository link.
- [ ] Live deployed link (verified reachable on mobile and desktop).
- [ ] 3–5 minute walkthrough video (unlisted YouTube or public Google Drive).
- [ ] Send to: `dev@hclguvi.com`.

---

*All the best with your submission and technical interview!*
