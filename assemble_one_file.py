#!/usr/bin/env python3
import os

root_dir = "/Users/akhil/.gemini/antigravity/scratch/pulsepoll"
output_file = os.path.join(root_dir, "FULL_PROJECT_IN_ONE_FILE.md")

files_to_bundle = [
    # Readme & configs
    "README.md",
    "docker-compose.yml",
    "render.yaml",
    "deploy.sh",
    
    # Backend
    "backend/main.go",
    "backend/config/config.go",
    "backend/models/models.go",
    "backend/database/store.go",
    "backend/database/redis.go",
    "backend/handlers/auth.go",
    "backend/handlers/poll.go",
    "backend/middleware/auth.go",
    "backend/middleware/cors.go",
    "backend/websocket/hub.go",
    "backend/go.mod",
    "backend/Dockerfile",
    
    # Frontend
    "frontend/package.json",
    "frontend/vite.config.js",
    "frontend/index.html",
    "frontend/nginx.conf",
    "frontend/Dockerfile",
    "frontend/src/main.jsx",
    "frontend/src/App.jsx",
    "frontend/src/index.css",
    "frontend/src/api/client.js",
    "frontend/src/context/AuthContext.jsx",
    "frontend/src/hooks/usePollWebSocket.js",
    "frontend/src/components/Navbar.jsx",
    "frontend/src/components/AuthModal.jsx",
    "frontend/src/components/CreatePollModal.jsx",
    "frontend/src/components/ShareModal.jsx",
    "frontend/src/pages/Dashboard.jsx",
    "frontend/src/pages/PollView.jsx",
]

def get_lang(filepath):
    ext = os.path.splitext(filepath)[1].lower()
    mapping = {
        '.go': 'go',
        '.jsx': 'jsx',
        '.js': 'javascript',
        '.json': 'json',
        '.css': 'css',
        '.html': 'html',
        '.yml': 'yaml',
        '.yaml': 'yaml',
        '.sh': 'bash',
        '.mod': 'text',
        '.conf': 'nginx',
        '.md': 'markdown'
    }
    if 'Dockerfile' in filepath:
        return 'dockerfile'
    return mapping.get(ext, 'text')

with open(output_file, 'w', encoding='utf-8') as out:
    out.write("# PulsePoll — NextGen Real-Time Live Polling Platform\n")
    out.write("### Complete Project Source Code & Documentation in One File\n\n")
    out.write("> **Task**: HCL GUVI Developer Internship Task\n")
    out.write("> **Tech Stack**: React 18 (Frontend) | Go 1.22 Gin (Backend) | Redis 7 (Realtime Engine) | MongoDB 6 (Persistence)\n")
    out.write("> **Submission Recipient**: dev@hclguvi.com\n\n")
    out.write("---\n\n")
    out.write("## Table of Contents\n")
    for f in files_to_bundle:
        anchor = f.replace('/', '').replace('.', '').replace('-', '').lower()
        out.write(f"- [{f}](#{anchor})\n")
    out.write("\n---\n\n")

    for f in files_to_bundle:
        full_path = os.path.join(root_dir, f)
        if not os.path.exists(full_path):
            continue
        anchor = f.replace('/', '').replace('.', '').replace('-', '').lower()
        out.write(f'<a id="{anchor}"></a>\n')
        out.write(f"## File: `{f}`\n\n")
        lang = get_lang(f)
        out.write(f"```{lang}\n")
        with open(full_path, 'r', encoding='utf-8', errors='replace') as src:
            out.write(src.read().strip())
        out.write("\n```\n\n---\n\n")

print(f"✔ Successfully assembled {output_file}")
