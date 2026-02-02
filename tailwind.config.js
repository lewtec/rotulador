/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./templates/**/*.html",
    "./annotation/templates/**/*.html",
    "./annotation/tmpl/**/*.html",
  ],
  daisyui: {
    themes: ["light", "dark"],
    darkTheme: "dark",
    base: true,
    styled: true,
    utils: true,
  },
};
