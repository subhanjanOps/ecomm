import React from 'react'
import Container from '@mui/material/Container'
import Typography from '@mui/material/Typography'

function Error({ statusCode }) {
  return (
    <Container sx={{ mt: 8 }}>
      <Typography variant="h4">{statusCode ? `${statusCode} Error` : 'An error occurred'}</Typography>
      <Typography variant="body1" sx={{ mt: 2 }}>Sorry, something went wrong.</Typography>
    </Container>
  )
}

Error.getInitialProps = ({ res, err }) => {
  const statusCode = res ? res.statusCode : err ? err.statusCode : 404
  return { statusCode }
}

export default Error