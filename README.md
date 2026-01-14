# Rotulador

> Simple and powerful tool to annotate image datasets for classification

Modern web-based annotation tool with a clean UI, keyboard shortcuts, and collaborative features. Built with Go, HTMX, DaisyUI, and TailwindCSS.

## Features

- **Modern UI** - Beautiful interface with DaisyUI and TailwindCSS
- **Keyboard Shortcuts** - Annotate faster with number keys (1-9) and `?` for unsure
- **Dark Mode** - Theme toggle with localStorage persistence
- **Authentication** - Multi-user support with password protection
- **Conditional Tasks** - Create annotation workflows with dependencies
- **Task Types** - Boolean, rotation, and custom classification tasks
- **i18n Support** - Internationalization for multiple languages
- **Responsive** - Works on desktop and mobile devices
- **Fast** - No CGO dependencies, pure Go with SQLite, SQLc to reduce overhead and indirection.

##  Quick Start

### Initialize a Project

```bash
# Create config, database and empty image folder
rotulador folder

# Ingest a folder of messy files to a images folder
rotulador ingest ./messy-folder ./images

```

### Start Annotating

```bash
rotulador folder/config.yaml
```

Then open http://localhost:8080 in your browser!

##  Configuration

There is a ready example in ./examples/test for you to play!
### Task Types

**Built-in types:**
- `boolean` - Yes/No questions
- `rotation` - Detect image rotation/flipping
- Custom - Define your own classes

**Conditional tasks:**
Use the `if` field to create dependent tasks:
```yaml
- id: second_task
  if:
    first_task: "expected_value"
```

### Authentication

Add users in the `auth` section. Passwords must be stored as bcrypt hashes.

To generate a hash, use the `hash-password` command:
```bash
rotulador hash-password 'your_secure_password'
# Output: $2a$10$... (copy this hash into your config)
```

Then, add the user to your `config.yaml`:
```yaml
auth:
  username:
    password: "$2a$10$..."
```

## Architecture

### Stack
- **Backend**: Go templates with HTMX for SPA-like interactions
- **Frontend**: DaisyUI + TailwindCSS with @tailwindcss/typography
- **Templates**: Mold for layout inheritance
- **Database**: SQLc + SQLite (modernc.org/sqlite - pure Go, no CGO)

### Project Structure
```
rotulador/
├── annotation/          # Core annotation logic
│   ├── templates/      # Mold templates
│   │   ├── layouts/    # Base layouts
│   │   └── pages/      # Page templates
│   └── assets/         # Generated CSS
├── cmd/                # CLI commands
│   ├── init.go        # Project initialization
│   ├── annotator.go   # Web server
│   ├── ingest.go      # Image import
│   └── query.go       # Database queries
└── examples/          # Sample projects
```

## Development

### Prerequisites
- Mise

See mise.toml for details on commands

### Progress Tracking
The system automatically tracks:
- Completed annotations
- Uncertain annotations (marked with `?`)
- User attribution
- Annotation order

---

**Made with ❤️ using Go, HTMX, DaisyUI and Claude Code**

> The problem is not using AI, it's not setting up the project to be testable and reviewing its outputs
