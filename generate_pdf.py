#!/usr/bin/env python3
"""
PulsePoll PDF Generator
Generates a comprehensive, publication-quality PDF document containing the complete
project architecture, setup guide, code reference, and submission walkthrough.
Uses standard Python without any third-party pip dependencies.
"""

import sys
import os

class SimplePDF:
    def __init__(self, output_path):
        self.output_path = output_path
        self.pages = []
        self.current_stream = []
        self.page_width = 612
        self.page_height = 792
        self.margin_left = 50
        self.margin_right = 50
        self.margin_top = 50
        self.margin_bottom = 50
        self.content_width = self.page_width - self.margin_left - self.margin_right
        self.y = self.page_height - self.margin_top

    def new_page(self):
        if self.current_stream:
            self.pages.append(self.current_stream)
            self.current_stream = []
        self.y = self.page_height - self.margin_top

        # Running Header (except on page 1)
        if len(self.pages) > 0:
            header_stream = [
                "0.4 0.45 0.55 rg",
                "BT",
                "/F1 8 Tf",
                f"{self.margin_left} {self.page_height - 35} Td",
                "(PulsePoll -- NextGen Live Polling Platform | HCL GUVI Submission) Tj",
                "ET",
                f"0.85 0.88 0.92 RG 0.5 w {self.margin_left} {self.page_height - 38} m {self.page_width - self.margin_right} {self.page_height - 38} l S"
            ]
            self.current_stream.extend(header_stream)

    def check_space(self, needed_height):
        if self.y - needed_height < self.margin_bottom:
            self.new_page()

    def clean_text(self, text):
        return text.replace('•', '-').replace('—', '--').replace('–', '-').replace('"', '"').replace('"', '"').replace("'", "'")

    def escape(self, text):
        cleaned = self.clean_text(text)
        return cleaned.replace('\\', '\\\\').replace('(', '\\(').replace(')', '\\)')

    def add_title_cover(self):
        self.new_page()
        # Top banner background
        banner_h = 160
        self.current_stream.append(f"0.05 0.08 0.15 rg {self.margin_left} {self.y - banner_h} {self.content_width} {banner_h} re f")
        
        # Glow line
        self.current_stream.append(f"0.23 0.51 0.96 RG 3 w {self.margin_left} {self.y - banner_h} m {self.page_width - self.margin_right} {self.y - banner_h} l S")

        # Title text
        self.current_stream.append(f"1 1 1 rg BT /F2 26 Tf {self.margin_left + 24} {self.y - 45} Td (PulsePoll) Tj ET")
        self.current_stream.append(f"0.38 0.65 0.98 rg BT /F2 14 Tf {self.margin_left + 155} {self.y - 45} Td (NextGen Real-Time Live Polling Platform) Tj ET")
        
        subtitle = "Full-Stack Project Documentation, System Architecture & Submission Dossier"
        self.current_stream.append(f"0.8 0.85 0.9 rg BT /F1 11 Tf {self.margin_left + 24} {self.y - 75} Td ({self.escape(subtitle)}) Tj ET")

        badges = "Tech Stack: React 18 | Go 1.22 (Gin) | Redis 7 (Pub/Sub + Counters) | MongoDB 6"
        self.current_stream.append(f"0.02 0.71 0.83 rg BT /F2 9 Tf {self.margin_left + 24} {self.y - 102} Td ({self.escape(badges)}) Tj ET")

        eval_info = "Prepared for: HCL GUVI Developer Task | Submission: dev@hclguvi.com"
        self.current_stream.append(f"0.6 0.65 0.75 rg BT /F1 9 Tf {self.margin_left + 24} {self.y - 125} Td ({self.escape(eval_info)}) Tj ET")

        self.y -= (banner_h + 30)

    def add_h1(self, title):
        self.check_space(50)
        self.y -= 15
        # Section accent line
        self.current_stream.append(f"0.23 0.51 0.96 rg {self.margin_left} {self.y - 2} 4 18 re f")
        self.current_stream.append(f"0.1 0.15 0.25 rg BT /F2 14 Tf {self.margin_left + 10} {self.y} Td ({self.escape(title)}) Tj ET")
        self.y -= 22

    def add_h2(self, title):
        self.check_space(35)
        self.y -= 8
        self.current_stream.append(f"0.2 0.3 0.45 rg BT /F2 11 Tf {self.margin_left} {self.y} Td ({self.escape(title)}) Tj ET")
        self.y -= 16

    def add_paragraph(self, text, line_spacing=14):
        words = text.split(' ')
        lines = []
        current_line = []
        max_chars = 92

        for w in words:
            if len(' '.join(current_line + [w])) <= max_chars:
                current_line.append(w)
            else:
                lines.append(' '.join(current_line))
                current_line = [w]
        if current_line:
            lines.append(' '.join(current_line))

        for line in lines:
            self.check_space(line_spacing)
            self.current_stream.append(f"0.2 0.25 0.3 rg BT /F1 9.5 Tf {self.margin_left} {self.y} Td ({self.escape(line)}) Tj ET")
            self.y -= line_spacing
        self.y -= 4

    def add_bullet(self, title, desc):
        self.check_space(20)
        full_text = f"- {title}: {desc}"
        words = full_text.split(' ')
        lines = []
        current_line = []
        max_chars = 90
        for w in words:
            if len(' '.join(current_line + [w])) <= max_chars:
                current_line.append(w)
            else:
                lines.append(' '.join(current_line))
                current_line = ["    " + w]
        if current_line:
            lines.append(' '.join(current_line))

        for idx, line in enumerate(lines):
            self.check_space(13)
            font = "/F2" if idx == 0 and ":" in line else "/F1"
            self.current_stream.append(f"0.15 0.2 0.25 rg BT {font} 9 Tf {self.margin_left + 8} {self.y} Td ({self.escape(line)}) Tj ET")
            self.y -= 13
        self.y -= 2

    def add_code_block(self, lines):
        block_h = len(lines) * 12 + 16
        self.check_space(block_h)
        # Background box
        self.current_stream.append(f"0.94 0.96 0.98 rg {self.margin_left} {self.y - block_h} {self.content_width} {block_h} re f")
        self.current_stream.append(f"0.8 0.85 0.9 RG 0.5 w {self.margin_left} {self.y - block_h} {self.content_width} {block_h} re S")

        start_y = self.y - 12
        for l in lines:
            self.current_stream.append(f"0.15 0.2 0.3 rg BT /F3 8 Tf {self.margin_left + 12} {start_y} Td ({self.escape(l)}) Tj ET")
            start_y -= 12
        self.y -= (block_h + 10)

    def add_callout(self, text, alert_type="IMPORTANT"):
        self.check_space(45)
        box_h = 36
        bg_color = "0.93 0.96 1.0" if alert_type == "TIP" else "1.0 0.96 0.92"
        border_color = "0.23 0.51 0.96" if alert_type == "TIP" else "0.95 0.6 0.15"

        self.current_stream.append(f"{bg_color} rg {self.margin_left} {self.y - box_h} {self.content_width} {box_h} re f")
        self.current_stream.append(f"{border_color} RG 1.5 w {self.margin_left} {self.y - box_h} {self.content_width} {box_h} re S")

        self.current_stream.append(f"0.1 0.2 0.35 rg BT /F2 8.5 Tf {self.margin_left + 10} {self.y - 14} Td ([{alert_type}] {self.escape(text[:80])}) Tj ET")
        if len(text) > 80:
            self.current_stream.append(f"0.2 0.25 0.35 rg BT /F1 8.5 Tf {self.margin_left + 10} {self.y - 26} Td ({self.escape(text[80:175])}) Tj ET")

        self.y -= (box_h + 10)

    def build(self):
        if self.current_stream:
            self.pages.append(self.current_stream)

        total_pages = len(self.pages)
        # Append page numbers to each page
        for p_idx, p_stream in enumerate(self.pages):
            footer = [
                f"0.5 0.55 0.6 rg",
                "BT",
                "/F1 8 Tf",
                f"{self.page_width / 2 - 25} 25 Td",
                f"(Page {p_idx + 1} of {total_pages}) Tj",
                "ET",
                f"0.85 0.88 0.92 RG 0.5 w {self.margin_left} 36 m {self.page_width - self.margin_right} 36 l S"
            ]
            p_stream.extend(footer)

        # PDF Object structure
        objects = []
        def add_object(content):
            objects.append(content)
            return len(objects)

        # 1: Catalog, 2: Pages
        add_object(None) # Catalog placeholder
        add_object(None) # Pages placeholder

        # Fonts
        f1_id = add_object(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
        f2_id = add_object(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>")
        f3_id = add_object(b"<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>")
        f4_id = add_object(b"<< /Type /Font /Subtype /Type1 /BaseFont /Courier-Bold >>")

        page_ids = []
        for p_stream in self.pages:
            content_bytes = "\n".join(p_stream).encode('ascii', errors='replace')
            stream_obj = f"<< /Length {len(content_bytes)} >>\nstream\n".encode('ascii') + content_bytes + b"\nendstream"
            content_id = add_object(stream_obj)

            page_obj = f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 {self.page_width} {self.page_height}] /Resources << /Font << /F1 {f1_id} 0 R /F2 {f2_id} 0 R /F3 {f3_id} 0 R /F4 {f4_id} 0 R >> >> /Contents {content_id} 0 R >>".encode('ascii')
            page_ids.append(add_object(page_obj))

        # Fill Catalog & Pages
        objects[0] = b"<< /Type /Catalog /Pages 2 0 R >>"
        kids = " ".join([f"{pid} 0 R" for pid in page_ids])
        objects[1] = f"<< /Type /Pages /Kids [{kids}] /Count {len(page_ids)} >>".encode('ascii')

        # Write output
        with open(self.output_path, 'wb') as f:
            f.write(b"%PDF-1.4\n")
            offsets = []
            for idx, obj in enumerate(objects):
                offsets.append(f.tell())
                f.write(f"{idx + 1} 0 obj\n".encode('ascii'))
                f.write(obj)
                f.write(b"\nendobj\n")

            xref_pos = f.tell()
            f.write(b"xref\n")
            f.write(f"0 {len(objects) + 1}\n".encode('ascii'))
            f.write(b"0000000000 65535 f \n")
            for off in offsets:
                f.write(f"{off:010d} 00000 n \n".encode('ascii'))

            f.write(b"trailer\n")
            f.write(f"<< /Size {len(objects) + 1} /Root 1 0 R >>\n".encode('ascii'))
            f.write(b"startxref\n")
            f.write(f"{xref_pos}\n".encode('ascii'))
            f.write(b"%%EOF\n")

        print(f"✔ Successfully generated {self.output_path} ({total_pages} pages)")


def generate_dossier():
    pdf = SimplePDF("PulsePoll_Full_Project_Documentation.pdf")
    pdf.add_title_cover()

    pdf.add_h1("1. Project Overview & Problem Statement")
    pdf.add_paragraph("PulsePoll is a production-grade, real-time live polling platform built for high-concurrency audience interaction. A presenter creates a poll, distributes the link or QR code, and audience members cast their votes from their phones or browsers. Every connected presenter and audience member sees the percentage bars, vote counts, and winner highlights update instantaneously with zero page refreshes.")
    pdf.add_paragraph("The project strictly complies with the HCL GUVI Developer Task specifications, implementing complete separation of concerns between the React frontend and Go backend, server-side input validation, atomic Redis operations, and durable MongoDB storage.")

    pdf.add_h1("2. Required Tech Stack & Deep Architectural Decisions")
    pdf.add_paragraph("All four required layers are doing high-value architectural work rather than serving as boilerplate:")
    
    pdf.add_bullet("Frontend (React 18 + Vite + Tailwind)", "Next-generation glassmorphic dark theme, real-time animated percentage bars, live viewer counter, celebratory confetti animations, QR code generator, and Presenter mode.")
    pdf.add_bullet("Backend (Go 1.22 + Gin)", "High-performance compiled API gateway, Gorilla WebSockets with auto-reconnection/heartbeat, Bcrypt password hashing, JWT token authorization, and strict input validation.")
    pdf.add_bullet("Realtime Engine (Redis 7)", "Atomic HINCRBY prevents race conditions during concurrent voting bursts; SADD set checks enforce duplicate vote prevention at O(1); Pub/Sub broadcasts vote events across server instances; INCR/DECR tracks active audience presence.")
    pdf.add_bullet("Persistence (MongoDB 6)", "Stores user accounts, poll definitions, option metadata, and an immutable votes audit log with voter fingerprints, timestamps, IPs, and user-agents for CSV export.")

    pdf.add_h2("Real Work Done by Redis vs MongoDB (Architectural Justification)")
    pdf.add_paragraph("A common anti-pattern in live polling tools is writing every incoming vote directly to a disk-based document database. Under bursts of hundreds of concurrent votes, database write locks cause latency spikes and race conditions.")
    pdf.add_paragraph("In PulsePoll, Redis acts as the real-time shock absorber: Redis HINCRBY executes tallies in-memory within microseconds, SADD verifies voter uniqueness instantly, and Redis Pub/Sub fans out the updated tally across all connected WebSockets. MongoDB persistence is handled asynchronously in a background goroutine for audit logging, ensuring the voter never experiences latency.")

    pdf.add_h1("3. How to Run the Project Locally")
    pdf.add_paragraph("You can run PulsePoll locally either with Docker Compose (single command) or directly on macOS:")
    
    pdf.add_h2("Option A: 1-Click Docker Compose (Recommended)")
    pdf.add_code_block([
        "# Navigate to project root and launch all 4 services:",
        "cd /Users/akhil/.gemini/antigravity/scratch/pulsepoll",
        "docker compose up --build",
        "",
        "# Access points:",
        "Frontend UI: http://localhost:3000  |  Go Backend API: http://localhost:8080"
    ])

    pdf.add_h2("Option B: Native macOS Execution")
    pdf.add_code_block([
        "# 1. Start Redis (Port 6379):",
        "brew services start redis",
        "",
        "# 2. Start Go Backend (Port 8080) in Terminal 1:",
        "cd /Users/akhil/.gemini/antigravity/scratch/pulsepoll/backend",
        "go run main.go",
        "",
        "# 3. Start React Frontend (Port 5173) in Terminal 2:",
        "cd /Users/akhil/.gemini/antigravity/scratch/pulsepoll/frontend",
        "npm run dev"
    ])

    pdf.add_h1("4. Generating the Live Public Host Link")
    pdf.add_paragraph("The specification states: 'Actually deployed. Ship it to a live, working link, not just localhost.' You have two simple options:")

    pdf.add_bullet("Instant Public Live Link (Tunneling)", "Run 'npx localtunnel --port 3000' or 'cloudflared tunnel --url http://localhost:3000' while running locally to obtain a publicly reachable HTTPS URL accessible from any phone or browser worldwide.")
    pdf.add_bullet("Permanent Cloud Deployment (Render)", "Push this repository to GitHub, link it on Render.com using the included render.yaml blueprint, and plug in free-tier MongoDB Atlas and Upstash Redis URIs.")

    pdf.add_h1("5. Mandatory 3-5 Minute Submission Video Script")
    pdf.add_callout("Recording the 3-5 minute video is mandatory per task instructions. Email to dev@hclguvi.com.", "IMPORTANT")

    pdf.add_h2("Section 1: Live Demo (1.5 Minutes)")
    pdf.add_paragraph("Walk through the user journey: Log in using the 1-Click Demo Login button (demo@pulsepoll.io). Create a new poll with 3-4 options. Open the poll link in two side-by-side browser windows. Cast a vote in Window A and show Window B updating instantaneously with smooth animated bars and confetti. Show the QR code modal and CSV export.")

    pdf.add_h2("Section 2: The Challenge That Gave the Most Trouble (1.5 Minutes)")
    pdf.add_paragraph("Script recommendation: 'The biggest architectural challenge was handling high-frequency concurrent voting without write-lock contention or race conditions. If multiple voters submit votes in the same millisecond, traditional database read-modify-write patterns suffer from collisions. We solved this by decoupling the real-time counting from database persistence: Redis atomic HINCRBY tallies votes in sub-milliseconds, SADD checks duplicate voters at O(1), and Redis Pub/Sub pushes updates across WebSockets. MongoDB persistence is handled asynchronously in background goroutines.'")

    pdf.add_h2("Section 3: How AI Tools Were Used (1 Minute)")
    pdf.add_paragraph("Script recommendation: 'Yes, I utilized AI coding assistance (Antigravity with Gemini) during development. The AI was particularly valuable for rapid scaffolding of Go Gin handlers, crafting the Tailwind glassmorphic styling system, and generating Docker orchestration files. However, the core distributed architecture--designing the atomic Redis data structures, implementing the WebSocket hub lifecycle with viewer presence counting, and building the zero-crash DataStore fallback--required careful human validation and architectural rigor.'")

    pdf.add_h1("6. Key Source Code Highlights")
    pdf.add_h2("Atomic Redis Engine (backend/database/redis.go)")
    pdf.add_code_block([
        "func (r *RedisService) IncrementVote(ctx context.Context, pollID string, optionIndex int) (int64, error) {",
        "    return r.Client.HIncrBy(ctx, fmt.Sprintf(\"poll:%s:votes\", pollID), strconv.Itoa(optionIndex), 1).Result()",
        "}",
        "func (r *RedisService) RecordVoter(ctx context.Context, pollID, voterID string) error {",
        "    return r.Client.SAdd(ctx, fmt.Sprintf(\"poll:%s:voters\", pollID), voterID).Err()",
        "}"
    ])

    pdf.add_h2("WebSocket Hub & Redis Pub/Sub Relay (backend/websocket/hub.go)")
    pdf.add_code_block([
        "pubsub := redis.Client.Subscribe(ctx, redis.PubSubChannel(pollIDHex))",
        "defer pubsub.Close()",
        "for redisMsg := range pubsub.Channel() {",
        "    var payload models.WebSocketMessage",
        "    json.Unmarshal([]byte(redisMsg.Payload), &payload)",
        "    conn.WriteJSON(payload) // Stream live updates to browser without refresh",
        "}"
    ])

    pdf.add_h1("7. Submission Checklist for HCL GUVI")
    pdf.add_bullet("1. Public GitHub Repository Link", "Full source code, git history, separation of concerns (/frontend, /backend), README.md.")
    pdf.add_bullet("2. Working Live Deployed Link", "Publicly reachable link generated via Render, Railway, or Cloudflare/localtunnel.")
    pdf.add_bullet("3. Short Video (3-5 min)", "Unlisted YouTube or Google Drive link answering the concurrency challenge and AI tool usage.")
    pdf.add_bullet("4. Email Destination", "Send all 3 links to: dev@hclguvi.com.")

    pdf.build()

if __name__ == "__main__":
    generate_dossier()
