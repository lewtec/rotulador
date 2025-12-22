#!/bin/bash
set -e

# Check if 'codegen' task exists in mise
if mise tasks | awk '{print $1}' | grep -q "^codegen$"; then
    echo "Task 'codegen' found. Running..."
    # Run the task, but don't fail the script if the task fails (allow non-blocking behavior)
    if ! mise run codegen; then
        echo "::warning::Codegen task failed"
        # We don't exit with error here because the requirement is "só avisa no status e segue o jogo"
    fi
else
    echo "Task 'codegen' not found."
fi
