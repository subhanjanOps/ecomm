import React, { useEffect, useMemo, useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper'
import MenuItem from '@mui/material/MenuItem'
import Divider from '@mui/material/Divider'
import Alert from '@mui/material/Alert'
import Stack from '@mui/material/Stack'
import axios from 'axios'

function parseHeaders(text) {
  const headers = {}
  if (!text) return headers
  text.split('\n').forEach((line) => {
    const idx = line.indexOf(':')
    if (idx > 0) {
      const k = line.slice(0, idx).trim()
      const v = line.slice(idx + 1).trim()
      if (k) headers[k] = v
    }
  })
  return headers
}

export default function ApiExplorer({ service }) {
  const isGrpc = (service.protocol || 'http') === 'grpc-json'

  const endpoints = useMemo(() => {
    if (isGrpc) return []
    const paths = service.swagger_json?.paths || {}
    const list = []
    Object.entries(paths).forEach(([p, methods]) => {
      Object.keys(methods || {}).forEach((m) => {
        list.push({ method: m.toUpperCase(), path: p })
      })
    })
    // Sort by path then method
    return list.sort((a, b) => (a.path === b.path ? a.method.localeCompare(b.method) : a.path.localeCompare(b.path)))
  }, [service, isGrpc])

  const httpMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']
  const [method, setMethod] = useState(isGrpc ? 'POST' : (endpoints[0]?.method || 'GET'))
  const [path, setPath] = useState(isGrpc ? '' : (endpoints[0]?.path || '/'))
  const [grpcRoutes, setGrpcRoutes] = useState([])
  const [headersText, setHeadersText] = useState('')
  const [bodyText, setBodyText] = useState(isGrpc ? '{}' : '')
  const [loading, setLoading] = useState(false)
  const [resp, setResp] = useState(null)
  const [error, setError] = useState(null)
  const [bearer, setBearer] = useState('')

  useEffect(() => {
    if (!isGrpc || !service?.id) return
    let mounted = true
    axios.get(`/admin/services/${service.id}/routes`).then((res) => {
      if (!mounted) return
      const list = (res.data || []).map(r => ({ method: (r.method || '').toUpperCase(), path: r.path, grpc_method: r.grpc_method }))
      setGrpcRoutes(list)
      if (list.length && !path) {
        setMethod(list[0].method || 'POST')
        setPath(list[0].path || '')
      }
    }).catch(() => {})
    return () => { mounted = false }
  }, [isGrpc, service?.id])

  // Extract params from a templated path like /users/{id}/notes/{noteId}
  const paramNames = useMemo(() => {
    const names = []
    const regex = /\{([^}]+)\}/g
    let m
    let target = ''
    if (isGrpc) {
      target = path || ''
    }
    while ((m = regex.exec(target)) !== null) {
      if (m[1]) names.push(m[1])
    }
    return names
  }, [isGrpc, path])
  const [paramValues, setParamValues] = useState({})
  useEffect(() => { setParamValues({}) }, [path])

  const buildActualPath = () => {
    if (!isGrpc || paramNames.length === 0) return path
    let p = path
    paramNames.forEach((name) => {
      const val = paramValues[name] || ''
      p = p.replace(new RegExp(`\\{${name}\\}`, 'g'), encodeURIComponent(val))
    })
    return p
  }

  const send = async () => {
    setLoading(true)
    setError(null)
    setResp(null)
    try {
      const actualPath = buildActualPath()
      const url = (service.public_prefix || service.publicPrefix || '/') + (actualPath.startsWith('/') ? actualPath.slice(1) : actualPath)
      const headers = parseHeaders(headersText)
      if (bearer) headers['Authorization'] = `Bearer ${bearer}`
      let data
      try {
        // Try parse JSON body, fall back to raw text
        data = bodyText ? JSON.parse(bodyText) : undefined
      } catch (e) {
        data = bodyText
      }
      const start = Date.now()
      const res = await axios({ url, method, headers, data })
      const duration = Date.now() - start
      setResp({
        status: res.status,
        statusText: res.statusText,
        duration,
        headers: res.headers,
        data: res.data,
      })
    } catch (e) {
      const r = e?.response
      setError({
        message: e?.message || 'Request failed',
        status: r?.status,
        data: r?.data,
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 2 }}>API Explorer</Typography>

      {isGrpc ? (
        <Alert severity="info" sx={{ mb: 2 }}>
          You are making REST requests to the API Gateway. For services using gRPC (protocol = grpc-json),
          the gateway transcodes your HTTP request into a gRPC call. Provide the full gRPC method as
          <code>package.Service/Method</code> (for example: <code>my.user.v1.UserService/ListUsers</code>), and a JSON body for the RPC input.
          The gateway handles the gRPC call and returns JSON back to you.
        </Alert>
      ) : (
        <Alert severity="info" sx={{ mb: 2 }}>
          Requests are sent to the API Gateway using the service's public prefix. Select an endpoint from the spec
          or type your own method and path, then send the request. The gateway will proxy it to the upstream service.
        </Alert>
      )}

      <Paper sx={{ p: 2, mb: 2 }}>
        {!isGrpc && endpoints.length > 0 && (
          <TextField
            label="Endpoints from Swagger"
            select
            fullWidth
            value={`${method} ${path}`}
            onChange={(e) => {
              const value = e.target.value
              const sp = value.indexOf(' ')
              setMethod(value.slice(0, sp))
              setPath(value.slice(sp + 1))
            }}
            helperText="Pick an endpoint from the service's OpenAPI/Swagger spec"
            sx={{ mb: 2 }}
          >
            {endpoints.map((ep, i) => (
              <MenuItem key={`${ep.method} ${ep.path} ${i}`} value={`${ep.method} ${ep.path}`}>
                {ep.method} {ep.path}
              </MenuItem>
            ))}
          </TextField>
        )}

        {isGrpc && grpcRoutes.length > 0 && (
          <TextField
            label="Mapped Endpoints (REST → gRPC)"
            select
            fullWidth
            value={`${method} ${path}`}
            onChange={(e) => {
              const value = e.target.value
              const sp = value.indexOf(' ')
              setMethod(value.slice(0, sp))
              setPath(value.slice(sp + 1))
            }}
            helperText="Pick a route mapping configured for this service"
            sx={{ mb: 2 }}
          >
            {grpcRoutes.map((ep, i) => (
              <MenuItem key={`${ep.method} ${ep.path} ${i}`} value={`${ep.method} ${ep.path}`}>
                {ep.method} {ep.path} → {ep.grpc_method}
              </MenuItem>
            ))}
          </TextField>
        )}

        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
          <TextField
            label="HTTP Method"
            select
            value={method}
            onChange={(e) => setMethod(e.target.value)}
            sx={{ width: 160 }}
            disabled={isGrpc}
            helperText={isGrpc ? 'gRPC transcoding uses POST' : ''}
          >
            {httpMethods.map((m) => (
              <MenuItem key={m} value={m}>{m}</MenuItem>
            ))}
          </TextField>

          <TextField
            label={isGrpc ? 'Gateway Path (package.Service/Method)' : 'Path'}
            placeholder={isGrpc ? 'my.user.v1.UserService/ListUsers' : '/api/v1/resource'}
            value={path}
            onChange={(e) => setPath(e.target.value)}
            fullWidth
          />
        </Box>

        {isGrpc && paramNames.length > 0 && (
          <Box sx={{ display: 'grid', gap: 2, mt: 2 }}>
            <Typography variant="subtitle2">Route parameters</Typography>
            <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
              {paramNames.map((n) => (
                <TextField key={n} label={n} value={paramValues[n] || ''} onChange={(e) => setParamValues(v => ({ ...v, [n]: e.target.value }))} />
              ))}
            </Box>
          </Box>
        )}

        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mt: 2 }}>
          <TextField label="Bearer token (optional)" placeholder="eyJhbGciOiJI..." value={bearer} onChange={(e) => setBearer(e.target.value)} fullWidth />
          <Button variant="outlined" onClick={() => setBearer('')}>Clear</Button>
        </Stack>

        <TextField
          label="Headers (Key: Value per line)"
          multiline
          minRows={3}
          value={headersText}
          onChange={(e) => setHeadersText(e.target.value)}
          fullWidth
          sx={{ mt: 2 }}
        />

        <TextField
          label={isGrpc ? 'JSON Body (RPC input)' : 'Body (JSON or text, optional)'}
          multiline
          minRows={6}
          value={bodyText}
          onChange={(e) => setBodyText(e.target.value)}
          fullWidth
          sx={{ mt: 2 }}
        />

        <Box sx={{ mt: 2 }}>
          <Button variant="contained" onClick={send} disabled={loading || !path}>
            {loading ? 'Sending...' : 'Send'}
          </Button>
        </Box>
      </Paper>

      {error && (
        <Paper sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle1" color="error">Error</Typography>
          <Typography variant="body2">{error.message}</Typography>
          {error.status ? <Typography variant="body2">Status: {error.status}</Typography> : null}
          {error.data ? (
            <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{
              typeof error.data === 'string' ? error.data : JSON.stringify(error.data, null, 2)
            }</pre>
          ) : null}
        </Paper>
      )}

      {resp && (
        <Paper sx={{ p: 2 }}>
          <Typography variant="subtitle1">Response</Typography>
          <Typography variant="body2">Status: {resp.status} {resp.statusText} • {resp.duration} ms</Typography>
          <Divider sx={{ my: 1 }} />
          <Typography variant="subtitle2">Headers</Typography>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{JSON.stringify(resp.headers || {}, null, 2)}</pre>
          <Typography variant="subtitle2">Body</Typography>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{
            typeof resp.data === 'string' ? resp.data : JSON.stringify(resp.data, null, 2)
          }</pre>
        </Paper>
      )}
    </Box>
  )
}
