import React from 'react'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemText from '@mui/material/ListItemText'
import CircularProgress from '@mui/material/CircularProgress'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Paper from '@mui/material/Paper'
import Link from 'next/link'

export default function ServiceList({ services, loading }) {
  if (loading) return <CircularProgress />

  if (!services || services.length === 0) {
    return <Typography>No services registered.</Typography>
  }

  return (
    <Paper>
      <List>
        {services.map((s) => (
          <Link key={s.id} href={`/services/${s.id}`} style={{ textDecoration: 'none', color: 'inherit' }}>
            <ListItem divider button component="a">
              <ListItemText
                primary={s.name || s.id}
                secondary={
                  s.protocol === 'grpc-json'
                    ? `${s.public_prefix} → (grpc) ${s.grpc_target || ''}`
                    : `${s.public_prefix} → ${s.base_url}`
                }
              />
            </ListItem>
          </Link>
        ))}
      </List>
    </Paper>
  )
}
