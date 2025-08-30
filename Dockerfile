# Python 3.11 슬림 버전을 기반 이미지로 사용
FROM python:3.11-alpine

# 작업 디렉토리 설정
WORKDIR /app

# 의존성 파일 복사 및 설치
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# 소스 코드 복사
COPY . .

ENV KAKAO_API_KEY=your-api-key-here

# 서버가 실행될 포트 노출 (main.py에서 8000번 포트 사용)
EXPOSE 8000

# 서버 실행
CMD ["uv", "index.py"]