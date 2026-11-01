export interface PhotoRecord {
  id: number
  original_path: string
  organized_path: string
  file_name: string
  file_size: number
  sha256: string
  width: number
  height: number
  date_taken: string
  year: number
  month: number
  day: number
  camera_make: string
  camera_model: string
  canonical_device: string
  platform: 'ios' | 'android' | 'camera' | 'unknown' | 'custom'
  is_screenshot: boolean
  screenshot_score: number
  route_folder: string
  show_in_all: boolean
  is_user_modified: boolean
  rotation: number
  rating: number
  is_favorite: boolean
  tags: string
  ocr_text: string
  ai_description: string
  face_count: number
  is_video: boolean
  duration_sec: number
  live_photo_pair: string
  created_at: string
  updated_at: string
}

export interface CatalogStats {
  total_photos: number
  total_videos: number
  total_bytes: number
  ios_count: number
  android_count: number
  camera_count: number
  screenshots: number
  unknown_count: number
  favorites: number
}

export interface TimelineBucket {
  year: number
  month: number
  count: number
}

export interface WatchedFolder {
  id: number
  path: string
  route_override: string
  show_in_all_photos: boolean
  enabled: boolean
  last_scanned_at: string
  created_at: string
}

export interface DeviceMapping {
  id: number
  device_name: string
  custom_folder: string
  created_at: string
}

export interface IndexerProgress {
  status: 'scanning' | 'cataloging' | 'organizing' | 'completed' | 'idle'
  scanned_files: number
  total_files: number
  current_file: string
  percentage: number
  new_photos: number
  screenshots: number
  error_count: number
}

export interface DetectedDriveInfo {
  driveLetter: string
  volumeLabel: string
  driveType: string
  totalGB: number
  freeGB: number
  recommended: boolean
}

export interface CommonFolderInfo {
  name: string
  path: string
  icon: string
}

export interface SetupHubData {
  drives: DetectedDriveInfo[]
  commonFolders: CommonFolderInfo[]
  suggestedDest: string
  hasFastNVMe: boolean
  fastDrive: string
}

export interface InstantSearchChips {
  topPlaces: string[]
  topDevices: string[]
  years: string[]
  categories: string[]
}
