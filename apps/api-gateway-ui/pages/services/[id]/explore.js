import React, { useEffect, useState } from 'react'
import { useRouter } from 'next/router'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import axios from 'axios'
import ApiExplorer from '../../../src/components/ApiExplorer'

export default function ExploreService() {
  const router = useRouter()
  const { id } = router.query
  const [svc, setSvc] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    let mounted = true
    axios.get(`/admin/services/${id}`).then((res) => { if (mounted) setSvc(res.data) }).catch(() => {}).finally(() => mounted && setLoading(false))
    return () => (mounted = false)
  }, [id])

  if (loading) return <Container sx={{ mt: 4 }}>Loading...</Container>
  if (!svc) return <Container sx={{ mt: 4 }}>Not found</Container>

  return (
    <Container sx={{ mt: 4, mb: 6 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h5">Explore: {svc.name || svc.id}</Typography>
      </Box>
      <ApiExplorer service={svc} />
    </Container>
  )
}

export async function getServerSideProps() {
  return { props: {} }
}
