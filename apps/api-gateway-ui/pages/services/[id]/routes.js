import React, { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import axios from 'axios'
import RouteForm from '../../../src/components/RouteForm'
import MenuItem from '@mui/material/MenuItem'
import TextField from '@mui/material/TextField'
import Alert from '@mui/material/Alert'

export default function ServiceRoutes() {
  const router = useRouter()
  const { id } = router.query
  const [svc, setSvc] = useState(null)
  const [routes, setRoutes] = useState([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [discovering, setDiscovering] = useState(false)
  const [discovered, setDiscovered] = useState([])
  const [drafts, setDrafts] = useState({}) // keyed by grpc_method -> {method, path}

  const load = async () => {
    setLoading(true)
    try {
      const [s, r] = await Promise.all([
        axios.get(`/admin/services/${id}`),
        axios.get(`/admin/services/${id}/routes`),
      ])
      setSvc(s.data)
      setRoutes(r.data || [])
    } catch (e) { console.error(e) }
    setLoading(false)
  }

  useEffect(() => { if (id) load() }, [id])

  const genDefaultPath = (svcName, methodName) => {
    // svcName like my.user.v1.UserService; prefer last segment without package
    const last = (svcName || '').split('.').pop() || 'Service'
    // methodName like ListUsers -> list-users
    const snake = methodName.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase()
    return `/${snake}`
  }

  const discover = async () => {
    setDiscovering(true)
    try {
      const res = await axios.get(`/admin/services/${id}/routes/discover`)
      const arr = res.data || []
      setDiscovered(arr)
      const init = {}
      arr.forEach((r) => {
        const defPath = genDefaultPath(r.service, r.method)
        init[r.grpc_method] = { method: 'GET', path: defPath }
      })
      setDrafts(init)
    } catch (e) { console.error(e); alert('Discovery failed') }
    setDiscovering(false)
  }

  const setDraft = (grpcMethod, field, value) => {
    setDrafts((d) => ({ ...d, [grpcMethod]: { ...(d[grpcMethod] || {}), [field]: value } }))
  }

  const addFromDiscovery = async (grpcMethod) => {
    const d = drafts[grpcMethod] || { method: 'GET', path: '/' }
    setSubmitting(true)
    try {
      await axios.post(`/admin/services/${id}/routes`, { method: d.method, path: d.path, grpc_method: grpcMethod })
      await load()
      // optionally remove from discovered list
      setDiscovered((arr) => arr.filter((x) => x.grpc_method !== grpcMethod))
    } catch (e) { console.error(e); alert('Create from discovery failed') }
    setSubmitting(false)
  }

  const addAllDiscovered = async () => {
    if (!confirm('Add routes for all discovered methods using default strategy?')) return
    setSubmitting(true)
    try {
      await axios.post(`/admin/services/${id}/routes/discover/bulk`)
      setDiscovered([])
      await load()
    } catch (e) { console.error(e); alert('Bulk add failed') }
    setSubmitting(false)
  }

  const create = async (payload) => {
    setSubmitting(true)
    try {
      await axios.post(`/admin/services/${id}/routes`, payload)
      setCreating(false)
      await load()
    } catch (e) { console.error(e); alert('Create failed') }
    setSubmitting(false)
  }

  const update = async (rt) => {
    setSubmitting(true)
    try {
      await axios.put(`/admin/services/${id}/routes/${rt.id}`, { method: rt.method, path: rt.path, grpc_method: rt.grpc_method })
      setEditing(null)
      await load()
    } catch (e) { console.error(e); alert('Update failed') }
    setSubmitting(false)
  }

  const del = async (rid) => {
    if (!confirm('Delete this route?')) return
    try { await axios.delete(`/admin/services/${id}/routes/${rid}`); await load() } catch (e) { console.error(e); alert('Delete failed') }
  }

  if (loading) return <Container sx={{ mt: 4 }}>Loading...</Container>
  if (!svc) return <Container sx={{ mt: 4 }}>Not found</Container>

  return (
    <Container sx={{ mt: 4, mb: 6 }}>
      <Stack direction={{ xs: 'column', md: 'row' }} alignItems={{ xs: 'stretch', md: 'center' }} justifyContent="space-between" sx={{ mb: 2 }} spacing={2}>
        <Typography variant="h5">Routes: {svc.name || svc.id}</Typography>
        <Stack direction="row" spacing={1}>
          <Button variant={creating ? 'outlined' : 'contained'} onClick={() => setCreating(c => !c)}>{creating ? 'Cancel' : 'New Route'}</Button>
          <Button variant="outlined" onClick={discover} disabled={discovering}>Discover gRPC Methods</Button>
          <Button variant="outlined" onClick={addAllDiscovered} disabled={submitting || discovering}>Add All (default)</Button>
        </Stack>
      </Stack>

      {creating && (
        <Paper sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle1" sx={{ mb: 1 }}>Create Route</Typography>
          <RouteForm onSubmit={create} submitting={submitting} />
        </Paper>
      )}

      {discovered.length > 0 && (
        <Paper sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle1" sx={{ mb: 1 }}>Discovered gRPC Methods</Typography>
          <Alert severity="info" sx={{ mb: 2 }}>Pick HTTP method and REST path, then click Add to create a mapping.</Alert>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>gRPC Service</TableCell>
                  <TableCell>Method</TableCell>
                  <TableCell>gRPC Full Method</TableCell>
                  <TableCell>HTTP Method</TableCell>
                  <TableCell>REST Path</TableCell>
                  <TableCell align="right">Action</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {discovered.map((drow) => {
                  const key = drow.grpc_method
                  const draft = drafts[key] || { method: 'GET', path: genDefaultPath(drow.service, drow.method) }
                  return (
                    <TableRow key={key}>
                      <TableCell>{drow.service}</TableCell>
                      <TableCell>{drow.method}</TableCell>
                      <TableCell>{drow.grpc_method}</TableCell>
                      <TableCell sx={{ minWidth: 140 }}>
                        <TextField select size="small" value={draft.method} onChange={(e) => setDraft(key, 'method', e.target.value)}>
                          {['GET','POST','PUT','PATCH','DELETE'].map(m => (<MenuItem key={m} value={m}>{m}</MenuItem>))}
                        </TextField>
                      </TableCell>
                      <TableCell sx={{ minWidth: 260 }}>
                        <TextField size="small" fullWidth value={draft.path} onChange={(e) => setDraft(key, 'path', e.target.value)} />
                      </TableCell>
                      <TableCell align="right">
                        <Button size="small" variant="contained" onClick={() => addFromDiscovery(key)} disabled={submitting}>Add</Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}

      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Method</TableCell>
              <TableCell>Path</TableCell>
              <TableCell>gRPC Method</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {routes.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{editing?.id === r.id ? (
                  <RouteForm initial={r} onSubmit={(payload) => update({ ...r, ...payload })} submitting={submitting} />
                ) : r.method}
                </TableCell>
                <TableCell>{editing?.id === r.id ? null : r.path}</TableCell>
                <TableCell>{editing?.id === r.id ? null : r.grpc_method}</TableCell>
                <TableCell align="right">
                  {editing?.id === r.id ? null : (
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button size="small" onClick={() => setEditing(r)}>Edit</Button>
                      <Button size="small" color="error" onClick={() => del(r.id)}>Delete</Button>
                    </Stack>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Container>
  )
}

export async function getServerSideProps() { return { props: {} } }
