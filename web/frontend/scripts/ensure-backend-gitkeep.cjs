const fs = require("node:fs")
const path = require("node:path")

const gitkeepPath = path.resolve(__dirname, "../../backend/dist/.gitkeep")
const gitkeepContents =
  "# Keep the embedded web backend dist directory in version control.\n"

fs.mkdirSync(path.dirname(gitkeepPath), { recursive: true })
fs.writeFileSync(gitkeepPath, gitkeepContents)
