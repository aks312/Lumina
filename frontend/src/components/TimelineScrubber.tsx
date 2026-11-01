import React from 'react'
import { TimelineBucket } from '../types'

interface TimelineScrubberProps {
  buckets: TimelineBucket[]
  onSelectMonth: (year: number, month: number) => void
  selectedYear?: number
  selectedMonth?: number
}

export const TimelineScrubber: React.FC<TimelineScrubberProps> = ({
  buckets,
  onSelectMonth,
  selectedYear,
  selectedMonth,
}) => {
  if (buckets.length === 0) return null

  const monthShort = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

  return (
    <div className="w-14 py-6 pr-2 flex flex-col items-center justify-center gap-1 select-none z-10">
      <div className="w-full py-2 px-1 rounded-2xl bg-slate-900/40 border border-white/5 backdrop-blur-md flex flex-col items-center gap-1.5 max-h-[70vh] overflow-y-auto scrollbar-none">
        {buckets.map((b) => {
          const mName = monthShort[b.month - 1] || `${b.month}`
          const isSelected = b.year === selectedYear && b.month === selectedMonth

          return (
            <button
              key={`${b.year}-${b.month}`}
              onClick={() => onSelectMonth(b.year, b.month)}
              className={`group relative flex flex-col items-center py-1 px-1 rounded-lg transition-all w-full ${
                isSelected
                  ? 'bg-sky-500/30 border border-sky-400/60 text-sky-200 font-bold shadow-md shadow-sky-500/25 ring-1 ring-sky-400/50'
                  : 'hover:bg-sky-500/20 text-slate-400 hover:text-white border border-transparent'
              }`}
              title={`${mName} ${b.year} (${b.count} photos)`}
            >
              <span className={`text-[10px] font-mono leading-none ${isSelected ? 'text-sky-200 font-bold' : 'font-medium'}`}>
                {b.year}
              </span>
              <span
                className={`text-[9px] uppercase leading-tight font-semibold ${
                  isSelected ? 'text-sky-300 font-bold' : 'text-slate-500 group-hover:text-sky-300'
                }`}
              >
                {mName}
              </span>

              {/* Hover pill showing exact count */}
              <div className="absolute right-full mr-2 px-2.5 py-1 rounded-md bg-slate-900/95 border border-white/10 text-xs text-white opacity-0 group-hover:opacity-100 pointer-events-none transition shadow-xl whitespace-nowrap z-30">
                {mName} {b.year} • <span className="text-sky-400 font-mono font-bold">{b.count}</span> items
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
