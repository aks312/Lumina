import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  X,
  ChevronLeft,
  ChevronRight,
  RotateCw,
  Maximize2,
  Info,
  Heart,
  Tag,
  FileText,
  Calendar,
  Camera,
  Cpu,
  Folder,
  RefreshCw,
} from 'lucide-react'
import { PhotoRecord } from '../types'
import { api } from '../services/api'

interface LightboxProps {
  photo: PhotoRecord | null
  photos: PhotoRecord[]
  onClose: () => void
  onUpdatePhoto: (updated: PhotoRecord) => void
}

export const Lightbox: React.FC<LightboxProps> = ({
  photo,
  photos,
  onClose,
  onUpdatePhoto,
}) => {
  const [showInfo, setShowInfo] = useState(false)
  const [highResLoaded, setHighResLoaded] = useState(false)
  const [rawError, setRawError] = useState(false)

  // Initialize currentIdx synchronously from photo prop (prevents flashing photos[0] on open)
  const [currentIdx, setCurrentIdx] = useState<number>(() => {
    if (!photo) return 0
    const idx = photos.findIndex((p) => p.id === photo.id)
    return idx !== -1 ? idx : 0
  })

  // Keep in sync if photo prop changes from outside
  useEffect(() => {
    if (photo) {
      const idx = photos.findIndex((p) => p.id === photo.id)
      if (idx !== -1 && idx !== currentIdx) {
        setCurrentIdx(idx)
      }
    }
  }, [photo?.id])

  // Reset loading & error states when switching photos
  const currentPhoto = photos[currentIdx] || photo
  useEffect(() => {
    setHighResLoaded(false)
    setRawError(false)
  }, [currentPhoto?.id])

  // Keyboard navigation: Left/Right, R (rotate), F (fullscreen), I (info), Esc (close)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!currentPhoto) return
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft') handlePrev()
      if (e.key === 'ArrowRight') handleNext()
      if (e.key === 'r' || e.key === 'R') handleRotate()
      if (e.key === 'f' || e.key === 'F') api.toggleFullscreen()
      if (e.key === 'i' || e.key === 'I') setShowInfo((prev) => !prev)
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [currentPhoto, currentIdx, photos])

  if (!currentPhoto) return null

  const handlePrev = () => {
    if (currentIdx > 0) setCurrentIdx(currentIdx - 1)
  }

  const handleNext = () => {
    if (currentIdx < photos.length - 1) setCurrentIdx(currentIdx + 1)
  }

  const handleRotate = async () => {
    const newRot = (currentPhoto.rotation + 90) % 360
    const updated = { ...currentPhoto, rotation: newRot, is_user_modified: true }
    onUpdatePhoto(updated)
    await api.updatePhoto(
      currentPhoto.id,
      currentPhoto.route_folder,
      currentPhoto.platform,
      newRot,
      currentPhoto.rating,
      currentPhoto.is_favorite,
      currentPhoto.show_in_all
    )
  }

  const handleToggleFavorite = async () => {
    const newFav = !currentPhoto.is_favorite
    const updated = { ...currentPhoto, is_favorite: newFav, is_user_modified: true }
    onUpdatePhoto(updated)
    await api.updatePhoto(
      currentPhoto.id,
      currentPhoto.route_folder,
      currentPhoto.platform,
      currentPhoto.rotation,
      currentPhoto.rating,
      newFav,
      currentPhoto.show_in_all
    )
  }

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center select-none overflow-hidden">
      {/* Ambient Chameleon Glow: Heavily blurred, low-opacity copy behind the main canvas */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none -z-10">
        <img
          key={`ambient-${currentPhoto.id}`}
          src={`/thumb/${currentPhoto.id}`}
          alt=""
          className="w-full h-full object-cover blur-3xl opacity-35 scale-110 transition-all duration-700"
        />
        <div className="absolute inset-0 bg-slate-950/75 backdrop-blur-2xl" />
      </div>

      {/* Top Control Bar */}
      <div className="absolute top-0 left-0 right-0 h-16 px-6 flex items-center justify-between z-20">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-slate-200 truncate max-w-sm">
            {currentPhoto.file_name}
          </span>
          <span className="text-xs text-slate-500 font-mono">
            {currentIdx + 1} / {photos.length}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* Rotate Button (R) */}
          <button
            onClick={handleRotate}
            className="p-2 rounded-xl bg-slate-900/60 border border-white/10 text-slate-300 hover:text-white hover:bg-slate-800 transition"
            title="Rotate 90° Clockwise (R)"
          >
            <RotateCw className="w-4 h-4" />
          </button>

          {/* Favorite Toggle */}
          <button
            onClick={handleToggleFavorite}
            className="p-2 rounded-xl bg-slate-900/60 border border-white/10 text-slate-300 hover:text-white hover:bg-slate-800 transition"
            title="Toggle Favorite"
          >
            <Heart
              className={`w-4 h-4 ${
                currentPhoto.is_favorite ? 'fill-rose-500 text-rose-500' : ''
              }`}
            />
          </button>

          {/* Fullscreen Button (F) */}
          <button
            onClick={() => api.toggleFullscreen()}
            className="p-2 rounded-xl bg-slate-900/60 border border-white/10 text-slate-300 hover:text-white hover:bg-slate-800 transition"
            title="Native Fullscreen (F)"
          >
            <Maximize2 className="w-4 h-4" />
          </button>

          {/* Info Drawer Toggle (I) */}
          <button
            onClick={() => setShowInfo((prev) => !prev)}
            className={`p-2 rounded-xl border transition ${
              showInfo
                ? 'bg-sky-500/20 border-sky-500/40 text-sky-400'
                : 'bg-slate-900/60 border-white/10 text-slate-300 hover:text-white hover:bg-slate-800'
            }`}
            title="Toggle Photo Details & EXIF (I)"
          >
            <Info className="w-4 h-4" />
          </button>

          {/* Close Lightbox */}
          <button
            onClick={onClose}
            className="p-2 rounded-xl bg-slate-900/60 border border-white/10 text-slate-300 hover:text-white hover:bg-rose-500/20 transition ml-2"
            title="Close (Escape)"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Navigation Arrows */}
      {currentIdx > 0 && (
        <button
          onClick={handlePrev}
          className="absolute left-6 top-1/2 -translate-y-1/2 p-3 rounded-full bg-slate-900/60 border border-white/10 text-slate-300 hover:text-white hover:bg-slate-800/90 transition shadow-2xl z-20"
        >
          <ChevronLeft className="w-6 h-6" />
        </button>
      )}

      {currentIdx < photos.length - 1 && (
        <button
          onClick={handleNext}
          className="absolute right-6 top-1/2 -translate-y-1/2 p-3 rounded-full bg-slate-900/60 border border-white/10 text-slate-300 hover:text-white hover:bg-slate-800/90 transition shadow-2xl z-20"
        >
          <ChevronRight className="w-6 h-6" />
        </button>
      )}

      {/* Central Stage */}
      <div className="w-full h-full p-16 flex items-center justify-center">
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          transition={{ type: 'spring', damping: 28, stiffness: 350 }}
          className="relative max-w-full max-h-full flex items-center justify-center shadow-2xl rounded-2xl overflow-hidden border border-white/10 bg-slate-950/40"
        >
          {currentPhoto.is_video ? (
            <video
              key={`video-${currentPhoto.id}`}
              src={`/raw/${currentPhoto.id}`}
              controls
              autoPlay
              playsInline
              className="max-w-[85vw] max-h-[80vh] object-contain rounded-xl shadow-2xl"
            />
          ) : (
            <div className="relative flex items-center justify-center max-w-[85vw] max-h-[80vh]">
              {/* Layer 1: Instant Base Thumbnail (0ms) */}
              <img
                src={`/thumb/${currentPhoto.id}`}
                alt={currentPhoto.file_name}
                className={`max-w-[85vw] max-h-[80vh] object-contain rounded-xl transition-all duration-300 ${
                  highResLoaded ? 'opacity-0 scale-100 absolute pointer-events-none' : 'opacity-100 scale-100 filter blur-xs'
                }`}
                style={{
                  transform: `rotate(${currentPhoto.rotation}deg)`,
                }}
              />

              {/* Layer 2: High-Res Master Image */}
              <img
                key={`master-${currentPhoto.id}`}
                src={`/raw/${currentPhoto.id}`}
                alt={currentPhoto.file_name}
                onLoad={() => setHighResLoaded(true)}
                onError={() => setRawError(true)}
                className={`max-w-[85vw] max-h-[80vh] object-contain rounded-xl transition-opacity duration-200 ${
                  highResLoaded ? 'opacity-100' : 'opacity-0'
                }`}
                style={{
                  transform: `rotate(${currentPhoto.rotation}deg)`,
                }}
              />

              {/* Graceful fallback if /raw/ has temporary network/disk error */}
              {rawError && (
                <div className="absolute bottom-4 left-1/2 -translate-x-1/2 px-3 py-1.5 rounded-xl bg-slate-900/95 border border-white/20 text-xs text-slate-300 flex items-center gap-2 shadow-2xl z-30">
                  <span>Displaying cached preview</span>
                  <button
                    onClick={() => {
                      setRawError(false)
                      setHighResLoaded(false)
                    }}
                    className="flex items-center gap-1 px-2 py-0.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white font-medium text-[11px] transition shadow"
                  >
                    <RefreshCw className="w-3 h-3" />
                    <span>Retry Full-Res</span>
                  </button>
                </div>
              )}
            </div>
          )}
        </motion.div>
      </div>

      {/* Slide-out Metadata Drawer */}
      <AnimatePresence>
        {showInfo && (
          <motion.div
            initial={{ x: 380, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: 380, opacity: 0 }}
            transition={{ type: 'spring', damping: 26, stiffness: 280 }}
            className="absolute right-0 top-0 bottom-0 w-80 glass-sidebar p-6 overflow-y-auto space-y-6 z-30 shadow-2xl"
          >
            <div className="flex items-center justify-between border-b border-white/10 pb-3">
              <h4 className="text-sm font-semibold text-white">Photo Details</h4>
              <button
                onClick={() => setShowInfo(false)}
                className="p-1 rounded-lg text-slate-400 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Basic File Specs */}
            <div className="space-y-3">
              <div className="flex items-center gap-2.5 text-xs text-slate-300">
                <Calendar className="w-4 h-4 text-sky-400" />
                <span>{new Date(currentPhoto.date_taken).toLocaleString()}</span>
              </div>
              <div className="flex items-center gap-2.5 text-xs text-slate-300">
                <Camera className="w-4 h-4 text-amber-400" />
                <span>
                  {currentPhoto.camera_make || currentPhoto.canonical_device || 'Unknown Camera'}{' '}
                  {currentPhoto.camera_model}
                </span>
              </div>
              <div className="flex items-center gap-2.5 text-xs text-slate-300">
                <Cpu className="w-4 h-4 text-emerald-400" />
                <span>
                  {currentPhoto.width} × {currentPhoto.height} • {formatBytes(currentPhoto.file_size)}
                </span>
              </div>
              <div className="flex items-center gap-2.5 text-xs text-slate-300">
                <Folder className="w-4 h-4 text-indigo-400" />
                <span>Route: {currentPhoto.route_folder}</span>
              </div>
            </div>

            {/* AI Tags */}
            <div className="space-y-2">
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                <Tag className="w-3.5 h-3.5 text-sky-400" />
                <span>Visual Tags</span>
              </div>
              {currentPhoto.tags ? (
                <div className="flex flex-wrap gap-1.5">
                  {currentPhoto.tags.split(',').map((t, i) => (
                    <span
                      key={i}
                      className="px-2 py-0.5 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-300 text-[11px]"
                    >
                      {t.trim()}
                    </span>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-slate-500 italic">No tags generated yet</p>
              )}
            </div>

            {/* OCR Text */}
            <div className="space-y-2">
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                <FileText className="w-3.5 h-3.5 text-purple-400" />
                <span>Native Windows OCR Text</span>
              </div>
              {currentPhoto.ocr_text ? (
                <div className="p-3 rounded-xl bg-slate-900/80 border border-white/5 text-xs text-slate-300 max-h-40 overflow-y-auto font-mono whitespace-pre-wrap leading-relaxed">
                  {currentPhoto.ocr_text}
                </div>
              ) : (
                <p className="text-xs text-slate-500 italic">No text recognized in image</p>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
