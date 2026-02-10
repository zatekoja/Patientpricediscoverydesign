FROM node:20-alpine

WORKDIR /app/Frontend

# Copy package files
COPY Frontend/package.json Frontend/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY Frontend/src ./src
COPY Frontend/public ./public
COPY Frontend/index.html vite.config.ts tsconfig.json ./

# Build arguments for environment variables
ARG VITE_GEOLOCATION_API_KEY=""
ENV VITE_GEOLOCATION_API_KEY=$VITE_GEOLOCATION_API_KEY

# Expose Vite dev server port
EXPOSE 5173

# Start Vite dev server with HMR configured for Docker
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
