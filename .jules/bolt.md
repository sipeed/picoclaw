## 2024-05-24 - Single Pass String Iteration for Token Estimation
**Learning:** `utf8.RuneCountInString(msg)` performs a full iteration over the string. If the string needs to be iterated over again (e.g., `for _, r := range msg`) to check specific rune properties, this results in two full passes over the string.
**Action:** Always count the total number of runes manually within the existing `for range` loop when a subsequent full iteration of the string is already required. This essentially halves the execution time.
## 2024-05-25 - Efficient String Building in Loops
**Learning:** In Go, string concatenation (`+=`) in a loop leads to $O(N^2)$ complexity due to immutability. Using `strings.Builder` provides $O(N)$ efficiency. Additionally, `fmt.Fprintf` has overhead due to format string parsing; direct `sb.WriteString` calls are significantly faster.
**Action:** Use `strings.Builder` for building strings in loops and prefer direct `WriteString` calls over `fmt.Fprintf` for maximum performance in hot paths.

## 2024-05-26 - Avoid Unnecessary `strings.ToLower`
**Learning:** Calling `strings.ToLower` on the entire message content allocates a new string and iterates over all runes. This causes measurable GC pressure and latency on hot paths like feature extraction during routing.
**Action:** Use fast paths to bypass `strings.ToLower`. For instance, check if a requisite character (like a dot `.`) exists, or check common casings directly (`DATA:IMAGE` vs `data:image`) before falling back to full case-normalization.
