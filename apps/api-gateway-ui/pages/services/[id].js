import React, { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Divider from '@mui/material/Divider'
import Alert from '@mui/material/Alert'
import Stack from '@mui/material/Stack'
import axios from 'axios'
import ServiceForm from '../../src/components/ServiceForm'
import Link from 'next/link'

export default function ServiceDetail() {
  const router = useRouter()
  const { id } = router.query
  const [svc, setSvc] = useState(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [showSwagger, setShowSwagger] = useState(false)

  useEffect(() => {
    if (!id) return
    let mounted = true
    axios.get(`/admin/services/${id}`).then((res) => { if (mounted) setSvc(res.data) }).catch(() => {}).finally(() => mounted && setLoading(false))
    return () => (mounted = false)
  }, [id])

  const onUpdate = async (payload) => {
    setSubmitting(true)
    try {
      await axios.put(`/admin/services/${id}`, payload)
      const res = await axios.get(`/admin/services/${id}`)
      setSvc(res.data)
      alert('Updated')
    } catch (e) {
      console.error(e)
      alert('Update failed')
    } finally { setSubmitting(false) }
  }

  const onDelete = async () => {
    if (!confirm('Delete this service?')) return
    try {
      await axios.delete(`/admin/services/${id}`)
      router.push('/')
    } catch (e) { console.error(e); alert('Delete failed') }
  }

  const onRefresh = async () => {
    try {
      await axios.post(`/admin/services/${id}/refresh`)
      const res = await axios.get(`/admin/services/${id}`)
      setSvc(res.data)
      alert('Refreshed')
    } catch (e) { console.error(e); alert('Refresh failed') }
  }

  if (loading) return <Container sx={{ mt: 4 }}>Loading...</Container>
  if (!svc) return <Container sx={{ mt: 4 }}>Not found</Container>

  return (
    <Container sx={{ mt: 4 }}>
      <Typography variant="h5" sx={{ mb: 2 }}>Service: {svc.name || svc.id}</Typography>

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <Button variant="outlined" onClick={onRefresh} disabled={svc.protocol === 'grpc-json'}>Refresh Swagger</Button>
        <Button color="error" variant="outlined" onClick={onDelete}>Delete</Button>
        <Button variant="contained" onClick={() => setShowSwagger(s => !s)} disabled={svc.protocol === 'grpc-json'}>
          {showSwagger ? 'Hide' : 'Show'} Swagger JSON
        </Button>
        <Link href={`/services/${id}/explore`} passHref legacyBehavior>
          <Button component="a" variant="outlined">Explore API</Button>
        </Link>
        <Link href={`/services/${id}/routes`} passHref legacyBehavior>
          <Button component="a" variant="outlined">Manage Routes</Button>
        </Link>
      </Stack>

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Typography variant="subtitle2">Status</Typography>
          <Box sx={{ mt: 1 }}>
            {svc.last_status ? <Alert severity={svc.last_status === 'ok' ? 'success' : 'warning'}>{svc.last_status}</Alert> : <Alert severity="info">No health checks yet</Alert>}
          </Box>
          <Divider sx={{ my: 2 }} />
          <Typography variant="body2">Last health: {svc.last_health_at || 'n/a'}</Typography>
          <Typography variant="body2">Last refreshed: {svc.last_refreshed || 'n/a'}</Typography>
        </CardContent>
      </Card>

      {showSwagger && svc.protocol !== 'grpc-json' && (
        <Card sx={{ mb: 2 }}>
          <CardContent>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>Swagger JSON</Typography>
            {svc.swagger_json ? (
              <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{typeof svc.swagger_json === 'string' ? svc.swagger_json : JSON.stringify(svc.swagger_json, null, 2)}</pre>
            ) : (
              <Typography variant="body2">No swagger JSON available</Typography>
            )}
          </CardContent>
        </Card>
      )}

      <ServiceForm initial={{
        name: svc.name,
        description: svc.description,
        public_prefix: svc.public_prefix,
        base_url: svc.base_url,
        swagger_url: svc.swagger_url,
        protocol: svc.protocol || 'http',
        grpc_target: svc.grpc_target,
        enabled: svc.enabled,
      }} onSubmit={onUpdate} submitting={submitting} />
    </Container>
  )
}

export async function getServerSideProps() {
  // Avoid static prerendering for dynamic route that relies on client router
  return { props: {} }
}
