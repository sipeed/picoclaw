import { atomWithStorage } from "jotai/utils"

export const CODE_BLOCK_WRAP_STORAGE_KEY = "picoclaw:code-block-wrap"
export const DEFAULT_CODE_BLOCK_WRAP = false

export const codeBlockWrapAtom = atomWithStorage<boolean>(
  CODE_BLOCK_WRAP_STORAGE_KEY,
  DEFAULT_CODE_BLOCK_WRAP,
  undefined,
  { getOnInit: true },
)
