# 🚀 Random Pics - Quick Start Guide

快速启动Random Pics项目的完整指南。

## 📋 前置要求

### 必需软件
- **Go 1.23+** - [下载安装](https://golang.org/dl/)
- **Node.js 18+** - [下载安装](https://nodejs.org/)
- **PostgreSQL 16+** 或 **SQLite**（开发用）

### 验证安装
```bash
go version    # 应显示 go version go1.23+
node --version # 应显示 v18+
npm --version  # 应显示 npm 版本
```

## 🗄️ 数据库设置

### 选项1：使用PostgreSQL（推荐生产环境）
```bash
# 安装PostgreSQL后，创建数据库
psql -U postgres -c "CREATE DATABASE randompic;"
psql -U postgres -c "CREATE USER randompic_user WITH PASSWORD 'your_password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE randompic TO randompic_user;"
```

### 选项2：使用SQLite（快速开发）
SQLite不需要额外安装，但需要CGO支持：
```bash
# Windows (需要GCC环境，如MinGW)
# 或者使用Docker方式启动
```

## ⚙️ 环境配置

### 1. 后端环境变量
在 `backend/` 目录创建 `.env` 文件：

```bash
# 数据库配置
DB_DRIVER=postgres  # 或 sqlite
DATABASE_URL=postgres://randompic_user:your_password@localhost:5432/randompic?sslmode=disable

# 如果使用SQLite（开发环境）
# DB_DRIVER=sqlite
# DATABASE_URL=./randompic.db

# JWT认证配置
JWT_SECRET=your-super-secure-jwt-secret-key-at-least-32-characters-long
JWT_ISSUER=randompic-api
JWT_AUDIENCE=randompic-users
JWT_EXPIRES_HOURS=24

# 服务器配置
PORT=8080
GIN_MODE=debug  # 生产环境用 release

# CORS配置
CORS_ORIGINS=http://localhost:4321,http://127.0.0.1:4321,http://localhost:3000
CORS_CREDENTIALS=true

# 存储配置
STORAGE_TYPE=local
LOCAL_STORAGE_PATH=./storage/images

# 速率限制
RATE_LIMIT_ENABLED=true
RATE_LIMIT_ANONYMOUS_REQUESTS=60
RATE_LIMIT_ANONYMOUS_WINDOW=900
RATE_LIMIT_AUTHENTICATED_REQUESTS=300
RATE_LIMIT_AUTHENTICATED_WINDOW=900

# 日志配置
REQUEST_LOGGING=true
SLOW_THRESHOLD=500
DB_LOG_LEVEL=warn

# 服务器超时配置
READ_TIMEOUT=10
WRITE_TIMEOUT=30
IDLE_TIMEOUT=60
MAX_BODY_SIZE=10485760
```

### 2. 前端环境变量
在 `frontend/` 目录创建 `.env` 文件：

```bash
PUBLIC_API_BASE_URL=http://localhost:8080/api
```

## 🚀 启动步骤

### 第一次启动完整流程

#### 1. 克隆项目
```bash
git clone https://github.com/ccc592/random-pics-gallery.git
cd random-pics-gallery
```

#### 2. 启动后端
```bash
# 进入后端目录
cd backend

# 安装Go依赖
go mod download

# 创建存储目录
mkdir -p storage/images

# 运行数据库迁移
go run ./cmd migrate

# 启动后端服务器
go run ./cmd
```

**后端启动成功标志**：
```
[GIN-debug] Listening and serving HTTP on :8080
Database migrations completed successfully
Server started on http://localhost:8080
```

#### 3. 启动前端（新终端窗口）
```bash
# 进入前端目录
cd frontend

# 安装Node.js依赖
npm install

# 启动开发服务器
npm run dev
```

**前端启动成功标志**：
```
🚀 astro v4.x ready in Xms
┃ Local    http://localhost:4321/
┃ Network  use --host to expose
```

## 🌐 访问应用

启动成功后，你可以访问：

- **前端应用**: http://localhost:4321
- **后端API**: http://localhost:8080/api
- **健康检查**: http://localhost:8080/api/health
- **API文档**: http://localhost:8080/api/metrics/json

## 🛠️ 开发命令

### 后端开发
```bash
cd backend

# 开发模式运行
go run ./cmd

# 运行测试
go test ./...

# 运行特定测试
go test ./tests/contract -v
go test ./tests/integration -v

# 构建生产版本
go build -o randompic-server ./cmd

# 代码格式化
go fmt ./...

# 代码检查
golangci-lint run
```

### 前端开发
```bash
cd frontend

# 开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览构建结果
npm run preview

# 类型检查
npm run check

# 运行测试
npm run test
```

## 🐳 Docker方式启动（推荐）

### 使用Docker Compose
项目根目录下有 `docker-compose.yml`：

```bash
# 一键启动所有服务（数据库 + 后端 + 前端）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 单独使用Docker
```bash
# 构建后端镜像
cd backend
docker build -t randompics-backend .
docker run -p 8080:8080 --env-file .env randompics-backend

# 构建前端镜像
cd frontend
docker build -t randompics-frontend .
docker run -p 4321:4321 randompics-frontend
```

## 🗃️ 初始数据

### 创建管理员用户
```bash
# 使用API创建管理员用户（后端启动后）
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@randompics.local",
    "username": "admin",
    "password": "SecurePass123!",
    "role": "admin"
  }'
```

### 上传示例图片
```bash
# 获取JWT Token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@randompics.local",
    "password": "SecurePass123!"
  }'

