import { test } from 'node:test'
import assert from 'node:assert/strict'
import { retry } from '../src/retry.ts'

test('initial call succeeds — no retry performed', async () => {
  let calls = 0
  const result = await retry(async () => {
    calls += 1
    return 'ok'
  })
  assert.equal(result, 'ok')
  assert.equal(calls, 1)
})

test('retries on failure and eventually succeeds', async () => {
  let calls = 0
  const sleepCalls: number[] = []
  const result = await retry(
    async () => {
      calls += 1
      if (calls < 3) throw new Error('transient')
      return 'recovered'
    },
    {
      maxAttempts: 5,
      sleep: async ms => {
        sleepCalls.push(ms)
      },
    },
  )
  assert.equal(result, 'recovered')
  assert.equal(calls, 3)
  assert.equal(sleepCalls.length, 2)
})

test('gives up after maxAttempts and throws last error', async () => {
  let calls = 0
  await assert.rejects(
    retry(
      async () => {
        calls += 1
        throw new Error(`fail-${calls}`)
      },
      {
        maxAttempts: 3,
        sleep: async () => {},
      },
    ),
    /fail-3/,
  )
  assert.equal(calls, 3)
})

test('retryIf=false stops retrying immediately', async () => {
  let calls = 0
  await assert.rejects(
    retry(
      async () => {
        calls += 1
        throw new Error('validation error')
      },
      {
        maxAttempts: 5,
        retryIf: err => !(err instanceof Error && err.message.includes('validation')),
        sleep: async () => {},
      },
    ),
    /validation/,
  )
  assert.equal(calls, 1)
})

test('exponential backoff grows and is capped by maxDelayMs (no jitter)', async () => {
  const sleepCalls: number[] = []
  await assert.rejects(
    retry(
      async () => {
        throw new Error('always')
      },
      {
        maxAttempts: 5,
        initialDelayMs: 100,
        maxDelayMs: 500,
        jitter: false,
        sleep: async ms => {
          sleepCalls.push(ms)
        },
      },
    ),
  )
  assert.deepEqual(sleepCalls, [100, 200, 400, 500])
})

test('jitter scales the delay by random() in [0, exp)', async () => {
  const sleepCalls: number[] = []
  const fixedRandom = () => 0.5
  await assert.rejects(
    retry(
      async () => {
        throw new Error('always')
      },
      {
        maxAttempts: 3,
        initialDelayMs: 200,
        maxDelayMs: 10_000,
        jitter: true,
        random: fixedRandom,
        sleep: async ms => {
          sleepCalls.push(ms)
        },
      },
    ),
  )
  assert.deepEqual(sleepCalls, [100, 200])
})
