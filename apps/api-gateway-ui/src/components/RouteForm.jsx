import React, { useMemo, useState } from 'react'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import Box from '@mui/material/Box'
import MenuItem from '@mui/material/MenuItem'
import Typography from '@mui/material/Typography'

export default function RouteForm({ initial = {}, onSubmit, submitting }) {
  const [form, setForm] = useState({
    method: (initial.method || 'GET').toUpperCase(),
    path: initial.path || '',
    grpc_method: initial.grpc_method || '',
    query_mapping: initial.query_mapping || {},
  })
  const change = (k) => (e) => setForm((s) => ({ ...s, [k]: e.target.value }))
  const methods = ['GET','POST','PUT','PATCH','DELETE']
  const errors = useMemo(() => {
    const e = {}
    if (!form.path) e.path = 'Path is required'
    if (!form.grpc_method) e.grpc_method = 'gRPC method is required (package.Service/Method)'
    return e
  }, [form])
  const isValid = Object.keys(errors).length === 0
  const submit = (e) => { e.preventDefault(); if (!isValid) return; onSubmit({ ...form }) }
  return (
    <Box component="form" onSubmit={submit} sx={{ display: 'grid', gap: 2 }}>
      <TextField label="HTTP Method" select value={form.method} onChange={change('method')}>
        {methods.map(m => (<MenuItem key={m} value={m}>{m}</MenuItem>))}
      </TextField>
      <TextField label="Path" value={form.path} onChange={change('path')} helperText={errors.path} error={!!errors.path} fullWidth />
      <TextField label="gRPC Method (package.Service/Method)" value={form.grpc_method} onChange={change('grpc_method')} helperText={errors.grpc_method} error={!!errors.grpc_method} fullWidth />
      <div>
        <Typography variant="subtitle2" sx={{ mb: 1 }}>Query Mapping (JSON, optional)</Typography>
        <TextField
          placeholder='{"page":{"field":"page","type":"int"},"search":{"field":"filter","type":"string"}}'
          multiline minRows={3} fullWidth
          value={JSON.stringify(form.query_mapping || {}, null, 2)}
          onChange={(e) => {
            try { const v = JSON.parse(e.target.value); setForm(s => ({ ...s, query_mapping: v })) } catch { /* ignore */ }
          }}
        />
      </div>
      <Box>
        <Button type="submit" variant="contained" disabled={submitting || !isValid}>{submitting ? 'Saving...' : 'Save'}</Button>
      </Box>
    </Box>
  )
}
