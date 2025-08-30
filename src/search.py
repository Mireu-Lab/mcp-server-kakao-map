import os
import asyncio
import json
from typing import List, Dict, Any

import httpx
from dotenv import load_dotenv
from pydantic import BaseModel, Field

# .env 파일에서 환경 변수를 로드합니다.
load_dotenv()

# --- Kakao API 설정 ---
KAKAO_API_KEY = os.getenv("KAKAO_API_KEY")
KAKAO_MAP_URL = "https://dapi.kakao.com/v2/local/search/keyword.json"
DAUM_SEARCH_URL = "https://dapi.kakao.com/v2/search"

# --- Pydantic 스키마 정의 ---
class SearchSchema(BaseModel):
    query: str = Field(
        description=(
            "Korean keywords for searching places in South Korea. Typically combines "
            "place type and location (e.g., '이태원 맛집', '서울 병원', '강남역 영화관')."
        )
    )

# --- 비동기 API 호출 함수 ---
async def fetch_map_documents(client: httpx.AsyncClient, query: str) -> List[Dict[str, Any]]:
    headers = {"Authorization": f"KakaoAK {KAKAO_API_KEY}"}
    params = {"query": query}
    try:
        response = await client.get(KAKAO_MAP_URL, headers=headers, params=params)
        response.raise_for_status()
        return response.json().get("documents", [])
    except httpx.HTTPStatusError as e:
        print(f"Error fetching map documents: {e}")
        return []

async def fetch_web_documents(client: httpx.AsyncClient, query: str) -> List[Dict[str, Any]]:
    headers = {"Authorization": f"KakaoAK {KAKAO_API_KEY}"}
    params = {"query": query, "page": 1, "size": 5}
    try:
        response = await client.get(f"{DAUM_SEARCH_URL}/web", headers=headers, params=params)
        response.raise_for_status()
        return response.json().get("documents", [])
    except httpx.HTTPStatusError as e:
        print(f"Error fetching web documents for '{query}': {e}")
        return []

async def fetch_image_document(client: httpx.AsyncClient, query: str) -> Dict[str, Any]:
    headers = {"Authorization": f"KakaoAK {KAKAO_API_KEY}"}
    params = {"query": query, "page": 1, "size": 1}
    try:
        response = await client.get(f"{DAUM_SEARCH_URL}/image", headers=headers, params=params)
        response.raise_for_status()
        documents = response.json().get("documents", [])
        return documents[0] if documents else {}
    except httpx.HTTPStatusError as e:
        print(f"Error fetching image document for '{query}': {e}")
        return {}
        
# --- 시스템 프롬프트 ---
SYSTEM_PROMPT = """
Using the provided JSON results, compile a detailed and visually appealing Markdown summary for the user.

Each place **MUST** include:

## [{place_name}]({place_url})

![Image]({image_url})

- **Address**: {address_name}
- **Category**: {category_name}
- **Contact**: {phone}
- **Summary**: Briefly summarize the overall sentiment or notable points based on provided comments. Consider aspects such as positive features, negative issues, and unique highlights.

Note:
- The summary should be directly derived by analyzing and condensing the provided comments.
- Ensure all listed elements (title with link, image, address, category, contact, and summary) are always included for every place."""

# --- MCP 도구 콜백 함수 (스트리밍 로직 구현) ---
async def search_tool(options: SearchSchema, context):
    if not KAKAO_API_KEY:
        return {
            "isError": True,
            "content": [{
                "type": "text",
                "text": "Tool Execution Failed: The KAKAO_API_KEY environment variable is not configured."
            }]
        }

    if not options.query:
        return {"isError": True, "content": [{"type": "text", "text": "Query is empty"}]}

    # 클라이언트에 렌더링 방법을 안내하는 시스템 프롬프트를 먼저 스트리밍합니다.
    await context.stream({"type": "text", "text": SYSTEM_PROMPT})

    async with httpx.AsyncClient() as client:
        map_documents = await fetch_map_documents(client, options.query)

        for document in map_documents:
            place_name = document.get("place_name", "Unknown Place")
            try:
                # 댓글과 이미지 정보를 병렬로 비동기 호출합니다.
                comments_task = fetch_web_documents(client, place_name)
                image_task = fetch_image_document(client, place_name)
                
                comments_res, image_res = await asyncio.gather(comments_task, image_task)

                result = {
                    "place_name": place_name,
                    "address_name": document.get("address_name", ""),
                    "category_name": document.get("category_name", ""),
                    "place_url": document.get("place_url", ""),
                    "phone": document.get("phone", ""),
                    "image_url": image_res.get("image_url", ""),
                    "comments": [
                        {"title": c.get("title", ""), "contents": c.get("contents", "")}
                        for c in comments_res
                    ],
                }
                
                # 완성된 장소 정보를 클라이언트로 스트리밍합니다.
                await context.stream({
                    "type": "text",
                    "text": json.dumps(result, ensure_ascii=False, indent=2)
                })
            
            except Exception as e:
                print(f"Error processing document for '{place_name}': {e}")
                # 특정 항목 처리 중 오류 발생 시, 오류 정보를 스트리밍할 수 있습니다.
                await context.stream({
                    "type": "text",
                    "text": json.dumps({"error": f"Failed to process {place_name}"})
                })
    
    # 모든 스트리밍이 완료되었음을 알리기 위해 빈 콘텐츠를 반환합니다.
    return {"content": []}