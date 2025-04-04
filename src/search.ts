import { ToolCallback } from '@modelcontextprotocol/sdk/server/mcp.js';
import { z } from 'zod';
import got from 'got';

const KAKAO_MAP_URL = 'https://dapi.kakao.com/v2/local/search/keyword.json';

const DAUM_SEARCH_URL = 'https://dapi.kakao.com/v2/search';

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

type DaumWebSearchResponse = {
  documents: {
    title: string;
    url: string;
    contents: string;
  }[];
};

type DaumImageSearchResponse = {
  documents: {
    thumbnail_url: string;
    image_url: string;
  }[];
};

type KakaoLocalSearchResult = Pick<
  KakaoLocalSearchResponse['documents'][number],
  'place_name' | 'address_name' | 'category_name' | 'place_url' | 'phone'
> & {
  comments: {
    title: string;
    contents: string;
  }[];
  image_url: string;
};

export const SearchSchema = {
  query: z
    .string()
    .describe(
      "Korean keywords for searching places in South Korea. Typically combines place type and location (e.g., '이태원 맛집', '서울 병원', '강남역 영화관')."
    ),
};

function fetchMapDocuments(query: string, apiKey: string) {
  return got
    .get<KakaoLocalSearchResponse>(KAKAO_MAP_URL, {
      headers: {
        Authorization: `KakaoAK ${apiKey}`,
      },
      searchParams: {
        query,
      },
      responseType: 'json',
    })
    .then((res) => res.body.documents);
}

function fetchWebDocuments(query: string, apiKey: string) {
  return got
    .get<DaumWebSearchResponse>(`${DAUM_SEARCH_URL}/web`, {
      headers: {
        Authorization: `KakaoAK ${apiKey}`,
      },
      searchParams: {
        query,
        page: 1,
        size: 3,
      },
      responseType: 'json',
    })
    .then((res) => res.body.documents);
}

function fetchImageDocument(query: string, apiKey: string) {
  return got
    .get<DaumImageSearchResponse>(`${DAUM_SEARCH_URL}/image`, {
      headers: {
        Authorization: `KakaoAK ${apiKey}`,
      },
      searchParams: {
        query,
        page: 1,
        size: 1,
      },
      responseType: 'json',
    })
    .then((res) => res.body.documents[0]);
}

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

  const mapDocuments = await fetchMapDocuments(options.query, KAKAO_API_KEY);

  const results = await Promise.allSettled(
    mapDocuments.map(async (document) => {
      const comments = await fetchWebDocuments(document.place_name, KAKAO_API_KEY).catch((err) => {
        console.error(err);
        return [];
      });
      const image = await fetchImageDocument(document.place_name, KAKAO_API_KEY).catch((err) => {
        console.error(err);
        return {
          image_url: '',
        };
      });
      return {
        place_name: document.place_name,
        address_name: document.address_name,
        category_name: document.category_name,
        place_url: document.place_url,
        phone: document.phone,
        image_url: image.image_url,
        comments: comments.map((comment) => ({
          title: comment.title,
          contents: comment.contents,
        })),
      } as KakaoLocalSearchResult;
    })
  ).then((res) => res.map((v) => (v.status == 'fulfilled' ? v.value : null)).filter(Boolean));

  return {
    content: [
      {
        type: 'text',
        text: `Using the provided JSON results, compile a detailed and visually appealing Markdown summary for the user. Each place should include:

- **Place Name**: Create a clickable Markdown link that opens \`place_url\`.

- **Address**: Clearly display the full address.

- **Category**: Mention the category clearly.

- **Contact**: Include the phone number if available; if not, indicate "Not available".

- **Image**: Display the image using the provided \`image_url\`.

- **Comments Summary**: Review the provided comments (\`title\` and \`contents\`) and summarize the overall sentiment or key points briefly (positive, negative, notable features, etc.).

Ensure the Markdown formatting is clean and easy to navigate, enhancing readability and user experience.`,
      },
      {
        type: 'text',
        text: JSON.stringify(results, null, 2),
      },
    ],
  };
};
