<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {Input} from '$lib/components/ui/input/index.js'
  import {Label} from '$lib/components/ui/label/index.js'
  import {Select} from '$lib/components/ui/select/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import {Separator} from '$lib/components/ui/separator/index.js'
  import TestConnectionButton from './TestConnectionButton.svelte'

  const MASK = '********'

  export let mode = 'new'
  export let initialProfile
  export let saving = false
  export let onSave = (_profile) => {}
  export let onCancel = () => {}

  let name = ''
  let protocol = 'sftp'
  let host = ''
  let port = ''
  let path = ''
  let authType = 'password'
  let metadata = {}
  let credentials = {}
  let credentialsMasked = {}
  let touched = {}

  $: if (initialProfile) {
    name = initialProfile.name ?? ''
    protocol = initialProfile.protocol ?? 'sftp'
    host = initialProfile.host ?? ''
    port = initialProfile.port ? String(initialProfile.port) : ''
    path = initialProfile.path ?? ''
    authType = initialProfile.authType ?? 'password'
    metadata = initialProfile.metadata ?? {}
    credentialsMasked = initialProfile.credentialsMasked ?? {}
    credentials = {}
    touched = {}

    const initialCredentials = initialProfile.credentials ?? {}

    for (const key of visibleCredentialKeys(protocol, authType)) {
      if (credentialsMasked[key]) {
        credentials[key] = MASK
        continue
      }

      if (initialCredentials[key] != null && String(initialCredentials[key]).trim() !== '') {
        credentials[key] = String(initialCredentials[key])
        continue
      }

      if ((key === 'username' || key === 'accessKeyId') && initialProfile.username) {
        credentials[key] = String(initialProfile.username)
        continue
      } else {
        credentials[key] = ''
      }
    }
  }

  $: if (!initialProfile) {
    name = ''
    protocol = 'sftp'
    host = ''
    port = ''
    path = ''
    authType = 'password'
    metadata = {}
    credentialsMasked = {}
    credentials = {
      username: '',
      password: ''
    }
    touched = {}
  }

  function visibleCredentialKeys(p, a) {
    if (p === 's3') return ['accessKeyId', 'secretAccessKey']
    if (p === 'nfs') return []
    if (p === 'sftp' && a === 'key') return ['username', 'privateKeyPath', 'passphrase']
    return ['username', 'password']
  }

  function visibleMetadataKeys(p) {
    if (p === 's3') return ['region']
    return []
  }

  function labelForKey(key) {
    switch (key) {
      case 'username': return 'Username'
      case 'password': return 'Password'
      case 'privateKeyPath': return 'Private key path'
      case 'passphrase': return 'Passphrase'
      case 'accessKeyId': return 'Access key ID'
      case 'secretAccessKey': return 'Secret access key'
      default: return key
    }
  }

  function markTouched(key) {
    touched = {...touched, [key]: true}
  }

  function updateCredential(key, value) {
    credentials = {...credentials, [key]: value}
    markTouched(key)
  }

  function updateMetadata(key, value) {
    metadata = {...metadata, [key]: value}
  }

  function readEventValue(event) {
    return (
      event?.detail?.target?.value ??
      event?.detail?.currentTarget?.value ??
      event?.target?.value ??
      event?.currentTarget?.value ??
      ''
    )
  }

  function buildPayload() {
    const base = {
      id: initialProfile?.id ?? '',
      name,
      protocol,
      host,
      port: Number(port) || 0,
      authType,
      path,
      metadata: {...metadata}
    }

    const outCredentials = {}
    for (const key of visibleCredentialKeys(protocol, authType)) {
      const value = credentials[key]
      const isSensitive = key === 'password' || key === 'secretAccessKey' || key === 'passphrase'
      if (isSensitive && credentialsMasked[key] && value === MASK && !touched[key]) {
        continue
      }
      outCredentials[key] = value === MASK ? '' : value
    }

    if (Object.keys(outCredentials).length > 0) {
      base.credentials = outCredentials
    }
    return base
  }

  function submit() {
    console.log("click save btn")
    onSave(buildPayload())
  }

  $: if (protocol === 'sftp' && (authType !== 'password' && authType !== 'key')) {
    authType = 'password'
  }

  $: if (protocol !== 'sftp') {
    authType = protocol === 's3' ? 's3_static' : (protocol === 'nfs' ? 'none' : 'password')
  }

  $: {
    for (const key of visibleCredentialKeys(protocol, authType)) {
      if (!(key in credentials)) {
        credentials = {...credentials, [key]: credentialsMasked[key] ? MASK : ''}
      }
    }
    for (const key of Object.keys(credentials)) {
      if (!visibleCredentialKeys(protocol, authType).includes(key)) {
        const copy = {...credentials}
        delete copy[key]
        credentials = copy
      }
    }
  }

  $: {
	for (const key of visibleMetadataKeys(protocol)) {
	  if (!(key in metadata)) metadata = {...metadata, [key]: ''}
	}
    for (const key of Object.keys(metadata)) {
      if (!visibleMetadataKeys(protocol).includes(key)) {
        const copy = {...metadata}
        delete copy[key]
        metadata = copy
      }
    }
  }
