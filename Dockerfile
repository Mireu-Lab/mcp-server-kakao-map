FROM node:22.12-alpine AS builder

WORKDIR /app
COPY package.json ./
RUN npm install


# Copy the rest of the application code
COPY src ./src
COPY tsconfig.json ./

# Build the application
RUN npm run build


FROM node:22-alpine AS release

WORKDIR /app

COPY --from=builder /app/dist /app/dist
COPY --from=builder /app/package.json /app/package.json



ENTRYPOINT ["node", "/app/dist/index.js"]