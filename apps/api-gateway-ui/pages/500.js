import React from 'react'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'

export default function Custom500() {
  return (
    <Container sx={{ mt: 8 }}>
      <Typography variant="h4">500 - Server Error</Typography>
      <Typography variant="body1" sx={{ mt: 2 }}>An unexpected error occurred. Try again later.</Typography>
    </Container>
  )
}
