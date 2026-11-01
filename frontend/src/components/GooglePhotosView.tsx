import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
  Cloud,
  UploadCloud,
  RefreshCw,
  ShieldCheck,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
  Layers,
  Sparkles,
  HardDrive,
} from 'lucide-react'
import { api } from '../services/api'
import { CatalogStats } from '../types'

interface GooglePhotosViewProps {
  stats: CatalogStats
}

export const GooglePhotosView: React.FC<GooglePhotosViewProps> = ({ stats }) => {
  const [isConnected, setIsConnected] = useState<boolean>(false)
  const [isSigningIn, setIsSigningIn] = useState<boolean>(false)
  const [isSyncing, setIsSyncing] = useState<boolean>(false)
  const [syncProgress, setSyncProgress] = useState<{ synced: number; total: number } | null>(null)
  const [authError, setAuthError] = useState<string>('')
  const [syncScope, setSyncScope] = useState<'all' | 'favorites' | 'clean'>('clean')

  useEffect(() => {
    // Check initial connection status
    api.isGoogleConnected().then((connected) => {
      setIsConnected(connected)
    })

    // Listen to Google connected event
    const unsubConnected = api.onEvent('gphotos:connected', (status: boolean) => {
      setIsConnected(status)
      setIsSigningIn(false)
    })

    // Listen to upload progress events
    const unsubProgress = api.onEvent('gphotos:upload_progress', (data: { synced: number; total: number }) => {
      setSyncProgress(data)
      setIsSyncing(true)
      if (data.synced >= data.total && data.total > 0) {
        setTimeout(() => {
          setIsSyncing(false)
        }, 2000)
      }
    })

    return () => {
      unsubConnected()
      unsubProgress()
    }
  }, [])

  const handleSignIn = async () => {
    try {
      setIsSigningIn(true)
      setAuthError('')
      const authUrl = await api.startGoogleSignIn()
      if (!authUrl) {
        setAuthError('Could not launch OAuth browser sign-in. Check your network or credentials.')
        setIsSigningIn(false)
      }
    } catch (err: any) {
      setAuthError(err?.message || 'Authentication error')
      setIsSigningIn(false)
    }
  }

  const handleStartSync = async () => {
    try {
      setIsSyncing(true)
      const count = await api.syncToGooglePhotos()
      setSyncProgress({ synced: 0, total: count })
    } catch (err: any) {
      setAuthError(err?.message || 'Sync failed')
      setIsSyncing(false)
    }
  }

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
  }

  const cleanPhotosCount = Math.max(0, stats.total_photos - stats.screenshots)

  return (
    <div className="flex-1 flex flex-col h-full overflow-y-auto px-8 py-8 select-none">
      <div className="max-w-4xl mx-auto w-full space-y-8">
        {/* Title Header */}
        <div className="flex items-center justify-between border-b border-white/10 pb-6">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-rose-500 via-amber-500 to-sky-500 p-0.5 shadow-xl shadow-sky-500/10">
              <div className="w-full h-full bg-slate-950 rounded-2xl flex items-center justify-center">
                <Cloud className="w-6 h-6 text-sky-400" />
              </div>
            </div>
            <div>
              <h2 className="text-xl font-bold text-white tracking-tight flex items-center gap-2">
                Google Photos Cloud Sync
                {isConnected && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 font-semibold flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                    Connected
                  </span>
                )}
              </h2>
              <p className="text-xs text-slate-400 mt-0.5">
                Seamlessly back up your curated library and custom folders to Google Photos with zero quality loss.
              </p>
            </div>
          </div>

          {isConnected ? (
            <button
              onClick={handleStartSync}
              disabled={isSyncing}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-medium text-xs shadow-lg shadow-sky-500/25 transition disabled:opacity-50"
            >
              {isSyncing ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  <span>Syncing...</span>
                </>
              ) : (
                <>
                  <UploadCloud className="w-4 h-4" />
                  <span>Start Cloud Backup</span>
                </>
              )}
            </button>
          ) : (
            <button
              onClick={handleSignIn}
              disabled={isSigningIn}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-white hover:bg-slate-100 text-slate-950 font-semibold text-xs shadow-xl transition disabled:opacity-60"
            >
              {isSigningIn ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin text-slate-900" />
                  <span>Signing In...</span>
                </>
              ) : (
                <>
                  <ExternalLink className="w-4 h-4 text-slate-900" />
                  <span>1-Click Google Sign-In</span>
                </>
              )}
            </button>
          )}
        </div>

        {/* Sync Progress Bar */}
        {syncProgress && isSyncing && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className="p-5 rounded-2xl bg-sky-950/40 border border-sky-500/30 backdrop-blur-xl shadow-xl space-y-3"
          >
            <div className="flex items-center justify-between text-xs">
              <span className="font-semibold text-sky-200 flex items-center gap-2">
                <UploadCloud className="w-4 h-4 text-sky-400 animate-pulse" />
                Uploading Batch to Google Photos...
              </span>
              <span className="font-mono text-sky-300 font-bold">
                {syncProgress.synced} / {syncProgress.total} items (
                {syncProgress.total > 0
                  ? Math.round((syncProgress.synced / syncProgress.total) * 100)
                  : 0}
                %)
              </span>
            </div>
            <div className="w-full bg-slate-900/80 rounded-full h-2 overflow-hidden border border-white/5">
              <div
                className="bg-gradient-to-r from-sky-400 to-indigo-400 h-full transition-all duration-300 rounded-full"
                style={{
                  width: `${
                    syncProgress.total > 0
                      ? Math.min(100, (syncProgress.synced / syncProgress.total) * 100)
                      : 0
                  }%`,
                }}
              />
            </div>
          </motion.div>
        )}

        {authError && (
          <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/25 flex items-center gap-3 text-xs text-rose-300">
            <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{authError}</span>
          </div>
        )}

        {/* Summary Stats Cards */}
        <div className="grid grid-cols-3 gap-4">
          <div className="p-5 rounded-2xl bg-slate-900/60 border border-white/10 backdrop-blur-xl shadow-lg">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Curated Photos</span>
              <Sparkles className="w-4 h-4 text-sky-400" />
            </div>
            <p className="text-2xl font-bold text-white font-mono mt-2">{cleanPhotosCount}</p>
            <p className="text-[11px] text-slate-500 mt-1">Screenshots filtered out</p>
          </div>

          <div className="p-5 rounded-2xl bg-slate-900/60 border border-white/10 backdrop-blur-xl shadow-lg">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Total Videos</span>
              <Layers className="w-4 h-4 text-amber-400" />
            </div>
            <p className="text-2xl font-bold text-white font-mono mt-2">{stats.total_videos}</p>
            <p className="text-[11px] text-slate-500 mt-1">High definition video clips</p>
          </div>

          <div className="p-5 rounded-2xl bg-slate-900/60 border border-white/10 backdrop-blur-xl shadow-lg">
            <div className="flex items-center justify-between text-slate-400 text-xs">
              <span>Library Size</span>
              <HardDrive className="w-4 h-4 text-emerald-400" />
            </div>
            <p className="text-2xl font-bold text-white font-mono mt-2">{formatBytes(stats.total_bytes)}</p>
            <p className="text-[11px] text-slate-500 mt-1">Ready for cloud backup</p>
          </div>
        </div>

        {/* Backup Scope Selector */}
        <div className="p-6 rounded-2xl bg-slate-900/40 border border-white/10 backdrop-blur-xl shadow-xl space-y-4">
          <h3 className="text-sm font-semibold text-white tracking-wide">Backup Scope</h3>
          <div className="grid grid-cols-3 gap-3">
            <button
              onClick={() => setSyncScope('clean')}
              className={`p-4 rounded-xl border text-left transition ${
                syncScope === 'clean'
                  ? 'bg-sky-500/20 border-sky-400/50 shadow-lg shadow-sky-500/10'
                  : 'bg-slate-900/50 border-white/5 hover:border-white/20'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-white">Cleaned Library</span>
                {syncScope === 'clean' && <CheckCircle2 className="w-4 h-4 text-sky-400" />}
              </div>
              <p className="text-[11px] text-slate-400 mt-1">
                Uploads real photos & videos; skips screenshots and temporary captures.
              </p>
            </button>

            <button
              onClick={() => setSyncScope('favorites')}
              className={`p-4 rounded-xl border text-left transition ${
                syncScope === 'favorites'
                  ? 'bg-sky-500/20 border-sky-400/50 shadow-lg shadow-sky-500/10'
                  : 'bg-slate-900/50 border-white/5 hover:border-white/20'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-white">Favorites Only</span>
                {syncScope === 'favorites' && <CheckCircle2 className="w-4 h-4 text-sky-400" />}
              </div>
              <p className="text-[11px] text-slate-400 mt-1">
                Uploads only photos marked with ♥ ({stats.favorites} items).
              </p>
            </button>

            <button
              onClick={() => setSyncScope('all')}
              className={`p-4 rounded-xl border text-left transition ${
                syncScope === 'all'
                  ? 'bg-sky-500/20 border-sky-400/50 shadow-lg shadow-sky-500/10'
                  : 'bg-slate-900/50 border-white/5 hover:border-white/20'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-white">All Library Media</span>
                {syncScope === 'all' && <CheckCircle2 className="w-4 h-4 text-sky-400" />}
              </div>
              <p className="text-[11px] text-slate-400 mt-1">
                Uploads every single asset indexed in Lumina without exclusions.
              </p>
            </button>
          </div>
        </div>

        {/* Security & Privacy Commitment Card */}
        <div className="p-5 rounded-2xl bg-slate-900/30 border border-white/5 flex items-start gap-4 text-slate-400 text-xs">
          <ShieldCheck className="w-5 h-5 text-sky-400 shrink-0 mt-0.5" />
          <div className="space-y-1">
            <span className="font-semibold text-slate-200">Zero-Telemetry & Direct Encrypted Upload</span>
            <p className="text-slate-400 leading-relaxed text-[11px]">
              Lumina communicates directly with Google's official REST APIs from your desktop via TLS 1.3 encryption.
              No intermediate servers, no telemetry, and your Google OAuth refresh tokens remain securely encrypted on your local Windows PC.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
