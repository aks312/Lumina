import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { FolderInput, Eye, EyeOff, Share2, X, Check, Tag } from 'lucide-react'

interface FloatingSelectionPillProps {
  selectedCount: number
  onClearSelection: () => void
  onMoveToFolder: (folderName: string) => void
  onToggleShowInAll: (show: boolean) => void
  onSmartTagSelected: () => void
}

export const FloatingSelectionPill: React.FC<FloatingSelectionPillProps> = ({
  selectedCount,
  onClearSelection,
  onMoveToFolder,
  onToggleShowInAll,
  onSmartTagSelected,
}) => {
  const [showFolderInput, setShowFolderInput] = useState(false)
  const [targetFolder, setTargetFolder] = useState('')

  if (selectedCount === 0) return null

  const handleFolderSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (targetFolder.trim()) {
      onMoveToFolder(targetFolder.trim())
      setTargetFolder('')
      setShowFolderInput(false)
    }
  }

  return (
    <AnimatePresence>
      <motion.div
        initial={{ y: 80, opacity: 0, scale: 0.9 }}
        animate={{ y: 0, opacity: 1, scale: 1 }}
        exit={{ y: 80, opacity: 0, scale: 0.9 }}
        transition={{ type: 'spring', damping: 25, stiffness: 300 }}
        className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 glass-pill px-4 py-2.5 rounded-full flex items-center gap-3 shadow-2xl border border-white/20 select-none"
      >
        {/* Count Badge */}
        <div className="flex items-center gap-2 pr-3 border-r border-white/10">
          <span className="w-5 h-5 rounded-full bg-sky-500 text-slate-950 font-bold text-xs flex items-center justify-center">
            {selectedCount}
          </span>
          <span className="text-xs font-medium text-slate-200">Selected</span>
        </div>

        {/* Action: Move to Folder */}
        {showFolderInput ? (
          <form onSubmit={handleFolderSubmit} className="flex items-center gap-1.5">
            <input
              type="text"
              value={targetFolder}
              onChange={(e) => setTargetFolder(e.target.value)}
              placeholder="Folder / Person Name..."
              autoFocus
              className="px-2.5 py-1 text-xs rounded-lg bg-slate-900 border border-sky-500/50 text-white focus:outline-none"
            />
            <button
              type="submit"
              className="p-1 rounded-lg bg-sky-500 text-slate-950 hover:bg-sky-400"
            >
              <Check className="w-3.5 h-3.5" />
            </button>
            <button
              type="button"
              onClick={() => setShowFolderInput(false)}
              className="p-1 rounded-lg hover:bg-white/10 text-slate-400"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </form>
        ) : (
          <button
            onClick={() => setShowFolderInput(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full hover:bg-white/10 text-xs font-medium text-slate-200 transition"
          >
            <FolderInput className="w-3.5 h-3.5 text-indigo-400" />
            <span>Move to Folder</span>
          </button>
        )}

        {/* Action: Smart Tag Selected */}
        <button
          onClick={onSmartTagSelected}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-full hover:bg-white/10 text-xs font-medium text-slate-200 transition"
        >
          <Tag className="w-3.5 h-3.5 text-sky-400" />
          <span>Smart Tag</span>
        </button>

        {/* Action: Hide / Show in All Photos */}
        <button
          onClick={() => onToggleShowInAll(false)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-full hover:bg-white/10 text-xs font-medium text-slate-200 transition"
          title="Hide from All Photos View"
        >
          <EyeOff className="w-3.5 h-3.5 text-amber-400" />
          <span>Hide from All</span>
        </button>

        {/* Action: Share / Export */}
        <button
          onClick={() => alert(`Ready to share ${selectedCount} photos securely.`)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-full hover:bg-white/10 text-xs font-medium text-slate-200 transition"
        >
          <Share2 className="w-3.5 h-3.5 text-emerald-400" />
          <span>Safe Share</span>
        </button>

        {/* Dismiss / Deselect All */}
        <button
          onClick={onClearSelection}
          className="p-1.5 rounded-full hover:bg-white/10 text-slate-400 hover:text-white transition ml-1"
          title="Clear Selection"
        >
          <X className="w-4 h-4" />
        </button>
      </motion.div>
    </AnimatePresence>
  )
}
