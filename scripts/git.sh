#!/bin/bash
set -e

FOLDER="node_modules"
BACKUP_BRANCH="backup-before-remove-node_modules"

# Check clean working directory
if [[ -n $(git status --porcelain) ]]; then
  echo "Please commit or stash your changes before running this script."
  exit 1
fi

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "Creating backup branch: $BACKUP_BRANCH"
git branch -f "$BACKUP_BRANCH"

echo "Removing folder '$FOLDER' from all commits..."
git filter-repo --path "$FOLDER" --invert-paths --force

echo "Done removing '$FOLDER'."

echo "Force pushing cleaned history to origin/$CURRENT_BRANCH"
git push origin "$CURRENT_BRANCH" --force

echo "Cleanup complete!"
echo "Inform your collaborators to reset their local repos:"
echo "  git fetch origin"
echo "  git reset --hard origin/$CURRENT_BRANCH"
