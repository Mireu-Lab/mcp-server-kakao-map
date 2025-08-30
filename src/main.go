package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	// 실제 존재하는 공식 SDK 경로
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- 데이터 구조 정의 (Structs) ---

type SearchSchema struct {
	Query string `json:"query" jsonschema:"Korean keywords for searching places in South Korea."`
}

type MapDocument struct {
	PlaceName      string `json:"place_name"`
	AddressName    string `json:"address_name"`
	CategoryName   string `json:"category_name"`
	PlaceURL       string `json:"place_url"`
	Phone          string `json:"phone"`
}

type KakaoLocalSearchResponse struct {
	Documents []MapDocument `json:"documents"`
}

type WebDocument struct {
	Title    string `json:"title"`
	Contents string `json:"contents"`
}

type DaumWebSearchResponse struct {
	Documents []WebDocument `json:"documents"`
}

type ImageDocument struct {
	ImageURL string `json:"image_url"`
}

type DaumImageSearchResponse struct {
	Documents []ImageDocument `json:"documents"`
}

type KakaoLocalSearchResult struct {
	PlaceName    string        `json:"place_name"`
	AddressName  string        `json:"address_name"`
	CategoryName string        `json:"category_name"`
	PlaceURL     string        `json:"place_url"`
	Phone        string        `json:"phone"`
	ImageURL     string        `json:"image_url"`
	Comments     []WebDocument `json:"comments"`
}

// --- 전역 변수 및 상수 ---

var (
	kakaoAPIKey string
	httpClient  = &http.Client{Timeout: 10 * time.Second}
)

const (
	kakaoMapURL   = "https://dapi.kakao.com/v2/local/search/keyword.json"
	daumSearchURL = "https://dapi.kakao.com/v2/search"
	systemPrompt  = `
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
- Ensure all listed elements (title with link, image, address, category, contact, and summary) are always included for every place.`
)

// --- Kakao API 호출 헬퍼 함수 ---

func makeKakaoRequest(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "KakaoAK "+kakaoAPIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func fetchMapDocuments(ctx context.Context, query string) ([]MapDocument, error) {
	var response KakaoLocalSearchResponse
	url := fmt.Sprintf("%s?query=%s", kakaoMapURL, query)
	err := makeKakaoRequest(ctx, url, &response)
	return response.Documents, err
}

func fetchWebDocuments(ctx context.Context, query string) ([]WebDocument, error) {
	var response DaumWebSearchResponse
	url := fmt.Sprintf("%s/web?query=%s&page=1&size=5", daumSearchURL, query)
	err := makeKakaoRequest(ctx, url, &response)
	return response.Documents, err
}

func fetchImageDocument(ctx context.Context, query string) (ImageDocument, error) {
	var response DaumImageSearchResponse
	url := fmt.Sprintf("%s/image?query=%s&page=1&size=1", daumSearchURL, query)
	err := makeKakaoRequest(ctx, url, &response)
	if err != nil {
		return ImageDocument{}, err
	}
	if len(response.Documents) > 0 {
		return response.Documents[0], nil
	}
	return ImageDocument{}, nil
}

// --- MCP 도구 콜백 함수 ---

func searchTool(ctx context.Context, req *mcp.CallToolRequest, options SearchSchema) (*mcp.CallToolResult, any, error) {
	// 1. GetSession()이 반환하는 mcp.Session 인터페이스를 *mcp.ServerSession 타입으로 단언합니다.
	serverSession, ok := req.GetSession().(*mcp.ServerSession)
	if !ok {
		return nil, nil, fmt.Errorf("internal error: could not cast session to *mcp.ServerSession")
	}

	if kakaoAPIKey == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Tool Execution Failed: KAKAO_API_KEY is not configured."}},
		}, nil, nil
	}
	if options.Query == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Query is empty"}},
		}, nil, nil
	}

	// 2. 타입 단언으로 얻은 serverSession 변수를 사용하여 NotifyProgress를 호출합니다.
	_ = serverSession.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: req.Params.GetProgressToken(),
		Message:       systemPrompt,
	})

	mapDocuments, err := fetchMapDocuments(ctx, options.Query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch map documents: %w", err)
	}

	for _, doc := range mapDocuments {
		var wg sync.WaitGroup
		var webDocs []WebDocument
		var imgDoc ImageDocument
		var webErr, imgErr error

		wg.Add(2)
		go func(d MapDocument) {
			defer wg.Done()
			webDocs, webErr = fetchWebDocuments(ctx, d.PlaceName)
		}(doc)
		go func(d MapDocument) {
			defer wg.Done()
			imgDoc, imgErr = fetchImageDocument(ctx, d.PlaceName)
		}(doc)
		wg.Wait()

		if webErr != nil || imgErr != nil {
			log.Printf("Error fetching details for %s: webErr=%v, imgErr=%v", doc.PlaceName, webErr, imgErr)
			continue
		}

		result := KakaoLocalSearchResult{
			PlaceName:    doc.PlaceName,
			AddressName:  doc.AddressName,
			CategoryName: doc.CategoryName,
			PlaceURL:     doc.PlaceURL,
			Phone:        doc.Phone,
			Comments:     webDocs,
			ImageURL:     imgDoc.ImageURL,
		}

		resultBytes, err := json.Marshal(result)
		if err != nil {
			log.Printf("Failed to marshal result for %s: %v", doc.PlaceName, err)
			continue
		}

		// 3. 여기서도 serverSession 변수를 사용합니다.
		_ = serverSession.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			ProgressToken: req.Params.GetProgressToken(),
			// Value 필드는 MCP 프로토콜의 ProgressNotificationParams에 없으므로 Message 필드를 사용합니다.
			Message: string(resultBytes),
		})
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Search complete. All results have been streamed."}},
	}, nil, nil
}


// --- 서버 실행 ---

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, reading from environment")
	}

	kakaoAPIKey = os.Getenv("KAKAO_API_KEY")
	if kakaoAPIKey == "" {
		log.Fatal("FATAL: KAKAO_API_KEY environment variable is not set.")
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-server-kakao-map-go",
		Version: "0.0.1",
	}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "kakao_map_place_recommender",
		Description: "Recommends relevant places in South Korea based on user queries.",
	}, searchTool)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	port := "8080"
	log.Printf("MCP server with SSE is running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}