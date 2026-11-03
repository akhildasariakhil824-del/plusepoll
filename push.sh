#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

echo "=========================================="
echo "   🚀 PulsePoll GitHub Push Helper        "
echo "=========================================="

if [ -z "$1" ]; then
    printf "Paste your GitHub Token (ghp_...): "
    read -r TOKEN
else
    TOKEN="$1"
fi

# Trim whitespace
TOKEN=$(echo "$TOKEN" | tr -d '[:space:]')

if [ -z "$TOKEN" ]; then
    echo "❌ Error: Token cannot be empty."
    exit 1
fi

echo "Configuring repository origin..."
git remote set-url origin "https://${TOKEN}@github.com/akhildasariakhil824-del/pulsepoll.git"

echo "Pushing branch 'main' to GitHub..."
git push -u origin main

echo ""
echo "🎉 SUCCESS! Your repository is now live at:"
echo "👉 https://github.com/akhildasariakhil824-del/pulsepoll"
