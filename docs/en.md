Kakao Map MCP Server

MCP Server for the Kakao Map API, enabling location-based place recommendations within South Korea. Optimized for Korean language queries.

## Tools

1.  kakao_map_place_recommender
    * Description: Recommends various relevant places (e.g., restaurants, shops, public facilities, attractions) in South Korea based on user queries seeking suggestions. Uses the Kakao Map API keyword search.
    * Inputs:
        * query (string): Korean keywords describing the type of place and location. Examples: '이태원 맛집', '서울 병원', '강남역 근처 카페'.
    * Returns: A JSON string containing a list of recommended places, each with details like name, address, category, URL, and phone number.

## Setup

### Kakao REST API Key

To use this tool, you need the REST API Key for your Kakao application.

1.  Register Application: Go to Kakao Developers (https://developers.kakao.com/), log in, and create an application (https://developers.kakao.com/docs/latest/ko/getting-started/quick-start#create) if you don't already have one.
2.  Get REST API Key: Navigate to your application's settings: `[My Applications] > [App Settings] > [Summary]`. Find and copy the REST API Key from the list of keys provided. This specific key is required for the tool.
3.  Enable Kakao Map API: Ensure the Kakao Map API is enabled for your application. Go to `[My Applications] > [Kakao Map] > [Activation Settings]` and set the `[Status]` to `ON`. (*Note: If adding the API to an existing app, additional permission requests might be necessary.*)
4.  Reference: For more details, consult the Kakao Local API Common Guide (https://developers.kakao.com/docs/latest/ko/local/common).

### Usage

You need to provide your Kakao REST API key as an environment variable named KAKAO_API_KEY when running the server.

### Client Usage (e.g., Claude)

#### NPX

Example client configuration (e.g., claude_desktop_config.json)
{
  "mcpServers": {
    "kakao_map": {
      "command": "npx",
      "args": [
        "-y",
        "@smithery/cli@latest",
        "run",
        "@cgoinglove/mcp-server-kakao-map",
        "--config",
        "\"{\\\"KAKAO_API_KEY\\\":\\\"YOUR_API_KEY\\\"}\""
      ]
    }
  }
}