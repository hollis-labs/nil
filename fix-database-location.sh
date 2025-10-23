#!/bin/bash

# This script fixes the nested directory issue

ICLOUD_BASE="/Users/chrispian/Library/Mobile Documents/com~apple~CloudDocs/planck-todo-data"

echo "Stopping any running instances of the app..."
# You should quit the app manually before running this

echo ""
echo "Current structure:"
ls -lh "$ICLOUD_BASE"
echo ""
ls -lh "$ICLOUD_BASE/data"

echo ""
echo "Moving files from nested data/ to root..."
# Remove the empty files at root
rm -f "$ICLOUD_BASE/todo.db"
rm -f "$ICLOUD_BASE/todo.db-shm"
rm -f "$ICLOUD_BASE/todo.db-wal"

# Move the real files up
mv "$ICLOUD_BASE/data/todo.db" "$ICLOUD_BASE/"
mv "$ICLOUD_BASE/data/todo.db-shm" "$ICLOUD_BASE/"
mv "$ICLOUD_BASE/data/todo.db-wal" "$ICLOUD_BASE/"

# Remove the empty nested directory
rmdir "$ICLOUD_BASE/data"

echo ""
echo "Fixed! New structure:"
ls -lh "$ICLOUD_BASE"

echo ""
echo "You can now restart the app!"
