#!/bin/bash
file="dashboard/src/app/(dashboard)/settings/page.tsx"
sed -i 's/loadNotificationPreferences();/void loadNotificationPreferences();/g' "$file"
