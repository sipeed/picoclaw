//  @ts-check

/** @type {import('prettier').Config} */
const config = {
  semi: false,
  printWidth: 80,
  tabWidth: 2,
  importOrder: ["<BUILTIN_MODULES>", "<THIRD_PARTY_MODULES>", "^@/", "^[./]"],
  importOrderSeparation: true,
  importOrderSortSpecifiers: true,
  plugins: [
    "@trivago/prettier-plugin-sort-imports",
    "prettier-plugin-tailwindcss",
  ],
}

export default config
