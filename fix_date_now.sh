#!/bin/bash
files=(
  "dashboard/src/app/(dashboard)/groups/page.tsx"
  "dashboard/src/app/(dashboard)/sessions/page.tsx"
  "dashboard/src/app/(dashboard)/projects/page.tsx"
  "dashboard/src/app/(dashboard)/chains/page.tsx"
  "dashboard/src/app/(dashboard)/skills/page.tsx"
  "dashboard/src/app/(dashboard)/api-keys/page.tsx"
  "dashboard/src/app/(dashboard)/webhooks/page.tsx"
)

for file in "${files[@]}"; do
  sed -i 's/|| Date.now()//g' "$file"
done
