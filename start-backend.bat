@echo off
cd backend
set DATABASE_URL=postgres://postgres:password@localhost:5432/randompic?sslmode=disable
set JWT_SECRET=your-super-secret-jwt-key-32-chars-minimum-length-updated
set STORAGE_TYPE=local
set S3_ENDPOINT=http://localhost:9000
set S3_BUCKET=images
set S3_ACCESS_KEY=minioadmin
set S3_SECRET_KEY=minioadmin
randompic-server.exe