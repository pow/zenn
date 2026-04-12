export type DataSource<T> =
  | { source: 'query'; data: T }
  | { source: 'initial'; data: T }
  | { source: 'none' }

export function resolveData<T>(params: {
  queryData: T | undefined
  initialData: T | undefined
}): DataSource<T> {
  const { queryData, initialData } = params
  if (queryData !== undefined) return { source: 'query', data: queryData }
  if (initialData !== undefined) return { source: 'initial', data: initialData }
  return { source: 'none' }
}

export function resolveDataWithGuard<T>(params: {
  queryData: T | undefined
  initialData: T | undefined
  initialDataKey: string
  currentKey: string
}): DataSource<T> {
  const { queryData, initialData, initialDataKey, currentKey } = params
  if (queryData !== undefined) return { source: 'query', data: queryData }
  if (initialData !== undefined && initialDataKey === currentKey) {
    return { source: 'initial', data: initialData }
  }
  return { source: 'none' }
}