# 使用token上传图片
curl -X POST http://localhost:8080/api/admin/images/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "image=@/path/to/your/image.jpg" \
  -F "alt=Beautiful landscape photo showing mountains and lakes" \
  -F "title=Mountain Lake Sunset" \
  -F "tags=nature,landscape,sunset,mountains" \
  -F "weight=8"
```

## 🔧 常见问题解决

### 后端启动问题

**1. 数据库连接失败**
```bash
# 检查PostgreSQL是否运行
sudo systemctl status postgresql  # Linux
brew services list | grep postgres  # macOS

# 检查数据库连接
psql -U randompic_user -d randompic -h localhost
```

**2. JWT Secret错误**
```bash
# 确保JWT_SECRET长度至少32字符且不是默认值
echo $JWT_SECRET | wc -c  # 应该 > 32
```

**3. SQLite CGO错误**
```bash
# 启用CGO编译
CGO_ENABLED=1 go run ./cmd

# 或使用PostgreSQL代替
```

### 前端启动问题

**1. 端口被占用**
```bash
# 使用不同端口
npm run dev -- --port 3000

# 或终止占用进程
lsof -ti:4321 | xargs kill -9  # macOS/Linux
netstat -ano | findstr :4321   # Windows
```

**2. API连接失败**
检查 `frontend/.env` 中的 `PUBLIC_API_BASE_URL` 设置

### 性能优化

**1. 开发环境优化**
```bash
# 后端热重载
go install github.com/cosmtrek/air@latest
air  # 在backend目录下运行

# 前端缓存清理
npm run build -- --force
```

**2. 生产环境配置**
```bash
# 设置环境变量
export GIN_MODE=release
export NODE_ENV=production

# 使用生产数据库
export DB_DRIVER=postgres
export DATABASE_URL="your-production-db-url"
```

## 🎯 下一步

启动成功后，你可以：

1. **浏览应用**: 访问 http://localhost:4321 查看前端界面
2. **测试API**: 使用Postman或curl测试API端点
3. **上传图片**: 通过管理员界面上传测试图片
4. **查看文档**: 阅读 `PROJECT_SUMMARY.md` 了解架构详情
5. **开发新功能**: 查看 `specs/` 目录了解设计文档

## 🆘 获得帮助

如果遇到问题：

1. **查看日志**: 检查后端和前端的控制台输出
2. **运行测试**: `go test ./...` 确保功能正常
3. **检查文档**: 阅读 `README.md` 和 `PROJECT_SUMMARY.md`
4. **GitHub Issues**: 在仓库中创建issue报告问题

**祝你使用愉快！** 🎉