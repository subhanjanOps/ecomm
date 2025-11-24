import React, { useState, useMemo } from 'react'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import Box from '@mui/material/Box'
import Switch from '@mui/material/Switch'
import FormControlLabel from '@mui/material/FormControlLabel'

function isValidURL(s) {
  if (!s) return false
  try {
    // allow http/https only
    const u = new URL(s)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch (e) {
    return false
  }
}

export default function ServiceForm({ initial = {}, onSubmit, submitting }) {
  const [form, setForm] = useState({
    name: initial.name || '',
    description: initial.description || '',
    public_prefix: initial.public_prefix || initial.publicPrefix || '',
    base_url: initial.base_url || initial.baseUrl || '',
    swagger_url: initial.swagger_url || initial.swaggerUrl || '',
    protocol: (initial.protocol || 'http'),
    grpc_target: initial.grpc_target || initial.grpcTarget || '',
    enabled: typeof initial.enabled === 'boolean' ? initial.enabled : true,
  })

  const [touched, setTouched] = useState({})

  const change = (k) => (e) => {
    const v = e && e.target ? e.target.value : e
    setForm((s) => ({ ...s, [k]: v }))
  }

  const markTouched = (k) => () => setTouched((t) => ({ ...t, [k]: true }))

  const errors = useMemo(() => {
    const e = {}
    if (!form.public_prefix || form.public_prefix.trim() === '') {
      e.public_prefix = 'Public prefix is required'
    } else if (!form.public_prefix.startsWith('/')) {
      e.public_prefix = 'Public prefix should start with a `/`'
    }
    if (form.protocol === 'http') {
      if (!form.swagger_url || form.swagger_url.trim() === '') {
        e.swagger_url = 'Swagger URL is required'
      } else if (!isValidURL(form.swagger_url)) {
        e.swagger_url = 'Swagger URL must be a valid http(s) URL'
      }
      if (form.base_url && form.base_url.trim() !== '' && !isValidURL(form.base_url)) {
        e.base_url = 'Base URL must be a valid http(s) URL'
      }
    } else if (form.protocol === 'grpc-json') {
      if (!form.grpc_target || form.grpc_target.trim() === '') {
        e.grpc_target = 'gRPC target is required (host:port)'
      }
    }
    return e
  }, [form])

  const isValid = Object.keys(errors).length === 0

  const handleSubmit = (e) => {
    e.preventDefault()
    setTouched({ public_prefix: true, swagger_url: true, base_url: true })
    if (!isValid) return
    // pass normalized payload keys expected by backend
    const payload = {
      name: form.name,
      description: form.description,
      public_prefix: form.public_prefix,
      enabled: form.enabled,
      protocol: form.protocol,
    }
    if (form.protocol === 'http') {
      payload.base_url = form.base_url
      payload.swagger_url = form.swagger_url
    } else if (form.protocol === 'grpc-json') {
      payload.grpc_target = form.grpc_target
    }
    onSubmit(payload)
  }

  return (
    <Box component="form" onSubmit={handleSubmit} sx={{ display: 'grid', gap: 2 }}>
      <TextField label="Name" value={form.name} onChange={change('name')} fullWidth onBlur={markTouched('name')} />
      <TextField label="Description" value={form.description} onChange={change('description')} fullWidth onBlur={markTouched('description')} />
      <TextField
        label="Public Prefix"
        value={form.public_prefix}
        onChange={change('public_prefix')}
        fullWidth
        helperText={touched.public_prefix && errors.public_prefix ? errors.public_prefix : 'Example: /api/users/'}
        error={!!(touched.public_prefix && errors.public_prefix)}
        onBlur={markTouched('public_prefix')}
      />
      {/* Protocol selection */}
      <TextField
        label="Protocol"
        select
        SelectProps={{ native: true }}
        value={form.protocol}
        onChange={change('protocol')}
        fullWidth
      >
        <option value="http">HTTP (Swagger/OpenAPI)</option>
        <option value="grpc-json">HTTP → gRPC (JSON transcoding)</option>
      </TextField>

      {form.protocol === 'http' && (
        <>
          <TextField
            label="Base URL (optional)"
            value={form.base_url}
            onChange={change('base_url')}
            fullWidth
            helperText={touched.base_url && errors.base_url ? errors.base_url : 'If empty, gateway will try to infer from swagger servers.'}
            error={!!(touched.base_url && errors.base_url)}
            onBlur={markTouched('base_url')}
          />
          <TextField
            label="Swagger URL"
            value={form.swagger_url}
            onChange={change('swagger_url')}
            fullWidth
            helperText={touched.swagger_url && errors.swagger_url ? errors.swagger_url : 'URL to swagger.json or OpenAPI document'}
            error={!!(touched.swagger_url && errors.swagger_url)}
            onBlur={markTouched('swagger_url')}
          />
        </>
      )}

      {form.protocol === 'grpc-json' && (
        <TextField
          label="gRPC Target (host:port)"
          value={form.grpc_target}
          onChange={change('grpc_target')}
          fullWidth
          helperText={touched.grpc_target && errors.grpc_target ? errors.grpc_target : 'Example: user-service:9090'}
          error={!!(touched.grpc_target && errors.grpc_target)}
          onBlur={markTouched('grpc_target')}
        />
      )}
      <FormControlLabel control={<Switch checked={form.enabled} onChange={(e) => setForm((s) => ({ ...s, enabled: e.target.checked }))} />} label="Enabled" />
      <Box>
        <Button type="submit" variant="contained" disabled={submitting || !isValid}>{submitting ? 'Saving...' : 'Save'}</Button>
      </Box>
    </Box>
  )
}
