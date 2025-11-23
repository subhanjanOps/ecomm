import React, { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import axios from 'axios'
import ServiceForm from '../../src/components/ServiceForm'

export default function ServiceDetail() {
  const router = useRouter()
  const { id } = router.query
  const [svc, setSvc] = useState(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

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
      <Box sx={{ mb: 2 }}>
        <Button onClick={onRefresh} sx={{ mr: 1 }}>Refresh Swagger</Button>
        <Button color="error" onClick={onDelete}>Delete</Button>
      </Box>
      <ServiceForm initial={{
        name: svc.name,
        description: svc.description,
        public_prefix: svc.public_prefix,
        base_url: svc.base_url,
        swagger_url: svc.swagger_url,
        enabled: svc.enabled,
      }} onSubmit={onUpdate} submitting={submitting} />
    </Container>
  )
}
