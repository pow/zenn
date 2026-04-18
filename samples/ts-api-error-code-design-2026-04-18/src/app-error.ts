import { type ErrorCode } from "./error-codes.ts"

export class AppError extends Error {
  readonly code: ErrorCode
  readonly detail?: string

  constructor(code: ErrorCode, message: string, detail?: string) {
    super(message)
    this.name = "AppError"
    this.code = code
    this.detail = detail
  }

  toJSON() {
    return {
      code: this.code,
      message: this.message,
      ...(this.detail !== undefined && { detail: this.detail }),
    }
  }
}
