import React, { useState, useEffect, useRef } from 'react'
import {
  Search,
  Sparkles,
  RefreshCw,
  Maximize2,
  Heart,
  Grid,
  MapPin,
  Smartphone,
  Calendar,
  Layers,
  FolderPlus,
  X
} from 'lucide-react'
import { InstantSearchChips } from '../types'
import { api } from '../services/api'

interface HeaderProps {
  currentTab: string
  searchQuery: string
  onSearchChange: (q: string) => void
  gridColumns: number
  onGridColumnsChange: (cols: number) => void
  favoritesOnly: boolean
  onToggleFavoritesOnly: () => void
  onSmartTagAll: () => void
  onStartSweep: () => void
  onToggleFullscreen: () => void
  onOpenSetupHub: () => void
  isScanning: boolean
}

export const Header: React.FC<HeaderProps> = ({
  currentTab,
  searchQuery,
  onSearchChange,
  gridColumns,
  onGridColumnsChange,
  favoritesOnly,
  onToggleFavoritesOnly,
  onSmartTagAll,
  onStartSweep,
  onToggleFullscreen,
  onOpenSetupHub,
  isScanning,
}) => {
  const [showChips, setShowChips] = useState(false)
  const [chips, setChips] = useState<InstantSearchChips>({
    topPlaces: ['Pune', 'Mumbai', 'Paris', 'New York'],
    topDevices: ['Apple_iPhone_15_Pro', 'Samsung_Galaxy_S23', 'Canon_EOS_6D'],
    years: ['2026', '2025', '2024'],
    categories: ['Documents & Receipts', 'Portraits & People', 'Videos', 'Live Photos', 'High Sharpness'],
  })

  const searchContainerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api.getInstantSearchChips().then((res) => {
      if (res) setChips(res)
    })
  }, [])

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (searchContainerRef.current && !searchContainerRef.current.contains(e.target as Node)) {
        setShowChips(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const getTabTitle = (tab: string) => {
    switch (tab.toLowerCase()) {
      case 'all':
        return 'All Photos'
      case 'ios':
        return 'Apple iOS'
      case 'android':
        return 'Android Devices'
      case 'camera':
        return 'Dedicated Cameras'
      case 'screenshots':
        return 'Screenshots & Captures'
      case 'people':
        return 'People & Faces'
      case 'cleanup':
        return 'Clean-Up Assistant'
      case 'google_photos':
        return 'Google Photos Cloud'
      case 'unknown':
        return 'Unclassified Media'
      default:
        return tab
    }
  }

  const handleSelectChip = (chipValue: string) => {
    onSearchChange(chipValue)
    setShowChips(false)
  }

  return (
    <header className="h-16 px-6 flex items-center justify-between border-b border-white/10 glass-panel z-20 select-none relative">
      {/* Title & Badge */}
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-bold text-white tracking-tight">
          {getTabTitle(currentTab)}
        </h2>
      </div>

      {/* Smart Search Bar with Instant Discovery Chips */}
      <div ref={searchContainerRef} className="flex-1 max-w-lg mx-6 relative">
        <div className="relative">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            type="text"
            value={searchQuery}
            onFocus={() => setShowChips(true)}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search date, place, device, OCR text, or tags..."
            className="w-full pl-10 pr-9 py-2 text-xs rounded-xl bg-slate-900/80 border border-white/15 text-white placeholder-slate-400 focus:outline-none focus:border-cyan-500/60 focus:ring-1 focus:ring-cyan-500/40 transition shadow-inner"
          />
          {searchQuery && (
            <button
              onClick={() => onSearchChange('')}
              className="absolute right-3 top-1/2 -translate-y-1/2 p-0.5 text-slate-400 hover:text-white"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>

        {/* Dropdown Instant Discovery Chips */}
        {showChips && (
          <div className="absolute top-full left-0 right-0 mt-2 p-4 rounded-2xl glass-card border border-white/20 shadow-2xl z-50 space-y-3">
            {/* Top Places */}
            {chips.topPlaces?.length > 0 && (
              <div>
                <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1 mb-1.5">
                  <MapPin className="w-3 h-3 text-rose-400" />
                  <span>Top Places</span>
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {chips.topPlaces.map((place) => (
                    <button
                      key={place}
                      onClick={() => handleSelectChip(place)}
                      className="px-2.5 py-1 rounded-lg bg-slate-800/80 hover:bg-rose-500/20 hover:text-rose-300 border border-white/10 text-xs text-slate-300 transition"
                    >
                      {place}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Top Devices */}
            {chips.topDevices?.length > 0 && (
              <div>
                <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1 mb-1.5">
                  <Smartphone className="w-3 h-3 text-emerald-400" />
                  <span>Devices</span>
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {chips.topDevices.map((dev) => (
                    <button
                      key={dev}
                      onClick={() => handleSelectChip(dev)}
                      className="px-2.5 py-1 rounded-lg bg-slate-800/80 hover:bg-emerald-500/20 hover:text-emerald-300 border border-white/10 text-xs text-slate-300 transition"
                    >
                      {dev.replace(/_/g, ' ')}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Years & Seasons */}
            {chips.years?.length > 0 && (
              <div>
                <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1 mb-1.5">
                  <Calendar className="w-3 h-3 text-sky-400" />
                  <span>Timeline</span>
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {chips.years.map((y) => (
                    <button
                      key={y}
                      onClick={() => handleSelectChip(y)}
                      className="px-2.5 py-1 rounded-lg bg-slate-800/80 hover:bg-sky-500/20 hover:text-sky-300 border border-white/10 text-xs text-slate-300 transition font-mono"
                    >
                      {y}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Smart Categories */}
            {chips.categories?.length > 0 && (
              <div>
                <span className="text-[10px] font-semibold uppercase tracking-wider text-slate-400 flex items-center gap-1 mb-1.5">
                  <Layers className="w-3 h-3 text-purple-400" />
                  <span>Smart Categories</span>
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {chips.categories.map((cat) => (
                    <button
                      key={cat}
                      onClick={() => handleSelectChip(cat)}
                      className="px-2.5 py-1 rounded-lg bg-slate-800/80 hover:bg-purple-500/20 hover:text-purple-300 border border-white/10 text-xs text-slate-300 transition"
                    >
                      {cat}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Control Tools */}
      <div className="flex items-center gap-2.5">
        {/* + Add Folders / Drives Setup Button */}
        <button
          onClick={onOpenSetupHub}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/15 text-slate-300 hover:text-white hover:bg-slate-800 text-xs font-medium transition"
          title="Open Setup & Drive Discovery Hub"
        >
          <FolderPlus className="w-3.5 h-3.5 text-sky-400" />
          <span>+ Sources / Drives</span>
        </button>

        {/* Real-time Grid Columns Slider (3 to 10 columns) */}
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/15">
          <Grid className="w-3.5 h-3.5 text-slate-400" />
          <input
            type="range"
            min={3}
            max={10}
            value={gridColumns}
            onChange={(e) => onGridColumnsChange(Number(e.target.value))}
            className="w-20 h-1 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-sky-400"
            title={`Grid size: ${gridColumns} columns`}
          />
          <span className="text-xs text-slate-400 font-mono w-4 text-center">{gridColumns}</span>
        </div>

        {/* Favorites Filter Toggle */}
        <button
          onClick={onToggleFavoritesOnly}
          className={`p-2 rounded-xl border transition ${
            favoritesOnly
              ? 'bg-rose-500/20 border-rose-500/40 text-rose-400'
              : 'bg-slate-900/80 border-white/15 text-slate-400 hover:text-white'
          }`}
          title="Filter Favorites"
        >
          <Heart className={`w-4 h-4 ${favoritesOnly ? 'fill-rose-400' : ''}`} />
        </button>

        {/* 100% Free Local AI & Native OCR Trigger */}
        <button
          onClick={onSmartTagAll}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-gradient-to-r from-sky-500/20 to-indigo-500/20 border border-sky-500/30 text-sky-300 hover:text-white hover:bg-sky-500/30 text-xs font-medium transition shadow-sm"
          title="Run 100% Free Local Visual Scene Analysis & Native Windows OCR"
        >
          <Sparkles className="w-3.5 h-3.5 text-sky-400" />
          <span>Smart Tag & OCR</span>
        </button>

        {/* Background Sweep & Organize Trigger */}
        <button
          onClick={onStartSweep}
          disabled={isScanning}
          className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-sky-600 hover:bg-sky-500 text-white text-xs font-medium transition shadow-lg shadow-sky-600/20 disabled:opacity-50"
          title="Sweep Watched Folders and Organize Photos"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isScanning ? 'animate-spin' : ''}`} />
          <span>{isScanning ? 'Scanning...' : 'Sweep & Organize'}</span>
        </button>

        {/* Native Wails Fullscreen Toggle */}
        <button
          onClick={onToggleFullscreen}
          className="p-2 rounded-xl bg-slate-900/80 border border-white/15 text-slate-400 hover:text-white transition"
          title="Toggle Fullscreen (F)"
        >
          <Maximize2 className="w-4 h-4" />
        </button>
      </div>
    </header>
  )
}
