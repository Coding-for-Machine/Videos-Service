#!/bin/bash

# YouTube Clone - Setup Script
echo "🎥 YouTube Clone - Loyiha Setup"
echo "=================================="

# Ranglar
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Papkalarni yaratish
echo -e "${YELLOW}📁 Papkalarni yaratish...${NC}"
mkdir -p config database models services handlers workers middleware public uploads/videos uploads/thumbnails

# 2. Go modulini boshlash
echo -e "${YELLOW}🔧 Go modulini boshlash...${NC}"
if [ ! -f "go.mod" ]; then
    go mod init youtube-clone
fi

# 3. Dependencies o'rnatish
echo -e "${YELLOW}📦 Dependencies o'rnatish...${NC}"
go get github.com/gofiber/fiber/v2
go get github.com/gocql/gocql
go get github.com/minio/minio-go/v7
go get github.com/redis/go-redis/v9
go get github.com/google/uuid

# 4. Environment faylini yaratish
echo -e "${YELLOW}⚙️  Environment faylini yaratish...${NC}"
if [ ! -f ".env" ]; then
    cp .env.example .env
    echo -e "${GREEN}✓ .env fayli yaratildi${NC}"
fi

# 5. Docker tekshirish
echo -e "${YELLOW}🐳 Docker tekshirilmoqda...${NC}"
if command -v docker &> /dev/null; then
    echo -e "${GREEN}✓ Docker o'rnatilgan${NC}"
else
    echo -e "${RED}✗ Docker topilmadi. Iltimos Docker o'rnating.${NC}"
    exit 1
fi

if command -v docker compose &> /dev/null; then
    echo -e "${GREEN}✓ Docker Compose o'rnatilgan${NC}"
else
    echo -e "${RED}✗ Docker Compose topilmadi. Iltimos o'rnating.${NC}"
    exit 1
fi

# 6. FFmpeg tekshirish
echo -e "${YELLOW}🎬 FFmpeg tekshirilmoqda...${NC}"
if command -v ffmpeg &> /dev/null; then
    echo -e "${GREEN}✓ FFmpeg o'rnatilgan${NC}"
else
    echo -e "${YELLOW}⚠ FFmpeg topilmadi. Video processing ishlamaydi.${NC}"
    echo "Linux: sudo apt install ffmpeg"
    echo "Mac: brew install ffmpeg"
fi

# 7. Docker Compose servicelarini ishga tushirish
echo -e "${YELLOW}🚀 Docker Compose servicelarini ishga tushirish...${NC}"
docker-compose up -d

echo ""
echo -e "${GREEN}✨ Setup tugadi!${NC}"
echo ""
echo "📍 Servicelar:"
echo "   Application:    http://localhost:3000"
echo "   MinIO Console:  http://localhost:9001"
echo "   Cassandra:      localhost:9042"
echo "   Redis:          localhost:6379"
echo ""
echo "🔧 Keyingi qadamlar:"
echo "   1. Application ishga tushguncha kuting (30-60 soniya)"
echo "   2. Browser ochib http://localhost:3000 ga kiring"
echo "   3. Video yuklang va test qiling"
echo ""
echo "📝 Commandlar:"
echo "   make logs    - Loglarni ko'rish"
echo "   make stop    - Servicelarni to'xtatish"
echo "   make clean   - Hamma narsani o'chirish"
echo ""
echo -e "${GREEN}Happy Coding! 🚀${NC}"
