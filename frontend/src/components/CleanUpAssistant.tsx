import React, { useMemo } from 'react'
import { Sparkles, Copy, EyeOff, Video, Trash2, CheckCircle } from 'lucide-react'
import { PhotoRecord } from '../types'

interface CleanUpAssistantProps {
  photos: PhotoRecord[]
  onOpenLightbox: (photo: PhotoRecord) => void
  onToggleSelect: (id: number, e: React.MouseEvent) => void
  selectedIDs: Set<number>
}

export const CleanUpAssistant: React.FC<CleanUpAssistantProps> = ({
  photos,
  onOpenLightbox,
  onToggleSelect,
  selectedIDs,
}) => {
  // 1. Duplicate detection: photos sharing same sha256
  const duplicates = useMemo(() => {
    const hashGroups: { [hash: string]: PhotoRecord[] } = {}
    photos.forEach((p) => {
      if (!hashGroups[p.sha256]) hashGroups[p.sha256] = []
      hashGroups[p.sha256].push(p)
    })
    const dupList: PhotoRecord[] = []
    Object.values(hashGroups).forEach((group) => {
      if (group.length > 1) {
        dupList.push(...group.slice(1)) // Keep first, flag remainder
      }
    })
    return dupList
  }, [photos])

  // 2. Large Videos (> 500MB)
  const largeVideos = useMemo(() => {
    return photos.filter((p) => p.is_video && p.file_size > 500 * 1024 * 1024)
  }, [photos])

  return (
    <div className="flex-1 overflow-y-auto p-8 space-y-8 select-none">
      <div>
        <h2 className="text-xl font-bold text-white flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-purple-400" />
          <span>Clean-Up Assistant</span>
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Intelligently reclaim disk space by identifying redundant duplicates and massive videos.
        </p>
      </div>

      {/* Duplicates Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between border-b border-white/5 pb-2">
          <div className="flex items-center gap-2">
            <Copy className="w-4 h-4 text-sky-400" />
            <h3 className="text-sm font-semibold text-white">Exact Duplicate Assets</h3>
          </div>
          <span className="text-xs text-slate-500 font-mono">{duplicates.length} redundant files</span>
        </div>

        {duplicates.length === 0 ? (
          <div className="p-6 rounded-2xl bg-slate-900/40 border border-white/5 flex items-center gap-3 text-emerald-400 text-xs">
            <CheckCircle className="w-4 h-4" />
            <span>Zero duplicates detected across your catalog!</span>
          </div>
        ) : (
          <div className="grid grid-cols-6 gap-3">
            {duplicates.map((photo) => (
              <div
                key={photo.id}
                onClick={() => onOpenLightbox(photo)}
                className="group relative aspect-square rounded-xl overflow-hidden bg-slate-900 border border-white/10 hover:border-sky-500 cursor-pointer"
              >
                <img src={`/thumb/${photo.id}`} alt="" className="w-full h-full object-cover" />
                <div className="absolute inset-0 bg-slate-950/40 opacity-0 group-hover:opacity-100 transition flex items-center justify-center">
                  <span className="text-[10px] text-white px-2 py-0.5 rounded bg-slate-900/80">Review</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Large Videos Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between border-b border-white/5 pb-2">
          <div className="flex items-center gap-2">
            <Video className="w-4 h-4 text-amber-400" />
            <h3 className="text-sm font-semibold text-white">Large Video Files (&gt;500MB)</h3>
          </div>
          <span className="text-xs text-slate-500 font-mono">{largeVideos.length} videos</span>
        </div>

        {largeVideos.length === 0 ? (
          <div className="p-6 rounded-2xl bg-slate-900/40 border border-white/5 flex items-center gap-3 text-slate-400 text-xs">
            <span>No oversized videos found.</span>
          </div>
        ) : (
          <div className="space-y-2">
            {largeVideos.map((video) => (
              <div
                key={video.id}
                className="p-3 rounded-xl bg-slate-900/60 border border-white/5 flex items-center justify-between text-xs"
              >
                <div className="truncate max-w-md">
                  <p className="font-medium text-slate-200 truncate">{video.file_name}</p>
                  <p className="text-slate-500 font-mono text-[11px]">
                    {(video.file_size / (1024 * 1024)).toFixed(1)} MB • {video.canonical_device}
                  </p>
                </div>
                <button
                  onClick={() => onOpenLightbox(video)}
                  className="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 transition"
                >
                  Play
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
