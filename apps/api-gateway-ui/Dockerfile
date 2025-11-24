FROM node:18-alpine AS builder
WORKDIR /app
ENV NODE_ENV=production
COPY package.json package-lock.json* ./
# install dev deps to allow building
RUN npm ci --silent || npm install
COPY . .
RUN mkdir -p public && npm run build

FROM node:18-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
# copy only what we need to run the app
COPY --from=builder /app/package.json ./
COPY --from=builder /app/package-lock.json ./
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public

# install only production deps in the final image for smaller size
RUN npm ci --production --silent || npm install --production --silent

EXPOSE 3000
CMD ["npm", "run", "start"]
