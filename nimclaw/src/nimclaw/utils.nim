import std/unicode

proc truncate*(s: string, maxLen: int): string =
  let runes = s.toRunes
  if runes.len <= maxLen:
    return s
  if maxLen <= 3:
    return $runes[0 ..< maxLen]
  return $runes[0 ..< maxLen - 3] & "..."
