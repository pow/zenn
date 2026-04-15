export type OrgContext = {
  orgId: string
  orgName: string // サイドバー表示用
  orgInterId: number // API リクエストに使う内部 ID
}

export type DisplayContext = { orgName: string }
export type ApiContext = { orgInterId: number }

export type SafeContext =
  | { status: 'active'; display: DisplayContext; api: ApiContext }
  | { status: 'display-only'; display: DisplayContext; api: null }
  | { status: 'unavailable'; display: null; api: null }

export type ErrorClassification = 'transient' | 'permanent'

// ❌ 危険な実装: stale データを丸ごと返す
export async function naiveResolveContext(
  fetchOrg: () => Promise<OrgContext>,
  staleContext: OrgContext | null,
): Promise<OrgContext | null> {
  try {
    return await fetchOrg()
  } catch {
    return staleContext // stale な orgInterId も返してしまう
  }
}

// ✅ 安全な実装: 表示用と API 用を分離する
export async function resolveSafeContext(
  fetchOrg: () => Promise<OrgContext>,
  staleContext: OrgContext | null,
): Promise<SafeContext> {
  try {
    const org = await fetchOrg()
    return {
      status: 'active',
      display: { orgName: org.orgName },
      api: { orgInterId: org.orgInterId },
    }
  } catch {
    if (staleContext) {
      return {
        status: 'display-only',
        display: { orgName: staleContext.orgName },
        api: null, // API 用データは返さない
      }
    }
    return { status: 'unavailable', display: null, api: null }
  }
}

// ✅ エラーの種類で戦略を変える実装
export async function resolveContextWithStrategy(
  fetchOrg: () => Promise<OrgContext>,
  staleContext: OrgContext | null,
  classifyError: (error: unknown) => ErrorClassification,
): Promise<SafeContext> {
  try {
    const org = await fetchOrg()
    return {
      status: 'active',
      display: { orgName: org.orgName },
      api: { orgInterId: org.orgInterId },
    }
  } catch (error) {
    const kind = classifyError(error)

    if (kind === 'transient' && staleContext) {
      // 一時的障害 → stale データを全部使う（すぐ復旧する想定）
      return {
        status: 'active',
        display: { orgName: staleContext.orgName },
        api: { orgInterId: staleContext.orgInterId },
      }
    }
    if (staleContext) {
      // 恒久的無効化 → 表示用だけ返す
      return {
        status: 'display-only',
        display: { orgName: staleContext.orgName },
        api: null,
      }
    }
    return { status: 'unavailable', display: null, api: null }
  }
}
