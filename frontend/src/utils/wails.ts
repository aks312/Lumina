// Safe Wails runtime helper ensuring frame 1 renders without blocking or blank crashes.

export async function safeWailsCall<T>(call: () => Promise<T>, fallback: T): Promise<T> {
  try {
    const res = await call()
    if (res === null || res === undefined) {
      return fallback
    }
    return res
  } catch (err) {
    console.warn('Wails backend call failed, using safe fallback:', err)
    return fallback
  }
}
