// 共通エラー: E0001–E0006
export const ERR_UNAUTHORIZED = "E0001" as const
export const ERR_FORBIDDEN = "E0002" as const
export const ERR_NOT_FOUND = "E0003" as const
export const ERR_CONFLICT = "E0004" as const
export const ERR_VALIDATION = "E0005" as const
export const ERR_INTERNAL = "E0006" as const

// ユーザーサービス: E0100–E0199
export const ERR_USER_EMAIL_TAKEN = "E0100" as const
export const ERR_USER_DEACTIVATED = "E0101" as const

// 注文サービス: E0200–E0299
export const ERR_ORDER_ALREADY_SHIPPED = "E0200" as const
export const ERR_ORDER_ITEM_OUT_OF_STOCK = "E0201" as const

export type ErrorCode =
  | typeof ERR_UNAUTHORIZED
  | typeof ERR_FORBIDDEN
  | typeof ERR_NOT_FOUND
  | typeof ERR_CONFLICT
  | typeof ERR_VALIDATION
  | typeof ERR_INTERNAL
  | typeof ERR_USER_EMAIL_TAKEN
  | typeof ERR_USER_DEACTIVATED
  | typeof ERR_ORDER_ALREADY_SHIPPED
  | typeof ERR_ORDER_ITEM_OUT_OF_STOCK
