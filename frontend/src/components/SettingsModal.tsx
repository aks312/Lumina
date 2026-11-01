import React, { useState, useEffect } from 'react'
import {
  X,
  HardDrive,
  FolderSync,
  Cloud,
  Cpu,
  Wrench,
  Gauge,
  CheckCircle2,
  Trash2,
  Plus,
  RefreshCw,
  ExternalLink,
  ChevronDown,
  ChevronRight,
  AlertTriangle
} from 'lucide-react'
import { WatchedFolder, DeviceMapping } from '../types'
import { api } from '../services/api'

interface SettingsModalProps {
  isOpen: boolean
  onClose: () => void
  onRefreshStats: () => void
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  onRefreshStats,
}) => {
  const [activeTab, setActiveTab] = useState<'storage' | 'folders' | 'devices' | 'google' | 'ai' | 'maintenance'>('storage')

  // Storage Settings
  const [primaryDest, setPrimaryDest] = useState('')
  const [secondaryMirror, setSecondaryMirror] = useState('')
  const [benchResults, setBenchResults] = useState<any>(null)
  const [isBenchmarking, setIsBenchmarking] = useState(false)

  // Watched Folders
  const [watchedFolders, setWatchedFolders] = useState<WatchedFolder[]>([])
  const [newFolderPath, setNewFolderPath] = useState('')
  const [newRouteOverride, setNewRouteOverride] = useState('')
  const [newShowInAll, setNewShowInAll] = useState(true)

  // Device Mappings
  const [deviceMappings, setDeviceMappings] = useState<DeviceMapping[]>([])
  const [newDeviceName, setNewDeviceName] = useState('')
  const [newCustomFolder, setNewCustomFolder] = useState('')

  // Google Photos
  const [isGoogleConnected, setIsGoogleConnected] = useState(false)
  const [showCustomOAuth, setShowCustomOAuth] = useState(false)
  const [customClientId, setCustomClientId] = useState('')
  const [customClientSecret, setCustomClientSecret] = useState('')

  // AI Boost
  const [geminiKey, setGeminiKey] = useState('')

  // Maintenance Status Message
  const [maintenanceMsg, setMaintenanceMsg] = useState('')

  useEffect(() => {
    if (isOpen) {
      loadSettings()
    }
  }, [isOpen])

  const loadSettings = async () => {
    const p = await api.getSetting('primary_destination', '')
    const m = await api.getSetting('secondary_mirror', '')
    const gKey = await api.getSetting('gemini_api_key', '')
    const gCid = await api.getSetting('gphotos_client_id', '')
    const gSec = await api.getSetting('gphotos_client_secret', '')
    const isGConnected = await api.isGoogleConnected()

    setPrimaryDest(p)
    setSecondaryMirror(m)
    setGeminiKey(gKey)
    setCustomClientId(gCid)
    setCustomClientSecret(gSec)
    setIsGoogleConnected(isGConnected)

    const wf = await api.getWatchedFolders()
    setWatchedFolders(wf)

    const dm = await api.getDeviceMappings()
    setDeviceMappings(dm)
  }

  if (!isOpen) return null

  const handleBrowsePrimary = async () => {
    const dir = await api.selectDirectoryDialog()
    if (dir) {
      setPrimaryDest(dir)
      await api.saveSetting('primary_destination', dir)
    }
  }

  const handleBrowseMirror = async () => {
    const dir = await api.selectDirectoryDialog()
    if (dir) {
      setSecondaryMirror(dir)
      await api.saveSetting('secondary_mirror', dir)
    }
  }

  const handleRunBenchmark = async () => {
    setIsBenchmarking(true)
    try {
      const res = await api.benchmarkDrives()
      setBenchResults(res)
    } finally {
      setIsBenchmarking(false)
    }
  }

  const handleAddWatchedFolder = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFolderPath.trim()) return
    await api.addWatchedFolder(newFolderPath.trim(), newRouteOverride.trim(), newShowInAll)
    setNewFolderPath('')
    setNewRouteOverride('')
    setNewShowInAll(true)
    const updated = await api.getWatchedFolders()
    setWatchedFolders(updated)
    onRefreshStats()
  }

  const handleRemoveWatchedFolder = async (id: number) => {
    await api.removeWatchedFolder(id)
    const updated = await api.getWatchedFolders()
    setWatchedFolders(updated)
    onRefreshStats()
  }

  const handleAddDeviceMapping = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newDeviceName.trim() || !newCustomFolder.trim()) return
    await api.setDeviceMapping(newDeviceName.trim(), newCustomFolder.trim())
    setNewDeviceName('')
    setNewCustomFolder('')
    const updated = await api.getDeviceMappings()
    setDeviceMappings(updated)
  }

  const handleDeleteDeviceMapping = async (dev: string) => {
    await api.deleteDeviceMapping(dev)
    const updated = await api.getDeviceMappings()
    setDeviceMappings(updated)
  }

  const handleSaveGeminiKey = async () => {
    await api.saveSetting('gemini_api_key', geminiKey.trim())
    setMaintenanceMsg('Gemini API Key saved successfully!')
    setTimeout(() => setMaintenanceMsg(''), 3000)
  }

  const handleGoogleSignIn = async () => {
    if (customClientId.trim()) {
      await api.saveSetting('gphotos_client_id', customClientId.trim())
    }
    if (customClientSecret.trim()) {
      await api.saveSetting('gphotos_client_secret', customClientSecret.trim())
    }
    await api.startGoogleSignIn()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-slate-950/80 backdrop-blur-xl select-none">
      <div className="w-full max-w-3xl h-[680px] glass-panel rounded-2xl flex flex-col overflow-hidden shadow-2xl border border-white/15">
        {/* Modal Header */}
        <div className="px-6 py-4 flex items-center justify-between border-b border-white/10">
          <div className="flex items-center gap-2.5">
            <h3 className="text-base font-semibold text-white">Settings & Preferences</h3>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Modal Body: Tabs & Content */}
        <div className="flex-1 flex overflow-hidden">
          {/* Sub-Nav Sidebar */}
          <div className="w-52 border-r border-white/10 p-3 space-y-1">
            <button
              onClick={() => setActiveTab('storage')}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition ${
                activeTab === 'storage'
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                  : 'text-slate-300 hover:bg-white/5'
              }`}
            >
              <HardDrive className="w-3.5 h-3.5" />
              <span>Storage & NVMe</span>
            </button>

            <button
              onClick={() => setActiveTab('folders')}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition ${
                activeTab === 'folders'
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                  : 'text-slate-300 hover:bg-white/5'
              }`}
            >
              <FolderSync className="w-3.5 h-3.5" />
              <span>Watched Folders</span>
            </button>

            <button
              onClick={() => setActiveTab('devices')}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition ${
                activeTab === 'devices'
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                  : 'text-slate-300 hover:bg-white/5'
              }`}
            >
              <Cpu className="w-3.5 h-3.5" />
              <span>Device Routing</span>
            </button>

            <button
              onClick={() => setActiveTab('google')}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition ${
                activeTab === 'google'
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                  : 'text-slate-300 hover:bg-white/5'
              }`}
            >
              <Cloud className="w-3.5 h-3.5" />
              <span>Google Photos</span>
            </button>

            <button
              onClick={() => setActiveTab('ai')}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition ${
                activeTab === 'ai'
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                  : 'text-slate-300 hover:bg-white/5'
              }`}
            >
              <Cpu className="w-3.5 h-3.5 text-fuchsia-400" />
              <span>AI Cloud Boost</span>
            </button>

            <div className="pt-2 border-t border-white/5">
              <button
                onClick={() => setActiveTab('maintenance')}
                className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition ${
                  activeTab === 'maintenance'
                    ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                    : 'text-slate-400 hover:text-rose-400 hover:bg-white/5'
                }`}
              >
                <Wrench className="w-3.5 h-3.5 text-rose-400" />
                <span>Catalog Maintenance</span>
              </button>
            </div>
          </div>

          {/* Active Tab Content Area */}
          <div className="flex-1 p-6 overflow-y-auto">
            {maintenanceMsg && (
              <div className="mb-4 px-4 py-2.5 rounded-xl bg-sky-500/20 border border-sky-500/40 text-xs text-sky-300 flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4" />
                <span>{maintenanceMsg}</span>
              </div>
            )}

            {/* TAB 1: Storage & NVMe */}
            {activeTab === 'storage' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-semibold text-white">Destination Drives & NVMe Benchmarking</h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Configure your primary storage location and optional secondary mirror drive.
                  </p>
                </div>

                <div className="space-y-4">
                  {/* Primary Destination */}
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-300">Primary Destination Directory</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={primaryDest}
                        readOnly
                        placeholder="Select primary photo directory..."
                        className="flex-1 px-3 py-2 text-xs rounded-xl bg-slate-900 border border-white/10 text-white font-mono"
                      />
                      <button
                        onClick={handleBrowsePrimary}
                        className="px-3 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-medium text-white transition border border-white/10"
                      >
                        Browse
                      </button>
                    </div>
                  </div>

                  {/* Secondary Mirror Destination */}
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-300">Secondary Mirror Directory (Optional Backup Drive)</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={secondaryMirror}
                        readOnly
                        placeholder="Optional mirror drive path..."
                        className="flex-1 px-3 py-2 text-xs rounded-xl bg-slate-900 border border-white/10 text-white font-mono"
                      />
                      <button
                        onClick={handleBrowseMirror}
                        className="px-3 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-medium text-white transition border border-white/10"
                      >
                        Browse
                      </button>
                    </div>
                  </div>

                  {/* Drive Benchmark */}
                  <div className="pt-4 border-t border-white/5 space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Gauge className="w-4 h-4 text-sky-400" />
                        <span className="text-xs font-semibold text-white">NVMe Throughput Benchmark</span>
                      </div>
                      <button
                        onClick={handleRunBenchmark}
                        disabled={isBenchmarking || !primaryDest}
                        className="px-3 py-1.5 rounded-xl bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium transition disabled:opacity-50"
                      >
                        {isBenchmarking ? 'Testing...' : 'Benchmark Drives'}
                      </button>
                    </div>

                    {benchResults && (
                      <div className="p-3 rounded-xl bg-slate-900/80 border border-white/10 space-y-2 text-xs">
                        <div className="flex items-center justify-between">
                          <span className="text-slate-400">Primary Drive Read:</span>
                          <span className="text-emerald-400 font-mono font-medium">
                            {benchResults.primary?.read_mbps?.toFixed(1) || 0} MB/s
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-slate-400">Primary Drive Write:</span>
                          <span className="text-sky-400 font-mono font-medium">
                            {benchResults.primary?.write_mbps?.toFixed(1) || 0} MB/s
                          </span>
                        </div>
                        {benchResults.fastest && (
                          <div className="pt-2 border-t border-white/5 flex items-center justify-between text-slate-300">
                            <span>Serving Photo Viewer from:</span>
                            <span className="text-sky-300 font-semibold truncate max-w-xs">{benchResults.fastest}</span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* TAB 2: Watched Folders */}
            {activeTab === 'folders' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-semibold text-white">Watched Source Folders</h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Directories automatically scanned for new photos. Custom Route Overrides (e.g. "Amruta") apply Priority 2 routing!
                  </p>
                </div>

                {/* Add Folder Form */}
                <form onSubmit={handleAddWatchedFolder} className="p-4 rounded-xl bg-slate-900/70 border border-white/10 space-y-3">
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-300">Source Folder Path</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        value={newFolderPath}
                        onChange={(e) => setNewFolderPath(e.target.value)}
                        placeholder="C:\Users\...\PhoneSync"
                        className="flex-1 px-3 py-1.5 text-xs rounded-lg bg-slate-800 border border-white/10 text-white font-mono"
                      />
                      <button
                        type="button"
                        onClick={async () => {
                          const dir = await api.selectDirectoryDialog()
                          if (dir) setNewFolderPath(dir)
                        }}
                        className="px-3 py-1.5 rounded-lg bg-slate-700 hover:bg-slate-600 text-xs text-white"
                      >
                        Browse
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1">
                      <label className="text-xs font-medium text-slate-300">Custom Route Override (Optional)</label>
                      <input
                        type="text"
                        value={newRouteOverride}
                        onChange={(e) => setNewRouteOverride(e.target.value)}
                        placeholder="e.g. Amruta"
                        className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-800 border border-white/10 text-white"
                      />
                    </div>

                    <div className="flex items-center gap-2 pt-6">
                      <input
                        type="checkbox"
                        id="showInAllToggle"
                        checked={newShowInAll}
                        onChange={(e) => setNewShowInAll(e.target.checked)}
                        className="rounded bg-slate-800 border-white/10 text-sky-500 focus:ring-0"
                      />
                      <label htmlFor="showInAllToggle" className="text-xs text-slate-300 cursor-pointer">
                        Show in "All Photos" View
                      </label>
                    </div>
                  </div>

                  <div className="pt-2 flex justify-end">
                    <button
                      type="submit"
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium transition"
                    >
                      <Plus className="w-3.5 h-3.5" />
                      <span>Add Watched Folder</span>
                    </button>
                  </div>
                </form>

                {/* Watched Folders List */}
                <div className="space-y-2">
                  {watchedFolders.length === 0 ? (
                    <p className="text-xs text-slate-500 italic">No watched folders configured yet.</p>
                  ) : (
                    watchedFolders.map((wf) => (
                      <div
                        key={wf.id}
                        className="p-3 rounded-xl bg-slate-900/50 border border-white/5 flex items-center justify-between text-xs"
                      >
                        <div className="space-y-0.5 truncate max-w-md">
                          <p className="font-mono text-slate-200 truncate">{wf.path}</p>
                          <p className="text-slate-400">
                            Route Override: <span className="text-sky-300">{wf.route_override || 'Default Platform'}</span> •{' '}
                            {wf.show_in_all_photos ? 'Visible in All' : 'Hidden from All'}
                          </p>
                        </div>
                        <button
                          onClick={() => handleRemoveWatchedFolder(wf.id)}
                          className="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}

            {/* TAB 3: Device Routing */}
            {activeTab === 'devices' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-semibold text-white">Device-to-Folder Mappings (Priority 4)</h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Assign any hardware device (e.g. "Samsung_Galaxy_S23") to a custom person/owner folder.
                  </p>
                </div>

                <form onSubmit={handleAddDeviceMapping} className="p-4 rounded-xl bg-slate-900/70 border border-white/10 flex items-center gap-3">
                  <div className="flex-1 space-y-1">
                    <label className="text-xs text-slate-400">Canonical Device Name</label>
                    <input
                      type="text"
                      value={newDeviceName}
                      onChange={(e) => setNewDeviceName(e.target.value)}
                      placeholder="e.g. Samsung_Galaxy_S23"
                      className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-800 border border-white/10 text-white font-mono"
                    />
                  </div>
                  <div className="flex-1 space-y-1">
                    <label className="text-xs text-slate-400">Target Folder / Person</label>
                    <input
                      type="text"
                      value={newCustomFolder}
                      onChange={(e) => setNewCustomFolder(e.target.value)}
                      placeholder="e.g. Akash"
                      className="w-full px-3 py-1.5 text-xs rounded-lg bg-slate-800 border border-white/10 text-white"
                    />
                  </div>
                  <button
                    type="submit"
                    className="self-end px-3 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium transition"
                  >
                    Map Device
                  </button>
                </form>

                <div className="space-y-2">
                  {deviceMappings.length === 0 ? (
                    <p className="text-xs text-slate-500 italic">No device mappings configured.</p>
                  ) : (
                    deviceMappings.map((dm) => (
                      <div
                        key={dm.id}
                        className="p-3 rounded-xl bg-slate-900/50 border border-white/5 flex items-center justify-between text-xs"
                      >
                        <div>
                          <span className="font-mono text-slate-200">{dm.device_name}</span>
                          <span className="text-slate-500 mx-2">→</span>
                          <span className="text-sky-300 font-medium">{dm.custom_folder}</span>
                        </div>
                        <button
                          onClick={() => handleDeleteDeviceMapping(dm.device_name)}
                          className="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}

            {/* TAB 4: Google Photos */}
            {activeTab === 'google' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-semibold text-white">Google Photos Cloud Sync</h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    1-Click secure sign-in with local PKCE authentication. Unlimited batch syncing and album routing.
                  </p>
                </div>

                <div className="p-6 rounded-2xl bg-slate-900/80 border border-white/10 text-center space-y-4">
                  <div className="w-12 h-12 rounded-2xl bg-blue-500/20 text-blue-400 mx-auto flex items-center justify-center">
                    <Cloud className="w-6 h-6" />
                  </div>

                  <div>
                    <h5 className="text-base font-semibold text-white">
                      {isGoogleConnected ? 'Google Photos Connected' : 'Connect Your Google Account'}
                    </h5>
                    <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
                      {isGoogleConnected
                        ? 'Your catalog is synced with device albums and metadata embedding.'
                        : 'Sign in with your browser to enable seamless cloud backup and sync.'}
                    </p>
                  </div>

                  <button
                    onClick={handleGoogleSignIn}
                    className="px-6 py-2.5 rounded-full bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 transition flex items-center gap-2 mx-auto"
                  >
                    <span>{isGoogleConnected ? 'Reconnect Google Account' : 'Continue with Google Photos'}</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </button>

                  {/* Collapsible Developer Credentials */}
                  <div className="pt-4 border-t border-white/5 text-left">
                    <button
                      onClick={() => setShowCustomOAuth(!showCustomOAuth)}
                      className="text-[11px] text-slate-500 hover:text-slate-400 flex items-center gap-1 transition"
                    >
                      {showCustomOAuth ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                      <span>Use Custom Developer Credentials (Optional)</span>
                    </button>

                    {showCustomOAuth && (
                      <div className="mt-3 p-3 rounded-xl bg-slate-950/60 border border-white/5 space-y-2">
                        <div className="space-y-1">
                          <label className="text-[10px] text-slate-400">OAuth Client ID</label>
                          <input
                            type="text"
                            value={customClientId}
                            onChange={(e) => setCustomClientId(e.target.value)}
                            placeholder="Custom Client ID"
                            className="w-full px-2.5 py-1 text-xs rounded bg-slate-900 border border-white/10 text-white font-mono"
                          />
                        </div>
                        <div className="space-y-1">
                          <label className="text-[10px] text-slate-400">OAuth Client Secret</label>
                          <input
                            type="password"
                            value={customClientSecret}
                            onChange={(e) => setCustomClientSecret(e.target.value)}
                            placeholder="Custom Client Secret"
                            className="w-full px-2.5 py-1 text-xs rounded bg-slate-900 border border-white/10 text-white font-mono"
                          />
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* TAB 5: AI Cloud Boost */}
            {activeTab === 'ai' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-semibold text-white">Optional Cloud AI Boost (Gemini)</h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Lumina automatically uses the 100% Free Local Visual & Windows Native OCR engine. Entering an optional key activates the Multi-Model Pool.
                  </p>
                </div>

                <div className="p-4 rounded-xl bg-slate-900/70 border border-white/10 space-y-3">
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-300">Gemini API Key</label>
                    <input
                      type="password"
                      value={geminiKey}
                      onChange={(e) => setGeminiKey(e.target.value)}
                      placeholder="AIzaSy..."
                      className="w-full px-3 py-2 text-xs rounded-xl bg-slate-800 border border-white/10 text-white font-mono"
                    />
                  </div>
                  <div className="flex justify-end">
                    <button
                      onClick={handleSaveGeminiKey}
                      className="px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium transition"
                    >
                      Save Key
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* TAB 6: Catalog Maintenance */}
            {activeTab === 'maintenance' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-sm font-semibold text-rose-400 flex items-center gap-1.5">
                    <Wrench className="w-4 h-4" />
                    <span>Catalog Maintenance & Repair</span>
                  </h4>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Strictly isolated administrative actions to repair metadata, re-verify screenshot classifications, and clean caches.
                  </p>
                </div>

                <div className="space-y-3">
                  {/* Action 1: Global Re-Analyze */}
                  <div className="p-4 rounded-xl bg-slate-900/60 border border-white/10 flex items-center justify-between">
                    <div>
                      <h5 className="text-xs font-semibold text-white">Global Re-Analyze</h5>
                      <p className="text-[11px] text-slate-400">Re-runs local visual scene tagging and Windows native OCR across all photos.</p>
                    </div>
                    <button
                      onClick={async () => {
                        await api.globalReAnalyze()
                        setMaintenanceMsg('Global Re-Analyze triggered in background.')
                      }}
                      className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-medium text-white transition border border-white/10"
                    >
                      Run Now
                    </button>
                  </div>

                  {/* Action 2: Re-Verify Screenshots */}
                  <div className="p-4 rounded-xl bg-slate-900/60 border border-white/10 flex items-center justify-between">
                    <div>
                      <h5 className="text-xs font-semibold text-white">Re-Verify Screenshots</h5>
                      <p className="text-[11px] text-slate-400">Re-evaluates the 100-pt screenshot scorer adhering strictly to Rule 1 & Rule 2.</p>
                    </div>
                    <button
                      onClick={async () => {
                        await api.reVerifyScreenshots()
                        setMaintenanceMsg('Screenshot re-verification completed.')
                        onRefreshStats()
                      }}
                      className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-medium text-white transition border border-white/10"
                    >
                      Re-Verify
                    </button>
                  </div>

                  {/* Action 3: Purge Source Thumbnails */}
                  <div className="p-4 rounded-xl bg-slate-900/60 border border-white/10 flex items-center justify-between">
                    <div>
                      <h5 className="text-xs font-semibold text-white">Purge Source Thumbnails</h5>
                      <p className="text-[11px] text-slate-400">Clears thumbnail cache and rebuilds thumbs.pack archive.</p>
                    </div>
                    <button
                      onClick={async () => {
                        await api.purgeThumbnails()
                        setMaintenanceMsg('Thumbnails purged.')
                      }}
                      className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-medium text-white transition border border-white/10"
                    >
                      Purge Cache
                    </button>
                  </div>

                  {/* Action 4: Factory Reset */}
                  <div className="p-4 rounded-xl bg-rose-950/20 border border-rose-500/20 flex items-center justify-between">
                    <div>
                      <h5 className="text-xs font-semibold text-rose-300 flex items-center gap-1.5">
                        <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />
                        <span>Factory Reset</span>
                      </h5>
                      <p className="text-[11px] text-rose-400/80">Wipes local database and re-initializes clean catalog.</p>
                    </div>
                    <button
                      onClick={async () => {
                        if (confirm('Are you sure you want to perform a factory reset? This will wipe the local Lumina index.')) {
                          await api.factoryReset()
                          onRefreshStats()
                          onClose()
                        }
                      }}
                      className="px-3 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-xs font-medium transition"
                    >
                      Reset Catalog
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
