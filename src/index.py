import asyncio
import json
import os
from typing import Any, List, Dict

import httpx
from pydantic import BaseModel, Field

# MCP SDK의 FastMCP 프레임워크를 import합니다.
from mcp.server.fastmcp import FastMCP, Context
from mcp.server.session import ServerSession

# --- Kakao API 설정 ---
KAKAO_API_KEY = os.getenv("KAKAO_API_KEY", None)
KAKAO_MAP_URL = "https://dapi.kakao.com/v2/local/search/keyword.json"
DAUM_SEARCH_URL = "https://dapi.kakao.com/v2/search"

if not KAKAO_API_KEY:
    print("env KAKAO_API_KEY가 존재 하지 않음")
    exit(1)

# --- Pydantic 스키마 정의 ---
# 도구의 입력 파라미터를 정의합니다. FastMCP가 이를 자동으로 입력 스키마로 변환합니다.
class SearchSchema(BaseModel):
    query: str = Field(
        description=(
            "Korean keywords for searching places in South Korea. Typically combines "
            "place type and location (e.g., '이태원 맛집', '서울 병원', '강남역 영화관')."
        )
    )

# --- 데이터 구조 정의 (Type Hinting용) ---
class WebDocument(BaseModel):
    title: str
    contents: str

class KakaoLocalSearchResult(BaseModel):
    place_name: str
    address_name: str
    category_name: str
    place_url: str
    phone: str
    image_url: str
    comments: List[WebDocument]

# --- 비동기 API 호출 함수 ---
# (이전 코드와 동일, 비동기 HTTP 요청을 처리합니다)
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

# --- MCP 서버 및 도구 정의 ---

# 1. FastMCP 서버 인스턴스를 생성합니다.
mcp = FastMCP("Kakao Map Search Server")

# 2. @mcp.tool() 데코레이터를 사용하여 도구를 등록합니다.
#    함수 인자에 타입 힌트를 사용하여 입력 스키마를 자동으로 생성합니다.
@mcp.tool()
async def search_tool(
    options: SearchSchema, 
    ctx: Context[ServerSession, None]
) -> str: # 최종 반환 타입은 간단한 성공 메시지입니다.
    """
    Recommends relevant places in South Korea based on user queries.
    Streams results back to the client.
    """
    if not KAKAO_API_KEY:
        await ctx.error("Tool Execution Failed: The KAKAO_API_KEY environment variable is not configured.")
        return "Error: KAKAO_API_KEY not set."

    if not options.query:
        await ctx.warning("Query is empty.")
        return "Warning: Query was empty."

    # 3. ctx.report_progress를 사용하여 시스템 프롬프트를 먼저 스트리밍합니다.
    await ctx.report_progress(message=SYSTEM_PROMPT)

    async with httpx.AsyncClient() as client:
        map_documents = await fetch_map_documents(client, options.query)

        for i, document in enumerate(map_documents):
            place_name = document.get("place_name", "Unknown Place")
            try:
                comments_task = fetch_web_documents(client, place_name)
                image_task = fetch_image_document(client, place_name)
                
                comments_res, image_res = await asyncio.gather(comments_task, image_task)

                result = KakaoLocalSearchResult(
                    place_name=place_name,
                    address_name=document.get("address_name", ""),
                    category_name=document.get("category_name", ""),
                    place_url=document.get("place_url", ""),
                    phone=document.get("phone", ""),
                    image_url=image_res.get("image_url", ""),
                    comments=[WebDocument(**c) for c in comments_res],
                )
                
                # 4. 개별 결과를 JSON으로 직렬화하여 progress 메시지로 스트리밍합니다.
                await ctx.report_progress(
                    progress=i + 1,
                    total=len(map_documents),
                    message=result.model_dump_json(indent=2)
                )
            
            except Exception as e:
                await ctx.error(f"Error processing '{place_name}': {e}")
                await ctx.report_progress(message=json.dumps({"error": f"Failed to process {place_name}"}))

    return "Search complete. All results have been streamed."

# --- 서버 실행 ---
if __name__ == "__main__":
    # 5. mcp.run()을 사용하여 streamable-http 전송 방식으로 서버를 실행합니다.
    mcp.run(transport="streamable-http")