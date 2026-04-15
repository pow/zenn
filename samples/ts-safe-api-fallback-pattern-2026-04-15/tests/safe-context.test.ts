import { describe, test } from 'node:test'
import assert from 'node:assert/strict'
import {
  naiveResolveContext,
  resolveSafeContext,
  resolveContextWithStrategy,
  type OrgContext,
  type ErrorClassification,
} from '../src/safe-context.ts'

const mockOrg: OrgContext = {
  orgId: 'org-1',
  orgName: 'Test Org',
  orgInterId: 42,
}

const fetchSuccess = async (): Promise<OrgContext> => mockOrg
const fetchFail = async (): Promise<OrgContext> => {
  throw new Error('API error')
}

describe('naiveResolveContext', () => {
  test('API 成功時はフレッシュなデータを返す', async () => {
    const result = await naiveResolveContext(fetchSuccess, null)
    assert.deepEqual(result, mockOrg)
  })

  test('API 失敗時は stale データを丸ごと返す（orgInterId を含む）', async () => {
    const result = await naiveResolveContext(fetchFail, mockOrg)
    assert.deepEqual(result, mockOrg) // stale な orgInterId も返る
    assert.equal(result?.orgInterId, 42)
  })
})

describe('resolveSafeContext', () => {
  test('API 成功時は active を返す（display と api の両方あり）', async () => {
    const result = await resolveSafeContext(fetchSuccess, null)
    assert.equal(result.status, 'active')
    assert.deepEqual(result.display, { orgName: 'Test Org' })
    assert.deepEqual(result.api, { orgInterId: 42 })
  })

  test('API 失敗 + stale あり → display-only を返す（api は null）', async () => {
    const result = await resolveSafeContext(fetchFail, mockOrg)
    assert.equal(result.status, 'display-only')
    assert.deepEqual(result.display, { orgName: 'Test Org' })
    assert.equal(result.api, null)
  })

  test('API 失敗 + stale なし → unavailable を返す', async () => {
    const result = await resolveSafeContext(fetchFail, null)
    assert.equal(result.status, 'unavailable')
    assert.equal(result.display, null)
    assert.equal(result.api, null)
  })
})

describe('resolveContextWithStrategy', () => {
  const alwaysTransient = (): ErrorClassification => 'transient'
  const alwaysPermanent = (): ErrorClassification => 'permanent'

  test('一時的障害 + stale あり → active を返す（api を含む）', async () => {
    const result = await resolveContextWithStrategy(
      fetchFail,
      mockOrg,
      alwaysTransient,
    )
    assert.equal(result.status, 'active')
    assert.deepEqual(result.api, { orgInterId: 42 })
  })

  test('恒久的無効化 + stale あり → display-only を返す（api は null）', async () => {
    const result = await resolveContextWithStrategy(
      fetchFail,
      mockOrg,
      alwaysPermanent,
    )
    assert.equal(result.status, 'display-only')
    assert.equal(result.api, null)
    assert.deepEqual(result.display, { orgName: 'Test Org' })
  })

  test('恒久的無効化 + stale なし → unavailable を返す', async () => {
    const result = await resolveContextWithStrategy(
      fetchFail,
      null,
      alwaysPermanent,
    )
    assert.equal(result.status, 'unavailable')
    assert.equal(result.display, null)
    assert.equal(result.api, null)
  })
})
