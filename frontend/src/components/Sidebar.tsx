import React from 'react'
import {
  Image,
  Apple,
  Smartphone,
  Camera,
  FolderPlus,
  Users,
  Sparkles,
  Cloud,
  HelpCircle,
  SmartphoneCharging,
  Settings,
  Folder
} from 'lucide-react'
import { CatalogStats, WatchedFolder } from '../types'

interface SidebarProps {
  currentTab: string
  onSelectTab: (tab: string) => void
  stats: CatalogStats
  watchedFolders: WatchedFolder[]
  onOpenSettings: () => void
  onAddCustomFolder: () => void
  onOpenSetupHub: () => void
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  stats,
  watchedFolders,
  onOpenSettings,
  onAddCustomFolder,
  onOpenSetupHub,
}) => {
  // Extract custom route folders
  const customRoutes = Array.from(
    new Set(
      watchedFolders
        .map((f) => f.route_override?.trim())
        .filter((r) => r && r !== '')
    )
  )

  const navItems = [
    { id: 'all', label: 'All Photos', icon: Image, count: stats.total_photos },
    { id: 'ios', label: 'iOS', icon: Apple, count: stats.ios_count, color: 'text-sky-400' },
    { id: 'android', label: 'Android', icon: Smartphone, count: stats.android_count, color: 'text-emerald-400' },
    { id: 'camera', label: 'Camera', icon: Camera, count: stats.camera_count, color: 'text-amber-400' },
  ]

  const toolItems = [
    { id: 'people', label: 'People (Faces)', icon: Users },
    { id: 'cleanup', label: 'Clean-Up Assistant', icon: Sparkles, color: 'text-purple-400' },
    { id: 'google_photos', label: 'Google Photos', icon: Cloud, color: 'text-blue-400' },
    { id: 'unknown', label: 'Unknown', icon: HelpCircle, count: stats.unknown_count },
    { id: 'screenshots', label: 'Screenshots', icon: SmartphoneCharging, count: stats.screenshots, color: 'text-rose-400' },
  ]

  return (
    <aside className="w-64 h-full flex flex-col glass-sidebar select-none z-20">
      {/* Brand Header */}
      <div className="p-5 flex items-center gap-3 border-b border-white/5">
        <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-sky-500 via-indigo-500 to-fuchsia-500 flex items-center justify-center shadow-lg shadow-indigo-500/20">
          <Sparkles className="w-5 h-5 text-white" />
        </div>
        <div>
          <h1 className="text-base font-semibold tracking-wide text-white flex items-center gap-1.5">
            Lumina <span className="text-xs px-1.5 py-0.5 rounded bg-sky-500/20 text-sky-400 font-mono">v3.0</span>
          </h1>
          <p className="text-xs text-slate-400">Glassmorphic Photo Suite</p>
        </div>
      </div>

      {/* Navigation List */}
      <div className="flex-1 overflow-y-auto px-3 py-4 space-y-5">
        {/* Quick Add Sources CTA */}
        <button
          onClick={onOpenSetupHub}
          className="w-full flex items-center justify-center gap-2 py-2 px-3 rounded-xl bg-gradient-to-r from-sky-500/20 to-indigo-500/20 border border-sky-500/30 text-sky-300 hover:text-white hover:bg-sky-500/30 text-xs font-semibold shadow-sm transition"
        >
          <FolderPlus className="w-4 h-4 text-sky-400" />
          <span>+ Add Folders / Drives</span>
        </button>

        {/* Core Library */}
        <div className="space-y-1">
          <div className="px-3 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            Library
          </div>
          {navItems.map((item) => {
            const Icon = item.icon
            const active = currentTab.toLowerCase() === item.id.toLowerCase()
            return (
              <button
                key={item.id}
                onClick={() => onSelectTab(item.id)}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-sm font-medium transition-all ${
                  active
                    ? 'bg-sky-500/15 text-sky-300 shadow-sm border border-sky-500/20'
                    : 'text-slate-300 hover:bg-white/5 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Icon className={`w-4 h-4 ${item.color || 'text-slate-400'}`} />
                  <span>{item.label}</span>
                </div>
                {item.count !== undefined && item.count > 0 && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono">
                    {item.count}
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* Custom People / Owner Folders */}
        <div className="space-y-1">
          <div className="px-3 flex items-center justify-between text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            <span>Folders</span>
            <button
              onClick={onAddCustomFolder}
              className="text-sky-400 hover:text-sky-300 p-0.5 hover:bg-sky-500/10 rounded transition"
              title="Add Custom Folder / Person Route"
            >
              <FolderPlus className="w-3.5 h-3.5" />
            </button>
          </div>
          {customRoutes.length === 0 ? (
            <p className="px-3 py-1.5 text-xs text-slate-500 italic">No custom folders</p>
          ) : (
            customRoutes.map((route) => {
              const active = currentTab.toLowerCase() === route.toLowerCase()
              return (
                <button
                  key={route}
                  onClick={() => onSelectTab(route)}
                  className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-sm font-medium transition-all ${
                    active
                      ? 'bg-indigo-500/15 text-indigo-300 border border-indigo-500/20'
                      : 'text-slate-300 hover:bg-white/5 hover:text-white'
                  }`}
                >
                  <div className="flex items-center gap-2.5 truncate">
                    <Folder className="w-4 h-4 text-indigo-400 shrink-0" />
                    <span className="truncate">{route}</span>
                  </div>
                </button>
              )
            })
          )}
        </div>

        {/* Intelligent Tools & Platforms */}
        <div className="space-y-1">
          <div className="px-3 text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            Intelligent Tools
          </div>
          {toolItems.map((item) => {
            const Icon = item.icon
            const active = currentTab.toLowerCase() === item.id.toLowerCase()
            return (
              <button
                key={item.id}
                onClick={() => onSelectTab(item.id)}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-sm font-medium transition-all ${
                  active
                    ? 'bg-sky-500/15 text-sky-300 border border-sky-500/20'
                    : 'text-slate-300 hover:bg-white/5 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Icon className={`w-4 h-4 ${item.color || 'text-slate-400'}`} />
                  <span>{item.label}</span>
                </div>
                {item.count !== undefined && item.count > 0 && (
                  <span className="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono">
                    {item.count}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      </div>

      {/* Footer Settings Button */}
      <div className="p-3 border-t border-white/5">
        <button
          onClick={onOpenSettings}
          className="w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-sm text-slate-300 hover:bg-white/5 hover:text-white transition"
        >
          <Settings className="w-4 h-4 text-slate-400" />
          <span>Settings</span>
        </button>
      </div>
    </aside>
  )
}
