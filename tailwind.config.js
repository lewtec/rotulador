/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./internal/ui/**/*.{templ,go}",
  ],
  daisyui: {
    themes: ["light", "dark"],
    darkTheme: "dark",
    base: true,
    styled: true,
    utils: true,
  },
}
