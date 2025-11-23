API Gateway UI
================

This is a minimal Next.js UI for the API Gateway. It uses Material UI and queries the gateway's
Admin API at `/admin/services` (dev server proxies to `http://localhost:8080`).

Run locally (requires Node):

```pwsh
cd apps/api-gateway-ui
npm install
npm run dev
```

Open http://localhost:3000

Production build:

```pwsh
npm run build
npm start
```

Docker (build and run):

```pwsh
docker build -t api-gateway-ui:local .
docker run -p 3000:3000 --env NEXT_PUBLIC_API_BASE=http://host.docker.internal:8080 api-gateway-ui:local
```

Notes
- The dev server proxies `/admin` to `http://localhost:8080` configured in `next.config.js` environment.
- For SSR and Material UI, Emotion cache and `_document.js` are wired for server-side CSS extraction.
