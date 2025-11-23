import React, { useEffect, useState } from 'react'
import axios from 'axios'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import AppBar from '@mui/material/AppBar'
import Toolbar from '@mui/material/Toolbar'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Link from 'next/link'
import ServiceList from '../src/components/ServiceList'

export default function Home() {
  const [services, setServices] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let mounted = true
    const fetchServices = () => {
      setLoading(true)
      axios
        .get('/admin/services')
        .then((res) => {
          if (mounted) setServices(res.data || [])
        })
        .catch(() => {
          if (mounted) setServices([])
        })
        .finally(() => mounted && setLoading(false))
    }

    fetchServices()
    // polling interval (30s)
    const iv = setInterval(fetchServices, 30000)
    return () => { mounted = false; clearInterval(iv) }
  }, [])

  return (
    <div>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6">Ecomm API Gateway</Typography>
        </Toolbar>
      </AppBar>
      <Container sx={{ mt: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Link href="/services/new"><Button variant="contained">Add Service</Button></Link>
          <Box>
            <Button onClick={() => { setLoading(true); axios.get('/admin/services').then(r => setServices(r.data || [])).finally(() => setLoading(false)) }} sx={{ mr: 1 }}>Refresh</Button>
            <Typography variant="caption">Polling every 30s</Typography>
          </Box>
        </Box>
        <Box sx={{ mb: 2 }}>
          <Typography variant="h5">Registered Services</Typography>
        </Box>
        <ServiceList services={services} loading={loading} />
      </Container>
    </div>
  )
}
