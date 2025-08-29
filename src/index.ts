#!/usr/bin/env node

import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { search, SearchSchema } from './search.js';
import dotenv from 'dotenv';

dotenv.config();

const server = new McpServer({
  name: 'mcp-server-kakao-map',
  version: '0.0.1',
});

server.tool(
  'kakao_map_place_recommender',
  'Recommends relevant places in South Korea, such as restaurants, cafes, parks, hospitals, banks, shops, or tourist attractions, based on user queries seeking suggestions.',
  SearchSchema,
  search
);

const transport = new StdioServerTransport();
await server.connect(transport);
console.log('MCP server is running with Kakao Map Place Recommender tool...');
