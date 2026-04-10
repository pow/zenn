export interface RetryOptions {
  /** 最大試行回数（初回含む）。デフォルト 3 */
  maxAttempts?: number
  /** 初回の遅延（ミリ秒）。デフォルト 300 */
  initialDelayMs?: number
  /** 遅延の上限（ミリ秒）。デフォルト 3000 */
  maxDelayMs?: number
  /** ジッターを有効化するか。デフォルト true */
  jitter?: boolean
  /** リトライすべきかを判定する関数。デフォルト: すべてリトライ */
  retryIf?: (error: unknown) => boolean
  /** sleep 実装。テスト時に差し替えるためのフック */
  sleep?: (ms: number) => Promise<void>
  /** 乱数生成器。テスト時に差し替えるためのフック。戻り値は [0, 1) */
  random?: () => number
}

const defaultSleep = (ms: number): Promise<void> =>
  new Promise(resolve => setTimeout(resolve, ms))

export async function retry<T>(
  fn: () => Promise<T>,
  options: RetryOptions = {},
): Promise<T> {
  const {
    maxAttempts = 3,
    initialDelayMs = 300,
    maxDelayMs = 3000,
    jitter = true,
    retryIf = () => true,
    sleep = defaultSleep,
    random = Math.random,
  } = options

  let lastError: unknown
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn()
    } catch (error) {
      lastError = error
      const isLastAttempt = attempt >= maxAttempts
      if (isLastAttempt || !retryIf(error)) {
        throw error
      }
      const exp = Math.min(initialDelayMs * 2 ** (attempt - 1), maxDelayMs)
      const delay = jitter ? Math.floor(random() * exp) : exp
      await sleep(delay)
    }
  }
  throw lastError
}