</script>

<Card className="h-full flex flex-col">
  <CardHeader className="flex-row items-start justify-between space-y-0">
    <div class="space-y-1 text-left">
      <CardTitle className="text-base">{mode === 'new' ? 'New connection' : 'Edit connection'}</CardTitle>
      <CardDescription>Protocol-specific fields update automatically.</CardDescription>
    </div>

    <div class="flex items-center gap-2">
      <TestConnectionButton getProfile={buildPayload} disabled={saving} />
      <Button variant="secondary" size="sm" on:click={onCancel} disabled={saving}>Cancel</Button>
      <Button size="sm" on:click={submit} disabled={saving}>Save</Button>
    </div>
  </CardHeader>

  <CardContent className="flex-1 min-h-0 space-y-6">
    <div class="grid grid-cols-2 gap-4">
      <div class="space-y-2 text-left">
        <Label>Name</Label>
        <Input bind:value={name} placeholder="My server" />
      </div>

      <div class="space-y-2 text-left">
        <Label>Protocol</Label>
        <Select bind:value={protocol}>
          <option value="ftp">FTP</option>
          <option value="sftp">SFTP</option>
          <option value="s3">S3</option>
          <option value="webdav">WebDAV</option>
          <option value="smb">SMB</option>
          <option value="nfs">NFS</option>
        </Select>
      </div>

      <div class="space-y-2 text-left">
        <Label>Host / Endpoint / URL</Label>
        <Input bind:value={host} placeholder={protocol === 's3' ? 'https://s3.amazonaws.com' : 'example.com'} />
      </div>

      <div class="space-y-2 text-left">
        <Label>Port (empty = default)</Label>
        <Input type="number" bind:value={port} min="0" max="65535" placeholder="0" />
      </div>

      <div class="space-y-2 text-left">
        <Label>Path / Bucket / Share</Label>
        <Input bind:value={path} placeholder={protocol === 's3' ? 'bucket-name' : '/'} />
      </div>

      {#if protocol === 'sftp'}
        <div class="space-y-2 text-left">
          <Label>Auth type</Label>
          <Select bind:value={authType}>
            <option value="password">Password</option>
            <option value="key">Private key</option>
          </Select>
        </div>
      {/if}
    </div>

    {#if visibleMetadataKeys(protocol).length > 0}
      <div class="space-y-3">
        <Separator />
        <div class="text-left text-sm font-medium">Metadata</div>
        <div class="grid grid-cols-2 gap-4">
          {#each visibleMetadataKeys(protocol) as key (key)}
            <div class="space-y-2 text-left">
              <Label>{labelForKey(key)}</Label>
              <Input
                value={metadata[key] ?? ''}
                on:input={(e) => updateMetadata(key, readEventValue(e))}
              />
            </div>
          {/each}
        </div>
      </div>
    {/if}

    {#if visibleCredentialKeys(protocol, authType).length > 0}
      <div class="space-y-3">
        <Separator />
        <div class="text-left text-sm font-medium">Credentials</div>
        <div class="grid grid-cols-2 gap-4">
          {#each visibleCredentialKeys(protocol, authType) as key (key)}
            <div class="space-y-2 text-left">
              <Label>{labelForKey(key)}</Label>
              <Input
                type={(key === 'password' || key === 'secretAccessKey' || key === 'passphrase') ? 'password' : 'text'}
                value={credentials[key] ?? ''}
                on:input={(e) => updateCredential(key, readEventValue(e))}
              />
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </CardContent>
</Card>
