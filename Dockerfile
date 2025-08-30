FROM ghcr.io/astral-sh/uv:python3.12-alpine

# 작업 디렉토리 설정
WORKDIR /app

# 소스 코드 복사
COPY . .

RUN uv venv && uv pip install -r requirements.txt

# 서버가 실행될 포트 노출 (main.py에서 8000번 포트 사용)
EXPOSE 8000

# 서버 실행
CMD ["uv", "run", "src/index.py"]