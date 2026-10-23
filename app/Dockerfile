FROM node:22-alpine AS build

WORKDIR /app
ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=$VITE_API_URL

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM node:22-alpine

WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=build /app/build ./build

EXPOSE 3000
CMD ["node", "build"]
