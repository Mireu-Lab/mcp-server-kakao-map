import { ToolCallback } from '@modelcontextprotocol/sdk/server/mcp.js';
import { z } from 'zod';
import got from 'got';

const KAKAO_MAP_URL = 'https://dapi.kakao.com/v2/local/search/keyword.json';

type KakaoLocalSearchResponse = {
  documents: {
    address_name: string;
    category_group_code: string;
    category_group_name: string;
    category_name: string;
    distance: string;
    id: string;
    phone: string;
    place_name: string;
    place_url: string;
    road_address_name: string;
  }[];
};

type KakaoLocalSearchResult = Pick<
  KakaoLocalSearchResponse['documents'][number],
  'place_name' | 'address_name' | 'category_name' | 'place_url' | 'phone'
>;

export const SearchSchema = {
  query: z
    .string()
    .describe(
      "Korean keywords for searching places in South Korea. Typically combines place type and location (e.g., '이태원 맛집', '서울 병원', '강남역 영화관')."
    ),
};

export const search: ToolCallback<typeof SearchSchema> = async (options) => {
  const KAKAO_API_KEY = process.env.KAKAO_API_KEY;

  if (!KAKAO_API_KEY) {
    return {
      isError: true,
      content: [
        {
          type: 'text',
          text: `Tool Execution Failed: The KAKAO_API_KEY environment variable is not configured for this tool. Inform the user (administrator/developer) that they need to set up the Kakao REST API key. Setup guide: https://developers.kakao.com/docs/latest/ko/local/common`,
        },
      ],
    };
  }

  if (options.query.length === 0) {
    return {
      isError: true,
      content: [{ type: 'text', text: 'Query is empty' }],
    };
  }

  const res = await got.get<KakaoLocalSearchResponse>(KAKAO_MAP_URL, {
    headers: {
      Authorization: `KakaoAK ${KAKAO_API_KEY}`,
    },
    searchParams: {
      query: options.query,
    },
    responseType: 'json',
  });

  const { documents } = res.body;

  const results: KakaoLocalSearchResult[] = documents.map((document) => ({
    place_name: document.place_name,
    address_name: document.address_name,
    category_name: document.category_name,
    place_url: document.place_url,
    phone: document.phone,
  }));

  return {
    content: [
      {
        type: 'text',
        text: `
        Please provide all details including place_name, address_name, category_name, place_url, and phone from the JSON results to the user.
        
        Recommended places JSON FORMAT:\n${JSON.stringify(results, null, 2)}`.trim(),
      },
    ],
  };
};
