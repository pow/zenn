import { describe, it } from "node:test"
import assert from "node:assert/strict"
import { toHttpStatus } from "../src/http-status.ts"
import {
  ERR_UNAUTHORIZED,
  ERR_FORBIDDEN,
  ERR_NOT_FOUND,
  ERR_CONFLICT,
  ERR_VALIDATION,
  ERR_INTERNAL,
  ERR_USER_EMAIL_TAKEN,
  ERR_USER_DEACTIVATED,
  ERR_ORDER_ALREADY_SHIPPED,
  ERR_ORDER_ITEM_OUT_OF_STOCK,
} from "../src/error-codes.ts"

describe("toHttpStatus", () => {
  it("ERR_UNAUTHORIZED returns 401", () => {
    assert.equal(toHttpStatus(ERR_UNAUTHORIZED), 401)
  })

  it("ERR_FORBIDDEN returns 403", () => {
    assert.equal(toHttpStatus(ERR_FORBIDDEN), 403)
  })

  it("ERR_NOT_FOUND returns 404", () => {
    assert.equal(toHttpStatus(ERR_NOT_FOUND), 404)
  })

  it("ERR_CONFLICT returns 409", () => {
    assert.equal(toHttpStatus(ERR_CONFLICT), 409)
  })

  it("ERR_VALIDATION returns 400", () => {
    assert.equal(toHttpStatus(ERR_VALIDATION), 400)
  })

  it("ERR_INTERNAL returns 500", () => {
    assert.equal(toHttpStatus(ERR_INTERNAL), 500)
  })

  it("ERR_USER_EMAIL_TAKEN returns 409", () => {
    assert.equal(toHttpStatus(ERR_USER_EMAIL_TAKEN), 409)
  })

  it("ERR_USER_DEACTIVATED returns 403", () => {
    assert.equal(toHttpStatus(ERR_USER_DEACTIVATED), 403)
  })

  it("ERR_ORDER_ALREADY_SHIPPED returns 409", () => {
    assert.equal(toHttpStatus(ERR_ORDER_ALREADY_SHIPPED), 409)
  })

  it("ERR_ORDER_ITEM_OUT_OF_STOCK returns 422", () => {
    assert.equal(toHttpStatus(ERR_ORDER_ITEM_OUT_OF_STOCK), 422)
  })
})
