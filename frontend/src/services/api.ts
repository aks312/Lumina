import { CatalogStats, PhotoRecord, TimelineBucket, WatchedFolder, DeviceMapping, SetupHubData, InstantSearchChips } from '../types'
import { safeWailsCall } from '../utils/wails'

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetStats: () => Promise<CatalogStats>
          GetPhotos: (tab: string, year: number, month: number, device: string, favoritesOnly: boolean, ratingAtLeast: number, query: string, offset: number, limit: number) => Promise<PhotoRecord[]>
          GetPhotoDetails: (id: number) => Promise<PhotoRecord>
          GetTimelineSummary: (tab: string) => Promise<TimelineBucket[]>
          UpdatePhoto: (id: number, routeFolder: string, platform: string, rotation: number, rating: number, isFavorite: boolean, showInAll: boolean) => Promise<void>
          BatchUpdatePhotos: (ids: number[], routeFolder: string, isFavorite: boolean | null, showInAll: boolean | null) => Promise<void>
          TriggerSmartTagAndOCR: (photoIDs: number[]) => Promise<void>
          GetWatchedFolders: () => Promise<WatchedFolder[]>
          AddWatchedFolder: (path: string, routeOverride: string, showInAll: boolean) => Promise<void>
          RemoveWatchedFolder: (id: number) => Promise<void>
          GetDeviceMappings: () => Promise<DeviceMapping[]>
          SetDeviceMapping: (deviceName: string, customFolder: string) => Promise<void>
          DeleteDeviceMapping: (deviceName: string) => Promise<void>
          GetSetting: (key: string, defaultValue: string) => Promise<string>
          SaveSetting: (key: string, val: string) => Promise<void>
          StartScan: (organize: boolean) => Promise<void>
          StartGoogleSignIn: () => Promise<string>
          SyncToGooglePhotos: () => Promise<number>
          IsGoogleConnected: () => Promise<boolean>
          GlobalReAnalyze: () => Promise<void>
          ReVerifyScreenshots: () => Promise<void>
          PurgeThumbnails: () => Promise<void>
          FactoryReset: () => Promise<void>
          BenchmarkDrives: () => Promise<any>
          SelectDirectoryDialog: () => Promise<string>
          ToggleFullscreen: () => Promise<void>
          GetSetupHubData: () => Promise<SetupHubData>
          GetInstantSearchChips: () => Promise<InstantSearchChips>
        }
      }
    }
    runtime?: {
      EventsOn: (eventName: string, callback: (...args: any[]) => void) => () => void
    }
  }
}

const mockStats: CatalogStats = {
  total_photos: 0,
  total_videos: 0,
  total_bytes: 0,
  ios_count: 0,
  android_count: 0,
  camera_count: 0,
  screenshots: 0,
  unknown_count: 0,
  favorites: 0
}

const mockSetupHubData: SetupHubData = {
  drives: [
    {
      driveLetter: "C:\\",
      volumeLabel: "NVMe SSD",
      driveType: "Fixed SSD/HDD",
      totalGB: 1024,
      freeGB: 420,
      recommended: false
    },
    {
      driveLetter: "D:\\",
      volumeLabel: "Mass Storage",
      driveType: "Fixed SSD/HDD",
      totalGB: 2048,
      freeGB: 1450,
      recommended: true
    }
  ],
  commonFolders: [
    { name: "Pictures", path: "C:\\Users\\User\\Pictures", icon: "pictures" },
    { name: "Desktop", path: "C:\\Users\\User\\Desktop", icon: "desktop" },
    { name: "Downloads", path: "C:\\Users\\User\\Downloads", icon: "downloads" }
  ],
  suggestedDest: "D:\\LuminaPhotos",
  hasFastNVMe: true,
  fastDrive: "C:\\"
}

const mockChips: InstantSearchChips = {
  topPlaces: ["Pune", "Mumbai", "Paris", "New York", "San Francisco"],
  topDevices: ["Apple_iPhone_15_Pro", "Samsung_Galaxy_S23", "Canon_EOS_6D", "Google_Pixel_8_Pro"],
  years: ["2026", "2025", "2024", "2023"],
  categories: [
    "Documents & Receipts",
    "Portraits & People",
    "Videos",
    "Live Photos",
    "High Sharpness"
  ]
}

