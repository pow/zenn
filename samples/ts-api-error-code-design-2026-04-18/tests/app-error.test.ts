import { describe, it } from "node:test"
import assert from "node:assert/strict"
import { AppError } from "../src/app-error.ts"
import { ERR_NOT_FOUND, ERR_USER_EMAIL_TAKEN } from "../src/error-codes.ts"

describe("AppError", () => {
  it("holds code, message, and detail", () => {
    const err = new AppError(ERR_NOT_FOUND, "User not found", "id: user-123")
    assert.equal(err.code, "E0003")
    assert.equal(err.message, "User not found")
    assert.equal(err.detail, "id: user-123")
  })

  it("is an instance of Error", () => {
    const err = new AppError(ERR_NOT_FOUND, "not found")
    assert.ok(err instanceof Error)
    assert.equal(err.name, "AppError")
  })

  it("toJSON includes code, message, and detail", () => {
    const err = new AppError(
      ERR_USER_EMAIL_TAKEN,
      "Email is already registered",
      "email: a@b.com",
    )
    assert.deepEqual(err.toJSON(), {
      code: "E0100",
      message: "Email is already registered",
      detail: "email: a@b.com",
    })
  })

  it("toJSON omits detail when undefined", () => {
    const err = new AppError(ERR_NOT_FOUND, "Not found")
    const json = err.toJSON()
    assert.equal(json.code, "E0003")
    assert.equal(json.message, "Not found")
    assert.equal("detail" in json, false)
  })

  it("toJSON does not include stack trace", () => {
    const err = new AppError(ERR_NOT_FOUND, "Not found")
    const json = JSON.parse(JSON.stringify(err))
    assert.equal("stack" in json, false)
  })
})
