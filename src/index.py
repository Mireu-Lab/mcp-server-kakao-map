# MCP SDK 서버 관련 모듈 임포트 (실제 SDK 패키지명에 따라 달라질 수 있음)
from mcp.server import Server
from mcp.server.fastmcp import FastMCP
from search import *

server = FastMCP("mcp-server-kakao-map")

# --- 서버 실행 메인 함수 ---
async def main():
    # server.tool(
    #     name="kakao_map_place_recommender",
    #     description="Recommends relevant places in South Korea based on user queries.",
    #     schema=SearchSchema,
    #     callback=search_tool
    # )
    
    # SSE 스트리밍을 위해 response_mode='stream'으로 설정합니다.
    # Python 웹 서버는 보통 8000 포트를 사용합니다.
    # transport = HttpServerTransport(port=8000, response_mode='stream')
    
    print("MCP server with SSE is running on http://localhost:8000")
    server.run('sse')

if __name__ == "__main__":
    asyncio.run(main())