import React from 'react'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'

export default function Custom404() {
  return (
    <Container sx={{ mt: 8 }}>
      <Typography variant="h4">404 - Page Not Found</Typography>
      <Typography variant="body1" sx={{ mt: 2 }}>The page you requested could not be found.</Typography>
    </Container>
  )
}
