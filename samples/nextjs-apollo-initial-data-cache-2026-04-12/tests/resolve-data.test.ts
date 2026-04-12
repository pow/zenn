import { test } from 'node:test'
import assert from 'node:assert/strict'
import { resolveData, resolveDataWithGuard } from '../src/resolve-data.ts'

// --- resolveData ---

test('resolveData: queryData が存在すれば query ソースを返す', () => {
  const result = resolveData({
    queryData: { title: 'from query' },
    initialData: { title: 'from server' },
  })
  assert.equal(result.source, 'query')
  assert.deepEqual(result.data, { title: 'from query' })
})

test('resolveData: queryData が undefined で initialData があれば initial ソースを返す', () => {
  const result = resolveData({
    queryData: undefined,
    initialData: { title: 'from server' },
  })
  assert.equal(result.source, 'initial')
  assert.deepEqual(result.data, { title: 'from server' })
})

test('resolveData: 両方 undefined なら none を返す', () => {
  const result = resolveData({
    queryData: undefined,
    initialData: undefined,
  })
  assert.equal(result.source, 'none')
})

test('resolveData: queryData が存在すれば initialData より優先される', () => {
  const result = resolveData({
    queryData: { title: 'query wins' },
    initialData: { title: 'initial loses' },
  })
  assert.equal(result.source, 'query')
  assert.deepEqual(result.data, { title: 'query wins' })
})

// --- resolveDataWithGuard ---

test('resolveDataWithGuard: queryData があれば key に関係なく query ソースを返す', () => {
  const result = resolveDataWithGuard({
    queryData: { title: 'from query' },
    initialData: { title: 'from server' },
    initialDataKey: 'case-a',
    currentKey: 'case-b',
  })
  assert.equal(result.source, 'query')
  assert.deepEqual(result.data, { title: 'from query' })
})

test('resolveDataWithGuard: key が一致すれば initialData にフォールバックする', () => {
  const result = resolveDataWithGuard({
    queryData: undefined,
    initialData: { title: 'Case A summary' },
    initialDataKey: 'case-a',
    currentKey: 'case-a',
  })
  assert.equal(result.source, 'initial')
  assert.deepEqual(result.data, { title: 'Case A summary' })
})

test('resolveDataWithGuard: key が不一致なら initialData を使わず none を返す', () => {
  const result = resolveDataWithGuard({
    queryData: undefined,
    initialData: { title: 'Case A summary' },
    initialDataKey: 'case-a',
    currentKey: 'case-b',
  })
  assert.equal(result.source, 'none')
})

test('resolveDataWithGuard: initialData も queryData も undefined なら none を返す', () => {
  const result = resolveDataWithGuard({
    queryData: undefined,
    initialData: undefined,
    initialDataKey: 'case-a',
    currentKey: 'case-a',
  })
  assert.equal(result.source, 'none')
})
