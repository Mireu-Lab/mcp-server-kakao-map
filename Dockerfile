# --- 1. 빌드 스테이지 ---
FROM golang:1.24-alpine AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 의존성 다운로드를 위해 go.mod와 go.sum 파일을 먼저 복사
COPY src/go.mod src/go.sum ./
RUN go mod download

# 소스 코드 복사
COPY src/ .

# 애플리케이션 빌드. CGO_ENABLED=0은 정적 바이너리를 생성하여 OS 의존성을 줄입니다.
RUN CGO_ENABLED=0 GOOS=linux go build -o /server main.go

# --- 2. 최종 스테이지 ---
FROM alpine:latest

# 작업 디렉토리 설정
WORKDIR /app

# 빌드 스테이지에서 컴파일된 바이너리만 복사
COPY --from=builder /server .

# 서버가 실행될 포트 노출
EXPOSE 8080

# 컨테이너 실행 시 서버 바이너리 실행
CMD ["/app/server"]