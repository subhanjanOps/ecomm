import React, { useState } from 'react'
import { useRouter } from 'next/router'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import axios from 'axios'
import ServiceForm from '../../src/components/ServiceForm'

export default function NewService() {
  const router = useRouter()
  const [submitting, setSubmitting] = useState(false)

  const handle = async (payload) => {
    setSubmitting(true)
    try {
      await axios.post('/admin/services', payload)
      router.push('/')
    } catch (e) {
      console.error(e)
      alert('Failed to create service: ' + (e?.response?.data || e.message))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Container sx={{ mt: 4 }}>
      <Typography variant="h5" sx={{ mb: 2 }}>Create Service</Typography>
      <Box>
        <ServiceForm onSubmit={handle} submitting={submitting} />
      </Box>
    </Container>
  )
}
