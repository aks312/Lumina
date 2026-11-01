import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Play, Heart, CheckCircle2, Circle, Zap, Tag } from 'lucide-react'
import { PhotoRecord } from '../types'

interface PhotoGridProps {
  photos: PhotoRecord[]
  selectedIDs: Set<number>
  onToggleSelect: (id: number, e: React.MouseEvent) => void
  onOpenLightbox: (photo: PhotoRecord) => void
  onToggleFavorite: (photo: PhotoRecord, e: React.MouseEvent) => void
  gridColumns: number
  onOpenSetupHub?: () => void
}

export const PhotoGrid: React.FC<PhotoGridProps> = ({
  photos,
  selectedIDs,
  onToggleSelect,
  onOpenLightbox,
  onToggleFavorite,
  gridColumns,
  onOpenSetupHub,
}) => {
  const safePhotos = Array.isArray(photos) ? photos : []

  if (safePhotos.length === 0) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-12 text-slate-400 select-none">
        <div className="w-16 h-16 rounded-2xl bg-slate-900/80 border border-white/15 flex items-center justify-center mb-4 shadow-xl">
          <Zap className="w-8 h-8 text-sky-400/70" />
        </div>
        <p className="text-base font-semibold text-slate-200">No photos in this view</p>
        <p className="text-xs text-slate-400 mt-1 max-w-sm text-center">
          Add a source folder or connected drive, or run "Sweep & Organize" to catalog your photos.
        </p>
        {onOpenSetupHub && (
          <button
            onClick={onOpenSetupHub}
            className="mt-4 px-4 py-2 rounded-xl bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold shadow-lg shadow-sky-600/30 transition"
          >
            + Add Folders / Drives
          </button>
        )}
      </div>
    )
  }

  // Group photos by Month-Year for seamless timeline layout
  const groups: { [key: string]: PhotoRecord[] } = {}
  safePhotos.forEach((p) => {
    const key = `${p.year}-${String(p.month).padStart(2, '0')}`
    if (!groups[key]) groups[key] = []
    groups[key].push(p)
  })

  const monthNames = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
  ]

  return (
    <div className="flex-1 overflow-y-auto px-6 py-6 space-y-8 select-none">
      {Object.entries(groups).map(([groupKey, groupPhotos]) => {
        const [yStr, mStr] = groupKey.split('-')
        const y = parseInt(yStr, 10)
        const m = parseInt(mStr, 10)
        const title = `${monthNames[m - 1] || 'Unknown'} ${y || ''}`

        return (
          <div key={groupKey} id={`timeline-${groupKey}`} className="space-y-3">
            {/* Group Header */}
            <div className="flex items-baseline justify-between border-b border-white/5 pb-2">
              <h3 className="text-sm font-semibold text-slate-200 tracking-wide">{title}</h3>
              <span className="text-xs text-slate-500 font-mono">{groupPhotos.length} items</span>
            </div>

            {/* Grid with dynamic columns */}
            <div
              className="grid gap-3"
              style={{
                gridTemplateColumns: `repeat(${gridColumns}, minmax(0, 1fr))`,
              }}
            >
              <AnimatePresence>
                {groupPhotos.map((photo) => {
                  const isSelected = selectedIDs.has(photo.id)

                  return (
                    <motion.div
                      layout
                      layoutId={`photo-${photo.id}`}
                      key={photo.id}
                      onClick={() => onOpenLightbox(photo)}
                      className={`group relative aspect-square rounded-xl overflow-hidden bg-slate-900 border cursor-pointer transition-all ${
                        isSelected
                          ? 'ring-2 ring-sky-500 border-sky-500 shadow-lg shadow-sky-500/20'
                          : 'border-white/10 hover:border-white/30 hover:shadow-xl'
                      }`}
                    >
                      {/* Thumbnail Image served directly from thumbs.pack */}
                      <img
                        src={`/thumb/${photo.id}`}
                        alt={photo.file_name}
                        loading="lazy"
                        className="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                      />

                      {/* Video Indicator */}
                      {photo.is_video && (
                        <div className="absolute top-2 left-2 px-2 py-0.5 rounded-md bg-black/60 backdrop-blur-md border border-white/20 flex items-center gap-1 text-[10px] text-white">
                          <Play className="w-2.5 h-2.5 fill-white" />
                          <span>Video</span>
                        </div>
                      )}

                      {/* Live Photo Indicator */}
                      {photo.live_photo_pair && !photo.is_video && (
                        <div className="absolute top-2 left-2 px-1.5 py-0.5 rounded-md bg-sky-950/70 backdrop-blur-md border border-sky-500/30 flex items-center gap-1 text-[9px] text-sky-300 font-mono">
                          <span>LIVE</span>
                        </div>
                      )}

                      {/* Multi-Selection Checkbox */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          onToggleSelect(photo.id, e)
                        }}
                        className={`absolute top-2 right-2 p-1 rounded-full transition-opacity ${
                          isSelected
                            ? 'opacity-100 text-sky-400 bg-slate-950/80 shadow'
                            : 'opacity-0 group-hover:opacity-100 text-white/80 hover:text-white bg-slate-950/50'
                        }`}
                      >
                        {isSelected ? (
                          <CheckCircle2 className="w-5 h-5 fill-sky-500 text-slate-950" />
                        ) : (
                          <Circle className="w-5 h-5" />
                        )}
                      </button>

                      {/* Favorite Heart Button */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          onToggleFavorite(photo, e)
                        }}
                        className={`absolute bottom-2 right-2 p-1.5 rounded-full transition-opacity ${
                          photo.is_favorite
                            ? 'opacity-100 text-rose-400 bg-slate-950/80 shadow'
                            : 'opacity-0 group-hover:opacity-100 text-white/70 hover:text-rose-400 bg-slate-950/50'
                        }`}
                        title="Toggle Favorite"
                      >
                        <Heart className={`w-3.5 h-3.5 ${photo.is_favorite ? 'fill-rose-400' : ''}`} />
                      </button>

                      {/* Tag Count Badge if tags exist */}
                      {photo.tags && photo.tags.length > 0 && (
                        <div className="absolute bottom-2 left-2 opacity-0 group-hover:opacity-100 transition-opacity px-1.5 py-0.5 rounded bg-slate-950/70 backdrop-blur-md text-[9px] text-slate-300 flex items-center gap-1">
                          <Tag className="w-2.5 h-2.5 text-sky-400" />
                          <span>{photo.tags.split(',').length}</span>
                        </div>
                      )}
                    </motion.div>
                  )
                })}
              </AnimatePresence>
            </div>
          </div>
        )
      })}
    </div>
  )
}
