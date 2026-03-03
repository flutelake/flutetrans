function parseServiceError(err) {
  if (err == null) return {message: 'Unknown error'}

  if (typeof err === 'string') {
    try {
      const parsed = JSON.parse(err)
      if (parsed && typeof parsed === 'object') return parsed
    } catch {
      return {message: err}
    }
    return {message: err}
  }

  if (typeof err === 'object') {
    const message = typeof err.message === 'string' ? err.message : ''
    if (message) {
      try {
        const parsed = JSON.parse(message)
        if (parsed && typeof parsed === 'object') return parsed
      } catch {
      }
    }
    if (typeof err.code === 'number' || (err.details && typeof err.details === 'object')) {
      return err
    }
    if (message) return {message}
    return err
  }

  return {message: String(err)}
}

function getService() {
  return globalThis?.go?.services?.ConnectionService
}

async function call(method, ...args) {
  const service = getService()
  if (!service || typeof service[method] !== 'function') {
    throw {message: `Wails binding not found: ConnectionService.${method}`}
  }
  try {
    return await service[method](...args)
  } catch (err) {
    throw parseServiceError(err)
  }
}

export function isWailsAvailable() {
  const service = getService()
  return !!service
}

export function setMasterPassword(passphrase) {
  return call('SetMasterPassword', passphrase)
}

export function getMasterPasswordStatus() {
  return call('GetMasterPasswordStatus')
}

export function initializeMasterPassword(passphrase) {
  return call('InitializeMasterPassword', passphrase)
}

export function lockMasterPassword() {
  return call('LockMasterPassword')
}

export function listConnections() {
  return call('ListConnections')
}

export function getConnection(id) {
  return call('GetConnection', id)
}

export function saveConnection(profile) {
  return call('SaveConnection', profile)
}

export function deleteConnection(id) {
  return call('DeleteConnection', id)
}

export function testConnection(profile) {
  return call('TestConnection', profile)
}

export function connect(id) {
  return call('Connect', id)
}

export function disconnect(sessionID) {
  return call('Disconnect', sessionID)
}

export function listFiles(sessionID, requestedPath = '') {
  return call('ListFiles', sessionID, requestedPath)
}

export function getTransfers() {
  return call('GetTransfers')
}

export function pickUploadFiles() {
  return call('PickUploadFiles')
}

export function startUpload(sessionID, localPaths, remoteDir = '') {
  return call('StartUpload', sessionID, localPaths, remoteDir)
}

export function startDownload(sessionID, remotePath) {
  return call('StartDownload', sessionID, remotePath)
}

export function deleteRemotePath(sessionID, remotePath, recursive = false) {
  return call('DeleteRemotePath', sessionID, remotePath, !!recursive)
}

export function createRemoteDir(sessionID, dirPath) {
  return call('CreateRemoteDir', sessionID, dirPath)
}
