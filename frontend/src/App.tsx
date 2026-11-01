import React, { useState, useEffect, useCallback } from 'react'
import { Sidebar } from './components/Sidebar'
import { Header } from './components/Header'
import { PhotoGrid } from './components/PhotoGrid'
import { TimelineScrubber } from './components/TimelineScrubber'
import { FloatingSelectionPill } from './components/FloatingSelectionPill'
import { Lightbox } from './components/Lightbox'
import { SettingsModal } from './components/SettingsModal'
import { CleanUpAssistant } from './components/CleanUpAssistant'
import { SetupHub } from './components/SetupHub'
import { GooglePhotosView } from './components/GooglePhotosView'
import { PeopleFacesView } from './components/PeopleFacesView'
import { Calendar, X } from 'lucide-react'
import { CatalogStats, PhotoRecord, TimelineBucket, WatchedFolder, IndexerProgress } from './types'
import { api } from './services/api'

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<string>('all')
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [gridColumns, setGridColumns] = useState<number>(5)
  const [favoritesOnly, setFavoritesOnly] = useState<boolean>(false)
  const [selectedYear, setSelectedYear] = useState<number>(0)
  const [selectedMonth, setSelectedMonth] = useState<number>(0)

  const [photos, setPhotos] = useState<PhotoRecord[]>([])
  const [stats, setStats] = useState<CatalogStats>({
    total_photos: 0,
    total_videos: 0,
    total_bytes: 0,
    ios_count: 0,
    android_count: 0,
    camera_count: 0,
    screenshots: 0,
    unknown_count: 0,
    favorites: 0,
  })
  const [timelineBuckets, setTimelineBuckets] = useState<TimelineBucket[]>([])
  const [watchedFolders, setWatchedFolders] = useState<WatchedFolder[]>([])

  const [selectedIDs, setSelectedIDs] = useState<Set<number>>(new Set())
  const [activeLightboxPhoto, setActiveLightboxPhoto] = useState<PhotoRecord | null>(null)
  const [isSettingsOpen, setIsSettingsOpen] = useState<boolean>(false)
  const [isSetupHubOpen, setIsSetupHubOpen] = useState<boolean>(false)
  const [isScanning, setIsScanning] = useState<boolean>(false)
  const [scanProgress, setScanProgress] = useState<IndexerProgress | null>(null)

  const monthNames = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
  ]

  // Load photos based on active filters
  const loadPhotos = useCallback(async () => {
    const list = await api.getPhotos(
      currentTab,
      selectedYear,
      selectedMonth,
      '',
      favoritesOnly,
      0,
      searchQuery,
      0,
      1000
    )
    setPhotos(list ?? [])

    const st = await api.getStats()
    if (st) setStats(st)

    const buckets = await api.getTimelineSummary(currentTab)
    setTimelineBuckets(buckets ?? [])
  }, [currentTab, selectedYear, selectedMonth, favoritesOnly, searchQuery])

  // Initial load & watched folders
  useEffect(() => {
    loadPhotos()
    api.getWatchedFolders().then((wf) => {
      const list = wf ?? []
      setWatchedFolders(list)
      // If zero watched folders and zero photos, show SetupHub on first run
      if (list.length === 0) {
        setIsSetupHubOpen(true)
      }
    })

    // Listen to background indexer progress events
    const unsubProgress = api.onEvent('indexer:progress', (p: IndexerProgress) => {
      setScanProgress(p)
      setIsScanning(p.status === 'scanning' || p.status === 'cataloging' || p.status === 'organizing')
      if (p.status === 'completed') {
        loadPhotos()
        setTimeout(() => setScanProgress(null), 3000)
      }
    })

    const unsubAi = api.onEvent('ai:analysis_complete', () => {
      loadPhotos()
    })

    return () => {
      unsubProgress()
      unsubAi()
    }
  }, [loadPhotos])

  // Select Tab
  const handleSelectTab = (tab: string) => {
    setCurrentTab(tab)
    setSelectedIDs(new Set())
    setIsSetupHubOpen(false)
    setSelectedYear(0)
    setSelectedMonth(0)
  }

  // Toggle selection
  const handleToggleSelect = (id: number, e: React.MouseEvent) => {
    setSelectedIDs((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  // Toggle favorite
  const handleToggleFavorite = async (photo: PhotoRecord, e: React.MouseEvent) => {
    const newFav = !photo.is_favorite
    setPhotos((prev) =>
      (prev ?? []).map((p) => (p.id === photo.id ? { ...p, is_favorite: newFav } : p))
    )
    await api.updatePhoto(
      photo.id,
      photo.route_folder,
      photo.platform,
      photo.rotation,
      photo.rating,
      newFav,
      photo.show_in_all
    )
    const st = await api.getStats()
    if (st) setStats(st)
  }

  // Batch Move to Folder
  const handleMoveToFolder = async (folderName: string) => {
    const ids = Array.from(selectedIDs)
    await api.batchUpdatePhotos(ids, folderName, null, null)
    setSelectedIDs(new Set())
    loadPhotos()
    const wf = await api.getWatchedFolders()
    setWatchedFolders(wf ?? [])
  }

  // Batch Show/Hide in All Photos
  const handleToggleShowInAll = async (show: boolean) => {
    const ids = Array.from(selectedIDs)
    await api.batchUpdatePhotos(ids, '', null, show)
    setSelectedIDs(new Set())
    loadPhotos()
  }

  // Smart Tag & OCR
  const handleSmartTagAll = async () => {
    const ids = (photos ?? []).slice(0, 100).map((p) => p.id)
    await api.triggerSmartTagAndOCR(ids)
  }

  const handleSmartTagSelected = async () => {
    const ids = Array.from(selectedIDs)
    await api.triggerSmartTagAndOCR(ids)
    setSelectedIDs(new Set())
  }

  // Sweep Watched Folders
  const handleStartSweep = async () => {
    setIsScanning(true)
    await api.startScan(true)
  }

  // Jump to Year-Month from Timeline Scrubber (Filters by Year & Month or scrolls smoothly)
  const handleSelectMonth = (year: number, month: number) => {
    if (selectedYear === year && selectedMonth === month) {
      // Toggle off to show full timeline
      setSelectedYear(0)
      setSelectedMonth(0)
    } else {
      setSelectedYear(year)
      setSelectedMonth(month)
      // If the section is already mounted in DOM, scroll smoothly to it
      const element = document.getElementById(`timeline-${year}-${String(month).padStart(2, '0')}`)
      if (element) {
        element.scrollIntoView({ behavior: 'smooth' })
      }
    }
  }

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-slate-950/75 text-slate-100 select-none opacity-100">
      {/* Glass Sidebar */}
      <Sidebar
        currentTab={isSetupHubOpen ? 'folders_hub' : currentTab}
        onSelectTab={handleSelectTab}
        stats={stats}
        watchedFolders={watchedFolders ?? []}
        onOpenSettings={() => setIsSettingsOpen(true)}
        onAddCustomFolder={() => setIsSettingsOpen(true)}
        onOpenSetupHub={() => setIsSetupHubOpen(true)}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col h-full overflow-hidden relative">
        {/* Top Header */}
        <Header
          currentTab={isSetupHubOpen ? 'Folders & Drives Setup Hub' : currentTab}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          gridColumns={gridColumns}
          onGridColumnsChange={setGridColumns}
          favoritesOnly={favoritesOnly}
          onToggleFavoritesOnly={() => setFavoritesOnly((prev) => !prev)}
          onSmartTagAll={handleSmartTagAll}
          onStartSweep={handleStartSweep}
          onToggleFullscreen={() => api.toggleFullscreen()}
          onOpenSetupHub={() => setIsSetupHubOpen(true)}
          isScanning={isScanning}
        />

        {/* Scan / Cataloging Progress Banner */}
        {scanProgress && (
          <div className="px-6 py-2 bg-sky-500/10 border-b border-sky-500/20 flex items-center justify-between text-xs text-sky-300">
            <div className="flex items-center gap-2">
              <span className="animate-spin text-sky-400">⚡</span>
              <span>
                {scanProgress.status === 'completed'
                  ? 'Sweep Completed Successfully!'
                  : `Scanning: ${scanProgress.current_file} (${scanProgress.scanned_files} / ${scanProgress.total_files})`}
              </span>
            </div>
            <div className="w-32 bg-slate-800 rounded-full h-1.5 overflow-hidden">
              <div
                className="bg-sky-400 h-full transition-all duration-200"
                style={{ width: `${scanProgress.percentage}%` }}
              />
            </div>
          </div>
        )}

        {/* Active Month/Year Filter Banner */}
        {selectedYear > 0 && selectedMonth > 0 && (
          <div className="px-6 py-2 bg-sky-500/15 border-b border-sky-500/30 flex items-center justify-between text-xs text-sky-200 backdrop-blur-md">
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-sky-400" />
              <span>
                Filtered by:{' '}
                <strong className="text-white">
                  {monthNames[selectedMonth - 1]} {selectedYear}
                </strong>
              </span>
              <span className="text-slate-400 font-mono">({photos.length} items)</span>
            </div>
            <button
              onClick={() => {
                setSelectedYear(0)
                setSelectedMonth(0)
              }}
              className="flex items-center gap-1 px-2.5 py-0.5 rounded-lg bg-sky-500/20 hover:bg-sky-500/30 text-sky-300 hover:text-white transition text-[11px]"
            >
              <X className="w-3 h-3" />
              <span>Show All Months</span>
            </button>
          </div>
        )}

        {/* Dynamic View Body */}
        <div className="flex-1 flex h-full overflow-hidden">
          {isSetupHubOpen ? (
            <SetupHub
              onComplete={() => {
                setIsSetupHubOpen(false)
                loadPhotos()
              }}
              onCancel={photos.length > 0 ? () => setIsSetupHubOpen(false) : undefined}
              isFirstRun={photos.length === 0}
            />
          ) : currentTab === 'cleanup' ? (
            <CleanUpAssistant
              photos={photos ?? []}
              onOpenLightbox={setActiveLightboxPhoto}
              onToggleSelect={handleToggleSelect}
              selectedIDs={selectedIDs}
            />
          ) : currentTab === 'google_photos' ? (
            <GooglePhotosView stats={stats} />
          ) : currentTab === 'people' ? (
            <PeopleFacesView
              photos={photos ?? []}
              onOpenLightbox={setActiveLightboxPhoto}
              onToggleFavorite={handleToggleFavorite}
            />
          ) : (
            <>
              {/* Photo Grid */}
              <PhotoGrid
                photos={photos ?? []}
                selectedIDs={selectedIDs}
                onToggleSelect={handleToggleSelect}
                onOpenLightbox={setActiveLightboxPhoto}
                onToggleFavorite={handleToggleFavorite}
                gridColumns={gridColumns}
                onOpenSetupHub={() => setIsSetupHubOpen(true)}
              />

              {/* Right-Hand Timeline Scrubber */}
              <TimelineScrubber
                buckets={timelineBuckets ?? []}
                onSelectMonth={handleSelectMonth}
                selectedYear={selectedYear}
                selectedMonth={selectedMonth}
              />
            </>
          )}
        </div>

        {/* Floating Glass Selection Pill */}
        <FloatingSelectionPill
          selectedCount={selectedIDs.size}
          onClearSelection={() => setSelectedIDs(new Set())}
          onMoveToFolder={handleMoveToFolder}
          onToggleShowInAll={handleToggleShowInAll}
          onSmartTagSelected={handleSmartTagSelected}
        />
      </div>

      {/* Lightbox Modal with Ambient Chameleon Glow */}
      {activeLightboxPhoto && (
        <Lightbox
          photo={activeLightboxPhoto}
          photos={photos ?? []}
          onClose={() => setActiveLightboxPhoto(null)}
          onUpdatePhoto={(updated) => {
            setActiveLightboxPhoto(updated)
            setPhotos((prev) => (prev ?? []).map((p) => (p.id === updated.id ? updated : p)))
          }}
        />
      )}

      {/* Settings & Catalog Maintenance Modal */}
      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        onRefreshStats={loadPhotos}
      />
    </div>
  )
}
