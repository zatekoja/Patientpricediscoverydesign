# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY ../package.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY .. .

# Build the application
WORKDIR /app/Frontend/src
RUN npx vite build --config vite.config.ts

# Serve stage
FROM nginx:alpine

# Copy built assets
COPY --from=builder /app/Frontend/src/dist /usr/share/nginx/html

# Copy custom nginx config
COPY ../nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
