Kakao Map MCP Server

카카오맵 API를 활용하여 대한민국 내 위치 기반 장소 추천을 제공하는 MCP 서버입니다. 한국어 쿼리에 최적화되어 있습니다.

## Tool: kakao_map_place_recommender

설명: 사용자 질의에 따라 대한민국 내 다양한 관련 장소(예: 식당, 상점, 공공시설, 관광명소)를 추천합니다. 카카오맵 API 키워드 검색을 사용합니다.

- `query` (필수): 장소 유형 및 위치를 설명하는 한국어 키워드. 예시: '이태원 맛집', '서울 병원', '강남역 근처 카페'.

## Configuration

### 환경 변수

- `KAKAO_API_KEY`: 카카오 API 키 (필수)

1.  **애플리케이션 등록**: [카카오 디벨로퍼스](https://developers.kakao.com/)에 로그인하고, 아직 애플리케이션이 없다면 [새로 만듭니다](https://developers.kakao.com/docs/latest/ko/getting-started/quick-start#create).
2.  **REST API 키 확인**: 애플리케이션 설정(`[내 애플리케이션] > [앱 설정] > [요약 정보]`)으로 이동합니다. 제공된 여러 키 중에서 **REST API 키**를 찾아 복사합니다. 이 도구에는 이 특정 키가 필요합니다.
3.  **카카오맵 API 활성화**: 애플리케이션에 카카오맵 API가 활성화되어 있는지 확인합니다. `[내 애플리케이션] > [카카오맵] > [활성화 설정]`으로 이동하여 `[상태]`를 `ON`으로 설정합니다. (*참고: 기존 앱에 API를 추가하는 경우, 추가적인 권한 신청 및 승인이 필요할 수 있습니다.*)
4.  **참고**: 자세한 내용은 공식 문서를 참조하세요: [카카오 로컬 API 공통 가이드](https://developers.kakao.com/docs/latest/ko/local/common).