export const api = {
  getStats: async (): Promise<CatalogStats> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetStats ? window.go.main.App.GetStats() : Promise.resolve(mockStats)),
      mockStats
    )
  },
  getPhotos: async (tab: string, year = 0, month = 0, device = '', favoritesOnly = false, ratingAtLeast = 0, query = '', offset = 0, limit = 100): Promise<PhotoRecord[]> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetPhotos ? window.go.main.App.GetPhotos(tab, year, month, device, favoritesOnly, ratingAtLeast, query, offset, limit) : Promise.resolve([])),
      []
    )
  },
  getPhotoDetails: async (id: number): Promise<PhotoRecord | null> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetPhotoDetails ? window.go.main.App.GetPhotoDetails(id) : Promise.resolve(null)),
      null
    )
  },
  getTimelineSummary: async (tab: string): Promise<TimelineBucket[]> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetTimelineSummary ? window.go.main.App.GetTimelineSummary(tab) : Promise.resolve([])),
      []
    )
  },
  updatePhoto: async (id: number, routeFolder: string, platform: string, rotation: number, rating: number, isFavorite: boolean, showInAll: boolean) => {
    if (window.go?.main?.App?.UpdatePhoto) {
      await window.go.main.App.UpdatePhoto(id, routeFolder, platform, rotation, rating, isFavorite, showInAll)
    }
  },
  batchUpdatePhotos: async (ids: number[], routeFolder: string, isFavorite: boolean | null, showInAll: boolean | null) => {
    if (window.go?.main?.App?.BatchUpdatePhotos) {
      await window.go.main.App.BatchUpdatePhotos(ids, routeFolder, isFavorite, showInAll)
    }
  },
  triggerSmartTagAndOCR: async (photoIDs: number[]) => {
    if (window.go?.main?.App?.TriggerSmartTagAndOCR) {
      await window.go.main.App.TriggerSmartTagAndOCR(photoIDs)
    }
  },
  getWatchedFolders: async (): Promise<WatchedFolder[]> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetWatchedFolders ? window.go.main.App.GetWatchedFolders() : Promise.resolve([])),
      []
    )
  },
  addWatchedFolder: async (path: string, routeOverride: string, showInAll: boolean) => {
    if (window.go?.main?.App?.AddWatchedFolder) {
      await window.go.main.App.AddWatchedFolder(path, routeOverride, showInAll)
    }
  },
  removeWatchedFolder: async (id: number) => {
    if (window.go?.main?.App?.RemoveWatchedFolder) {
      await window.go.main.App.RemoveWatchedFolder(id)
    }
  },
  getDeviceMappings: async (): Promise<DeviceMapping[]> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetDeviceMappings ? window.go.main.App.GetDeviceMappings() : Promise.resolve([])),
      []
    )
  },
  setDeviceMapping: async (deviceName: string, customFolder: string) => {
    if (window.go?.main?.App?.SetDeviceMapping) {
      await window.go.main.App.SetDeviceMapping(deviceName, customFolder)
    }
  },
  deleteDeviceMapping: async (deviceName: string) => {
    if (window.go?.main?.App?.DeleteDeviceMapping) {
      await window.go.main.App.DeleteDeviceMapping(deviceName)
    }
  },
  getSetting: async (key: string, defaultValue = ''): Promise<string> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetSetting ? window.go.main.App.GetSetting(key, defaultValue) : Promise.resolve(defaultValue)),
      defaultValue
    )
  },
  saveSetting: async (key: string, val: string) => {
    if (window.go?.main?.App?.SaveSetting) {
      await window.go.main.App.SaveSetting(key, val)
    }
  },
  startScan: async (organize = false) => {
    if (window.go?.main?.App?.StartScan) {
      await window.go.main.App.StartScan(organize)
    }
  },
  startGoogleSignIn: async () => {
    if (window.go?.main?.App?.StartGoogleSignIn) return await window.go.main.App.StartGoogleSignIn()
    return ''
  },
  syncToGooglePhotos: async () => {
    if (window.go?.main?.App?.SyncToGooglePhotos) return await window.go.main.App.SyncToGooglePhotos()
    return 0
  },
  isGoogleConnected: async (): Promise<boolean> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.IsGoogleConnected ? window.go.main.App.IsGoogleConnected() : Promise.resolve(false)),
      false
    )
  },
  globalReAnalyze: async () => {
    if (window.go?.main?.App?.GlobalReAnalyze) await window.go.main.App.GlobalReAnalyze()
  },
  reVerifyScreenshots: async () => {
    if (window.go?.main?.App?.ReVerifyScreenshots) await window.go.main.App.ReVerifyScreenshots()
  },
  purgeThumbnails: async () => {
    if (window.go?.main?.App?.PurgeThumbnails) await window.go.main.App.PurgeThumbnails()
  },
  factoryReset: async () => {
    if (window.go?.main?.App?.FactoryReset) await window.go.main.App.FactoryReset()
  },
  benchmarkDrives: async () => {
    if (window.go?.main?.App?.BenchmarkDrives) return await window.go.main.App.BenchmarkDrives()
    return null
  },
  selectDirectoryDialog: async (): Promise<string> => {
    if (window.go?.main?.App?.SelectDirectoryDialog) return await window.go.main.App.SelectDirectoryDialog()
    return ''
  },
  toggleFullscreen: async () => {
    if (window.go?.main?.App?.ToggleFullscreen) await window.go.main.App.ToggleFullscreen()
  },
  getSetupHubData: async (): Promise<SetupHubData> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetSetupHubData ? window.go.main.App.GetSetupHubData() : Promise.resolve(mockSetupHubData)),
      mockSetupHubData
    )
  },
  getInstantSearchChips: async (): Promise<InstantSearchChips> => {
    return safeWailsCall(
      () => (window.go?.main?.App?.GetInstantSearchChips ? window.go.main.App.GetInstantSearchChips() : Promise.resolve(mockChips)),
      mockChips
    )
  },
  onEvent: (eventName: string, cb: (...args: any[]) => void) => {
    if (window.runtime?.EventsOn) {
      return window.runtime.EventsOn(eventName, cb)
    }
    return () => {}
  }
}
