export const supportedLocales = [
  {id: 'zh', label: '中文'},
  {id: 'en', label: 'English'}
]

export const defaultLocale = 'zh'

export const messages = {
  en: {
    common: {
      actions: 'Actions',
      cancel: 'Cancel',
      close: 'Close',
      confirm: 'Confirm',
      connect: 'Connect',
      copy: 'Copy',
      copied: 'Copied',
      delete: 'Delete',
      details: 'Details',
      edit: 'Edit',
      empty: 'Empty',
      loading: 'Loading…',
      open: 'Open',
      refresh: 'Refresh',
      retry: 'Retry',
      save: 'Save',
      success: 'Success',
      failed: 'Failed'
    },
    nav: {
      connections: 'Connections',
      transfers: 'Transfers',
      fileBrowser: 'File Browser',
      language: 'Language',
      lock: 'Lock'
    },
    security: {
      lockedTitle: 'Locked',
      lockedDescription: 'Enter your password to continue.',
      masterPasswordPlaceholder: 'Master password',
      unlock: 'Unlock',
      unlockFailed: 'Unlock failed',
      errors: {
        required: 'Master password required',
        invalidMasterPassword: 'Incorrect master password',
        invalidEncryptedStore: 'Encrypted store is corrupted'
      },
      setup: {
        title: 'Set master password',
        description: 'This password encrypts stored credentials on your device.',
        createPlaceholder: 'Create master password',
        confirmPlaceholder: 'Confirm master password',
        continue: 'Continue',
        forgetHint: 'If you forget this password, encrypted credentials cannot be recovered.',
        errors: {
          required: 'Master password required',
          minLength: 'Use at least 8 characters',
          mismatch: 'Passwords do not match',
          failed: 'Setup failed'
        }
      }
    },
    connections: {
      manageTitle: 'Manage Connections',
      manageDescription: 'Create, edit, and switch saved connections.',
      list: {
        title: 'Connections',
        description: 'Saved connection profiles.',
        emptyTitle: 'No connections',
        emptyDescription: 'Create your first connection to get started.'
      },
      actions: {
        new: 'New connection',
        export: 'Export connections',
        exportSuccessTitle: 'Exported',
        exportSuccessMessage: 'connections-export.json downloaded'
      },
      confirmDelete: 'Delete this connection?',
      toasts: {
        deletedTitle: 'Deleted',
        deletedMessage: 'Connection removed',
        connectingTitle: 'Connecting',
        connectingMessage: 'Session {session}',
        savedTitle: 'Saved',
        savedMessage: 'Connection updated',
        masterPasswordSetTitle: 'Master password set',
        masterPasswordSetMessage: 'Encrypted store initialized'
      },
      errors: {
        loadFailedTitle: 'Load failed',
        failedToLoadConnections: 'Failed to load connections',
        actionFailed: 'Action failed',
        connectFailedTitle: 'Connect failed',
        saveFailedTitle: 'Save failed',
        deleteFailedTitle: 'Delete failed',
        exportFailedTitle: 'Export failed',
        unknownError: 'Unknown error'
      },
      form: {
        titleNew: 'New connection',
        titleEdit: 'Edit connection',
        description: 'Protocol-specific fields update automatically.',
        test: 'Test',
        fields: {
          name: 'Name',
          protocol: 'Protocol',
          host: 'Host / Endpoint / URL',
          port: 'Port (empty = default)',
          path: 'Path / Bucket / Share',
          authType: 'Auth type',
          metadata: 'Metadata',
          credentials: 'Credentials',
          username: 'Username',
          password: 'Password',
          privateKeyPath: 'Private key path',
          passphrase: 'Passphrase',
          accessKeyId: 'Access key ID',
          secretAccessKey: 'Secret access key',
          region: 'Region'
        },
        placeholders: {
          name: 'My server'
        },
        authOptions: {
          password: 'Password',
          key: 'Private key'
        }
      }
    },
    fileBrowser: {
      title: 'File Browser',
      fromConnection: 'From connection: {name}',
      selectConnectionHint: 'Please select a connection to browse files.',
      noActiveSessionTitle: 'No active session',
      noActiveSessionDescription: 'Connect to a server to browse files.',
      connectingTitle: 'Connecting',
      pleaseWait: 'Please wait…',
      up: 'Up',
      disconnect: 'Disconnect',
      newFolder: 'New folder',
      createFolderTitle: 'Create folder',
      folderNamePlaceholder: 'Folder name',
      upload: 'Upload',
      table: {
        name: 'Name',
        size: 'Size',
        modified: 'Modified'
      },
      menu: {
        download: 'Download',
        delete: 'Delete'
      },
      toasts: {
        loadFailedTitle: 'Load files failed',
        downloadStartedTitle: 'Download started',
        downloadFailedTitle: 'Download failed',
        deletedTitle: 'Deleted',
        deleteFailedTitle: 'Delete failed',
        dirCreatedTitle: 'Folder created',
        dirCreateFailedTitle: 'Create folder failed',
        uploadStartedTitle: 'Upload started',
        uploadStartedMessage: '{count} item(s)',
        uploadFailedTitle: 'Upload failed'
      },
      confirmDeleteTitle: 'Delete file',
      confirmDeleteDescription: 'Are you sure you want to delete this item?',
      confirmDeleteWarningDir: 'Deleting a folder will recursively delete its contents.',
      confirmDeleteWarningFile: 'Deleted files cannot be recovered.',
      cancel: 'Cancel',
      delete: 'Delete'
    },
    transfers: {
      title: 'Transfers',
      description: 'Manage upload/download tasks across protocols.',
      loadFailedTitle: 'Load transfers failed',
      failedToLoad: 'Failed to load transfers',
      sessionLabel: 'Session: {id}',
      noTransfers: 'No transfers',
      direction: {
        upload: 'Upload',
        download: 'Download'
      },
      protocol: {
        all: 'All'
      }
    },
    testConnection: {
      testing: 'Testing…',
      statusSuccess: 'Success',
      statusFailed: 'Failed',
      dialogTitle: 'Test connection',
      toastSuccessTitle: 'Test succeeded',
      toastFailedTitle: 'Test failed',
      failedFallback: 'Test failed'
    },
    status: {
      connected: 'Connected',
      connecting: 'Connecting',
      error: 'Error',
      disconnected: 'Disconnected'
    }
  },
  zh: {
    common: {
      actions: '操作',
      cancel: '取消',
      close: '关闭',
      confirm: '确认',
      connect: '连接',
      copy: '复制',
      copied: '已复制',
      delete: '删除',
      details: '详情',
      edit: '编辑',
      empty: '空',
      loading: '加载中…',
      open: '打开',
      refresh: '刷新',
      retry: '重试',
      save: '保存',
      success: '成功',
      failed: '失败'
    },
    nav: {
      connections: '连接',
      transfers: '传输',
      fileBrowser: '文件浏览',
      language: '语言',
      lock: '锁定'
    },
    security: {
      lockedTitle: '已锁定',
      lockedDescription: '请输入解锁密码继续使用。',
      masterPasswordPlaceholder: '主密码',
      unlock: '解锁',
      unlockFailed: '解锁失败',
      errors: {
        required: '需要主密码',
        invalidMasterPassword: '主密码不正确',
        invalidEncryptedStore: '加密存储已损坏'
      },
      setup: {
        title: '设置主密码',
        description: '该密码会在本机加密保存的凭据。',
        createPlaceholder: '创建主密码',
        confirmPlaceholder: '确认主密码',
        continue: '继续',
        forgetHint: '如果忘记该密码，加密的凭据将无法恢复。',
        errors: {
          required: '需要主密码',
          minLength: '至少使用 8 个字符',
          mismatch: '两次输入的密码不一致',
          failed: '设置失败'
        }
      }
    },
    connections: {
      manageTitle: '连接管理',
      manageDescription: '创建、编辑并切换已保存的连接。',
      list: {
        title: '连接',
        description: '已保存的连接配置。',
        emptyTitle: '暂无连接',
        emptyDescription: '创建你的第一个连接以开始使用。'
      },
      actions: {
        new: '新建连接',
        export: '导出连接',
        exportSuccessTitle: '已导出',
        exportSuccessMessage: '已下载 connections-export.json'
      },
      confirmDelete: '确定删除该连接吗？',
      toasts: {
        deletedTitle: '已删除',
        deletedMessage: '连接已移除',
        connectingTitle: '正在连接',
        connectingMessage: '会话 {session}',
        savedTitle: '已保存',
        savedMessage: '连接已更新',
        masterPasswordSetTitle: '主密码已设置',
        masterPasswordSetMessage: '已初始化加密存储'
      },
      errors: {
        loadFailedTitle: '加载失败',
        failedToLoadConnections: '加载连接失败',
        actionFailed: '操作失败',
        connectFailedTitle: '连接失败',
        saveFailedTitle: '保存失败',
        deleteFailedTitle: '删除失败',
        exportFailedTitle: '导出失败',
        unknownError: '未知错误'
      },
      form: {
        titleNew: '新建连接',
        titleEdit: '编辑连接',
        description: '字段会根据协议自动调整。',
        test: '测试',
        fields: {
          name: '名称',
          protocol: '协议',
          host: '主机 / Endpoint / URL',
          port: '端口（留空 = 默认）',
          path: '路径 / Bucket / Share',
          authType: '认证方式',
          metadata: '元数据',
          credentials: '凭据',
          username: '用户名',
          password: '密码',
          privateKeyPath: '私钥路径',
          passphrase: '口令',
          accessKeyId: 'Access key ID',
          secretAccessKey: 'Secret access key',
          region: '区域'
        },
        placeholders: {
          name: '我的服务器'
        },
        authOptions: {
          password: '密码',
          key: '私钥'
        }
      }
    },
    fileBrowser: {
      title: '文件浏览',
      fromConnection: '来自连接：{name}',
      selectConnectionHint: '请选择一个连接以浏览文件。',
      noActiveSessionTitle: '没有可用会话',
      noActiveSessionDescription: '连接到服务器后即可浏览文件。',
      connectingTitle: '连接中',
      pleaseWait: '请稍候…',
      up: '上一级',
      disconnect: '断开',
      newFolder: '新建目录',
      createFolderTitle: '创建目录',
      folderNamePlaceholder: '目录名称',
      upload: '上传',
      table: {
        name: '名称',
        size: '大小',
        modified: '修改时间'
      },
      menu: {
        download: '下载',
        delete: '删除'
      },
      toasts: {
        loadFailedTitle: '加载文件失败',
        downloadStartedTitle: '开始下载',
        downloadFailedTitle: '下载失败',
        deletedTitle: '已删除',
        deleteFailedTitle: '删除失败',
        dirCreatedTitle: '目录已创建',
        dirCreateFailedTitle: '创建目录失败',
        uploadStartedTitle: '开始上传',
        uploadStartedMessage: '{count} 个项目',
        uploadFailedTitle: '上传失败'
      },
      confirmDeleteTitle: '删除文件',
      confirmDeleteDescription: '确定要删除该条目吗？',
      confirmDeleteWarningDir: '删除文件夹将递归删除其中所有内容。',
      confirmDeleteWarningFile: '删除文件将无法恢复。',
      cancel: '取消',
      delete: '删除'
    },
    transfers: {
      title: '传输',
      description: '集中管理所有协议的上传/下载任务。',
      loadFailedTitle: '加载传输失败',
      failedToLoad: '加载传输失败',
      sessionLabel: '会话：{id}',
      noTransfers: '暂无传输任务',
      direction: {
        upload: '上传',
        download: '下载'
      },
      protocol: {
        all: '全部'
      }
    },
    testConnection: {
      testing: '测试中…',
      statusSuccess: '成功',
      statusFailed: '失败',
      dialogTitle: '测试连接',
      toastSuccessTitle: '测试成功',
      toastFailedTitle: '测试失败',
      failedFallback: '测试失败'
    },
    status: {
      connected: '已连接',
      connecting: '连接中',
      error: '错误',
      disconnected: '已断开'
    }
  }
}
