/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./internal/web/templates/**/*.html",
  ],
  daisyui: {
    themes: ["light", "dark"],
    darkTheme: "dark",
    base: true,
    styled: true,
    utils: true,
  },
}
