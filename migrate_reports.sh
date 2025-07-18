#!/bin/bash

# Batch replacement script for migrating report syntax
# Run this from the compiler directory

echo "Starting report syntax migration..."

# Function to replace patterns in files
replace_in_files() {
    local pattern="$1"
    local replacement="$2"
    local description="$3"
    
    echo "Replacing $description..."
    find . -name "*.go" -type f -exec grep -l "Reports\.Add.*SetLevel" {} \; | \
    xargs sed -i.bak "s|$pattern|$replacement|g"
}

# Replace SEMANTIC_ERROR
replace_in_files \
    "\.Reports\.Add(\([^)]*\))\.SetLevel(report\.SEMANTIC_ERROR)" \
    ".Reports.AddSemanticError(\1)" \
    "SEMANTIC_ERROR patterns"

# Replace CRITICAL_ERROR  
replace_in_files \
    "\.Reports\.Add(\([^)]*\))\.SetLevel(report\.CRITICAL_ERROR)" \
    ".Reports.AddCriticalError(\1)" \
    "CRITICAL_ERROR patterns"

# Replace SYNTAX_ERROR
replace_in_files \
    "\.Reports\.Add(\([^)]*\))\.SetLevel(report\.SYNTAX_ERROR)" \
    ".Reports.AddSyntaxError(\1)" \
    "SYNTAX_ERROR patterns"

# Replace NORMAL_ERROR
replace_in_files \
    "\.Reports\.Add(\([^)]*\))\.SetLevel(report\.NORMAL_ERROR)" \
    ".Reports.AddError(\1)" \
    "NORMAL_ERROR patterns"

# Replace WARNING
replace_in_files \
    "\.Reports\.Add(\([^)]*\))\.SetLevel(report\.WARNING)" \
    ".Reports.AddWarning(\1)" \
    "WARNING patterns"

# Replace INFO
replace_in_files \
    "\.Reports\.Add(\([^)]*\))\.SetLevel(report\.INFO)" \
    ".Reports.AddInfo(\1)" \
    "INFO patterns"

echo "Migration complete! Check the .bak files for backups."
echo "Run 'find . -name \"*.bak\" -delete' to remove backup files when satisfied."
