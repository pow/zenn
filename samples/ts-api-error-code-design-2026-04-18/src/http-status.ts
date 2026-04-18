import { type ErrorCode } from "./error-codes.ts"

const STATUS_MAP: Record<ErrorCode, number> = {
  E0001: 401,
  E0002: 403,
  E0003: 404,
  E0004: 409,
  E0005: 400,
  E0006: 500,
  E0100: 409,
  E0101: 403,
  E0200: 409,
  E0201: 422,
}

export const toHttpStatus = (code: ErrorCode): number => STATUS_MAP[code]
