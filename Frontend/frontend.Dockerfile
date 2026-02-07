# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY ../package.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY .. .

# Pass through Vite build-time env vars
ARG VITE_GOOGLE_MAPS_API_KEY
ENV VITE_GOOGLE_MAPS_API_KEY=${VITE_GOOGLE_MAPS_API_KEY}

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
