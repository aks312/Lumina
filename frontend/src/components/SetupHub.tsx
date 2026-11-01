import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
  HardDrive,
  FolderPlus,
  Folder,
  Eye,
  EyeOff,
  Sparkles,
  Zap,
  ArrowRight,
  Trash2,
  CheckCircle2,
  ShieldCheck,
  Smartphone,
  Apple,
  Camera,
  FolderSync,
  Layers,
  UploadCloud
} from 'lucide-react'
import { DetectedDriveInfo, CommonFolderInfo, WatchedFolder } from '../types'
import { api } from '../services/api'

interface SetupHubProps {
  onComplete: () => void
  onCancel?: () => void
  isFirstRun?: boolean
}

export const SetupHub: React.FC<SetupHubProps> = ({
  onComplete,
  onCancel,
  isFirstRun = false,
}) => {
  const [drives, setDrives] = useState<DetectedDriveInfo[]>([])
  const [commonFolders, setCommonFolders] = useState<CommonFolderInfo[]>([])
  const [destinationPath, setDestinationPath] = useState<string>('')
  const [enableNvmeMirror, setEnableNvmeMirror] = useState<boolean>(true)
  const [fastDrive, setFastDrive] = useState<string>('C:\\')

  // Watched folders in setup
  const [folders, setFolders] = useState<WatchedFolder[]>([])
  const [customRouteTarget, setCustomRouteTarget] = useState<{ [id: number]: string }>({})
  const [isStarting, setIsStarting] = useState(false)

  useEffect(() => {
    loadSetupData()
  }, [])

  const loadSetupData = async () => {
    const data = await api.getSetupHubData()
    setDrives(data.drives ?? [])
    setCommonFolders(data.commonFolders ?? [])
    setFastDrive(data.fastDrive ?? 'C:\\')

    const existingDest = await api.getSetting('primary_destination', '')
    if (existingDest) {
      setDestinationPath(existingDest)
    } else {
      setDestinationPath(data.suggestedDest || 'C:\\LuminaPhotos')
    }

    const existingFolders = await api.getWatchedFolders()
    setFolders(existingFolders ?? [])
  }

  const handleAddFolder = async () => {
    const dir = await api.selectDirectoryDialog()
    if (dir) {
      await api.addWatchedFolder(dir, '', true)
      const updated = await api.getWatchedFolders()
      setFolders(updated ?? [])
    }
  }

  const handleAddDriveAsSource = async (driveLetter: string) => {
    // If not already added
    const alreadyAdded = folders.some((f) => f.path.toLowerCase() === driveLetter.toLowerCase())
    if (!alreadyAdded) {
      await api.addWatchedFolder(driveLetter, '', true)
      const updated = await api.getWatchedFolders()
      setFolders(updated ?? [])
    }
  }

  const handleAddCommonFolder = async (folderPath: string) => {
    const alreadyAdded = folders.some((f) => f.path.toLowerCase() === folderPath.toLowerCase())
    if (!alreadyAdded) {
      await api.addWatchedFolder(folderPath, '', true)
      const updated = await api.getWatchedFolders()
      setFolders(updated ?? [])
    }
  }

  const handleToggleShowInAll = async (folder: WatchedFolder) => {
    await api.addWatchedFolder(folder.path, folder.route_override, !folder.show_in_all_photos)
    const updated = await api.getWatchedFolders()
    setFolders(updated ?? [])
  }

  const handleRouteChange = async (folder: WatchedFolder, newRoute: string) => {
    await api.addWatchedFolder(folder.path, newRoute, folder.show_in_all_photos)
    const updated = await api.getWatchedFolders()
    setFolders(updated ?? [])
  }

  const handleRemoveFolder = async (id: number) => {
    await api.removeWatchedFolder(id)
    const updated = await api.getWatchedFolders()
    setFolders(updated ?? [])
  }

  const handleBrowseDestination = async () => {
    const dir = await api.selectDirectoryDialog()
    if (dir) {
      setDestinationPath(dir)
    }
  }

  const handleStartOrganizing = async () => {
    if (!destinationPath) return
    setIsStarting(true)

    // Save Primary Destination
    await api.saveSetting('primary_destination', destinationPath)

    // If NVMe Mirror enabled, set secondary mirror to fast drive
    if (enableNvmeMirror && fastDrive) {
      const mirrorPath = `${fastDrive}\\LuminaFastMirror`
      await api.saveSetting('secondary_mirror', mirrorPath)
    } else {
      await api.saveSetting('secondary_mirror', '')
    }

    // Start background catalog sweep and organization
    await api.startScan(true)
    setIsStarting(false)
    onComplete()
  }

  return (
    <div className="flex-1 h-full overflow-y-auto p-8 select-none">
      <div className="max-w-5xl mx-auto space-y-8">
        {/* Welcome Header */}
        <div className="text-center space-y-2">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-sky-500/15 border border-sky-500/30 text-sky-300 text-xs font-medium">
            <Sparkles className="w-3.5 h-3.5" />
            <span>Smart Guided Onboarding</span>
          </div>
          <h2 className="text-3xl font-extrabold text-white tracking-tight">
            {isFirstRun ? 'Welcome to Lumina Photos' : 'Library Folders & Organization Hub'}
          </h2>
          <p className="text-sm text-slate-400 max-w-xl mx-auto">
            Set up your photo sources, configure your clean organized destination, and let Lumina's 5-tier zero-key AI engine catalog your media.
          </p>
        </div>

        {/* STEP 1: Source Discovery */}
        <div className="glass-card rounded-2xl p-6 border border-white/15 shadow-2xl space-y-5">
          <div className="flex items-center justify-between border-b border-white/10 pb-3">
            <div className="flex items-center gap-2.5">
              <span className="w-6 h-6 rounded-full bg-sky-500 text-slate-950 font-bold text-xs flex items-center justify-center">
                1
              </span>
              <h3 className="text-base font-semibold text-white">Where are your photos?</h3>
            </div>
            <button
              onClick={handleAddFolder}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium transition shadow-lg shadow-sky-600/20"
            >
              <FolderPlus className="w-4 h-4" />
              <span>+ Add Folder or Drive</span>
            </button>
          </div>

          {/* 1-Click Detected Drives Cards */}
          <div>
            <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider block mb-2.5">
              Detected Storage Drives & Connected Devices
            </label>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              {(drives ?? []).map((drive) => {
                const usedPercent = Math.min(100, Math.max(0, ((drive.totalGB - drive.freeGB) / drive.totalGB) * 100))
                const isAdded = folders.some((f) => f.path.toLowerCase() === drive.driveLetter.toLowerCase())

                return (
                  <div
                    key={drive.driveLetter}
                    onClick={() => handleAddDriveAsSource(drive.driveLetter)}
                    className={`p-3.5 rounded-xl border transition cursor-pointer relative overflow-hidden ${
                      isAdded
                        ? 'bg-sky-950/40 border-sky-500/50 ring-1 ring-sky-500/30'
                        : 'bg-slate-900/60 border-white/10 hover:border-sky-500/40 hover:bg-slate-900/90'
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2.5">
                        <HardDrive className={`w-5 h-5 ${drive.driveType.includes('Removable') ? 'text-amber-400' : 'text-sky-400'}`} />
                        <div>
                          <p className="text-xs font-bold text-white flex items-center gap-1.5">
                            <span>{drive.driveLetter}</span>
                            <span className="text-[11px] font-normal text-slate-400 truncate max-w-[120px]">
                              {drive.volumeLabel}
                            </span>
                          </p>
                          <p className="text-[10px] text-slate-500">{drive.driveType}</p>
                        </div>
                      </div>
                      {drive.recommended && (
                        <span className="text-[9px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 font-medium">
                          Recommended
                        </span>
                      )}
                    </div>

                    {/* Capacity Bar */}
                    <div className="mt-3 space-y-1">
                      <div className="w-full bg-slate-800 rounded-full h-1.5 overflow-hidden">
                        <div
                          className="bg-sky-400 h-full rounded-full"
                          style={{ width: `${usedPercent}%` }}
                        />
                      </div>
                      <div className="flex justify-between text-[10px] text-slate-400 font-mono">
                        <span>{drive.freeGB} GB Free</span>
                        <span>{drive.totalGB} GB</span>
                      </div>
                    </div>

                    {isAdded && (
                      <div className="mt-2 flex items-center gap-1 text-[11px] text-sky-400 font-medium">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        <span>Source Added</span>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>

          {/* Common Media Folders Quick Adds */}
          <div>
            <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider block mb-2">
              Common Media Locations
            </label>
            <div className="flex flex-wrap gap-2">
              {(commonFolders ?? []).map((cf) => {
                const isAdded = folders.some((f) => f.path.toLowerCase() === cf.path.toLowerCase())
                return (
                  <button
                    key={cf.path}
                    onClick={() => handleAddCommonFolder(cf.path)}
                    className={`px-3 py-1.5 rounded-xl border text-xs flex items-center gap-2 transition ${
                      isAdded
                        ? 'bg-sky-500/20 border-sky-500/40 text-sky-300'
                        : 'bg-slate-900/60 border-white/10 text-slate-300 hover:text-white hover:bg-slate-800'
                    }`}
                  >
                    <Folder className="w-3.5 h-3.5 text-indigo-400" />
                    <span>{cf.name}</span>
                    {isAdded && <CheckCircle2 className="w-3 h-3 text-sky-400" />}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Active Source Folders with In-Place Route & Show Toggles */}
          {folders.length > 0 && (
            <div className="pt-3 border-t border-white/5 space-y-2">
              <label className="text-xs font-semibold text-slate-300 block">
                Configured Source Folders ({folders.length})
              </label>
              <div className="space-y-2">
                {folders.map((f) => (
                  <div
                    key={f.id}
                    className="p-3 rounded-xl bg-slate-900/70 border border-white/10 flex flex-col md:flex-row md:items-center justify-between gap-3 text-xs"
                  >
                    <div className="truncate max-w-sm">
                      <p className="font-mono text-slate-200 truncate">{f.path}</p>
                      <span className="text-[10px] text-slate-500">Recursive scan enabled</span>
                    </div>

                    <div className="flex items-center gap-3">
                      {/* Show in All Photos Toggle */}
                      <button
                        onClick={() => handleToggleShowInAll(f)}
                        className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs transition ${
                          f.show_in_all_photos
                            ? 'bg-emerald-500/15 border-emerald-500/30 text-emerald-300'
                            : 'bg-slate-800 border-white/10 text-slate-400'
                        }`}
                        title="Include photos from this folder in the main All Photos gallery"
                      >
                        {f.show_in_all_photos ? <Eye className="w-3.5 h-3.5" /> : <EyeOff className="w-3.5 h-3.5" />}
                        <span>{f.show_in_all_photos ? 'In All Photos' : 'Isolated'}</span>
                      </button>

                      {/* Route To Dropdown */}
                      <div className="flex items-center gap-1.5 bg-slate-800 border border-white/10 rounded-lg px-2.5 py-1">
                        <span className="text-slate-400 text-[11px]">Route:</span>
                        <select
                          value={f.route_override || ''}
                          onChange={(e) => handleRouteChange(f, e.target.value)}
                          className="bg-transparent text-sky-300 font-medium focus:outline-none text-xs cursor-pointer"
                        >
                          <option value="" className="bg-slate-900 text-slate-200">
                            Auto (iOS / Android / Camera)
                          </option>
                          <option value="Amruta" className="bg-slate-900 text-slate-200">
                            Amruta (Priority 2)
                          </option>
                          <option value="Family" className="bg-slate-900 text-slate-200">
                            Family
                          </option>
                          <option value="Work" className="bg-slate-900 text-slate-200">
                            Work
                          </option>
                        </select>
                      </div>

                      {/* Remove Button */}
                      <button
                        onClick={() => handleRemoveFolder(f.id)}
                        className="p-1 rounded-lg text-slate-400 hover:text-rose-400 transition"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* STEP 2: Destination & NVMe Speed Boost */}
        <div className="glass-card rounded-2xl p-6 border border-white/15 shadow-2xl space-y-5">
          <div className="flex items-center gap-2.5 border-b border-white/10 pb-3">
            <span className="w-6 h-6 rounded-full bg-indigo-500 text-slate-950 font-bold text-xs flex items-center justify-center">
              2
            </span>
            <h3 className="text-base font-semibold text-white">Where should Lumina organize your clean library?</h3>
          </div>

          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-300">Clean Destination Directory</label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={destinationPath}
                  readOnly
                  className="flex-1 px-3.5 py-2 text-xs rounded-xl bg-slate-900 border border-white/10 text-white font-mono"
                />
                <button
                  onClick={handleBrowseDestination}
                  className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-medium text-white transition border border-white/10"
                >
                  Change...
                </button>
              </div>
            </div>

            {/* 1-Click NVMe Speed Mirror Toggle */}
            <div className="p-4 rounded-xl bg-slate-900/60 border border-cyan-500/20 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-cyan-500/10 text-cyan-400 flex items-center justify-center">
                  <Zap className="w-5 h-5" />
                </div>
                <div>
                  <h5 className="text-xs font-semibold text-white">Enable Dual-Drive NVMe Speed Mirror</h5>
                  <p className="text-[11px] text-slate-400">
                    Automatically benchmarks fast SSD ({fastDrive}) and serves photo viewer at 120fps from fastest drive.
                  </p>
                </div>
              </div>
              <input
                type="checkbox"
                checked={enableNvmeMirror}
                onChange={(e) => setEnableNvmeMirror(e.target.checked)}
                className="w-4 h-4 rounded bg-slate-800 border-white/20 text-cyan-500 focus:ring-0 cursor-pointer"
              />
            </div>
          </div>
        </div>

        {/* STEP 3: Live Interactive Architecture Preview & CTA */}
        <div className="glass-card rounded-2xl p-6 border border-white/15 shadow-2xl space-y-5">
          <div className="flex items-center justify-between border-b border-white/10 pb-3">
            <div className="flex items-center gap-2.5">
              <span className="w-6 h-6 rounded-full bg-emerald-500 text-slate-950 font-bold text-xs flex items-center justify-center">
                3
              </span>
              <h3 className="text-base font-semibold text-white">Live Organization Preview</h3>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-emerald-400">
              <ShieldCheck className="w-4 h-4" />
              <span>Strict 1-Copy Deduplication Guaranteed</span>
            </div>
          </div>

          {/* Visual Architecture Tree Diagram */}
          <div className="p-4 rounded-xl bg-slate-950/80 border border-white/5 space-y-2.5 font-mono text-xs">
            <div className="flex items-center gap-2 text-sky-300">
              <Apple className="w-3.5 h-3.5 text-sky-400" />
              <span>📁 iOS / 2025 / 2025-07 / 20250709_195013_iPhone_15_Pune_60f974.jpg</span>
            </div>
            <div className="flex items-center gap-2 text-emerald-300">
              <Smartphone className="w-3.5 h-3.5 text-emerald-400" />
              <span>📁 Android / 2025 / 2025-07 / 20250709_142000_Samsung_Galaxy_S23_Mumbai_a1b2c3.jpg</span>
            </div>
            <div className="flex items-center gap-2 text-amber-300">
              <Camera className="w-3.5 h-3.5 text-amber-400" />
              <span>📁 Camera / 2024 / 2024-12 / 20241215_083011_Canon_EOS_6D_Mark_II_e87f72.jpg</span>
            </div>
            <div className="flex items-center gap-2 text-indigo-300">
              <Folder className="w-3.5 h-3.5 text-indigo-400" />
              <span>📁 Amruta (Private Folder) / 2025 / ...</span>
            </div>
            <div className="flex items-center gap-2 text-rose-300">
              <span>📱 Screenshots (Strictly Isolated from Main Gallery)</span>
            </div>
          </div>

          {/* Start CTA */}
          <div className="pt-2 flex items-center justify-between">
            {onCancel ? (
              <button
                onClick={onCancel}
                className="px-4 py-2 rounded-xl text-xs font-medium text-slate-400 hover:text-white transition"
              >
                Back to Gallery
              </button>
            ) : <div />}

            <button
              onClick={handleStartOrganizing}
              disabled={isStarting || folders.length === 0}
              className="px-8 py-3 rounded-xl bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-semibold text-sm shadow-xl shadow-sky-500/25 transition flex items-center gap-2.5 disabled:opacity-40"
            >
              <span>{isStarting ? 'Starting Sweep...' : '🚀 Start Organizing Library'}</span>
              <ArrowRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
